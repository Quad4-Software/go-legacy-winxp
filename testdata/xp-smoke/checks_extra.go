package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"cmp"
	"compress/flate"
	"compress/zlib"
	"container/heap"
	"container/list"
	"container/ring"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base32"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc64"
	"hash/fnv"
	"html"
	"io"
	"io/fs"
	"maps"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"
)

func extraChecks(passed *int) {
	section("extra_io")
	runCheck("io_pipe", checkIOPipe, passed)
	runCheck("io_tee_limit_section", checkIOTeeLimitSection, passed)
	runCheck("io_copy_n", checkIOCopyN, passed)
	runCheck("bufio_scan_string", checkBufioScanString, passed)
	runCheck("bufio_readstring", checkBufioReadString, passed)

	section("extra_strings_bytes")
	runCheck("strings_fields_split", checkStringsFieldsSplit, passed)
	runCheck("strings_replace_repeat", checkStringsReplaceRepeat, passed)
	runCheck("strings_reader", checkStringsReader, passed)
	runCheck("bytes_equal_compare", checkBytesEqualCompare, passed)
	runCheck("bytes_replace_reader", checkBytesReplaceReader, passed)

	section("extra_strconv_fmt_errors")
	runCheck("strconv_parse_format", checkStrconvParseFormat, passed)
	runCheck("fmt_sprintf_scanf", checkFmtSprintfScanf, passed)
	runCheck("errors_is_as_join", checkErrorsIsAsJoin, passed)

	section("extra_time_context")
	runCheck("time_parse_duration_arith", checkTimeParseDurationArith, passed)
	runCheck("context_value_deadline", checkContextValueDeadline, passed)
	runCheck("context_afterfunc", checkContextAfterFunc, passed)

	section("extra_os")
	dir := os.TempDir()
	runCheck("os_args_tempdir_devnull", checkOSArgsTempdirDevnull, passed)
	runCheck("os_file_seek_readat", func() error { return checkOSFileSeekReadAt(dir) }, passed)
	runCheck("os_fileinfo_stat", func() error { return checkOSFileInfoStat(dir) }, passed)
	runCheck("os_hardlink_samefile", func() error { return checkOSHardlinkSameFile(dir) }, passed)
	runCheck("os_file_sync", func() error { return checkOSFileSync(dir) }, passed)
	runCheck("filepath_extras", checkFilepathExtras, passed)
	runCheck("os_signal_notify", checkOSSignalNotify, passed)
	runCheck("os_signal_notifyctx", checkOSSignalNotifyCtx, passed)

	section("extra_exec")
	runCheck("exec_self_args", checkExecSelfArgs, passed)
	runCheck("exec_env_dir", func() error { return checkExecEnvDir(dir) }, passed)
	runCheck("exec_stdin_stdout", checkExecStdinStdout, passed)
	runCheck("exec_kill", checkExecKill, passed)
	runCheck("exec_wait_delay", checkExecWaitDelay, passed)

	section("extra_syscall")
	runCheck("syscall_utf16", checkSyscallUTF16, passed)
	runCheck("syscall_filetime", checkSyscallFileTime, passed)

	section("extra_net")
	runCheck("net_tcp_deadlines", checkNetTCPDeadlines, passed)
	runCheck("net_udp_listenpacket", checkNetUDPListenPacket, passed)
	runCheck("net_udp_deadline", checkNetUDPDeadline, passed)
	runCheck("net_pipe", checkNetPipe, passed)
	runCheck("net_parseip_cidr", checkNetParseIPCIDR, passed)
	runCheck("netip_parse", checkNetIPParse, passed)
	runCheck("net_url_parse", checkNetURLParse, passed)
	runCheck("net_mail_parse", checkNetMailParse, passed)
	runCheck("net_http_httptest", checkNetHTTPHttptest, passed)
	runCheck("net_http_post_form", checkNetHTTPPostForm, passed)

	section("extra_encoding")
	runCheck("encoding_csv", checkEncodingCSV, passed)
	runCheck("encoding_gob", checkEncodingGob, passed)
	runCheck("encoding_base32", checkEncodingBase32, passed)
	runCheck("encoding_pem", checkEncodingPEM, passed)
	runCheck("encoding_json_stream", checkJSONStream, passed)

	section("extra_crypto_hash")
	runCheck("crypto_md5_sha1_sha512", checkCryptoHashes, passed)
	runCheck("crypto_aes_des", checkCryptoAESDES, passed)
	runCheck("crypto_subtle", checkCryptoSubtle, passed)
	runCheck("crypto_rand_reader", checkCryptoRandReader, passed)
	runCheck("hash_fnv_adler_crc64", checkHashExtras, passed)

	section("extra_compress_archive")
	runCheck("compress_zlib_flate", checkCompressZlibFlate, passed)
	runCheck("archive_tar", checkArchiveTar, passed)

	section("extra_containers")
	runCheck("container_list_ring", checkContainerListRing, passed)
	runCheck("container_heap", checkContainerHeap, passed)

	section("extra_reflect_unicodedata")
	runCheck("reflect_basic", checkReflectBasic, passed)
	runCheck("unicode_utf8", checkUnicodeUTF8, passed)

	section("extra_math")
	runCheck("math_basic", checkMathBasic, passed)
	runCheck("math_big", checkMathBig, passed)

	section("extra_sort_iter_cmp")
	runCheck("sort_search_slice", checkSortSearchSlice, passed)
	runCheck("iter_slices_values", checkIterSlicesValues, passed)
	runCheck("cmp_compare", checkCmpCompare, passed)

	section("extra_template_flag_html")
	runCheck("template_execute", checkTemplateExecute, passed)
	runCheck("flag_parse", checkFlagParse, passed)
	runCheck("html_escape", checkHTMLEscape, passed)

	section("extra_runtime_debug_fs")
	runCheck("runtime_numgoroutine_memstats", checkRuntimeNumGoroutineMemStats, passed)
	runCheck("debug_gcpercent", checkDebugGCPercent, passed)
	runCheck("io_fs_readdir_validpath", func() error { return checkIOFSReadDir(os.TempDir()) }, passed)
}

