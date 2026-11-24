// archiver zips a directory tree, extracts it elsewhere, and verifies
// every file's SHA-256 matches. Usage:
//
//	archiver.exe <srcdir> <zipfile> <outdir>
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: archiver <srcdir> <zipfile> <outdir>")
		os.Exit(2)
	}
	src, zipPath, out := os.Args[1], os.Args[2], os.Args[3]

	// collect files + hashes of source
	srcFiles := map[string]string{}
	var paths []string
	filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(src, p)
			paths = append(paths, p)
			srcFiles[filepath.ToSlash(rel)] = mustHash(p)
		}
		return nil
	})
	sort.Strings(paths)

	// write zip
	zf, err := os.Create(zipPath)
	if err != nil {
		fatal("create zip:", err)
	}
	zw := zip.NewWriter(zf)
	for _, p := range paths {
		rel, _ := filepath.Rel(src, p)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			fatal("zip create:", err)
		}
		in, err := os.Open(p)
		if err != nil {
			fatal("open:", err)
		}
		if _, err := io.Copy(w, in); err != nil {
			fatal("copy:", err)
		}
		in.Close()
	}
	if err := zw.Close(); err != nil {
		fatal("zip close:", err)
	}
	zf.Close()
	zi, _ := os.Stat(zipPath)
	fmt.Printf("zip written: %s (%d bytes, %d files)\n", zipPath, zi.Size(), len(paths))

	// extract
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		fatal("open zip:", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		dst := filepath.Join(out, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			fatal("mkdir:", err)
		}
		rc, err := f.Open()
		if err != nil {
			fatal("zip open member:", err)
		}
		w, err := os.Create(dst)
		if err != nil {
			rc.Close()
			fatal("create out:", err)
		}
		if _, err := io.Copy(w, rc); err != nil {
			fatal("extract copy:", err)
		}
		w.Close()
		rc.Close()
	}

	// verify
	bad := 0
	for rel, want := range srcFiles {
		got := mustHash(filepath.Join(out, filepath.FromSlash(rel)))
		if got != want {
			fmt.Println("MISMATCH", rel)
			bad++
		}
	}
	if bad > 0 {
		fmt.Printf("archiver: FAIL (%d mismatches)\n", bad)
		os.Exit(1)
	}
	fmt.Printf("archiver: ALL PASS (%d files verified)\n", len(srcFiles))
}

func mustHash(p string) string {
	f, err := os.Open(p)
	if err != nil {
		fatal("hash open:", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fatal("hash copy:", err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func fatal(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
