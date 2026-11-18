package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

// mobile-init runs offline on a trusted host. A fresh private directory
// keeps existing manager identities safe from replacement.
func mobileInit(args []string, registry string) error {
	if len(args) < 3 || registry == "" {
		return errors.New("usage: -registry device.json mobile-init NEW_DIRECTORY HOST [HOST ...]")
	}
	enroll, err := hostprovision.LoadEnrollment(registry)
	if err != nil {
		return err
	}
	defer clear(enroll.DeviceKey[:])
	cert, key, err := mobileCertificate(args[2:])
	if err != nil {
		return err
	}
	dir := args[1]
	if err = os.Mkdir(dir, 0700); err != nil {
		return err
	}
	mac := hmac.New(sha256.New, enroll.DeviceKey[:])
	mac.Write([]byte("epaper/mobile-ap/v1"))
	card := fmt.Appendf(nil, "Wi-Fi: Epaper-%x\nPassword: %x\nSetup: http://192.168.4.1/\n", enroll.DeviceID[:3], mac.Sum(nil)[:16])
	defer clear(card)
	defer clear(key)
	for name, data := range map[string][]byte{"device-tls.crt": cert, "device-tls.key": key, "phone-setup.txt": card} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			return err
		}
	}
	return nil
}

func mobileCertificate(hosts []string) ([]byte, []byte, error) {
	t := &x509.Certificate{Subject: pkix.Name{CommonName: "E-paper device manager"},
		NotBefore: time.Now().Add(-5 * time.Minute), NotAfter: time.Now().AddDate(5, 0, 0),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	for _, host := range hosts {
		if !provision.ValidESP32Endpoint("tcp://" + host + ":443") {
			return nil, nil, errors.New("invalid IPv4 address or DNS name")
		}
		if ip := net.ParseIP(host); ip != nil {
			t.IPAddresses = append(t.IPAddresses, ip)
		} else {
			t.DNSNames = append(t.DNSNames, host)
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	t.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, t, t, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	raw, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	defer clear(raw)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: raw}), nil
}

func mobileEndpoint(hostPort, certPath string) (string, error) {
	if !strings.Contains(hostPort, ":") {
		hostPort = net.JoinHostPort(hostPort, "443")
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(certPath)
	if err != nil {
		return "", err
	}
	block, rest := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) != 0 {
		return "", errors.New("expected one public manager certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	if err = cert.VerifyHostname(host); err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("tls://%s#%x", hostPort, sha256.Sum256(block.Bytes))
	if !provision.ValidESP32Endpoint(endpoint) {
		return "", errors.New("invalid or oversized TLS endpoint")
	}
	return endpoint, nil
}
