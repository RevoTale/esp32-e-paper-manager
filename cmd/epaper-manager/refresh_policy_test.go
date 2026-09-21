package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRefreshOverrideIsExplicitAndBounded(t *testing.T) {
	args := append(screenArgs(), "-refresh-policy", "-full-interval", "10s", "-urgent-interval", "2s")
	c, err := parseConfig(args)
	if err != nil || c.screen.interval != 10*time.Second || c.screen.urgent != 2*time.Second {
		t.Fatal(c, err)
	}
	for _, extra := range [][]string{{"-urgent-interval", "0"}, {"-urgent-interval", "1ns"}, {"-full-interval", "0"}, {"-full-interval", "1200h"}} {
		if _, err := parseConfig(append(args, extra...)); err == nil {
			t.Fatal(extra)
		}
	}
}

func TestRefreshPolicyRunUsesNegotiatedSender(t *testing.T) {
	c, err := parseConfig(append(screenArgs(), "-refresh-policy"))
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("stopped")
	old := serveTLS
	defer func() { serveTLS = old }()
	serveTLS = func(*http.Server, string, string) error { return want }
	if err := runScreen(c, []byte(strings.Repeat("x", 32))); !errors.Is(err, want) {
		t.Fatal(err)
	}
}
