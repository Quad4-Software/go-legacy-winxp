// jsonapi is a small JSON HTTP service with a built-in client mode.
// Usage:
//
//	jsonapi.exe serve            start server on :8377
//	jsonapi.exe client           run self-test requests against it
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price float64 `json:"price"`
}

var items = []item{
	{1, "widget", 9.99},
	{2, "gadget", 19.95},
	{3, "sprocket", 4.50},
}

var hits int64

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: jsonapi serve|client")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		serve()
	case "client":
		client()
	default:
		fmt.Fprintln(os.Stderr, "unknown mode", os.Args[1])
		os.Exit(2)
	}
}

func serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		writeJSON(w, items)
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		writeJSON(w, map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"body":   string(body),
			"echo":   hex.EncodeToString(func() []byte { h := sha256.Sum256(body); return h[:] }()),
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "jsonapi hits=%d\n", atomic.LoadInt64(&hits))
	})
	fmt.Println("jsonapi listening on :8377")
	srv := &http.Server{Addr: ":8377", Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		os.Exit(1)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
	}
}

func client() {
	base := "http://127.0.0.1:8377"
	fail := 0

	resp, err := http.Get(base + "/health")
	if err != nil {
		fmt.Println("FAIL health:", err)
		os.Exit(1)
	}
	var health map[string]string
	json.NewDecoder(resp.Body).Decode(&health)
	resp.Body.Close()
	if health["status"] != "ok" {
		fmt.Println("FAIL health body:", health)
		fail++
	} else {
		fmt.Println("PASS health")
	}

	resp, err = http.Get(base + "/items")
	if err != nil {
		fmt.Println("FAIL items:", err)
		os.Exit(1)
	}
	var got []item
	json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if len(got) != len(items) {
		fmt.Println("FAIL items count:", len(got))
		fail++
	} else {
		fmt.Println("PASS items", len(got), "items")
	}

	payload := []byte(`{"query":"windows xp","n":42}`)
	resp, err = http.Post(base+"/echo", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("FAIL echo:", err)
		os.Exit(1)
	}
	var echo map[string]string
	json.NewDecoder(resp.Body).Decode(&echo)
	resp.Body.Close()
	if echo["body"] != string(payload) {
		fmt.Println("FAIL echo body:", echo["body"])
		fail++
	} else {
		fmt.Println("PASS echo sha256=" + echo["echo"][:16])
	}

	resp, err = http.Get(base + "/")
	if err != nil {
		fmt.Println("FAIL root:", err)
		os.Exit(1)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("PASS root: %s", b)

	if fail > 0 {
		fmt.Println("jsonapi client: FAIL")
		os.Exit(1)
	}
	fmt.Println("jsonapi client: ALL PASS")
}
