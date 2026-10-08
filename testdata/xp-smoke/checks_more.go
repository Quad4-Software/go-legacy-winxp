package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"maps"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func checkTimeZone() error {
	name, offset := time.Now().Zone()
	if issue10RFC3339 == "" {
		return fmt.Errorf("init Format RFC3339 was empty")
	}
	loc := time.Now().Location()
	if loc == nil {
		return fmt.Errorf("nil location")
	}
	fmt.Printf("smoke: zone name=%q offset=%d loc=%q init_name=%q init_offset=%d init_loc=%q init_rfc3339=%s unix=%d\n",
		name, offset, loc.String(), issue10Name, issue10Offset, issue10LocationName, issue10RFC3339, issue10UnixLocal)
	return nil
}

func checkTimeFormat() error {
	now := time.Now()
	rfc3339 := now.Format(time.RFC3339)
	rfc1123 := now.Format(time.RFC1123)
	stamp := now.Format(time.Stamp)
	if rfc3339 == "" || rfc1123 == "" || stamp == "" {
		return fmt.Errorf("empty format")
	}
	if !strings.Contains(rfc3339, "T") && !strings.Contains(rfc3339, "t") {
		return fmt.Errorf("rfc3339=%q", rfc3339)
	}
	return nil
}

func checkTimeParse() error {
	const raw = "2006-01-02T15:04:05Z"
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return err
	}
	if t.UTC().Year() != 2006 || t.UTC().Month() != time.January || t.UTC().Day() != 2 {
		return fmt.Errorf("parsed=%v", t)
	}
	return nil
}

func checkTimeJSON() error {
	type payload struct {
		When time.Time `json:"when"`
	}
	in := payload{When: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	var out payload
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	if !out.When.Equal(in.When) {
		return fmt.Errorf("json time=%v want=%v", out.When, in.When)
	}
	return nil
}

func checkTimeSleep() error {
	start := time.Now()
	time.Sleep(15 * time.Millisecond)
	if time.Since(start) < 10*time.Millisecond {
		return fmt.Errorf("sleep too short")
	}
	return nil
}

func checkTimeTicker() error {
	t := time.NewTicker(20 * time.Millisecond)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("ticker timeout")
	}
}

func checkTimeFixedZone() error {
	loc := time.FixedZone("XPTEST", -5*3600)
	t := time.Date(2026, 1, 2, 12, 0, 0, 0, loc)
	name, offset := t.Zone()
	if name != "XPTEST" || offset != -5*3600 {
		return fmt.Errorf("zone=%s offset=%d", name, offset)
	}
	return nil
}

func checkTimeDate() error {
	t := time.Date(2026, 10, 8, 14, 30, 0, 0, time.UTC)
	if t.Year() != 2026 || t.Month() != time.October || t.Day() != 8 {
		return fmt.Errorf("date=%v", t)
	}
	if t.Unix() <= 0 {
		return fmt.Errorf("unix=%d", t.Unix())
	}
	return nil
}

func checkMkdirTemp() error {
	dir, err := os.MkdirTemp("", "go-legacy-winxp-mkdirtemp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func checkCreateTemp() error {
	f, err := os.CreateTemp("", "go-legacy-winxp-create-*.txt")
	if err != nil {
		return err
	}
	name := f.Name()
	if _, err := f.WriteString("tmp"); err != nil {
		f.Close()
		os.Remove(name)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return err
	}
	data, err := os.ReadFile(name)
	os.Remove(name)
	if err != nil {
		return err
	}
	if string(data) != "tmp" {
		return fmt.Errorf("contents=%q", data)
	}
	return nil
}

func checkUserHomeDir() error {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		return fmt.Errorf("home: %w", err)
	}
	return nil
}

func checkUserCacheDir() error {
	dir, err := os.UserCacheDir()
	if err != nil {
		if strings.Contains(err.Error(), "LocalAppData") {
			fmt.Printf("smoke: UserCacheDir not set (%v)\n", err)
			return nil
		}
		return fmt.Errorf("cache: %w", err)
	}
	if dir == "" {
		return fmt.Errorf("empty cache dir")
	}
	return nil
}

