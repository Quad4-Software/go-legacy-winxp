// XP integration smoke test for go-legacy-winxp Docker and CI runs.
package main

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func check(name string, fn func() error) {
	fmt.Printf("check %s: ", name)
	if err := fn(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func main() {
	fmt.Println("smoke: starting go-legacy-winxp guest checks")
	fmt.Printf("go-legacy-winxp smoke ok version=%s goos=%s goarch=%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	passed := 0
	run := func(name string, fn func() error) {
		check(name, fn)
		passed++
	}

	run("runtime_cpus", func() error {
		if runtime.GOMAXPROCS(0) < 1 {
			return fmt.Errorf("GOMAXPROCS=%d", runtime.GOMAXPROCS(0))
		}
		return nil
	})

	run("goroutines", func() error {
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
	})

	run("time", func() error {
		if time.Now().Unix() <= 0 {
			return fmt.Errorf("clock looks unset")
		}
		return nil
	})

	run("env_path", func() error {
		if os.Getenv("PATH") == "" {
			return fmt.Errorf("PATH is empty")
		}
		return nil
	})

	dir := os.TempDir()
	fmt.Printf("smoke: tempdir=%s\n", dir)

	run("temp_file_io", func() error {
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
	})

	run("read_dir", func() error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return fmt.Errorf("temp dir is empty")
		}
		return nil
	})

	run("crypto_rand", func() error {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return err
		}
		zero := true
		for _, b := range buf {
			if b != 0 {
				zero = false
				break
			}
		}
		if zero {
			return fmt.Errorf("rand returned all zeros")
		}
		return nil
	})

	run("net_interfaces", func() error {
		ifaces, err := net.Interfaces()
		if err != nil {
			return err
		}
		if len(ifaces) == 0 {
			return fmt.Errorf("no interfaces found")
		}
		return nil
	})

	run("net_lookup_localhost", func() error {
		addrs, err := net.LookupHost("localhost")
		if err != nil {
			return err
		}
		if len(addrs) == 0 {
			return fmt.Errorf("no addresses for localhost")
		}
		return nil
	})

	run("net_resolve_tcp", func() error {
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		if addr.IP == nil {
			return fmt.Errorf("nil IP")
		}
		return nil
	})

	fmt.Printf("smoke: %d checks passed\n", passed)
	fmt.Println("smoke: all checks passed")
}
