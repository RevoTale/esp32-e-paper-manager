package screenusbhost

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestProxyFailureReportsStageWithoutPrivateCause(t *testing.T) {
	old := openESP32Port
	t.Cleanup(func() { openESP32Port = old })
	openESP32Port = func(string) (serialPort, error) { return nil, errors.New("private-port-secret") }
	err := SerialProxy("esp32:private-port-secret", bytes.NewReader(nil), io.Discard)
	if ProxyExitCode(err) != 20 || strings.Contains(err.Error(), "private-port-secret") {
		t.Fatalf("missing safe open-stage diagnostic: %v", err)
	}
}

func TestProxyExchangeFailureCategories(t *testing.T) {
	r := screenwire.Record{Kind: screenwire.Abort, Epoch: 1}
	for _, tc := range []struct {
		name string
		port io.ReadWriter
		code int
	}{
		{"write", faultIO{err: io.ErrClosedPipe}, 22},
		{"read", &proxyPort{}, 23},
		{"reply", &proxyPort{input: *bytes.NewBuffer(make([]byte, screenwire.HeaderSize))}, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := serveProxy(bytes.NewReader(encoded(t, r)), io.Discard, tc.port)
			if ProxyExitCode(err) != tc.code {
				t.Fatalf("code=%d want=%d: %v", ProxyExitCode(err), tc.code, err)
			}
		})
	}
}

func TestSubprocessEOFRetainsSafeFailureCategory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	w, err := startProcess(executable, "diagnostic", make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	var b [1]byte
	_, err = w.Read(b[:])
	if !errors.Is(err, io.EOF) || !strings.Contains(err.Error(), "serial-read") || strings.Contains(err.Error(), "private-secret") {
		t.Fatalf("lost or unsafe worker diagnostic: %v", err)
	}
}

func TestProxyExitCodeIsFixedAndPreservesLocalCause(t *testing.T) {
	if ProxyExitCode(nil) != 0 || ProxyExitCode(io.EOF) != 1 {
		t.Fatal("legacy exit codes changed")
	}
	for code := 20; code <= 24; code++ {
		err := proxyFailure{code, io.EOF}
		if ProxyExitCode(errors.Join(ErrWorker, err)) != code || !errors.Is(err, io.EOF) {
			t.Fatal("category or local cause lost", code)
		}
		if strings.Contains(err.Error(), "unknown") {
			t.Fatal(err)
		}
	}
	if !strings.Contains(proxyStage(99), "without a known diagnostic") {
		t.Fatal("unrecognized exit")
	}
}

func TestProcessEOFAwaitsOnlyBoundedExitPublication(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	p := &process{output: r, done: make(chan struct{})}
	var data [1]byte
	if _, err = p.Read(data[:]); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	close(p.done) // A normal successful worker exit must not fabricate a failure.
	if _, err = p.Read(data[:]); err != io.EOF {
		t.Fatal(err)
	}
}