func checkIOPipe() error {
	r, w := io.Pipe()
	go func() {
		w.Write([]byte("pipe-data"))
		w.Close()
	}()
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if string(data) != "pipe-data" {
		return fmt.Errorf("io.Pipe=%q", data)
	}
	return nil
}

func checkIOTeeLimitSection() error {
	var out bytes.Buffer
	t := io.TeeReader(strings.NewReader("tee-limit"), &out)
	data, err := io.ReadAll(io.LimitReader(t, 3))
	if err != nil {
		return err
	}
	if string(data) != "tee" {
		return fmt.Errorf("limit=%q", data)
	}
	if out.String() != "tee" {
		return fmt.Errorf("tee wrote %q", out.String())
	}
	sr := io.NewSectionReader(strings.NewReader("0123456789"), 2, 4)
	sdata, err := io.ReadAll(sr)
	if err != nil {
		return err
	}
	if string(sdata) != "2345" {
		return fmt.Errorf("section=%q", sdata)
	}
	return nil
}

func checkIOCopyN() error {
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, strings.NewReader("abcdef"), 4)
	if err != nil || n != 4 {
		return fmt.Errorf("copyn n=%d err=%v", n, err)
	}
	if buf.String() != "abcd" {
		return fmt.Errorf("copyn=%q", buf.String())
	}
	return nil
}

func checkBufioScanString() error {
	s := bufio.NewScanner(strings.NewReader("one\ntwo\nthree\n"))
	var got []string
	for s.Scan() {
		got = append(got, s.Text())
	}
	if err := s.Err(); err != nil {
		return err
	}
	if len(got) != 3 || got[1] != "two" {
		return fmt.Errorf("scan=%v", got)
	}
	return nil
}

func checkBufioReadString() error {
	r := bufio.NewReader(strings.NewReader("line1\nline2\n"))
	l1, err := r.ReadString('\n')
	if err != nil || l1 != "line1\n" {
		return fmt.Errorf("readstring1=%q err=%v", l1, err)
	}
	l2, err := r.ReadBytes('\n')
	if err != nil || string(l2) != "line2\n" {
		return fmt.Errorf("readbytes=%q err=%v", l2, err)
	}
	return nil
}

func checkStringsFieldsSplit() error {
	f := strings.Fields("  a  b\tc  ")
	if len(f) != 3 || f[0] != "a" || f[2] != "c" {
		return fmt.Errorf("fields=%v", f)
	}
	p := strings.SplitN("a,b,c,d", ",", 2)
	if len(p) != 2 || p[1] != "b,c,d" {
		return fmt.Errorf("splitn=%v", p)
	}
	if !strings.HasPrefix("prefix-x", "prefix") || !strings.HasSuffix("x-suffix", "suffix") {
		return fmt.Errorf("prefix/suffix")
	}
	if strings.Count("aaa", "a") != 3 || strings.Index("abc", "b") != 1 {
		return fmt.Errorf("count/index")
	}
	if strings.TrimSpace("  x  ") != "x" || strings.TrimPrefix("pre-x", "pre-") != "x" {
		return fmt.Errorf("trim")
	}
	return nil
}

func checkStringsReplaceRepeat() error {
	r := strings.Replace("a-b-c", "-", "+", 2)
	if r != "a+b+c" {
		return fmt.Errorf("replace=%q", r)
	}
	if strings.Repeat("ab", 3) != "ababab" {
		return fmt.Errorf("repeat")
	}
	if strings.ToUpper("ab") != "AB" || strings.ToLower("AB") != "ab" {
		return fmt.Errorf("case")
	}
	if !strings.Contains("hello", "ell") {
		return fmt.Errorf("contains")
	}
	return nil
}

func checkStringsReader() error {
	data, err := io.ReadAll(strings.NewReader("reader-data"))
	if err != nil {
		return err
	}
	if string(data) != "reader-data" {
		return fmt.Errorf("reader=%q", data)
	}
	return nil
}

func checkBytesEqualCompare() error {
	if !bytes.Equal([]byte("x"), []byte("x")) {
		return fmt.Errorf("equal")
	}
	if bytes.Compare([]byte("a"), []byte("b")) >= 0 {
		return fmt.Errorf("compare")
	}
	if bytes.Count([]byte("aaa"), []byte("a")) != 3 {
		return fmt.Errorf("count")
	}
	if bytes.Index([]byte("abc"), []byte("b")) != 1 {
		return fmt.Errorf("index")
	}
	if string(bytes.Repeat([]byte("z"), 2)) != "zz" {
		return fmt.Errorf("repeat")
	}
	return nil
}

func checkBytesReplaceReader() error {
	r := bytes.Replace([]byte("a-b-c"), []byte("-"), []byte("+"), -1)
	if string(r) != "a+b+c" {
		return fmt.Errorf("replace=%q", r)
	}
	data, err := io.ReadAll(bytes.NewReader([]byte("reader")))
	if err != nil || string(data) != "reader" {
		return fmt.Errorf("reader=%q err=%v", data, err)
	}
	return nil
}

