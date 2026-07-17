// XP smoke test for go-legacy-winxp CI and local Docker runs.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	fmt.Printf("go-legacy-winxp smoke ok version=%s goos=%s goarch=%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	dir := os.TempDir()
	path := filepath.Join(dir, "go-legacy-winxp-smoke.txt")
	if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write temp file: %v\n", err)
		os.Exit(1)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read temp file: %v\n", err)
		os.Exit(1)
	}
	if string(data) != "ok\n" {
		fmt.Fprintf(os.Stderr, "unexpected temp file contents: %q\n", data)
		os.Exit(1)
	}
	_ = os.Remove(path)
}
