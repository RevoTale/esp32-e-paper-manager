package screenusbhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type idleUARTWorker struct {
	*testWorker
	kinds []screenwire.Kind
	binds [][]byte
}

func (w *idleUARTWorker) Write(data []byte) (int, error) {
	record, err := screenwire.Decode(data)
	if err != nil {
		return 0, err
	}
	w.kinds = append(w.kinds, record.Kind)
	if record.Kind == screenwire.Bind {
		w.binds = append(w.binds, bytes.Clone(record.Payload))
	}
	return w.testWorker.Write(data)
}

func (w *idleUARTWorker) retire(t *testing.T, device *screenlink.Device) {
	t.Helper()
	if err := w.link.Disconnect(); err != nil {
		t.Fatal(err)
	}
	w.link = device.Open() // UART stays open while the device retires its logical binding.
	w.kinds = nil
}

func TestESP32RebindsRetiredUARTBeforeReadinessAndSend(t *testing.T) {
	s, device, sink, frame := usbFixture(t)
	s.rebind = true
	w := &idleUARTWorker{testWorker: &testWorker{link: device.Open(), done: make(chan struct{})}}
	s.start = func() (worker, error) { return w, nil }
	ready, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	w.retire(t, device)
	got, err := s.WaitReady(context.Background())
	if err != nil || !got.NotBefore.Equal(ready.NotBefore) || sink.commits != 0 {
		t.Fatal("readiness changed floor or refreshed", err)
	}
	if !reflect.DeepEqual(w.kinds, []screenwire.Kind{screenwire.Hello, screenwire.Bind}) {
		t.Fatal("readiness did more than rebind", w.kinds)
	}
	w.retire(t, device)
	s.now = func() time.Time { return ready.NotBefore }
	if err := s.Send(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	if len(w.kinds) < 3 || !reflect.DeepEqual(w.kinds[:3], []screenwire.Kind{screenwire.Hello, screenwire.Bind, screenwire.Begin}) {
		t.Fatal("send did not rebind before pixels", w.kinds)
	}
	checkRebindEvidence(t, w, sink)
}

func checkRebindEvidence(t *testing.T, w *idleUARTWorker, sink *testSink) {
	t.Helper()
	if sink.commits != 1 || len(w.binds) != 3 || !bytes.Equal(w.binds[0], w.binds[1]) || !bytes.Equal(w.binds[0], w.binds[2]) {
		t.Fatal("rebind changed claim or replayed")
	}
}

func TestESP32LostCommitReconcilesWithoutFrameReplay(t *testing.T) {
	s, device, sink, frame := usbFixture(t)
	s.rebind = true
	starts := 0
	s.start = func() (worker, error) {
		w := &testWorker{link: device.Open(), done: make(chan struct{})}
		if starts == 0 {
			w.drop = screenwire.Commit
		}
		starts++
		return w, nil
	}
	s.retry = func(context.Context) error { return nil }
	ready, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return ready.NotBefore }
	if err := s.Send(context.Background(), frame); !errors.Is(err, screendelivery.ErrTransportLost) {
		t.Fatal(err)
	}
	ready, err = s.WaitReady(context.Background())
	if err != nil || ready.Pending != screendelivery.PendingConfirmed || sink.commits != 1 {
		t.Fatal("uncertain frame replayed or lost", err, sink.commits)
	}
}

type timedESP32Port struct {
	proxyPort
	timeout time.Duration
}

func (p *timedESP32Port) SetReadTimeout(timeout time.Duration) error {
	p.timeout = timeout
	return p.err
}

func TestESP32ProxyUsesDedicatedOpenerAndNeverSetsDTR(t *testing.T) {
	originalESP, originalPico := openESP32Port, openPort
	t.Cleanup(func() { openESP32Port, openPort = originalESP, originalPico })
	openPort = func(string) (serialPort, error) { t.Fatal("Pico opener used"); return nil, ErrWorker }
	for _, timeoutErr := range []error{nil, io.ErrClosedPipe} {
		p := &timedESP32Port{proxyPort: proxyPort{err: timeoutErr}}
		openESP32Port = func(name string) (serialPort, error) {
			if name != "/dev/example" {
				t.Fatal("ESP32 prefix reached serial opener")
			}
			return p, nil
		}
		err := SerialProxy("esp32:/dev/example", bytes.NewReader(nil), io.Discard)
		if (err != nil) != (timeoutErr != nil) || !p.closed || len(p.dtr) != 0 || p.timeout != 180*time.Second {
			t.Fatal("proxy lifecycle or modem-line policy", err)
		}
	}
}

func TestOnlyExplicitESP32EndpointEnablesRebinding(t *testing.T) {
	for _, port := range []string{"/dev/example", "esp32:/dev/example"} {
		s, err := New("unused-proxy", port, display.Size{Width: 8, Height: 1})
		if err != nil {
			t.Fatal(err)
		}
		if s.rebind != (port == "esp32:/dev/example") {
			t.Fatal("unexpected transport rebind policy")
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	s, device, _, _ := usbFixture(t)
	w := &idleUARTWorker{testWorker: &testWorker{link: device.Open(), done: make(chan struct{})}}
	s.start = func() (worker, error) { return w, nil }
	if _, err := s.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	w.kinds = nil
	if _, err := s.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.kinds, []screenwire.Kind{screenwire.Hello}) {
		t.Fatal("Pico readiness began rebinding", w.kinds)
	}
}