func checkStrconvParseFormat() error {
	n, err := strconv.ParseInt("0x10", 0, 32)
	if err != nil || n != 16 {
		return fmt.Errorf("parseint=%d %v", n, err)
	}
	f, err := strconv.ParseFloat("1.5", 64)
	if err != nil || f != 1.5 {
		return fmt.Errorf("parsefloat=%v %v", f, err)
	}
	if strconv.FormatFloat(1.5, 'f', -1, 64) != "1.5" {
		return fmt.Errorf("formatfloat")
	}
	b, err := strconv.ParseBool("true")
	if err != nil || !b {
		return fmt.Errorf("parsebool=%v %v", b, err)
	}
	if strconv.FormatBool(false) != "false" {
		return fmt.Errorf("formatbool")
	}
	if strconv.FormatInt(-42, 10) != "-42" {
		return fmt.Errorf("formatint")
	}
	q := strconv.Quote("a\"b")
	if u, err := strconv.Unquote(q); err != nil || u != "a\"b" {
		return fmt.Errorf("quote roundtrip=%q %v", u, err)
	}
	var buf []byte
	buf = strconv.AppendInt(buf, 7, 10)
	if string(buf) != "7" {
		return fmt.Errorf("appendint=%q", buf)
	}
	return nil
}

func checkFmtSprintfScanf() error {
	s := fmt.Sprintf("%s-%d-%v", "x", 42, true)
	if s != "x-42-true" {
		return fmt.Errorf("sprintf=%q", s)
	}
	var n int
	var str string
	if _, err := fmt.Sscanf("7 abc", "%d %s", &n, &str); err != nil {
		return fmt.Errorf("sscanf: %v", err)
	}
	if n != 7 || str != "abc" {
		return fmt.Errorf("sscanf=%d %q", n, str)
	}
	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "%04d", 9); err != nil {
		return err
	}
	if buf.String() != "0009" {
		return fmt.Errorf("fprintf=%q", buf.String())
	}
	if fmt.Sprintf("%.2f", 1.234) != "1.23" {
		return fmt.Errorf("float fmt")
	}
	return nil
}

type smokeWrapped struct{ msg string }

func (e *smokeWrapped) Error() string { return "wrapped: " + e.msg }
func (e *smokeWrapped) Unwrap() error { return errors.New(e.msg) }

func checkErrorsIsAsJoin() error {
	sentinel := errors.New("sentinel")
	err := fmt.Errorf("outer: %w", sentinel)
	if !errors.Is(err, sentinel) {
		return fmt.Errorf("errors.Is failed")
	}
	var w *smokeWrapped
	if !errors.As(fmt.Errorf("x: %w", &smokeWrapped{msg: "m"}), &w) {
		return fmt.Errorf("errors.As failed")
	}
	j := errors.Join(errors.New("a"), errors.New("b"))
	if j == nil || !strings.Contains(j.Error(), "a") || !strings.Contains(j.Error(), "b") {
		return fmt.Errorf("join=%v", j)
	}
	if errors.Unwrap(sentinel) != nil {
		return fmt.Errorf("unexpected unwrap")
	}
	return nil
}

func checkTimeParseDurationArith() error {
	d, err := time.ParseDuration("1h30m")
	if err != nil || d != 90*time.Minute {
		return fmt.Errorf("duration=%v %v", d, err)
	}
	now := time.Now()
	later := now.Add(2 * time.Hour)
	if !later.After(now) || later.Sub(now) < time.Hour {
		return fmt.Errorf("add/sub")
	}
	if !now.Before(later) || !later.After(now) || !now.Equal(now) {
		return fmt.Errorf("compare")
	}
	if now.UTC().Location() != time.UTC {
		return fmt.Errorf("utc")
	}
	r := now.Round(time.Second)
	if r.Sub(now).Abs() > time.Second {
		return fmt.Errorf("round")
	}
	tr := now.Truncate(time.Minute)
	if tr.After(now) {
		return fmt.Errorf("truncate")
	}
	if time.Until(later) <= 0 {
		return fmt.Errorf("until")
	}
	if time.Since(now) < 0 {
		return fmt.Errorf("since")
	}
	if now.UnixMilli() <= 0 || now.UnixMicro() <= 0 || now.UnixNano() <= 0 {
		return fmt.Errorf("unix variants")
	}
	if _, err := time.ParseDuration("bogus"); err == nil {
		return fmt.Errorf("bad duration accepted")
	}
	return nil
}

func checkContextValueDeadline() error {
	type keyT struct{}
	ctx := context.WithValue(context.Background(), keyT{}, "v")
	if ctx.Value(keyT{}) != "v" {
		return fmt.Errorf("value")
	}
	dctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	select {
	case <-dctx.Done():
	case <-time.After(time.Second):
		return fmt.Errorf("deadline ctx not done")
	}
	if dctx.Err() != context.DeadlineExceeded {
		return fmt.Errorf("err=%v", dctx.Err())
	}
	return nil
}

func checkContextAfterFunc() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fired := make(chan struct{}, 1)
	stop := context.AfterFunc(ctx, func() { fired <- struct{}{} })
	cancel()
	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		return fmt.Errorf("afterfunc did not run")
	}
	if stop != nil && !stop() {
		// stop reports false when the function already ran or is running.
	}
	return nil
}

func checkOSArgsTempdirDevnull() error {
	if len(os.Args) < 1 || os.Args[0] == "" {
		return fmt.Errorf("args empty")
	}
	if os.TempDir() == "" {
		return fmt.Errorf("tempdir empty")
	}
	if os.DevNull == "" {
		return fmt.Errorf("devnull empty")
	}
	return nil
}

