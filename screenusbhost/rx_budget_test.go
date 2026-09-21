package screenusbhost

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestUSBSenderFitsRXBurstAndDeliversExactTwoPlanes(t *testing.T) {
	for _, kind := range []string{"raw_noise", "packed_mixed", "packed_solid"} {
		t.Run(kind, func(t *testing.T) {
			s, port, sink, frame := rxBudgetFixture(t, kind)
			ready, err := s.WaitReady(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			s.now = func() time.Time { return ready.NotBefore }
			if err = s.Send(context.Background(), frame); err != nil {
				t.Fatalf("send failed: %v; largest USB burst=%d dropped=%d raw=%d packed=%d commits=%d",
					err, port.largest, port.dropped, port.raw, port.packed, sink.commits)
			}
			assertRXBudgetDelivery(t, port, sink, frame, kind)
		})
	}
}

func assertRXBudgetDelivery(t *testing.T, port *rxBudgetPort, sink *rxBudgetSink, frame display.Frame, kind string) {
	t.Helper()
	if port.dropped != 0 || port.largest > len(port.rx) {
		t.Fatalf("RX overflow: largest=%d capacity=%d dropped=%d", port.largest, len(port.rx), port.dropped)
	}
	for pass, plane := range sink.planes {
		if !bytes.Equal(plane, frame.Bytes()) {
			t.Fatalf("pass %d differs: got %d bytes, want %d", pass, len(plane), len(frame.Bytes()))
		}
	}
	if sink.commits != 1 {
		t.Fatalf("commits=%d, want 1", sink.commits)
	}
	if kind != "packed_solid" && port.raw == 0 || kind != "raw_noise" && port.packed == 0 {
		t.Fatalf("fixture missed its encoding path: raw=%d packed=%d", port.raw, port.packed)
	}
}

func rxBudgetFixture(t *testing.T, kind string) (*Sender, *rxBudgetPort, *rxBudgetSink, display.Frame) {
	t.Helper()
	size := display.Size{Width: 800, Height: 480}
	s, err := New("worker", "port", size)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	sink := &rxBudgetSink{}
	caps := screenwire.Capabilities{Width: 800, Height: 480, Stride: 100, MaxChunk: 1000, Passes: 2,
		Format: screenwire.Mono1, Features: screenwire.RawFull, Profile: 99, ProfileVersion: 2, MinimumFullMS: 1000}
	if kind != "raw_noise" {
		caps.Features |= screenwire.FeaturePackBits
	}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	port := &rxBudgetPort{testWorker: &testWorker{link: d.Open(), done: make(chan struct{})}}
	s.start = func() (worker, error) { return &rxBudgetWorker{port: port}, nil }
	s.now = func() time.Time { return time.Unix(1000, 0) }
	pixels := make([]byte, 48000)
	for i := range pixels {
		if kind == "raw_noise" || kind == "packed_mixed" && (i/1000)%2 == 1 {
			pixels[i] = byte((i*37 + i/251) % 256)
		}
	}
	frame, err := display.NewFrame(size, 100, pixels)
	if err != nil {
		t.Fatal(err)
	}
	return s, port, sink, frame
}

type rxBudgetSink struct {
	testSink
	planes [2][]byte
}

func (s *rxBudgetSink) Write(pass uint8, offset uint32, data []byte) error {
	if int(pass) >= len(s.planes) || int(offset) != len(s.planes[pass]) {
		return fmt.Errorf("unexpected sink coordinates: pass=%d offset=%d", pass, offset)
	}
	s.planes[pass] = append(s.planes[pass], data...)
	return nil
}

type rxBudgetPort struct {
	*testWorker
	rx                            [512]byte
	largest, dropped, raw, packed int
}

func (p *rxBudgetPort) Write(data []byte) (int, error) {
	r, err := screenwire.Decode(data)
	if err != nil {
		return 0, err
	}
	switch r.Kind {
	case screenwire.Data:
		p.raw++
	case screenwire.DataPacked:
		p.packed++
	}
	p.largest = max(p.largest, len(data))
	// TinyGo 0.42.0 machine/usb/cdc/{usbcdc,ring}.go: cdcCallbackRx copies
	// only min(packet length, ring512.Free()), dropping the remaining bytes.
	// Model a complete burst before the owner drains RX; the host still
	// reports every byte written.
	n := copy(p.rx[:], data)
	p.dropped += len(data) - n
	err = p.link.Push(p.rx[:n], time.Hour, func(reply []byte) error {
		_, err := p.input.Write(reply)
		return err
	})
	return len(data), err
}

// Keep the real stop-and-wait proxy between Sender and the bounded USB port.
// An incomplete record yields no reply; bytes.Buffer returns EOF immediately
// instead of waiting for a physical serial timeout. No sleep or retry is used.
type rxBudgetWorker struct {
	port   *rxBudgetPort
	output bytes.Buffer
}

func (w *rxBudgetWorker) Done() <-chan struct{}      { return w.port.Done() }
func (w *rxBudgetWorker) Close() error               { return w.port.Close() }
func (w *rxBudgetWorker) Read(p []byte) (int, error) { return w.output.Read(p) }
func (w *rxBudgetWorker) Write(p []byte) (int, error) {
	var scratch [screenwire.MaxRecord]byte
	if len(p) > len(scratch) {
		return 0, screenwire.ErrRecord
	}
	n := copy(scratch[:], p)
	if err := proxyExchange(w.port, &w.output, scratch[:], n); err != nil {
		return 0, err
	}
	return n, nil
}