func checkExecutable() error {
	path, err := os.Executable()
	if err != nil || path == "" {
		return fmt.Errorf("executable: %w", err)
	}
	return nil
}

func checkChdir() error {
	orig, err := os.Getwd()
	if err != nil {
		return err
	}
	tmp := os.TempDir()
	if err := os.Chdir(tmp); err != nil {
		return err
	}
	defer os.Chdir(orig)
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if wd == "" {
		return fmt.Errorf("empty wd after chdir")
	}
	return nil
}

func checkTruncate(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-truncate.bin")
	if err := os.WriteFile(path, []byte("abcdef"), 0o644); err != nil {
		return err
	}
	defer os.Remove(path)
	if err := os.Truncate(path, 3); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(data) != "abc" {
		return fmt.Errorf("truncated=%q", data)
	}
	return nil
}

func checkSHA256() error {
	sum := sha256.Sum256([]byte("go-legacy-winxp"))
	if sum == ([32]byte{}) {
		return fmt.Errorf("zero sha256")
	}
	mac := hmac.New(sha256.New, []byte("key"))
	mac.Write([]byte("msg"))
	if len(mac.Sum(nil)) != 32 {
		return fmt.Errorf("hmac size")
	}
	return nil
}

func checkBase64() error {
	enc := base64.StdEncoding.EncodeToString([]byte("xp"))
	dec, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return err
	}
	if string(dec) != "xp" {
		return fmt.Errorf("base64=%q", dec)
	}
	return nil
}

func checkHex() error {
	enc := hex.EncodeToString([]byte{0xab, 0xcd})
	if enc != "abcd" {
		return fmt.Errorf("hex=%q", enc)
	}
	return nil
}

func checkBinary() error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0x01020304)
	if binary.LittleEndian.Uint32(buf) != 0x01020304 {
		return fmt.Errorf("binary roundtrip")
	}
	return nil
}

func checkXML() error {
	type row struct {
		XMLName xml.Name `xml:"row"`
		Name    string   `xml:"name"`
	}
	in := row{Name: "xp"}
	data, err := xml.Marshal(in)
	if err != nil {
		return err
	}
	var out row
	if err := xml.Unmarshal(data, &out); err != nil {
		return err
	}
	if out.Name != in.Name {
		return fmt.Errorf("xml=%q", out.Name)
	}
	return nil
}

func checkGzip() error {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte("gzip-xp")); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	r, err := gzip.NewReader(&buf)
	if err != nil {
		return err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if string(data) != "gzip-xp" {
		return fmt.Errorf("gzip=%q", data)
	}
	return nil
}

func checkStringsBuilder() error {
	var b strings.Builder
	b.WriteString("go")
	b.WriteString("-xp")
	if b.String() != "go-xp" {
		return fmt.Errorf("builder=%q", b.String())
	}
	return nil
}

func checkBytesBuffer() error {
	var b bytes.Buffer
	b.WriteString("buf")
	if b.String() != "buf" {
		return fmt.Errorf("buffer=%q", b.String())
	}
	return nil
}

func checkStrconv() error {
	n, err := strconv.Atoi("386")
	if err != nil || n != 386 {
		return fmt.Errorf("atoi=%d %v", n, err)
	}
	if strconv.Itoa(27) != "27" {
		return fmt.Errorf("itoa")
	}
	return nil
}

func checkRegexp() error {
	re := regexp.MustCompile(`go-legacy-winxp`)
	if !re.MatchString("prefix go-legacy-winxp suffix") {
		return fmt.Errorf("regexp no match")
	}
	return nil
}

func checkSort() error {
	a := []int{3, 1, 2}
	sort.Ints(a)
	if a[0] != 1 || a[2] != 3 {
		return fmt.Errorf("sort=%v", a)
	}
	return nil
}

