// fetch downloads a URL and reports status, size, and SHA-256.
// Usage: fetch [-k] [-v] <url>...
package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

func main() {
	insecure := false
	verbose := false
	args := os.Args[1:]
	for len(args) > 0 && args[0] == "-k" || len(args) > 0 && args[0] == "-v" {
		if args[0] == "-k" {
			insecure = true
		} else {
			verbose = true
		}
		args = args[1:]
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: fetch [-k] [-v] <url>...")
		os.Exit(2)
	}
	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: tr}
	fail := 0
	for _, url := range args {
		if err := fetchOne(client, url, verbose); err != nil {
			fmt.Printf("FAIL %s: %v\n", url, err)
			fail++
		}
	}
	if fail > 0 {
		os.Exit(1)
	}
}

func fetchOne(c *http.Client, url string, verbose bool) error {
	start := time.Now()
	ctx := context.Background()
	if verbose {
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
			GetConn:              func(h string) { fmt.Println("  trace: GetConn", h) },
			GotConn:              func(ci httptrace.GotConnInfo) { fmt.Println("  trace: GotConn", ci.Reused, ci.WasIdle) },
			PutIdleConn:          func(e error) { fmt.Println("  trace: PutIdleConn", e) },
			GotFirstResponseByte: func() { fmt.Println("  trace: GotFirstResponseByte") },
			Got100Continue:       func() { fmt.Println("  trace: 100-continue") },
			WroteRequest:         func(info httptrace.WroteRequestInfo) { fmt.Println("  trace: WroteRequest err=", info.Err) },
			DNSStart:             func(i httptrace.DNSStartInfo) { fmt.Println("  trace: DNSStart", i.Host) },
			DNSDone:              func(i httptrace.DNSDoneInfo) { fmt.Println("  trace: DNSDone", i.Addrs, i.Err) },
			ConnectStart:         func(n, a string) { fmt.Println("  trace: ConnectStart", n, a) },
			ConnectDone:          func(n, a string, err error) { fmt.Println("  trace: ConnectDone", err) },
			TLSHandshakeStart:    func() { fmt.Println("  trace: TLSHandshakeStart") },
			TLSHandshakeDone:     func(cs tls.ConnectionState, err error) { fmt.Printf("  trace: TLSHandshakeDone v=%x err=%v\n", cs.Version, err) },
		})
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	h := sha256.New()
	n, err := io.Copy(h, resp.Body)
	if err != nil {
		return err
	}
	ms := time.Since(start).Milliseconds()
	fmt.Printf("OK %s status=%d bytes=%d sha256=%s ms=%d\n",
		url, resp.StatusCode, n, hex.EncodeToString(h.Sum(nil))[:16], ms)
	return nil
}
