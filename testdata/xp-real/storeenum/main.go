// storeenum lists subjects in the Windows ROOT and CA certificate stores.
package main

import (
	"crypto/x509"
	"fmt"
	"syscall"
	"unsafe"
)

func enumStore(storeFlag uint32, name string) {
	name16, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		fmt.Println("utf16:", err)
		return
	}
	store, err := syscall.CertOpenSystemStore(0, name16)
	if err != nil {
		fmt.Printf("%s: open err=%v\n", name, err)
		return
	}
	defer syscall.CertCloseStore(store, 0)
	count := 0
	var prev *syscall.CertContext
	for {
		ctx, err := syscall.CertEnumCertificatesInStore(store, prev)
		if err != nil || ctx == nil {
			break
		}
		count++
		// CertContext -> pbCertEncoded at offset? use CertContext struct fields
		cert, perr := x509.ParseCertificate(unsafe.Slice((*byte)(unsafe.Pointer(ctx.EncodedCert)), ctx.Length))
		if perr == nil {
			fmt.Printf("  %s\n", cert.Subject.CommonName)
		} else {
			fmt.Printf("  <unparsed>\n")
		}
		prev = ctx
		if count > 500 {
			fmt.Println("  ...truncated")
			break
		}
	}
	fmt.Printf("%s: %d certs\n", name, count)
}

func main() {
	const (
		CERT_SYSTEM_STORE_CURRENT_USER   = 0x00010000
		CERT_SYSTEM_STORE_LOCAL_MACHINE  = 0x00020000
	)
	enumStore(CERT_SYSTEM_STORE_LOCAL_MACHINE, "ROOT")
	enumStore(CERT_SYSTEM_STORE_LOCAL_MACHINE, "CA")
	enumStore(CERT_SYSTEM_STORE_CURRENT_USER, "ROOT")
}