func checkSlices() error {
	a := []int{3, 1, 2}
	slices.Sort(a)
	if !slices.Equal(a, []int{1, 2, 3}) {
		return fmt.Errorf("slices=%v", a)
	}
	return nil
}

func checkMaps() error {
	src := map[string]int{"a": 1, "b": 2}
	dst := maps.Clone(src)
	if dst["a"] != 1 || dst["b"] != 2 || len(dst) != 2 {
		return fmt.Errorf("maps=%v", dst)
	}
	return nil
}

func checkCRC32() error {
	sum := crc32.ChecksumIEEE([]byte("xp"))
	if sum == 0 {
		return fmt.Errorf("crc32 zero")
	}
	return nil
}

func checkContextCancel() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	select {
	case <-ctx.Done():
		return nil
	default:
		return fmt.Errorf("context not canceled")
	}
}

func checkSyncOnce() error {
	var once sync.Once
	var n int
	once.Do(func() { n++ })
	once.Do(func() { n++ })
	if n != 1 {
		return fmt.Errorf("once=%d", n)
	}
	return nil
}

func checkRWmutex() error {
	var mu sync.RWMutex
	var n int
	mu.Lock()
	n = 1
	mu.Unlock()
	mu.RLock()
	got := n
	mu.RUnlock()
	if got != 1 {
		return fmt.Errorf("rwmutex=%d", got)
	}
	return nil
}

func checkExecEcho() error {
	out, err := exec.Command("cmd.exe", "/c", "echo", "xp-exec-ok").CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec: %w output=%q", err, out)
	}
	if !strings.Contains(string(out), "xp-exec-ok") {
		return fmt.Errorf("exec output=%q", out)
	}
	return nil
}

func checkFilepathGlob(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-glob.txt")
	if err := os.WriteFile(path, []byte("g"), 0o644); err != nil {
		return err
	}
	defer os.Remove(path)
	matches, err := filepath.Glob(filepath.Join(dir, "go-legacy-winxp-glob.*"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no glob matches")
	}
	return nil
}

func checkArchiveZip(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-smoke.zip")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("hello.txt")
	if err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if _, err := w.Write([]byte("zip-xp")); err != nil {
		zw.Close()
		f.Close()
		os.Remove(path)
		return err
	}
	if err := zw.Close(); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return err
	}
	r, err := zip.OpenReader(path)
	os.Remove(path)
	if err != nil {
		return err
	}
	defer r.Close()
	if len(r.File) != 1 {
		return fmt.Errorf("zip files=%d", len(r.File))
	}
	rc, err := r.File[0].Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	if string(data) != "zip-xp" {
		return fmt.Errorf("zip data=%q", data)
	}
	return nil
}

func checkIOMulti() error {
	var a, b bytes.Buffer
	mw := io.MultiWriter(&a, &b)
	if _, err := mw.Write([]byte("xy")); err != nil {
		return err
	}
	if a.String() != "xy" || b.String() != "xy" {
		return fmt.Errorf("multiwriter a=%q b=%q", a.String(), b.String())
	}
	return nil
}

func checkSyscallTimezone() error {
	var tzi syscall.Timezoneinformation
	if _, err := syscall.GetTimeZoneInformation(&tzi); err != nil {
		return err
	}
	std := syscall.UTF16ToString(tzi.StandardName[:])
	if std == "" {
		return fmt.Errorf("empty timezone name")
	}
	fmt.Printf("smoke: timezone std=%q\n", std)
	return nil
}

func checkLogBytes() error {
	var buf bytes.Buffer
	l := log.New(&buf, "xp ", 0)
	l.Print("ok")
	if !strings.Contains(buf.String(), "ok") {
		return fmt.Errorf("log=%q", buf.String())
	}
	return nil
}

func checkNetLookupIP() error {
	ips, err := net.LookupIP("localhost")
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("lookup ip: %w", err)
	}
	return nil
}
