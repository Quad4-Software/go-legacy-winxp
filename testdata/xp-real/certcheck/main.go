// certcheck stages Windows CAPI chain building to find the XP hang.
// Usage: certcheck.exe
package main

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const certB64 = "MIID5jCCA42gAwIBAgIQAe7mqrtSHV4U/DFf2ZhWkDAKBggqhkjOPQQDAjBRMQswCQYDVQQGEwJVUzEYMBYGA1UECgwPU1NMIENvcnBvcmF0aW9uMSgwJgYDVQQDDB9DbG91ZGZsYXJlIFRMUyBJc3N1aW5nIEVDQyBDQSAzMB4XDTI2MDkyNjIyNDkxMVoXDTI2MTIyNTIyNTYzNVowFjEUMBIGA1UEAwwLZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARqadExWzxTJ4S3LSzKIzAAeyF09q0Qt0F0XmLiZiaJSD0SQPhVK36pUKlM55TUstJnrU7f+7elE0kZESKzzcbto4ICgDCCAnwwDAYDVR0TAQH/BAIwADAfBgNVHSMEGDAWgBSDA/3n9vVKTRVB9O0iFtMyCj7KZjBsBggrBgEFBQcBAQRgMF4wOQYIKwYBBQUHMAKGLWh0dHA6Ly9pLmNmLWkuc3NsLmNvbS9DbG91ZGZsYXJlLVRMUy1JLUUzLmNlcjAhBggrBgEFBQcwAYYVaHR0cDovL28uY2YtaS5zc2wuY29tMCUGA1UdEQQeMByCC2V4YW1wbGUuY29tgg0qLmV4YW1wbGUuY29tMCMGA1UdIAQcMBowCAYGZ4EMAQIBMA4GDCsGAQQBgqkwAQMBATATBgNVHSUEDDAKBggrBgEFBQcDATBTBgNVHR8ETDBKMEigRqBEhkJodHRwOi8vYy5jZi1pLnNzbC5jb20vYWU4MDFlZDFjNTViYjU3OWQ3OTIwOGIwZDc3MmFjZmI4Y2MzYTIwOC5jcmwwDgYDVR0PAQH/BAQDAgeAMA8GCSsGAQQBgtpLLAQCBQAwggEEBgorBgEEAdZ5AgQCBIH1BIHyAPAAdgCUTkOH+uzB74HzGSQmqBhlAcfTXzgCAT9yZ31VNy4Z2AAAAaDf8a1iAAAEAwBHMEUCIQCuCUahRazwOcqj3A77y8eXdQBpLlGCjv9D/xvXd3X7hAIgbc0KMS+MsANY5WLGpHgTk3Uxk0g1VWTQB78vYRMga1EAdgDLOPcViXyEoURfW8Hd+8lu8ppZzUcKaQWFsMsUwxRY5wAAAaDf8a1+AAAEAwBHMEUCIB2YN/lZP6eSBMtV0BTrGD7KSyKVu/i68Zza97li96FcAiEA1mtvTnM8zbNKj6ArUpWmgTRURDDgOC4R1C2W5cSK6fIwCgYIKoZIzj0EAwIDRwAwRAIgNGlXxWLTTqn5+XQOkP06fTgaEbjDMAHUILmIapx8i5sCIAeTI/xpafGZNB9HwNfEeU05yh+aSv95pD8SNtIAWNr0"

func stage(name string, f func() error) {
	t := time.Now()
	err := f()
	fmt.Printf("%-40s %6dms err=%v\n", name, time.Since(t).Milliseconds(), err)
}

func main() {
	der, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		panic(err)
	}
	stage("x509.ParseCertificate", func() error {
		_, err := x509.ParseCertificate(der)
		return err
	})

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		panic(err)
	}
	fmt.Println("subject:", cert.Subject.CommonName, "issuer:", cert.Issuer.CommonName)

	// Stage 1: createStoreContext equivalent
	var storeCtx *syscall.CertContext
	stage("CertCreateCertificateContext", func() error {
		var err error
		storeCtx, err = syscall.CertCreateCertificateContext(syscall.X509_ASN_ENCODING|syscall.PKCS_7_ASN_ENCODING, &cert.Raw[0], uint32(len(cert.Raw)))
		return err
	})
	defer func() {
		if storeCtx != nil {
			syscall.CertFreeCertificateContext(storeCtx)
		}
	}()

	var store syscall.Handle
	stage("CertOpenStore(memory)", func() error {
		var err error
		store, err = syscall.CertOpenStore(syscall.CERT_STORE_PROV_MEMORY, 0, 0, syscall.CERT_STORE_DEFER_CLOSE_UNTIL_LAST_FREE_FLAG, 0)
		return err
	})
	if store != 0 {
		defer syscall.CertCloseStore(store, 0)
	}

	stage("CertAddCertificateContextToStore", func() error {
		return syscall.CertAddCertificateContextToStore(store, storeCtx, syscall.CERT_STORE_ADD_ALWAYS, nil)
	})

	// Stage 2: CertGetCertificateChain
	para := new(syscall.CertChainPara)
	para.Size = uint32(unsafe.Sizeof(*para))
	var topCtx *syscall.CertChainContext
	stage("CertGetCertificateChain", func() error {
		return syscall.CertGetCertificateChain(0, storeCtx, nil, store, para, 0x84, 0, &topCtx) // CERT_CHAIN_CACHE_ONLY_URL_RETRIEVAL|CERT_CHAIN_RETURN_LOWER_QUALITY_CONTEXTS
	})
	if topCtx != nil {
		defer syscall.CertFreeCertificateChain(topCtx)
		fmt.Println("chain count:", topCtx.ChainCount)
		st := topCtx.TrustStatus.ErrorStatus
		fmt.Printf("trust error status: 0x%x\n", st)
	}

	// Stage 3: full Go Verify
	stage("cert.Verify(DNSName)", func() error {
		_, err := cert.Verify(x509.VerifyOptions{DNSName: "example.com"})
		return err
	})

	_ = pem.Block{}
	_ = os.Args
}
