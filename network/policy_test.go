package network

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

func TestEndpointDualStackShapeAndIPv4Selection(t *testing.T) {
	for _, input := range []string{"tcp://manager.example:9757", "tcp://192.0.2.1:9757", "tcp://[2001:db8::1]:9757"} {
		if _, err := ParseEndpoint(input); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
	}
	endpoint, _ := ParseEndpoint("tcp://manager.example:9757")
	selected, err := SelectIPv4([]netip.Addr{netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.1")}, endpoint.Port)
	if err != nil || selected.String() != "192.0.2.1:9757" {
		t.Fatalf("selected=%s err=%v", selected, err)
	}
	if _, err = SelectIPv4([]netip.Addr{netip.MustParseAddr("2001:db8::1")}, 1); !errors.Is(err, ErrIPv6Unavailable) {
		t.Fatalf("IPv6-only=%v", err)
	}
}

func TestRetryDelayIsBounded(t *testing.T) {
	if RetryDelay(0) != 2*time.Second || RetryDelay(100) != 5*time.Minute {
		t.Fatalf("retry bounds: %s %s", RetryDelay(0), RetryDelay(100))
	}
}

func TestParseEndpointRejectsMalformedAndUnsafeForms(t *testing.T) {
	invalid := []string{
		"%", "http://manager.example:9757", "tcp://manager.example:9757/path",
		"tcp://manager.example:9757?query=1", "tcp://manager.example:9757#fragment",
		"tcp://:9757", "tcp://manager.example", "tcp://manager.example:0",
		"tcp://manager.example:not-a-port",
	}
	for _, value := range invalid {
		if _, err := ParseEndpoint(value); !errors.Is(err, ErrEndpoint) {
			t.Fatalf("%q: %v", value, err)
		}
	}
}
