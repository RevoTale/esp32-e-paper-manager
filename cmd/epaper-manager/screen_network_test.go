package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

func TestDeviceTLSAdmission(t *testing.T) {
	c := networkConfig(t)
	cert, key, roots := testCertificate(t)
	c.screen.deviceCert, c.screen.deviceKey = cert, key
	listener, err := deviceListener(c)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	for _, tc := range []struct {
		name    string
		roots   *x509.CertPool
		version uint16
		want    bool
	}{
		{"trusted", roots, tls.VersionTLS13, true},
		{"unknown-manager", x509.NewCertPool(), tls.VersionTLS13, false},
		{"old-tls", roots, tls.VersionTLS12, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan error, 1)
			go func() {
				conn, acceptErr := listener.Accept()
				if acceptErr != nil {
					done <- acceptErr
					return
				}
				defer func() { _ = conn.Close() }()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				done <- conn.(*tls.Conn).Handshake()
			}()
			conn, dialErr := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp", listener.Addr().String(), &tls.Config{
				RootCAs: tc.roots, MinVersion: tc.version, MaxVersion: tc.version,
			})
			if conn != nil {
				_ = conn.Close()
			}
			serverErr := <-done
			if (dialErr == nil) != tc.want || (serverErr == nil) != tc.want {
				t.Fatal(dialErr, serverErr)
			}
		})
	}
	c.screen.deviceKey = "missing"
	if _, err := deviceListener(c); err == nil {
		t.Fatal("accepted missing key")
	}
	for _, args := range [][]string{
		append(wifiArgs(), "-device-tls-cert", cert),
		append(screenArgs(), "-device-tls-cert", cert, "-device-tls-key", key),
	} {
		if _, err := parseConfig(args); err == nil {
			t.Fatal("accepted invalid device TLS flags")
		}
	}
}
