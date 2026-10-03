package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeHTTPSRequiresTLS13(t *testing.T) {
	server := httpServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || r.TLS.Version != tls.VersionTLS13 {
			t.Error("request without TLS 1.3")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	listener, roots := startNativeHTTPS(t, server)
	for _, version := range []uint16{tls.VersionTLS13, tls.VersionTLS12} {
		transport := &http.Transport{ForceAttemptHTTP2: true, TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: version, MaxVersion: version}}
		client := &http.Client{Transport: transport, Timeout: time.Second}
		response, err := client.Get("https://" + listener.Addr().String())
		transport.CloseIdleConnections()
		if version == tls.VersionTLS12 {
			if err == nil {
				_ = response.Body.Close()
				t.Fatal("TLS 1.2 accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusUnauthorized || response.ProtoMajor != 2 {
			t.Fatal(response.StatusCode, response.Proto)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

type observedListener struct {
	net.Listener
	accepted chan struct{}
}

func (l *observedListener) Accept() (net.Conn, error) {
	connection, err := l.Listener.Accept()
	if err == nil {
		l.accepted <- struct{}{}
	}
	return connection, err
}

func TestHTTPSBoundsHalfOpenTLSConnections(t *testing.T) {
	server := httpServer("127.0.0.1:0", http.NotFoundHandler())
	listener, _ := startNativeHTTPS(t, server)
	for range maxHTTPSConnections + 1 {
		connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = connection.Close() })
	}
	for range maxHTTPSConnections {
		select {
		case <-listener.accepted:
		case <-time.After(time.Second):
			t.Fatal("connection not admitted")
		}
	}
	select {
	case <-listener.accepted:
		t.Fatal("TLS connection cap exceeded")
	case <-time.After(100 * time.Millisecond):
	}
}

func startNativeHTTPS(t *testing.T, server *http.Server) (*observedListener, *x509.CertPool) {
	t.Helper()
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	observed := &observedListener{Listener: listener, accepted: make(chan struct{}, 64)}
	original := listenTCP
	listenTCP = func(string, string) (net.Listener, error) { return observed, nil }
	certificate, key, roots := testCertificate(t)
	done := make(chan error, 1)
	go func() { done <- serveHTTPS(server, certificate, key) }()
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
		_ = observed.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("HTTPS did not stop")
		}
		listenTCP = original
	})
	return observed, roots
}

func testCertificate(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	certificate, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey})
	certPath, keyPath := filepath.Join(t.TempDir(), "cert.pem"), filepath.Join(t.TempDir(), "key.pem")
	if err = os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("test certificate invalid")
	}
	return certPath, keyPath, roots
}

func TestServeHTTPSListenerAndCertificateErrors(t *testing.T) {
	original := listenTCP
	t.Cleanup(func() { listenTCP = original })
	listenTCP = func(string, string) (net.Listener, error) { return nil, io.ErrClosedPipe }
	server := httpServer("127.0.0.1:0", http.NotFoundHandler())
	if err := serveHTTPS(server, "missing", "missing"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	listenTCP = net.Listen
	if err := serveHTTPS(server, "missing", "missing"); err == nil {
		t.Fatal("missing certificate accepted")
	}
}
