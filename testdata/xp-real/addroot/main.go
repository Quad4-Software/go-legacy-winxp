// addroot adds a DER certificate to the Windows ROOT store.
// Usage: addroot.exe <cert.der>
package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: addroot <cert.der>")
		os.Exit(2)
	}
	der, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	name, err := syscall.UTF16PtrFromString("ROOT")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	store, err := syscall.CertOpenSystemStore(0, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		os.Exit(1)
	}
	defer syscall.CertCloseStore(store, 0)
	ctx, err := syscall.CertCreateCertificateContext(syscall.X509_ASN_ENCODING, &der[0], uint32(len(der)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create ctx:", err)
		os.Exit(1)
	}
	defer syscall.CertFreeCertificateContext(ctx)
	if err := syscall.CertAddCertificateContextToStore(store, ctx, syscall.CERT_STORE_ADD_ALWAYS, nil); err != nil {
		fmt.Fprintln(os.Stderr, "add:", err)
		os.Exit(1)
	}
	fmt.Println("added", os.Args[1])
}
