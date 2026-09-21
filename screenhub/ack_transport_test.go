package screenhub

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/screenpeer"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

type ackStream struct {
	io.ReadWriter
	io.Closer
}

type ackClose func() error

func (f ackClose) Close() error { return f() }

func authenticatedACKStream(t *testing.T, conn net.Conn, cfg Config) io.ReadWriteCloser {
	t.Helper()
	t.Cleanup(func() { _ = conn.Close() })
	records, _, err := screenpeer.Manager(conn, func(id securetransport.DeviceID) (securetransport.Key, error) {
		if id != cfg.ID {
			return securetransport.Key{}, securetransport.ErrAuthentication
		}
		return cfg.Key, nil
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	s, err := screenpeer.NewHostScreen(conn, records)
	if err != nil {
		t.Fatal(err)
	}
	return ackStream{s, conn}
}

func simulatedACKStream(t *testing.T, f *deviceFixture) io.ReadWriteCloser {
	t.Helper()
	host, device := net.Pipe()
	done := make(chan error, 1)
	go func() {
		defer func() { _ = device.Close() }()
		r, err := screenpeer.Device(device, testConfig().Key, testConfig().ID, f.lifetime, time.Now())
		if err == nil {
			err = f.loop(device, r, 0)
		}
		done <- err
	}()
	var once sync.Once
	closePeer := ackClose(func() error {
		once.Do(func() { _ = host.Close(); awaitPeer(t, done) })
		return nil
	})
	t.Cleanup(func() { _ = closePeer.Close() })
	return ackStream{authenticatedACKStream(t, host, testConfig()), closePeer}
}

func liveACKEnvironment(t *testing.T) (ackConnect, func(time.Duration), display.Frame, func()) {
	t.Helper()
	if !*liveACKConfirm || *liveACKEnrollment == "" || *liveACKListen == "" {
		t.Fatal("physical test requires enrollment, listen and explicit confirm; no device access")
	}
	e, err := hostprovision.LoadEnrollment(*liveACKEnrollment)
	if err != nil {
		t.Fatal("cannot load private enrollment")
	}
	cfg := Config{ID: securetransport.DeviceID(e.DeviceID), Key: securetransport.Key(e.DeviceKey)}
	listener, err := net.Listen("tcp", *liveACKListen)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	ctx := boundedACKContext(t)
	connect := func() io.ReadWriteCloser {
		if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
			t.Fatal(err)
		}
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		return authenticatedACKStream(t, conn, cfg)
	}
	wait := func(d time.Duration) {
		t.Logf("authenticated; conservative guard=%s; one frame only", d)
		timer := time.NewTimer(max(d, 180*time.Second))
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	return connect, wait, ackPattern(t), func() { t.Log("physical pixels still require user observation; no direct refresh-counter telemetry") }
}

func ackPattern(t *testing.T) display.Frame {
	t.Helper()
	pixels := make([]byte, 48000)
	for y := 0; y < 480; y++ {
		for x := 0; x < 100; x++ {
			if (x < 50) == (y < 240) {
				pixels[y*100+x] = 0xff
			}
		}
	}
	frame, err := display.NewFrame(display.Size{Width: 800, Height: 480}, 100, pixels)
	if err != nil {
		t.Fatal(err)
	}
	return frame
}
