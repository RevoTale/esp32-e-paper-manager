package screenhub

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestLatestAuthenticatedPeerReplacesOnlyQueuedPeer(t *testing.T) {
	f := deviceTest(t, 1)
	first := f.offer(t, 0)
	second := f.offer(t, 0)
	awaitPeer(t, first)
	f.hub.mu.Lock()
	count := len(f.hub.sockets)
	f.hub.mu.Unlock()
	if count != 1 {
		t.Fatal("queued sockets unbounded", count)
	}
	ready, err := f.hub.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Millisecond)
	still, err := f.hub.WaitReady(context.Background())
	if err != nil || still != ready {
		t.Fatal("readiness renewed cooldown", still, ready, err)
	}
	f.hub.disconnect()
	awaitPeer(t, second)
}

func TestProfileMismatchAndInvalidFrameAreFatal(t *testing.T) {
	f := deviceTest(t, 1)
	f.hub.config.ProfileVersion = 2
	done := f.offer(t, 0)
	if _, err := f.hub.WaitReady(context.Background()); !errors.Is(err, ErrProfile) {
		t.Fatal(err)
	}
	awaitPeer(t, done)
	f.hub.config.ProfileVersion = 1
	done = f.offer(t, 0)
	ready, err := f.hub.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.now = ready.NotBefore
	err = f.hub.Send(context.Background(), display.Frame{})
	if !errors.Is(err, screenwire.ErrRecord) || errors.Is(err, screendelivery.ErrTransportLost) {
		t.Fatal(err)
	}
	awaitPeer(t, done)
	if f.sink.commits.Load() != 0 {
		t.Fatal("invalid frame reached panel")
	}
}

func TestTransientHelloFailureWaitsForAnotherPeer(t *testing.T) {
	f := deviceTest(t, 1)
	done := f.offer(t, screenwire.Hello)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := f.hub.WaitReady(ctx); result <- err }()
	awaitPeer(t, done)
	done = f.offer(t, 0)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	f.hub.disconnect()
	awaitPeer(t, done)
}

func TestNoRetryForProtocolErrorsAndCanceledSend(t *testing.T) {
	if transient(screenwire.ErrRecord) || transient(securetransport.ErrAuthentication) || transient(io.ErrNoProgress) {
		t.Fatal("protocol error retryable")
	}
	for _, err := range []error{io.EOF, io.ErrUnexpectedEOF, io.ErrClosedPipe, net.ErrClosed} {
		if !transient(err) {
			t.Fatal(err)
		}
	}
	f := deviceTest(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.hub.Send(ctx, f.frame); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := f.hub.Offer(ctx, nil); !errors.Is(err, securetransport.ErrConfig) {
		t.Fatal(err)
	}
	if err := f.hub.Serve(ctx, nil); !errors.Is(err, securetransport.ErrConfig) {
		t.Fatal(err)
	}
}

func TestProtocolFailureWinsOverJoinedCloseFailure(t *testing.T) {
	for _, err := range []error{screenwire.ErrRecord, screenclient.ErrResync, screenclient.ErrPending,
		securetransport.ErrAuthentication, securetransport.ErrBounds, securetransport.ErrConfig, securetransport.ErrSequence,
		screenclient.RemoteError{Status: screenwire.Status{Code: screenwire.CodeHardware}}} {
		if transient(errors.Join(err, net.ErrClosed)) {
			t.Fatal("cleanup hid protocol failure", err)
		}
	}
}

func TestLookupNeverReturnsAnotherDeviceKey(t *testing.T) {
	h, _ := New(testConfig())
	t.Cleanup(func() { _ = h.Close() })
	key, err := h.lookup(securetransport.DeviceID{9})
	if !errors.Is(err, securetransport.ErrAuthentication) || key != (securetransport.Key{}) {
		t.Fatal(key, err)
	}
	a, b := net.Pipe()
	_ = b.Close()
	if err := h.enqueue(&peer{socket: a}); err != nil {
		t.Fatal(err)
	}
	_ = h.Close()
	a, b = net.Pipe()
	defer func() { _ = b.Close() }()
	if err := h.enqueue(&peer{socket: a}); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
}

func TestServeClosesListenerAndBoundedHandshakeOnCancellation(t *testing.T) {
	h, _ := New(testConfig())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- h.Serve(ctx, listener) }()
	socket, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = socket.Close() }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve leaked")
	}
}
