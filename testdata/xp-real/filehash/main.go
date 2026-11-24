// filehash walks a directory tree and computes SHA-256 for every file
// using a worker pool, then prints a JSON summary. Realistic mix of
// filesystem traversal, concurrency, and crypto.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type fileResult struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Hash  string `json:"hash"`
	Error string `json:"error,omitempty"`
}

type summary struct {
	Root        string       `json:"root"`
	Files       int          `json:"files"`
	Bytes       int64        `json:"bytes"`
	Errors      int          `json:"errors"`
	ElapsedMS   int64        `json:"elapsed_ms"`
	Goroutines  int          `json:"goroutines"`
	TopLargest  []fileResult `json:"top_largest,omitempty"`
	CombinedSum string       `json:"combined_sha256"`
}

func main() {
	root := "."
	workers := runtime.NumCPU() * 2
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if len(os.Args) > 2 {
		fmt.Sscan(os.Args[2], &workers)
	}
	if workers < 1 {
		workers = 1
	}

	var paths []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})

	jobs := make(chan string)
	results := make(chan fileResult, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- hashFile(path)
			}
		}()
	}

	start := time.Now()
	go func() {
		for _, p := range paths {
			jobs <- p
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var files int64
	var totalBytes int64
	var errs int64
	combined := sha256.New()
	var all []fileResult
	for r := range results {
		files++
		totalBytes += r.Size
		if r.Error != "" {
			errs++
		} else {
			combined.Write([]byte(r.Hash))
		}
		all = append(all, r)
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Size > all[j].Size })
	top := all
	if len(top) > 10 {
		top = top[:10]
	}

	out := summary{
		Root:        root,
		Files:       int(files),
		Bytes:       totalBytes,
		Errors:      int(errs),
		ElapsedMS:   time.Since(start).Milliseconds(),
		Goroutines:  runtime.NumGoroutine(),
		TopLargest:  top,
		CombinedSum: hex.EncodeToString(combined.Sum(nil)),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	_ = atomic.LoadInt64(&files)
}

func hashFile(path string) fileResult {
	f, err := os.Open(path)
	if err != nil {
		return fileResult{Path: path, Error: err.Error()}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return fileResult{Path: path, Error: err.Error()}
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fileResult{Path: path, Size: info.Size(), Error: err.Error()}
	}
	return fileResult{Path: path, Size: info.Size(), Hash: hex.EncodeToString(h.Sum(nil))}
}
