package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSecretRejectsSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(strings.Repeat("t", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, path+".link"); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecret(path + ".link"); err == nil {
		t.Fatal("secret symlink accepted")
	}
}

func TestHTTP2HasExplicitAdmissionAndReceiveBounds(t *testing.T) {
	server := httpServer("127.0.0.1:0", http.NotFoundHandler())
	if server.HTTP2 == nil {
		t.Fatal("HTTP/2 admission defaults are not a chosen bound")
	}
	c := server.HTTP2
	if c.MaxConcurrentStreams <= 0 || c.MaxConcurrentStreams > 8 || c.MaxReceiveBufferPerConnection != 65536 || c.MaxReceiveBufferPerStream != 32768 || c.MaxReadFrameSize != 16384 {
		t.Fatal("HTTP/2 admission is not bounded for the screen API")
	}
}
