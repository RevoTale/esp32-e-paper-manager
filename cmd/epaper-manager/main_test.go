package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestConfigRequiresAllSecretAndTLSPaths(t *testing.T) {
	_, err := parseConfig([]string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", "token",
		"-enrollment", "device", "-state", "state"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parseConfig([]string{"-tls-cert", "cert"}); err == nil {
		t.Fatal("incomplete config accepted")
	}
}

func TestConfigRejectsInvalidFlagsAndArguments(t *testing.T) {
	complete := []string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", "token",
		"-enrollment", "device", "-state", "state"}
	if _, err := parseConfig(append(complete, "unexpected")); err == nil {
		t.Fatal("positional argument accepted")
	}
	if _, err := parseConfig([]string{"-unknown"}); err == nil {
		t.Fatal("unknown flag accepted")
	}
}

func TestServeDeviceRejectsUnknownIdentity(t *testing.T) {
	registry, _ := manager.NewStaticRegistry(nil)
	serverSide, deviceSide := net.Pipe()
	done := make(chan struct{})
	go func() { serveDevice(serverSide, registry, manager.NewStore()); close(done) }()
	id := securetransport.DeviceID{9}
	_, _ = deviceSide.Write(id[:])
	_ = deviceSide.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("serveDevice did not stop")
	}
}

func TestServeDevicesStopsWhenListenerFails(t *testing.T) {
	registry, _ := manager.NewStaticRegistry(nil)
	listener := &failingListener{}
	serveDevices(listener, registry, manager.NewStore())
	if listener.calls != 1 {
		t.Fatalf("calls=%d", listener.calls)
	}
}

type failingListener struct{ calls int }

func (listener *failingListener) Accept() (net.Conn, error) {
	listener.calls++
	return nil, errors.New("stop")
}
func (*failingListener) Close() error   { return nil }
func (*failingListener) Addr() net.Addr { return fakeAddress("test") }

type fakeAddress string

func (address fakeAddress) Network() string { return string(address) }
func (address fakeAddress) String() string  { return string(address) }

func TestRunReachesNetworkConfigurationWithoutExposingSecrets(t *testing.T) {
	directory := t.TempDir()
	token := filepath.Join(directory, "token")
	enrollment := filepath.Join(directory, "device.json")
	state := filepath.Join(directory, "state.json")
	if err := os.WriteFile(token, []byte(strings.Repeat("t", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv"})
	if err := os.WriteFile(enrollment, record, 0o600); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", token,
		"-enrollment", enrollment, "-state", state, "-device-listen", "bad:::address"})
	if err == nil {
		t.Fatal("invalid listen address accepted")
	}
}

func TestReadSecretRequiresPrivateBoundedToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 32)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := readSecret(path)
	if err != nil || len(value) != 32 {
		t.Fatalf("length=%d err=%v", len(value), err)
	}
	if err = os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err = readSecret(path); err == nil {
		t.Fatal("public secret file accepted")
	}
}

func TestRunConfiguresTLS13AndDualListeners(t *testing.T) {
	directory := t.TempDir()
	token := filepath.Join(directory, "token")
	enrollment := filepath.Join(directory, "device.json")
	state := filepath.Join(directory, "state.json")
	if err := os.WriteFile(token, []byte(strings.Repeat("t", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv"})
	if err := os.WriteFile(enrollment, record, 0o600); err != nil {
		t.Fatal(err)
	}
	originalListen, originalTLS := listenTCP, serveTLS
	defer func() { listenTCP, serveTLS = originalListen, originalTLS }()
	listenTCP = func(network, address string) (net.Listener, error) {
		if network != "tcp" || address != "[::]:9757" {
			return nil, errors.New("unexpected listener")
		}
		return &failingListener{}, nil
	}
	want := errors.New("stop TLS")
	serveTLS = func(server *http.Server, certificate, key string) error {
		if server.TLSConfig.MinVersion != tls.VersionTLS13 || certificate != "cert" || key != "key" {
			t.Fatal("TLS policy not enforced")
		}
		return want
	}
	err := run([]string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", token,
		"-enrollment", enrollment, "-state", state})
	if !errors.Is(err, want) {
		t.Fatalf("run=%v", err)
	}
}

func TestReadSecretRejectsShortAndOversizedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	for _, value := range []string{"short", strings.Repeat("x", 4097)} {
		if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readSecret(path); err == nil {
			t.Fatalf("accepted length=%d", len(value))
		}
	}
}

func TestReadSecretRejectsMissingPathAndDirectory(t *testing.T) {
	directory := t.TempDir()
	if _, err := readSecret(filepath.Join(directory, "missing")); err == nil {
		t.Fatal("missing secret accepted")
	}
	if _, err := readSecret(directory); err == nil {
		t.Fatal("directory accepted as a secret")
	}
}

func TestRunRejectsMissingEnrollmentAndPublicState(t *testing.T) {
	directory := t.TempDir()
	token := filepath.Join(directory, "token")
	enrollment := filepath.Join(directory, "device.json")
	state := filepath.Join(directory, "state.json")
	if err := os.WriteFile(token, []byte(strings.Repeat("t", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := []string{"-tls-cert", "cert", "-tls-key", "key", "-token-file", token,
		"-enrollment", enrollment, "-state", state}
	if err := run(arguments); err == nil {
		t.Fatal("missing enrollment accepted")
	}
	record, _ := json.Marshal(hostprovision.Enrollment{DeviceID: [16]byte{1}, DeviceKey: [32]byte{2}, Timezone: "Europe/Kyiv"})
	if err := os.WriteFile(enrollment, record, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte(`{"devices":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(arguments); err == nil {
		t.Fatal("public state accepted")
	}
}
