package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestPartialCLIRequiresExplicitCompatibleProfile(t *testing.T) {
	args := append(screenArgs(), "-refresh-policy", "-experimental-partial")
	if _, err := parseConfig(args); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{
		{"-refresh-policy=false"}, {"-width", "200"}, {"-height", "200"},
		{"-partial-interval", "0"}, {"-partial-urgent-interval", "1ns"},
		{"-partial-max-updates", "0"}, {"-partial-max-updates", "65536"},
		{"-partial-max-bytes", "3"}, {"-partial-max-bytes", "48001"},
	} {
		if _, err := parseConfig(append(args, extra...)); err == nil {
			t.Fatal(extra)
		}
	}
}

func TestPartialCLIConstructsTransportAndScreen(t *testing.T) {
	c, err := parseConfig(append(screenArgs(), "-refresh-policy", "-experimental-partial"))
	if err != nil {
		t.Fatal(err)
	}
	old := serveTLS
	defer func() { serveTLS = old }()
	want := errors.New("stopped before hardware")
	serveTLS = func(*http.Server, string, string) error { return want }
	if err := runScreen(c, []byte(strings.Repeat("x", 32))); !errors.Is(err, want) {
		t.Fatal(err)
	}
}
