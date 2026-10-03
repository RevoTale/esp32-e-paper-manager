package interop

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/screenhub"
	"github.com/RevoTale/esp32-e-paper-manager/screenpeer"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

type nativeRegionBridge struct {
	done chan struct{}
	err  error
}

func offerNativeRegion(t *testing.T, hub *screenhub.Hub, config screenhub.Config, lifetime *network.Lifetime, peer *nativePeer, drop bool) *nativeRegionBridge {
	t.Helper()
	host, device := net.Pipe()
	b := &nativeRegionBridge{done: make(chan struct{})}
	go func() {
		defer close(b.done)
		defer func() { _ = device.Close() }()
		records, err := screenpeer.Device(device, config.Key, config.ID, lifetime, time.Now())
		if err == nil {
			err = bridgeRegionRecords(device, records, peer, drop)
		}
		b.err = err
	}()
	t.Cleanup(func() { _ = host.Close(); _ = device.Close(); b.wait(t) })
	if err := hub.Offer(t.Context(), host); err != nil {
		t.Fatal(err)
	}
	return b
}

func (b *nativeRegionBridge) wait(t *testing.T) {
	t.Helper()
	select {
	case <-b.done:
		if !errors.Is(b.err, io.EOF) && !errors.Is(b.err, io.ErrUnexpectedEOF) && !errors.Is(b.err, net.ErrClosed) && !errors.Is(b.err, io.ErrClosedPipe) {
			t.Errorf("encrypted bridge failed: %v", b.err)
		}
	case <-time.After(time.Second):
		t.Error("encrypted bridge did not stop")
	}
}

// Only EPN2 socket/pipe adaptation is modeled. EPS2 admission, hashes, old/new
// streaming and panel I/O run in the actual native receiver/panel implementation.
func bridgeRegionRecords(socket net.Conn, records *securetransport.RecordStream, peer *nativePeer, drop bool) error {
	var request, reply [screenwire.MaxRecord]byte
	limits := securetransport.ReadLimits{Idle: time.Second, Active: time.Second, SetDeadline: socket.SetDeadline}
	for {
		n, err := records.ReadRecord(request[:], limits)
		if err != nil {
			return err
		}
		r, err := screenwire.Decode(request[:n])
		if err != nil {
			return err
		}
		if _, err = peer.input.Write(request[:n]); err != nil {
			return err
		}
		n, err = nativeReply(peer.output, reply[:])
		if err != nil {
			return err
		}
		if drop && r.Kind == screenwire.Commit && r.ID == 2 {
			return io.ErrUnexpectedEOF
		}
		if _, err = records.Write(reply[:n]); err != nil {
			return err
		}
	}
}

func nativeReply(reader io.Reader, reply []byte) (int, error) {
	if _, err := io.ReadFull(reader, reply[:screenwire.HeaderSize]); err != nil {
		return 0, err
	}
	n, err := screenwire.Size(reply[:screenwire.HeaderSize])
	if err != nil {
		return 0, err
	}
	if _, err = io.ReadFull(reader, reply[screenwire.HeaderSize:n]); err != nil {
		return 0, err
	}
	_, err = screenwire.Decode(reply[:n])
	return n, err
}