func checkOSFileSeekReadAt(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-seek.bin")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if _, err := f.Write([]byte("0123456789")); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Seek(2, io.SeekStart); err != nil {
		f.Close()
		return err
	}
	buf := make([]byte, 3)
	if _, err := io.ReadFull(f, buf); err != nil {
		f.Close()
		return err
	}
	if string(buf) != "234" {
		f.Close()
		return fmt.Errorf("seek read=%q", buf)
	}
	if err := f.Close(); err != nil {
		return err
	}
	f, err = os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.ReadAt(buf, 4); err != nil {
		return err
	}
	if string(buf) != "456" {
		return fmt.Errorf("readat=%q", buf)
	}
	if _, err := f.WriteAt([]byte("XY"), 0); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	end, err := f.Seek(0, io.SeekCurrent)
	if err != nil || end != 10 {
		return fmt.Errorf("seek end=%d err=%v", end, err)
	}
	return nil
}

func checkOSFileInfoStat(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-stat.txt")
	content := []byte("stat-data")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return err
	}
	defer os.Remove(path)
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Name() != "go-legacy-winxp-stat.txt" {
		return fmt.Errorf("name=%q", fi.Name())
	}
	if fi.Size() != int64(len(content)) {
		return fmt.Errorf("size=%d", fi.Size())
	}
	if fi.IsDir() {
		return fmt.Errorf("isdir")
	}
	if fi.Mode().IsRegular() == false {
		return fmt.Errorf("mode=%v", fi.Mode())
	}
	if fi.ModTime().Unix() <= 0 {
		return fmt.Errorf("modtime")
	}
	if fi.Sys() == nil {
		return fmt.Errorf("sys nil")
	}
	return nil
}

func checkOSHardlinkSameFile(dir string) error {
	a := filepath.Join(dir, "go-legacy-winxp-link-a.txt")
	b := filepath.Join(dir, "go-legacy-winxp-link-b.txt")
	c := filepath.Join(dir, "go-legacy-winxp-link-c.txt")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		return err
	}
	defer os.Remove(a)
	defer os.Remove(b)
	defer os.Remove(c)
	if err := os.Link(a, b); err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if err := os.WriteFile(c, []byte("y"), 0o644); err != nil {
		return err
	}
	fa, err := os.Stat(a)
	if err != nil {
		return err
	}
	fb, err := os.Stat(b)
	if err != nil {
		return err
	}
	fc, err := os.Stat(c)
	if err != nil {
		return err
	}
	if !os.SameFile(fa, fb) {
		return fmt.Errorf("hardlink not SameFile")
	}
	if os.SameFile(fa, fc) {
		return fmt.Errorf("different files SameFile")
	}
	return nil
}

func checkOSFileSync(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-sync.bin")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if _, err := f.Write([]byte("sync")); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func checkFilepathExtras() error {
	if filepath.VolumeName(`C:\Windows\x`) != "C:" {
		return fmt.Errorf("volume=%q", filepath.VolumeName(`C:\Windows\x`))
	}
	if !filepath.IsAbs(`C:\Windows`) {
		return fmt.Errorf("isabs")
	}
	if filepath.Ext("a.txt") != ".txt" {
		return fmt.Errorf("ext")
	}
	if filepath.Base(`C:\a\b.txt`) != "b.txt" {
		return fmt.Errorf("base")
	}
	if filepath.Dir(`C:\a\b.txt`) != `C:\a` {
		return fmt.Errorf("dir=%q", filepath.Dir(`C:\a\b.txt`))
	}
	rel, err := filepath.Rel(`C:\a`, `C:\a\b\c.txt`)
	if err != nil || rel != filepath.Join("b", "c.txt") {
		return fmt.Errorf("rel=%q err=%v", rel, err)
	}
	return nil
}

func checkOSSignalNotify() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Stop(c)
	select {
	case <-c:
		return fmt.Errorf("unexpected signal")
	default:
	}
	return nil
}

func checkOSSignalNotifyCtx() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("ctx done without signal")
	case <-time.After(50 * time.Millisecond):
	}
	return nil
}

func smokeExe() (string, error) {
	return os.Executable()
}

func checkExecSelfArgs() error {
	exe, err := smokeExe()
	if err != nil {
		return err
	}
	out, err := exec.Command(exe, "__helper_args").CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec: %w output=%q", err, out)
	}
	if !strings.Contains(string(out), "argc=2") || !strings.Contains(string(out), "arg1=__helper_args") {
		return fmt.Errorf("args output=%q", out)
	}
	return nil
}

func checkExecEnvDir(dir string) error {
	exe, err := smokeExe()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__helper_env")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XP_SMOKE_ENV=env-ok")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec: %w output=%q", err, out)
	}
	if strings.TrimSpace(string(out)) != "env-ok" {
		return fmt.Errorf("env output=%q", out)
	}
	return nil
}

func checkExecStdinStdout() error {
	exe, err := smokeExe()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__helper_echo")
	cmd.Stdin = strings.NewReader("stdin-data")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	if string(out) != "stdin-data" {
		return fmt.Errorf("stdout=%q", out)
	}
	return nil
}

func checkExecKill() error {
	exe, err := smokeExe()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__helper_sleep", "60")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		cmd.Wait()
		return fmt.Errorf("kill: %w", err)
	}
	err = cmd.Wait()
	if err == nil {
		return fmt.Errorf("wait after kill returned nil")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() == 0 {
		return fmt.Errorf("wait err=%v", err)
	}
	return nil
}

