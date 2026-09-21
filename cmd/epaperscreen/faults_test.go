package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestUsageFailuresAndProxyMainExit(t *testing.T) {
	for _, args := range [][]string{nil, {"--status"}, {"--serial-proxy"}, {"-bad"}, {"-width", "0", "file", "port"}} {
		if err := run(context.Background(), args, io.Discard); err == nil {
			t.Fatal(args)
		}
	}
	oldProxy, oldArgs := proxy, os.Args
	defer func() { proxy, os.Args = oldProxy, oldArgs }()
	proxy = func(port string, _ io.Reader, _ io.Writer) error {
		if port != "port" {
			t.Fatal(port)
		}
		return nil
	}
	os.Args = []string{"epaperscreen", "--serial-proxy", "port"}
	if code := mainExit(); code != 0 {
		t.Fatal(code)
	}
	os.Args = []string{"epaperscreen", "--status"}
	if code := mainExit(); code != 1 {
		t.Fatal(code)
	}
}

func TestStatusReportsOnlyNumericSnapshotAndCloses(t *testing.T) {
	s := &fakeTransport{inspect: func() (screenusbhost.Snapshot, error) {
		return screenusbhost.Snapshot{Capabilities: screenwire.Capabilities{Profile: 99, ProfileVersion: 2, Width: 17, Height: 9},
			Health: screenwire.HealthStatus{State: 7, LastFailure: 7, Failures: 1, UptimeSeconds: 42}}, nil
	}}
	restore, _ := cliFixture(t, s)
	defer restore()
	var out bytes.Buffer
	if err := run(context.Background(), []string{"--status", "port"}, &out); err != nil {
		t.Fatal(err)
	}
	if !s.closed || !strings.Contains(out.String(), "profile=99 version=2 size=17x9") || !strings.Contains(out.String(), "state=7 last_failure=7") {
		t.Fatal(out.String())
	}
	s.inspect = func() (screenusbhost.Snapshot, error) { return screenusbhost.Snapshot{}, screenusbhost.ErrWorker }
	if err := run(context.Background(), []string{"--status", "port"}, io.Discard); !errors.Is(err, screenusbhost.ErrWorker) {
		t.Fatal(err)
	}
}

func TestTransportConstructionFailures(t *testing.T) {
	old := openTransport
	defer func() { openTransport = old }()
	openTransport = func(string, display.Size) (transport, error) { return nil, screenusbhost.ErrConfiguration }
	for _, args := range [][]string{{"file", "port"}, {"--status", "port"}} {
		if err := run(context.Background(), args, io.Discard); !errors.Is(err, screenusbhost.ErrConfiguration) {
			t.Fatal(err)
		}
	}
}

func TestDeliveryReadAndRenderFailuresNeverSend(t *testing.T) {
	s := &fakeTransport{ready: func() (screendelivery.Readiness, error) { return screendelivery.Readiness{}, nil }}
	restore, path := cliFixture(t, s)
	defer restore()
	if err := run(context.Background(), []string{path + "missing", "port"}, io.Discard); !errors.Is(err, os.ErrNotExist) || !s.closed {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("<script>reject</script>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{path, "port"}, io.Discard); err == nil {
		t.Fatal("active content accepted")
	}
	s.ready = func() (screendelivery.Readiness, error) { return screendelivery.Readiness{}, context.Canceled }
	if err := run(context.Background(), []string{path, "port"}, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestReadHTMLBoundsAndRegularFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := readHTML(dir); err == nil {
		t.Fatal("directory accepted")
	}
	path := dir + "/large.html"
	if err := os.WriteFile(path, bytes.Repeat([]byte{'x'}, 32769), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readHTML(path); err == nil {
		t.Fatal("oversize accepted")
	}
}

func TestReadinessAndReconciliationFailures(t *testing.T) {
	s := &fakeTransport{ready: func() (screendelivery.Readiness, error) {
		return screendelivery.Readiness{Pending: screendelivery.PendingConfirmed}, nil
	}}
	if err := awaitReady(context.Background(), s); !errors.Is(err, ErrUnconfirmed) {
		t.Fatal(err)
	}
	s.ready = func() (screendelivery.Readiness, error) {
		return screendelivery.Readiness{NotBefore: time.Now().Add(time.Hour)}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := awaitReady(ctx, s); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	s.ready = func() (screendelivery.Readiness, error) {
		return screendelivery.Readiness{}, screendelivery.ErrTransportResync
	}
	s.send = func(display.Frame) error { return screendelivery.ErrTransportLost }
	if err := sendOnce(context.Background(), s, display.Frame{}); !errors.Is(err, ErrUnconfirmed) {
		t.Fatal(err)
	}
	s.send = func(display.Frame) error { return screendelivery.ErrTransportResync }
	if err := sendOnce(context.Background(), s, display.Frame{}); !errors.Is(err, screendelivery.ErrTransportResync) {
		t.Fatal(err)
	}
}
