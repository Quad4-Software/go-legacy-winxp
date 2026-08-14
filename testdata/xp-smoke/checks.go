package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func runAllChecks(passed *int) {
	section("runtime")
	runCheck("runtime_cpus", checkRuntimeCPUs, passed)
	runCheck("runtime_goroutines", checkGoroutines, passed)
	runCheck("runtime_gc", checkRuntimeGC, passed)

	section("sync")
	runCheck("sync_mutex", checkSyncMutex, passed)
	runCheck("sync_atomic", checkSyncAtomic, passed)

	section("time")
	runCheck("time_clock", checkTimeClock, passed)
	runCheck("time_timer", checkTimeTimer, passed)

	section("env_paths")
	runCheck("env_path", checkEnvPath, passed)
	runCheck("os_hostname", checkHostname, passed)
	runCheck("os_getwd", checkGetwd, passed)
	runCheck("filepath_abs", checkFilepathAbs, passed)
	runCheck("filepath_join_clean", checkFilepathJoinClean, passed)

	section("os_files")
	dir := os.TempDir()
	fmt.Printf("smoke: tempdir=%s\n", dir)
	runCheck("os_mkdir_stat", func() error { return checkMkdirStat(dir) }, passed)
	runCheck("os_read_write_file", func() error { return checkReadWriteFile(dir) }, passed)
	runCheck("os_open_read_write", func() error { return checkOpenReadWrite(dir) }, passed)
	runCheck("os_rename", func() error { return checkRename(dir) }, passed)
	runCheck("os_read_dir", func() error { return checkReadDir(dir) }, passed)
	runCheck("os_walk_dir", func() error { return checkWalkDir(dir) }, passed)
	runCheck("os_pipe", checkPipe, passed)
	runCheck("os_remove_all", func() error { return checkRemoveAll(dir) }, passed)

	section("syscall")
	runCheck("syscall_computer_name", checkSyscallComputerName, passed)
	runCheck("syscall_getenv", checkSyscallGetenv, passed)

	section("crypto")
	runCheck("crypto_rand", checkCryptoRand, passed)

	section("encoding_io")
	runCheck("encoding_json", checkEncodingJSON, passed)
	runCheck("bufio_copy", func() error { return checkBufioCopy(dir) }, passed)

	section("net")
	runCheck("net_interfaces", checkNetInterfaces, passed)
	runCheck("net_lookup_localhost", checkNetLookupLocalhost, passed)
	runCheck("net_resolve_tcp", checkNetResolveTCP, passed)
	runCheck("net_tcp_listen_dial", checkNetTCPListenDial, passed)
	runCheck("net_udp_packet", checkNetUDPPacket, passed)
	runCheck("net_http_local", checkNetHTTPLocal, passed)
	runCheck("net_context_timeout", checkNetContextTimeout, passed)
}

func checkRuntimeCPUs() error {
	if runtime.GOMAXPROCS(0) < 1 {
		return fmt.Errorf("GOMAXPROCS=%d", runtime.GOMAXPROCS(0))
	}
	if runtime.NumCPU() < 1 {
		return fmt.Errorf("NumCPU=%d", runtime.NumCPU())
	}
	return nil
}

func checkGoroutines() error {
	const workers = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("goroutine wait timeout")
	}
}

func checkRuntimeGC() error {
	runtime.GC()
	return nil
}

func checkSyncMutex() error {
	var mu sync.Mutex
	var count int
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			count++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if count != 16 {
		return fmt.Errorf("count=%d", count)
	}
	return nil
}

func checkSyncAtomic() error {
	var n atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.Add(1)
		}()
	}
	wg.Wait()
	if n.Load() != 32 {
		return fmt.Errorf("atomic=%d", n.Load())
	}
	return nil
}

func checkTimeClock() error {
	if time.Now().Unix() <= 0 {
		return fmt.Errorf("clock unset")
	}
	return nil
}

func checkTimeTimer() error {
	t := time.NewTimer(20 * time.Millisecond)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timer timeout")
	}
}

func checkEnvPath() error {
	if os.Getenv("PATH") == "" {
		return fmt.Errorf("PATH empty")
	}
	if len(os.Environ()) == 0 {
		return fmt.Errorf("environ empty")
	}
	return nil
}

func checkHostname() error {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return fmt.Errorf("hostname: %w", err)
	}
	return nil
}

func checkGetwd() error {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return fmt.Errorf("getwd: %w", err)
	}
	return nil
}

func checkFilepathAbs() error {
	abs, err := filepath.Abs(".")
	if err != nil || abs == "" {
		return fmt.Errorf("abs: %w", err)
	}
	return nil
}

func checkFilepathJoinClean() error {
	p := filepath.Join("C:\\", "Windows", "Temp", "..", "Temp")
	if p == "" {
		return fmt.Errorf("empty join result")
	}
	clean := filepath.Clean(p)
	if clean == "" {
		return fmt.Errorf("empty clean result")
	}
	return nil
}

