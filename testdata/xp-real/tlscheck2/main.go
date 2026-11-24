// tlscheck2 isolates the TLS handshake: plain TCP dial, wrap in tls.Client,
// optionally SetDeadline before handshake, then handshake + GET.
// Usage: tlscheck2.exe <host:port> [deadline]
package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tlscheck2 <host:port> [deadline]")
		os.Exit(2)
	}
	addr := os.Args[1]
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	useDeadline := len(os.Args) > 2 && os.Args[2] == "deadline"

	start := time.Now()
	raw, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		fmt.Println("FAIL dial:", err)
		os.Exit(1)
	}
	conn := tls.Client(raw, &tls.Config{InsecureSkipVerify: true, ServerName: host})
	if useDeadline {
		conn.SetDeadline(time.Now().Add(15 * time.Second))
		fmt.Println("deadline set before handshake")
	}
	if err := conn.Handshake(); err != nil {
		fmt.Printf("FAIL handshake: %v (after %dms)\n", err, time.Since(start).Milliseconds())
		os.Exit(1)
	}
	st := conn.ConnectionState()
	fmt.Printf("OK handshake v=%x ms=%d\n", st.Version, time.Since(start).Milliseconds())

	req := "GET / HTTP/1.1\r\nHost: " + host + "\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		fmt.Println("FAIL write:", err)
		os.Exit(1)
	}
	buf, err := io.ReadAll(io.LimitReader(conn, 2048))
	if err != nil {
		fmt.Println("FAIL read:", err)
		os.Exit(1)
	}
	fmt.Printf("OK response %d bytes %.50q\n", len(buf), string(buf))
	conn.Close()
}
