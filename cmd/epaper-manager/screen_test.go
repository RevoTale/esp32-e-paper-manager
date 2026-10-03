package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/manager"
)

func screenArgs() []string {
	return []string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", "token", "-screen", "-usb-worker", "epaperscreen", "-serial", "port", "-trusted-html"}
}

func TestScreenConfigIsOptInAndLoopbackOnly(t *testing.T) {
	c, err := parseConfig(screenArgs())
	if err != nil || c.httpsAddress != "127.0.0.1:8443" || c.screen.interval != 180*time.Second {
		t.Fatal(c, err)
	}
	for _, extra := range [][]string{
		{"-https-listen", ":8443"}, {"-https-listen", "192.168.1.4:8443"},
		{"-trusted-html=false"}, {"-usb-worker", ""}, {"-serial", ""},
		{"-full-interval", "1s"}, {"-debounce-interval", "-1s"}, {"-max-wait", "-1s"},
		{"-screen=false", "-enrollment", "device", "-state", "state"},
		{"-width", "0"}, {"-height", "2049"}, {"-width", "2048", "-height", "2048"},
	} {
		if _, err = parseConfig(append(screenArgs(), extra...)); err == nil {
			t.Fatal(extra)
		}
	}
	if _, err = parseConfig(append(screenArgs(), "-https-listen", "[::1]:8443", "-debounce-interval", "1s", "-max-wait", "2s")); err != nil {
		t.Fatal(err)
	}
}

func TestScreenServerUsesExistingTLSAndClosesPump(t *testing.T) {
	c, _ := parseConfig(screenArgs())
	want := errors.New("server stopped")
	original := serveTLS
	defer func() { serveTLS = original }()
	serveTLS = func(s *http.Server, cert, key string) error {
		if _, ok := s.Handler.(*manager.ScreenAPI); !ok {
			t.Fatal("wrong handler")
		}
		if s.TLSConfig.MinVersion != tls.VersionTLS13 || s.ReadTimeout == 0 || cert != "cert" || key != "key" {
			t.Fatal(s, cert, key)
		}
		return want
	}
	if err := runScreen(c, []byte(strings.Repeat("x", 32))); !errors.Is(err, want) {
		t.Fatal(err)
	}
}

func TestScreenServingStopsOnDeliveryFailure(t *testing.T) {
	want := errors.New("USB disconnected")
	s := &http.Server{}
	serve := func() error { <-time.After(time.Millisecond); return nil }
	if err := serveScreen(context.Background(), s, serve, func(context.Context) error { return want }); !errors.Is(err, want) {
		t.Fatal(err)
	}
}
