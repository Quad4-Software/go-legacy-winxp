// tlscheck dials TLS endpoints, reports handshake timing, then sends a
// minimal HTTP/1.1 GET and reports whether a response arrives.
// Usage: tlscheck.exe <host:port> [tls12]
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
		fmt.Fprintln(os.Stderr, "usage: tlscheck <host:port> [tls12]")
		os.Exit(2)
	}
	addr := os.Args[1]
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	maxV := uint16(tls.VersionTLS13)
	if len(os.Args) > 2 && os.Args[2] == "tls12" {
		maxV = tls.VersionTLS12
	}
	start := time.Now()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         maxV,
		ServerName:         host,
	})
	if err != nil {
		fmt.Printf("FAIL dial %s: %v (after %dms)\n", addr, err, time.Since(start).Milliseconds())
		os.Exit(1)
	}
	state := conn.ConnectionState()
	fmt.Printf("OK handshake %s version=%x cipher=%x ms=%d\n",
		addr, state.Version, state.CipherSuite, time.Since(start).Milliseconds())

	req := "GET / HTTP/1.1\r\nHost: " + host + "\r\nUser-Agent: tlscheck\r\nAccept: */*\r\nConnection: close\r\n\r\n"
	conn.SetDeadline(time.Now().Add(20 * time.Second))
	if _, err := conn.Write([]byte(req)); err != nil {
		fmt.Println("FAIL write:", err)
		os.Exit(1)
	}
	buf, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		fmt.Println("FAIL read:", err)
		os.Exit(1)
	}
	fmt.Printf("OK response %d bytes, head: %.60q\n", len(buf), string(buf))
	conn.Close()
}
