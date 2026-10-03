package main

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/manager"
)

type queuedListener struct {
	connections chan net.Conn
	accepted    chan struct{}
	closed      chan struct{}
	once        sync.Once
}

func (l *queuedListener) Accept() (net.Conn, error) {
	select {
	case connection := <-l.connections:
		l.accepted <- struct{}{}
		return connection, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *queuedListener) Close() error { l.once.Do(func() { close(l.closed) }); return nil }
func (*queuedListener) Addr() net.Addr { return fakeAddress("queued") }

func TestLegacyDeviceHandshakeAdmissionIsBounded(t *testing.T) {
	listener := &queuedListener{connections: make(chan net.Conn, 5), accepted: make(chan struct{}, 5), closed: make(chan struct{})}
	registry, err := manager.NewStaticRegistry(nil)
	if err != nil {
		t.Fatal(err)
	}
	var peers []net.Conn
	for range 5 {
		server, client := net.Pipe()
		listener.connections <- server
		peers = append(peers, server, client)
	}
	done := make(chan struct{})
	go func() { serveDevices(listener, registry, manager.NewStore()); close(done) }()
	t.Cleanup(func() { closeQueuedDevices(t, listener, peers, done) })
	for range 4 {
		select {
		case <-listener.accepted:
		case <-time.After(time.Second):
			t.Fatal("handshake not admitted")
		}
	}
	select {
	case <-listener.accepted:
		t.Fatal("fifth concurrent handshake admitted")
	case <-time.After(100 * time.Millisecond):
	}
}

func closeQueuedDevices(t *testing.T, listener *queuedListener, peers []net.Conn, done <-chan struct{}) {
	t.Helper()
	_ = listener.Close()
	for _, peer := range peers {
		_ = peer.Close()
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("legacy listener did not stop")
	}
}