func checkExecWaitDelay() error {
	exe, err := smokeExe()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "__helper_sleep", "30")
	cmd.WaitDelay = 500 * time.Millisecond
	start := time.Now()
	err = cmd.Run()
	if err == nil {
		return fmt.Errorf("expected context kill error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		return fmt.Errorf("waitdelay bound=%v", elapsed)
	}
	return nil
}

func checkSyscallUTF16() error {
	u16, err := syscall.UTF16FromString("xp-utf16")
	if err != nil {
		return err
	}
	if syscall.UTF16ToString(u16) != "xp-utf16" {
		return fmt.Errorf("utf16 roundtrip")
	}
	p, err := syscall.UTF16PtrFromString("ptr")
	if err != nil || p == nil {
		return fmt.Errorf("utf16ptr: %v", err)
	}
	return nil
}

func checkSyscallFileTime() error {
	var ft syscall.Filetime
	syscall.GetSystemTimeAsFileTime(&ft)
	if ft.LowDateTime == 0 && ft.HighDateTime == 0 {
		return fmt.Errorf("zero filetime")
	}
	return nil
}

func checkNetTCPDeadlines() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		buf := make([]byte, 1)
		// Read with a deadline already in the past must time out.
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetReadDeadline(time.Now().Add(-time.Second))
		}
		_, err = conn.Read(buf)
		if err == nil {
			done <- fmt.Errorf("read past deadline succeeded")
			return
		}
		if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
			done <- fmt.Errorf("read err=%v not timeout", err)
			return
		}
		done <- nil
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		return err
	}
	defer client.Close()
	// The server side times out on its own; keep the client open so the
	// accept cannot fail first.
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("deadline read did not finish")
	}
}

func checkNetUDPListenPacket() error {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer pc.Close()
	if _, err := pc.WriteTo([]byte("pkt"), pc.LocalAddr()); err != nil {
		return err
	}
	buf := make([]byte, 8)
	pc.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "pkt" {
		return fmt.Errorf("packet=%q", buf[:n])
	}
	return nil
}

func checkNetUDPDeadline() error {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(-time.Second))
	buf := make([]byte, 4)
	_, _, err = conn.ReadFromUDP(buf)
	if err == nil {
		return fmt.Errorf("past-deadline read succeeded")
	}
	if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
		return fmt.Errorf("err=%v not timeout", err)
	}
	return nil
}

func checkNetPipe() error {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	go func() {
		c2.Write([]byte("np"))
	}()
	buf := make([]byte, 2)
	if _, err := io.ReadFull(c1, buf); err != nil {
		return err
	}
	if string(buf) != "np" {
		return fmt.Errorf("pipe=%q", buf)
	}
	return nil
}

func checkNetParseIPCIDR() error {
	ip := net.ParseIP("127.0.0.1")
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("parseip")
	}
	if !ip.IsLoopback() {
		return fmt.Errorf("loopback")
	}
	_, ipnet, err := net.ParseCIDR("192.168.0.0/24")
	if err != nil {
		return err
	}
	if !ipnet.Contains(net.ParseIP("192.168.0.7")) {
		return fmt.Errorf("cidr contains")
	}
	if ipnet.Contains(net.ParseIP("10.0.0.1")) {
		return fmt.Errorf("cidr false positive")
	}
	if net.IPv4(127, 0, 0, 1).To16() == nil {
		return fmt.Errorf("ipv4 to16")
	}
	mask := net.CIDRMask(24, 32)
	if mask == nil {
		return fmt.Errorf("cidrmask nil")
	}
	ones, bits := mask.Size()
	if ones != 24 || bits != 32 {
		return fmt.Errorf("cidrmask=%d/%d", ones, bits)
	}
	return nil
}

func checkNetIPParse() error {
	a, err := netip.ParseAddr("127.0.0.1")
	if err != nil {
		return err
	}
	if !a.IsLoopback() || !a.Is4() {
		return fmt.Errorf("addr=%v", a)
	}
	p, err := netip.ParsePrefix("10.0.0.0/8")
	if err != nil {
		return err
	}
	if !p.Contains(netip.MustParseAddr("10.1.2.3")) {
		return fmt.Errorf("prefix")
	}
	if _, err := netip.ParseAddr("not-an-ip"); err == nil {
		return fmt.Errorf("bad addr accepted")
	}
	return nil
}

func checkNetURLParse() error {
	u, err := url.Parse("http://example.com:8080/path?q=1&r=2#frag")
	if err != nil {
		return err
	}
	if u.Host != "example.com:8080" || u.Path != "/path" {
		return fmt.Errorf("url=%v", u)
	}
	q := u.Query()
	if q.Get("q") != "1" || q.Get("r") != "2" {
		return fmt.Errorf("query=%v", q)
	}
	esc := url.QueryEscape("a b&c")
	if !strings.Contains(esc, "%20") {
		return fmt.Errorf("escape=%q", esc)
	}
	if u.Fragment != "frag" {
		return fmt.Errorf("fragment=%q", u.Fragment)
	}
	return nil
}

func checkNetMailParse() error {
	addr, err := mail.ParseAddress("User <user@example.com>")
	if err != nil {
		return err
	}
	if addr.Address != "user@example.com" || addr.Name != "User" {
		return fmt.Errorf("addr=%+v", addr)
	}
	list, err := mail.ParseAddressList("a@x.com, b@y.com")
	if err != nil || len(list) != 2 {
		return fmt.Errorf("list=%v err=%v", list, err)
	}
	return nil
}

func checkNetHTTPHttptest() error {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.Write([]byte("httptest-ok"))
	}))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Test") != "yes" {
		return fmt.Errorf("header missing")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "httptest-ok" {
		return fmt.Errorf("body=%q", body)
	}
	return nil
}

