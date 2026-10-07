package managersetup

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"time"
)

func credentials(enrollment []byte) (map[string][]byte, error) {
	return credentialsWithRandom(enrollment, rand.Reader)
}

func credentialsWithRandom(enrollment []byte, random io.Reader) (map[string][]byte, error) {
	certificate, key, err := loopbackCertificate(random)
	if err != nil {
		return nil, err
	}
	var token [32]byte
	if _, err = io.ReadFull(random, token[:]); err != nil {
		return nil, err
	}
	defer clear(token[:])
	return map[string][]byte{
		"device.json": bytes.Clone(enrollment), "api-token": []byte(hex.EncodeToString(token[:]) + "\n"),
		"tls.crt": certificate, "tls.key": key,
	}, nil
}

func loopbackCertificate(random io.Reader) ([]byte, []byte, error) {
	public, private, err := ed25519.GenerateKey(random)
	if err != nil {
		return nil, nil, err
	}
	defer clear(private)
	serial, err := rand.Int(random, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "epaper-manager loopback"},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: now.AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(random, template, template, public, private)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, nil, err
	}
	defer clear(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), nil
}

func validateTLS(files map[string][]byte) error {
	pair, err := tls.X509KeyPair(files["tls.crt"], files["tls.key"])
	if err != nil {
		return errors.New("setup: invalid TLS pair; existing credentials preserved")
	}
	certificate, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	_, err = certificate.Verify(x509.VerifyOptions{Roots: roots, DNSName: "127.0.0.1"})
	return err
}
