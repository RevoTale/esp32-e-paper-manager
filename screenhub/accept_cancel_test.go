package screenhub

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// Forces closure after Accept succeeds but before Serve reserves the socket.
type closingListener struct {
	accept func() (net.Conn, error)
}

func (l closingListener) Accept() (net.Conn, error) { return l.accept() }
func (closingListener) Close() error                { return nil }
func (closingListener) Addr() net.Addr              { return &net.TCPAddr{} }

func TestServeClosureDuringAdmissionPreservesCause(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "hub_closed"
		if canceled {
			name = "context_canceled"
		}
		t.Run(name, func(t *testing.T) {
			h, err := New(testConfig())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a, b := net.Pipe()
			defer func() { _ = a.Close(); _ = b.Close() }()
			if err := b.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			listener := closingListener{accept: func() (net.Conn, error) {
				if canceled {
					cancel()
				}
				if err := h.Close(); err != nil {
					t.Fatal(err)
				}
				return a, nil
			}}
			want := net.ErrClosed
			if canceled {
				want = context.Canceled
			}
			if err := h.Serve(ctx, listener); !errors.Is(err, want) {
				t.Fatalf("Serve = %v, want %v", err, want)
			}
			if _, err := b.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
				t.Fatalf("unregistered socket not closed: %v", err)
			}
		})
	}
}