func checkNetHTTPPostForm() error {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte("field=" + r.Form.Get("field")))
	}))
	defer srv.Close()
	resp, err := http.PostForm(srv.URL, url.Values{"field": {"value"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "field=value" {
		return fmt.Errorf("post body=%q", body)
	}
	return nil
}

func checkEncodingCSV() error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"a", "b,c", "d\"e"}); err != nil {
		return err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	r := csv.NewReader(&buf)
	rec, err := r.Read()
	if err != nil {
		return err
	}
	if len(rec) != 3 || rec[1] != "b,c" || rec[2] != `d"e` {
		return fmt.Errorf("csv=%v", rec)
	}
	return nil
}

type gobPayload struct {
	Name string
	N    int
	Tags []string
}

func checkEncodingGob() error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	in := gobPayload{Name: "xp", N: 386, Tags: []string{"a", "b"}}
	if err := enc.Encode(in); err != nil {
		return err
	}
	var out gobPayload
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		return err
	}
	if out.Name != in.Name || out.N != in.N || !slices.Equal(out.Tags, in.Tags) {
		return fmt.Errorf("gob=%+v", out)
	}
	return nil
}

func checkEncodingBase32() error {
	enc := base32.StdEncoding.EncodeToString([]byte("xp"))
	dec, err := base32.StdEncoding.DecodeString(enc)
	if err != nil {
		return err
	}
	if string(dec) != "xp" {
		return fmt.Errorf("base32=%q", dec)
	}
	return nil
}

func checkEncodingPEM() error {
	block := &pem.Block{Type: "SMOKE", Bytes: []byte("pem-data")}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, block); err != nil {
		return err
	}
	out, _ := pem.Decode(buf.Bytes())
	if out == nil || out.Type != "SMOKE" || string(out.Bytes) != "pem-data" {
		return fmt.Errorf("pem roundtrip")
	}
	return nil
}

func checkJSONStream() error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(map[string]int{"a": 1}); err != nil {
		return err
	}
	if err := enc.Encode(map[string]int{"b": 2}); err != nil {
		return err
	}
	dec := json.NewDecoder(&buf)
	var m1, m2 map[string]int
	if err := dec.Decode(&m1); err != nil {
		return err
	}
	if err := dec.Decode(&m2); err != nil {
		return err
	}
	if m1["a"] != 1 || m2["b"] != 2 {
		return fmt.Errorf("stream=%v %v", m1, m2)
	}
	data, err := json.Marshal(map[string]any{"k": "v", "n": 1.5})
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m["k"] != "v" {
		return fmt.Errorf("map=%v", m)
	}
	return nil
}

func checkCryptoHashes() error {
	in := []byte("hash-input")
	if got := md5.Sum(in); got == [16]byte{} {
		return fmt.Errorf("md5 zero")
	}
	if got := sha1.Sum(in); got == [20]byte{} {
		return fmt.Errorf("sha1 zero")
	}
	if got := sha512.Sum512(in); got == [64]byte{} {
		return fmt.Errorf("sha512 zero")
	}
	if got := sha512.Sum512_256(in); got == [32]byte{} {
		return fmt.Errorf("sha512_256 zero")
	}
	h := sha256.New()
	h.Write(in)
	if len(h.Sum(nil)) != 32 {
		return fmt.Errorf("sha256 size")
	}
	return nil
}

func checkCryptoAESDES() error {
	key := make([]byte, 16)
	rand.Read(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	iv := make([]byte, aes.BlockSize)
	enc := cipher.NewCBCEncrypter(block, iv)
	plain := []byte("0123456789abcdef")
	ct := make([]byte, len(plain))
	enc.CryptBlocks(ct, plain)
	dec := cipher.NewCBCDecrypter(block, iv)
	out := make([]byte, len(plain))
	dec.CryptBlocks(out, ct)
	if !bytes.Equal(out, plain) {
		return fmt.Errorf("aes roundtrip")
	}
	dkey := make([]byte, 8)
	dblock, err := des.NewCipher(dkey)
	if err != nil {
		return err
	}
	div := make([]byte, des.BlockSize)
	denc := cipher.NewCBCEncrypter(dblock, div)
	dct := make([]byte, 8)
	denc.CryptBlocks(dct, []byte("12345678"))
	ddec := cipher.NewCBCDecrypter(dblock, div)
	dout := make([]byte, 8)
	ddec.CryptBlocks(dout, dct)
	if string(dout) != "12345678" {
		return fmt.Errorf("des roundtrip")
	}
	return nil
}

func checkCryptoSubtle() error {
	if subtle.ConstantTimeCompare([]byte("same"), []byte("same")) != 1 {
		return fmt.Errorf("cteq same")
	}
	if subtle.ConstantTimeCompare([]byte("same"), []byte("diff")) != 0 {
		return fmt.Errorf("cteq diff")
	}
	if subtle.ConstantTimeByteEq('a', 'a') != 1 {
		return fmt.Errorf("ctbyteeq")
	}
	return nil
}

func checkCryptoRandReader() error {
	buf := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return err
	}
	for _, b := range buf {
		if b != 0 {
			return nil
		}
	}
	return fmt.Errorf("crypto/rand all zeros")
}

func checkHashExtras() error {
	h := fnv.New64a()
	h.Write([]byte("fnv"))
	if h.Sum64() == 0 {
		return fmt.Errorf("fnv64a zero")
	}
	h32 := fnv.New32()
	h32.Write([]byte("fnv"))
	if h32.Sum32() == 0 {
		return fmt.Errorf("fnv32 zero")
	}
	if adler32.Checksum([]byte("adler")) == 1 {
		return fmt.Errorf("adler suspicious")
	}
	c64 := crc64.Checksum([]byte("crc64"), crc64.MakeTable(crc64.ISO))
	if c64 == 0 {
		return fmt.Errorf("crc64 zero")
	}
	var hh hash.Hash = sha256.New()
	if hh.Size() != 32 {
		return fmt.Errorf("hash size")
	}
	return nil
}

