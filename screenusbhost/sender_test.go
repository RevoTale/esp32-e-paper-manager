package screenusbhost

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type testSink struct{ commits int }

func (*testSink) Begin() error                      { return nil }
func (*testSink) Write(uint8, uint32, []byte) error { return nil }
func (s *testSink) Commit() error                   { s.commits++; return nil }
func (*testSink) Abort() error                      { return nil }

type testWorker struct {
	link  *screenlink.Connection
	input bytes.Buffer
	done  chan struct{}
	drop  screenwire.Kind
}

func (w *testWorker) Done() <-chan struct{}      { return w.done }
func (w *testWorker) Read(p []byte) (int, error) { return w.input.Read(p) }
func (w *testWorker) Close() error {
	select {
	case <-w.done:
		return nil
	default:
		close(w.done)
	}
	return w.link.Disconnect()
}
func (w *testWorker) Write(p []byte) (int, error) {
	r, err := screenwire.Decode(p)
	if err != nil {
		return 0, err
	}
	err = w.link.Push(p, time.Hour, func(reply []byte) error {
		if r.Kind == w.drop {
			return io.ErrUnexpectedEOF
		}
		_, err := w.input.Write(reply)
		return err
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func usbFixture(t *testing.T) (*Sender, *screenlink.Device, *testSink, display.Frame) {
	t.Helper()
	size := display.Size{Width: 17, Height: 9}
	s, err := New("worker", "port", size)
	if err != nil {
		t.Fatal(err)
	}
	sink := &testSink{}
	caps := screenwire.Capabilities{Width: 17, Height: 9, Stride: 3, MaxChunk: 10, Passes: 2,
		Format: screenwire.Mono1, Features: screenwire.RawFull, Profile: 99, ProfileVersion: 2, MinimumFullMS: 1000}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.start = func() (worker, error) { return &testWorker{link: d.Open(), done: make(chan struct{})}, nil }
	s.now = func() time.Time { return time.Unix(1000, 0) }
	f, err := display.NewFrame(size, 3, bytes.Repeat([]byte{0x80}, 27))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, d, sink, f
}

func TestUSBRetainsClientAndCooldownAcrossReconnect(t *testing.T) {
	s, _, sink, frame := usbFixture(t)
	r, err := s.WaitReady(context.Background())
	if err != nil || r.Pending != screendelivery.NoPending || !r.NotBefore.Equal(s.now().Add(time.Second)) {
		t.Fatal(r, err)
	}
	if err = s.Send(context.Background(), frame); !errors.Is(err, ErrCooldown) {
		t.Fatal(err)
	}
	s.now = func() time.Time { return r.NotBefore }
	if err = s.Send(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	floor := s.floor
	if err = s.disconnect(); err != nil {
		t.Fatal(err)
	}
	r, err = s.WaitReady(context.Background())
	if err != nil || !r.NotBefore.Equal(floor) || sink.commits != 1 {
		t.Fatal(r, err, sink.commits)
	}
}

func TestUSBLostRepliesReconcileWithoutReusingFrame(t *testing.T) {
	for _, kind := range []screenwire.Kind{screenwire.Commit, screenwire.Data} {
		t.Run(string(rune('A'+kind)), func(t *testing.T) {
			s, d, sink, frame := usbFixture(t)
			count := 0
			s.start = func() (worker, error) {
				w := &testWorker{link: d.Open(), done: make(chan struct{})}
				if count == 0 {
					w.drop = kind
				}
				count++
				return w, nil
			}
			s.retry = func(context.Context) error { return nil }
			r, err := s.WaitReady(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			s.now = func() time.Time { return r.NotBefore }
			if err = s.Send(context.Background(), frame); !errors.Is(err, screendelivery.ErrTransportLost) {
				t.Fatal(err)
			}
			r, err = s.WaitReady(context.Background())
			want, commits := screendelivery.PendingUnconfirmed, 0
			if kind == screenwire.Commit {
				want, commits = screendelivery.PendingConfirmed, 1
			}
			if err != nil || r.Pending != want || sink.commits != commits {
				t.Fatal(r, err, sink.commits)
			}
		})
	}
}

func TestUSBLostAcquireRetainsPrivateClaim(t *testing.T) {
	s, d, sink, _ := usbFixture(t)
	count := 0
	s.start = func() (worker, error) {
		w := &testWorker{link: d.Open(), done: make(chan struct{})}
		if count == 0 {
			w.drop = screenwire.Acquire
		}
		count++
		return w, nil
	}
	s.retry = func(context.Context) error { return nil }
	if _, err := s.WaitReady(context.Background()); err != nil || count != 2 || sink.commits != 0 {
		t.Fatal(err, count, sink.commits)
	}
}
