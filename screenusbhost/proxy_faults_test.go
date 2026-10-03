package screenusbhost

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type setupPort struct {
	proxyPort
	stage string
}

func (p *setupPort) SetDTR(v bool) error {
	p.dtr = append(p.dtr, v)
	if p.stage == "dtr" {
		return ErrWorker
	}
	return nil
}
func (p *setupPort) SetReadTimeout(time.Duration) error {
	if p.stage == "timeout" {
		return ErrWorker
	}
	return nil
}

func TestSerialProxyLifecycleAndFailures(t *testing.T) {
	old := openPort
	defer func() { openPort = old }()
	for _, stage := range []string{"open", "timeout", "dtr", "ok"} {
		t.Run(stage, func(t *testing.T) {
			p := &setupPort{stage: stage}
			openPort = func(string) (serialPort, error) {
				if stage == "open" {
					return nil, ErrWorker
				}
				return p, nil
			}
			err := SerialProxy("port", bytes.NewReader(nil), io.Discard)
			if (err == nil) != (stage == "ok") {
				t.Fatal(err)
			}
			if stage != "open" && !p.closed {
				t.Fatal("port leaked")
			}
			if stage == "ok" && (len(p.dtr) != 2 || !p.dtr[0] || p.dtr[1]) {
				t.Fatal(p.dtr)
			}
		})
	}
}

type faultIO struct {
	n   int
	err error
}

func (f faultIO) Read([]byte) (int, error)  { return f.n, f.err }
func (f faultIO) Write([]byte) (int, error) { return f.n, f.err }

func TestProxyHandlesNoProgressAndWriterFailures(t *testing.T) {
	var data [screenwire.MaxRecord]byte
	if _, err := readRecord(faultIO{}, data[:]); !errors.Is(err, io.ErrNoProgress) {
		t.Fatal(err)
	}
	for _, f := range []faultIO{{}, {n: 2}, {err: io.ErrClosedPipe}} {
		if err := writeAll(f, []byte{1}); err == nil {
			t.Fatal(f)
		}
	}
	r := screenwire.Record{Kind: screenwire.Abort, Epoch: 1}
	if err := serveProxy(bytes.NewReader(encoded(t, r)), io.Discard, faultIO{err: io.ErrClosedPipe}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if err := serveProxy(bytes.NewReader(response(t, r)), io.Discard, &proxyPort{}); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	p := &proxyPort{}
	if err := serveProxy(bytes.NewReader(encoded(t, r)), io.Discard, p); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	p.input.Write(response(t, r))
	if err := serveProxy(bytes.NewReader(encoded(t, r)), faultIO{err: io.ErrClosedPipe}, p); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func TestProxyRejectsMalformedReplyAndHeader(t *testing.T) {
	r := screenwire.Record{Kind: screenwire.Abort, Epoch: 1}
	p := &proxyPort{}
	reply := response(t, r)
	reply[len(reply)-1] ^= 1
	p.input.Write(reply)
	if err := serveProxy(bytes.NewReader(encoded(t, r)), io.Discard, p); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	var data [screenwire.MaxRecord]byte
	if _, err := readRecord(bytes.NewReader(make([]byte, 32)), data[:]); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
}