func checkMkdirStat(dir string) error {
	root := filepath.Join(dir, "go-legacy-winxp-smoke-tree")
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return err
	}
	info, err := os.Stat(sub)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func checkReadWriteFile(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-smoke.txt")
	if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if string(data) != "ok\n" {
		return fmt.Errorf("contents=%q", data)
	}
	return os.Remove(path)
}

func checkOpenReadWrite(dir string) error {
	path := filepath.Join(dir, "go-legacy-winxp-open.bin")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte("data")); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	f, err = os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 4)
	if _, err := io.ReadFull(f, buf); err != nil {
		return err
	}
	if string(buf) != "data" {
		return fmt.Errorf("read=%q", buf)
	}
	return os.Remove(path)
}

func checkRename(dir string) error {
	from := filepath.Join(dir, "go-legacy-winxp-rename.src")
	to := filepath.Join(dir, "go-legacy-winxp-rename.dst")
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	if _, err := os.Stat(from); err == nil {
		return fmt.Errorf("source still exists")
	}
	return os.Remove(to)
}

func checkReadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("empty dir listing")
	}
	return nil
}

func checkWalkDir(dir string) error {
	root := filepath.Join(dir, "go-legacy-winxp-walk")
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	leaf := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(leaf, "leaf.txt"), []byte("leaf"), 0o644); err != nil {
		return err
	}
	var count int
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("walk files=%d", count)
	}
	return os.RemoveAll(root)
}

func checkPipe() error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()
	defer w.Close()
	msg := []byte("pipe")
	if _, err := w.Write(msg); err != nil {
		return err
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if !bytes.Equal(buf, msg) {
		return fmt.Errorf("pipe data mismatch")
	}
	return nil
}

func checkRemoveAll(dir string) error {
	root := filepath.Join(dir, "go-legacy-winxp-removeall")
	if err := os.MkdirAll(filepath.Join(root, "x"), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("tree still exists")
	}
	return nil
}

func checkSyscallComputerName() error {
	var buf [256]uint16
	n := uint32(len(buf))
	if err := syscall.GetComputerName(&buf[0], &n); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("empty computer name")
	}
	return nil
}

func checkSyscallGetenv() error {
	v, found := syscall.Getenv("PATH")
	if !found || v == "" {
		return fmt.Errorf("PATH empty via syscall.Getenv")
	}
	return nil
}

func checkCryptoRand() error {
	buf := make([]byte, 32)
	if _, err := randRead(buf); err != nil {
		return err
	}
	for _, b := range buf {
		if b != 0 {
			return nil
		}
	}
	return fmt.Errorf("rand all zeros")
}

func randRead(buf []byte) (int, error) {
	return randReadPackage(buf)
}

func checkEncodingJSON() error {
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	in := payload{Name: "xp", N: 386}
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	var out payload
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	if out != in {
		return fmt.Errorf("json roundtrip=%+v", out)
	}
	return nil
}

func checkBufioCopy(dir string) error {
	src := filepath.Join(dir, "go-legacy-winxp-bufio.src")
	dst := filepath.Join(dir, "go-legacy-winxp-bufio.dst")
	content := "buffered copy test\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(out)
	if _, err := io.Copy(w, bufio.NewReader(in)); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		return err
	}
	if string(data) != content {
		return fmt.Errorf("copy=%q", data)
	}
	os.Remove(src)
	os.Remove(dst)
	return nil
}

func checkNetInterfaces() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	if len(ifaces) == 0 {
		return fmt.Errorf("no interfaces")
	}
	return nil
}

func checkNetLookupLocalhost() error {
	addrs, err := net.LookupHost("localhost")
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("lookup localhost: %w", err)
	}
	return nil
}

func checkNetResolveTCP() error {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil || addr.IP == nil {
		return fmt.Errorf("resolve tcp: %w", err)
	}
	return nil
}

func checkNetTCPListenDial() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer ln.Close()
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()
		_, err = conn.Write([]byte("ack"))
		errCh <- err
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		return err
	}
	defer client.Close()
	buf := make([]byte, 3)
	if _, err := io.ReadFull(client, buf); err != nil {
		return err
	}
	if string(buf) != "ack" {
		return fmt.Errorf("tcp payload=%q", buf)
	}
	return <-errCh
}

func checkNetUDPPacket() error {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	msg := []byte("udp")
	if _, err := conn.WriteToUDP(msg, conn.LocalAddr().(*net.UDPAddr)); err != nil {
		return err
	}
	buf := make([]byte, 8)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "udp" {
		return fmt.Errorf("udp payload=%q", buf[:n])
	}
	return nil
}

func checkNetHTTPLocal() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("http ok"))
		}),
	}
	go srv.Serve(ln)
	defer srv.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	url := "http://" + ln.Addr().String()
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "http ok" {
		return fmt.Errorf("http body=%q", body)
	}
	return nil
}

func checkNetContextTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	d := net.Dialer{Timeout: 50 * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:9")
	if err == nil {
		conn.Close()
		return fmt.Errorf("expected timeout dial to fail")
	}
	return nil
}
