package interop

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"strings"
	"testing"
	"time"
)

func TestReceiverPinnedTransport(t *testing.T) {
	cert := pinnedLeaf(t)
	pin := sha256.Sum256(cert.Certificate[0])
	digest := hex.EncodeToString(pin[:])
	for _, tc := range []struct {
		name, host, pin string
		version         uint16
		accept          bool
	}{
		{"dns", "example.com", digest, tls.VersionTLS13, true},
		{"ipv4", "127.0.0.1", digest, tls.VersionTLS13, true},
		{"wrong pin", "127.0.0.1", strings.Repeat("01", 32), tls.VersionTLS13, false},
		{"wrong host", "other.invalid", digest, tls.VersionTLS13, false},
		{"downgrade", "127.0.0.1", digest, tls.VersionTLS12, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, err := tls.Listen("tcp4", "127.0.0.1:0", &tls.Config{MinVersion: tc.version, MaxVersion: tc.version, Certificates: []tls.Certificate{cert}, SessionTicketsDisabled: true})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = l.Close() }()
			done := make(chan bool, 1)
			go func() {
				c, e := l.Accept()
				if e != nil {
					done <- false
					return
				}
				defer func() { _ = c.Close() }()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				var frame [5]byte
				_, e = io.ReadFull(c, frame[:])
				if e != nil || string(frame[:]) != "frame" {
					done <- false
					return
				}
				_, e = c.Write([]byte("ack"))
				done <- e == nil
			}()
			h, p, _ := net.SplitHostPort(l.Addr().String())
			cmd := exec.Command(os.Getenv("EP_TLS_CLI"), h, p, tc.host, tc.pin)
			b, err := cmd.CombinedOutput()
			_ = l.Close()
			if (<-done) != tc.accept || (err == nil) != tc.accept {
				t.Fatalf("want success=%v, client %v %s", tc.accept, err, b)
			}
		})
	}
}

func pinnedLeaf(t *testing.T) tls.Certificate {
	t.Helper()
	dir := t.TempDir()
	reg := filepath.Join(dir, "device.json")
	if err := hostprovision.SaveEnrollment(reg, hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "UTC"}); err != nil {
		t.Fatal(err)
	}
	mobile := filepath.Join(dir, "mobile")
	cmd := exec.Command(os.Getenv("EP_PROVISION_CLI"), "-registry", reg, "mobile-init", mobile, "example.com", "127.0.0.1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	pair, err := tls.LoadX509KeyPair(filepath.Join(mobile, "device-tls.crt"), filepath.Join(mobile, "device-tls.key"))
	if err != nil {
		t.Fatal(err)
	}
	return pair
}
