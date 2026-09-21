package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/screenhub"
)

func wifiArgs() []string {
	return append(screenArgs(), "-usb-worker", "", "-serial", "", "-screen-transport", "wifi", "-enrollment", "registry")
}

func TestWiFiScreenRequiresExplicitEnrollmentAndOneTransport(t *testing.T) {
	c, err := parseConfig(wifiArgs())
	if err != nil || c.screen.transport != "wifi" || c.screen.maintenance != 10*time.Minute {
		t.Fatal(c, err)
	}
	for _, extra := range [][]string{
		{"-enrollment", ""}, {"-serial", "port"}, {"-usb-worker", "worker"},
		{"-screen-transport", "auto"}, {"-maintenance-interval", "0"}, {"-timezone", "invalid/zone"},
	} {
		if _, err = parseConfig(append(wifiArgs(), extra...)); err == nil {
			t.Fatal(extra)
		}
	}
	c, err = parseConfig(append(wifiArgs(), "-maintenance-interval", "15m", "-timezone", "Europe/Kyiv"))
	if err != nil || c.screen.maintenance != 15*time.Minute {
		t.Fatal(c, err)
	}
}

func networkConfig(t *testing.T) config {
	t.Helper()
	c, err := parseConfig(wifiArgs())
	if err != nil {
		t.Fatal(err)
	}
	c.enrollmentFile = filepath.Join(t.TempDir(), "enrollment.json")
	err = hostprovision.SaveEnrollment(c.enrollmentFile, hostprovision.Enrollment{
		DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	c.deviceAddress = "127.0.0.1:0"
	return c
}

func TestNetworkCompositionUsesEnrollmentZoneAndEPN2Hub(t *testing.T) {
	c := networkConfig(t)
	transport, zone, err := screenDelivery(c)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transport.close() }()
	if _, ok := transport.sender.(*screenhub.Hub); !ok {
		t.Fatal("legacy sender")
	}
	if zone.String() != "UTC" {
		t.Fatal("zone not from USB enrollment", zone)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = transport.run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestNetworkManagerStopsAllThreeOwners(t *testing.T) {
	c := networkConfig(t)
	want := errors.New("HTTPS stopped")
	original := serveTLS
	defer func() { serveTLS = original }()
	serveTLS = func(*http.Server, string, string) error { return want }
	if err := runScreen(c, []byte(strings.Repeat("x", 32))); !errors.Is(err, want) {
		t.Fatal(err)
	}
}

func TestNetworkCompositionFailsBeforeServingInvalidEnrollmentOrListener(t *testing.T) {
	c := networkConfig(t)
	original := listenTCP
	defer func() { listenTCP = original }()
	want := errors.New("listen rejected")
	listenTCP = func(string, string) (net.Listener, error) { return nil, want }
	if _, _, err := screenNetwork(c); !errors.Is(err, want) {
		t.Fatal(err)
	}
	c.enrollmentFile += ".missing"
	if _, _, err := screenNetwork(c); err == nil {
		t.Fatal("missing enrollment accepted")
	}
	if err := runScreen(c, make([]byte, 32)); err == nil {
		t.Fatal("invalid composition served")
	}
}