func checkCompressZlibFlate() error {
	in := []byte("compress-me-compress-me")
	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	if _, err := zw.Write(in); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	zr, err := zlib.NewReader(&zbuf)
	if err != nil {
		return err
	}
	zout, err := io.ReadAll(zr)
	zr.Close()
	if err != nil {
		return err
	}
	if !bytes.Equal(zout, in) {
		return fmt.Errorf("zlib=%q", zout)
	}
	var fbuf bytes.Buffer
	fw, err := flate.NewWriter(&fbuf, flate.DefaultCompression)
	if err != nil {
		return err
	}
	if _, err := fw.Write(in); err != nil {
		return err
	}
	if err := fw.Close(); err != nil {
		return err
	}
	fr := flate.NewReader(&fbuf)
	fout, err := io.ReadAll(fr)
	fr.Close()
	if err != nil {
		return err
	}
	if !bytes.Equal(fout, in) {
		return fmt.Errorf("flate=%q", fout)
	}
	return nil
}

func checkArchiveTar() error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte("tar-data")
	if err := tw.WriteHeader(&tar.Header{Name: "f.txt", Mode: 0o644, Size: int64(len(content))}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	if err != nil {
		return err
	}
	if hdr.Name != "f.txt" {
		return fmt.Errorf("tar name=%q", hdr.Name)
	}
	data, err := io.ReadAll(tr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, content) {
		return fmt.Errorf("tar data=%q", data)
	}
	return nil
}

func checkContainerListRing() error {
	l := list.New()
	l.PushBack("a")
	l.PushFront("z")
	var names []string
	for e := l.Front(); e != nil; e = e.Next() {
		names = append(names, e.Value.(string))
	}
	if len(names) != 2 || names[0] != "z" || names[1] != "a" {
		return fmt.Errorf("list=%v", names)
	}
	e := l.Back()
	l.Remove(e)
	if l.Len() != 1 {
		return fmt.Errorf("list len=%d", l.Len())
	}
	r := ring.New(3)
	for i := 0; i < 3; i++ {
		r.Value = i
		r = r.Next()
	}
	sum := 0
	for i := 0; i < 3; i++ {
		sum += r.Value.(int)
		r = r.Next()
	}
	if sum != 3 {
		return fmt.Errorf("ring sum=%d", sum)
	}
	return nil
}

type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intHeap) Pop() any {
	old := *h
	n := len(old)
	v := old[n-1]
	*h = old[:n-1]
	return v
}

func checkContainerHeap() error {
	h := &intHeap{3, 1, 4, 1, 5}
	heap.Init(h)
	var got []int
	for h.Len() > 0 {
		got = append(got, heap.Pop(h).(int))
	}
	if !slices.Equal(got, []int{1, 1, 3, 4, 5}) {
		return fmt.Errorf("heap=%v", got)
	}
	return nil
}

type smokeStruct struct {
	Name string
	Age  int
}

func checkReflectBasic() error {
	v := reflect.ValueOf(smokeStruct{Name: "xp", Age: 27})
	if v.Type().Name() != "smokeStruct" {
		return fmt.Errorf("type=%s", v.Type().Name())
	}
	f := v.FieldByName("Name")
	if !f.IsValid() || f.String() != "xp" {
		return fmt.Errorf("field=%v", f)
	}
	pv := reflect.ValueOf(&smokeStruct{})
	pv.Elem().FieldByName("Age").SetInt(30)
	if pv.Elem().Interface().(smokeStruct).Age != 30 {
		return fmt.Errorf("set")
	}
	if reflect.TypeOf(42).Kind() != reflect.Int {
		return fmt.Errorf("kind")
	}
	if !reflect.TypeOf((*error)(nil)).Elem().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return fmt.Errorf("implements")
	}
	return nil
}

func checkUnicodeUTF8() error {
	if !utf8.ValidString("héllo") {
		return fmt.Errorf("validstring")
	}
	if utf8.ValidString("\xff\xfe bad") {
		return fmt.Errorf("invalid accepted")
	}
	if utf8.RuneCountInString("héllo") != 5 {
		return fmt.Errorf("runecount=%d", utf8.RuneCountInString("héllo"))
	}
	r, size := utf8.DecodeRuneInString("éx")
	if r != 'é' || size != 2 {
		return fmt.Errorf("decode=%c %d", r, size)
	}
	if !unicode.IsLetter('a') || !unicode.IsDigit('5') || unicode.IsLetter('5') {
		return fmt.Errorf("unicode classes")
	}
	if unicode.ToUpper('a') != 'A' {
		return fmt.Errorf("toupper")
	}
	return nil
}

func checkMathBasic() error {
	if math.Sqrt(16) != 4 {
		return fmt.Errorf("sqrt")
	}
	if math.Pow(2, 10) != 1024 {
		return fmt.Errorf("pow")
	}
	if !math.IsNaN(math.NaN()) || !math.IsInf(math.Inf(1), 1) {
		return fmt.Errorf("nan/inf")
	}
	if math.Abs(-3) != 3 || math.Floor(1.9) != 1 || math.Ceil(1.1) != 2 {
		return fmt.Errorf("abs/floor/ceil")
	}
	if math.Max(1, 2) != 2 || math.Min(1, 2) != 1 {
		return fmt.Errorf("max/min")
	}
	if math.Mod(7, 3) != 1 {
		return fmt.Errorf("mod")
	}
	return nil
}

