package managersetup

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestTLSIsTrustedOnlyForLoopback(t *testing.T) {
	files, err := credentials(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateTLS(files); err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(files["tls.crt"], files["tls.key"])
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	for _, host := range []string{"127.0.0.1", "::1", "localhost"} {
		if _, err = certificate.Verify(x509.VerifyOptions{Roots: roots, DNSName: host}); err != nil {
			t.Fatal(err)
		}
	}
	if err = certificate.VerifyHostname("manager.example"); err == nil {
		t.Fatal("loopback certificate accepted external hostname")
	}
}

func TestEntropyFailuresDoNotProduceCredentials(t *testing.T) {
	for _, size := range []int{0, 32, 48} {
		files, err := credentialsWithRandom(nil, bytes.NewReader(bytes.Repeat([]byte{1}, size)))
		if err == nil || files != nil {
			t.Fatal("incomplete entropy accepted")
		}
	}
}
