package screenhub

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenpeer"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

type panelSink struct {
	commits atomic.Int32
	planes  [2][]byte
}

func (p *panelSink) Begin() error { p.planes = [2][]byte{}; return nil }
func (p *panelSink) Write(pass uint8, _ uint32, data []byte) error {
	p.planes[pass] = append(p.planes[pass], data...)
	return nil
}
func (p *panelSink) Commit() error { p.commits.Add(1); return nil }
func (p *panelSink) Abort() error  { return nil }

type deviceFixture struct {
	hub      *Hub
	device   *screenlink.Device
	sink     *panelSink
	lifetime *network.Lifetime
	now      time.Time
	frame    display.Frame
}

func deviceTest(t *testing.T, boot byte) *deviceFixture {
	t.Helper()
	config := testConfig()
	h, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	f := &deviceFixture{hub: h, sink: &panelSink{}, now: time.Unix(1000, 0)}
	h.now = func() time.Time { return f.now }
	caps := screenwire.Capabilities{Width: 17, Height: 9, Stride: 3, MaxChunk: 10, Passes: 2,
		Format: screenwire.Mono1, Features: screenwire.RawFull, Profile: 1, ProfileVersion: 1,
		MinimumFullMS: 1000, DeviceID: config.ID}
	f.device, err = screenlink.New(caps, [16]byte{boot}, f.sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.lifetime, err = network.NewLifetime([8]byte{1}, uint64(boot))
	if err != nil {
		t.Fatal(err)
	}
	f.frame, err = display.NewFrame(config.Size, 3, bytes.Repeat([]byte{0x80}, 27))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return f
}

func (f *deviceFixture) offer(t *testing.T, drop screenwire.Kind) <-chan error {
	t.Helper()
	host, device := net.Pipe()
	done := make(chan error, 1)
	go func() {
		defer func() { _ = device.Close() }()
		records, err := screenpeer.Device(device, testConfig().Key, testConfig().ID, f.lifetime, time.Now())
		if err == nil {
			err = f.loop(device, records, drop)
		}
		done <- err
	}()
	if err := f.hub.Offer(context.Background(), host); err != nil {
		t.Fatal(err)
	}
	return done
}

func (f *deviceFixture) loop(socket net.Conn, records *securetransport.RecordStream, drop screenwire.Kind) error {
	connection := f.device.Open()
	defer func() { _ = connection.Disconnect() }()
	var data [screenwire.MaxRecord]byte
	limits := securetransport.ReadLimits{Idle: time.Second, Active: time.Second, SetDeadline: socket.SetDeadline}
	for {
		n, err := records.ReadRecord(data[:], limits)
		if err != nil {
			return err
		}
		request, err := screenwire.Decode(data[:n])
		if err != nil {
			return err
		}
		err = connection.Push(data[:n], 10*time.Second, func(reply []byte) error {
			if request.Kind == drop {
				return io.ErrUnexpectedEOF
			}
			_, err := records.Write(reply)
			return err
		})
		if err != nil {
			return err
		}
	}
}

func awaitPeer(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("peer leaked")
	}
}