func checkMathBig() error {
	a := big.NewInt(1)
	a.Lsh(a, 100) // 2^100
	if a.BitLen() != 101 {
		return fmt.Errorf("bitlen=%d", a.BitLen())
	}
	b := big.NewInt(12345)
	c := new(big.Int).Mul(b, b)
	if c.Cmp(big.NewInt(12345*12345)) != 0 {
		return fmt.Errorf("mul=%v", c)
	}
	q, r := new(big.Int).QuoRem(c, b, new(big.Int))
	if q.Cmp(b) != 0 || r.Sign() != 0 {
		return fmt.Errorf("quorem q=%v r=%v", q, r)
	}
	f := new(big.Float).SetFloat64(1.5)
	if f.Sign() <= 0 {
		return fmt.Errorf("bigfloat")
	}
	return nil
}

func checkSortSearchSlice() error {
	a := []int{5, 3, 8, 1}
	sort.Ints(a)
	if !slices.Equal(a, []int{1, 3, 5, 8}) {
		return fmt.Errorf("sort=%v", a)
	}
	if sort.SearchInts(a, 5) != 2 {
		return fmt.Errorf("search=%d", sort.SearchInts(a, 5))
	}
	b := []string{"b", "a", "c"}
	sort.Strings(b)
	if !slices.Equal(b, []string{"a", "b", "c"}) {
		return fmt.Errorf("strings=%v", b)
	}
	sort.Slice(a, func(i, j int) bool { return a[i] > a[j] })
	if !slices.Equal(a, []int{8, 5, 3, 1}) {
		return fmt.Errorf("slice desc=%v", a)
	}
	if !slices.Contains(a, 5) || slices.Index(a, 3) != 2 {
		return fmt.Errorf("slices contains/index")
	}
	if slices.Index(a, 99) != -1 {
		return fmt.Errorf("index missing")
	}
	fs := []float64{2.5, 1.5}
	sort.Float64s(fs)
	if !slices.Equal(fs, []float64{1.5, 2.5}) {
		return fmt.Errorf("floats=%v", fs)
	}
	return nil
}

func checkIterSlicesValues() error {
	var sum int
	for v := range slices.Values([]int{1, 2, 3}) {
		sum += v
	}
	if sum != 6 {
		return fmt.Errorf("values sum=%d", sum)
	}
	keys := slices.Collect(maps.Keys(map[string]int{"a": 1}))
	if len(keys) != 1 || keys[0] != "a" {
		return fmt.Errorf("keys=%v", keys)
	}
	vals := slices.Collect(maps.Values(map[string]int{"a": 7}))
	if len(vals) != 1 || vals[0] != 7 {
		return fmt.Errorf("values=%v", vals)
	}
	return nil
}

func checkCmpCompare() error {
	if cmp.Compare(1, 2) != -1 || cmp.Compare(2, 1) != 1 || cmp.Compare(1, 1) != 0 {
		return fmt.Errorf("cmp")
	}
	if cmp.Or(0, 0, 3) != 3 || cmp.Or(1, 2, 3) != 1 {
		return fmt.Errorf("or")
	}
	if !cmp.Less("a", "b") || cmp.Less("b", "a") {
		return fmt.Errorf("less")
	}
	return nil
}

func checkTemplateExecute() error {
	t := template.Must(template.New("t").Parse("hello {{.Name}} {{.Age}}"))
	var buf bytes.Buffer
	if err := t.Execute(&buf, smokeStruct{Name: "xp", Age: 386}); err != nil {
		return err
	}
	if buf.String() != "hello xp 386" {
		return fmt.Errorf("template=%q", buf.String())
	}
	return nil
}

func checkFlagParse() error {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var name string
	var n int
	fs.StringVar(&name, "name", "", "name")
	fs.IntVar(&n, "n", 0, "n")
	if err := fs.Parse([]string{"-name", "xp", "-n", "5", "pos"}); err != nil {
		return err
	}
	if name != "xp" || n != 5 || fs.Arg(0) != "pos" {
		return fmt.Errorf("flag name=%q n=%d arg=%q", name, n, fs.Arg(0))
	}
	return nil
}

func checkHTMLEscape() error {
	e := html.EscapeString(`<a href="x">&'`)
	if !strings.Contains(e, "&lt;") || !strings.Contains(e, "&amp;") {
		return fmt.Errorf("escape=%q", e)
	}
	u := html.UnescapeString("&lt;a&gt;")
	if u != "<a>" {
		return fmt.Errorf("unescape=%q", u)
	}
	return nil
}

func checkRuntimeNumGoroutineMemStats() error {
	if runtime.NumGoroutine() < 1 {
		return fmt.Errorf("numgoroutine=%d", runtime.NumGoroutine())
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if ms.Alloc == 0 && ms.HeapAlloc == 0 {
		return fmt.Errorf("memstats zero")
	}
	if ms.NumGC < 0 {
		return fmt.Errorf("numgc")
	}
	return nil
}

func checkDebugGCPercent() error {
	old := debug.SetGCPercent(200)
	debug.SetGCPercent(old)
	debug.FreeOSMemory()
	return nil
}

func checkIOFSReadDir(dir string) error {
	entries, err := fs.ReadDir(os.DirFS(dir), ".")
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("empty dirfs")
	}
	if !fs.ValidPath("a/b/c") || fs.ValidPath("../x") || fs.ValidPath("a//b") {
		return fmt.Errorf("validpath")
	}
	return nil
}
