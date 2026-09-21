package screenusbhost

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type proxyPort struct {
	input  bytes.Buffer
	output bytes.Buffer
	dtr    []bool
	closed bool
	err    error
}

func (p *proxyPort) Read(b []byte) (int, error)         { return p.input.Read(b[:min(len(b), 7)]) }
func (p *proxyPort) Write(b []byte) (int, error)        { return p.output.Write(b) }
func (p *proxyPort) SetDTR(v bool) error                { p.dtr = append(p.dtr, v); return p.err }
func (p *proxyPort) SetReadTimeout(time.Duration) error { return p.err }
func (p *proxyPort) Close() error                       { p.closed = true; return nil }

func encoded(t *testing.T, r screenwire.Record) []byte {
	t.Helper()
	b := make([]byte, screenwire.MaxRecord)
	n, err := screenwire.Encode(b, r)
	if err != nil {
		t.Fatal(err)
	}
	return b[:n]
}

func response(t *testing.T, r screenwire.Record) []byte {
	t.Helper()
	p := make([]byte, screenwire.StatusSize)
	if err := screenwire.EncodeStatus(p, screenwire.Status{Operation: r.Kind, Boot: [16]byte{1}}); err != nil {
		t.Fatal(err)
	}
	return encoded(t, screenwire.Record{Kind: screenwire.Reply, Epoch: r.Epoch, ID: r.ID, Payload: p})
}

func TestProxyKeepsCompleteRecordIncludingPayloadMagic(t *testing.T) {
	for _, kind := range []screenwire.Kind{screenwire.Acquire, screenwire.Data} {
		t.Run(string(rune('A'+kind)), func(t *testing.T) {
			payload := bytes.Repeat([]byte{3}, 32)
			r := screenwire.Record{Kind: kind, Payload: payload}
			if kind == screenwire.Data {
				r.Epoch, r.ID = 1, 1
				r.Payload = bytes.Repeat([]byte("EPS2"), 256)
			}
			input, reply := encoded(t, r), response(t, r)
			p := &proxyPort{}
			p.input.Write(reply)
			var output bytes.Buffer
			if err := serveProxy(bytes.NewReader(input), &output, p); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(p.output.Bytes(), input) || !bytes.Equal(output.Bytes(), reply) {
				t.Fatal("record changed")
			}
		})
	}
}

func TestProxyRejectsEveryTruncationAndCorruptRecordBeforeSerial(t *testing.T) {
	r := screenwire.Record{Kind: screenwire.Acquire, Payload: make([]byte, 32)}
	data := encoded(t, r)
	for n := 1; n < len(data); n++ {
		p := &proxyPort{}
		if err := serveProxy(bytes.NewReader(data[:n]), io.Discard, p); err == nil || p.output.Len() != 0 {
			t.Fatal(n, err)
		}
	}
	data[len(data)-1] ^= 1
	p := &proxyPort{}
	if err := serveProxy(bytes.NewReader(data), io.Discard, p); !errors.Is(err, screenwire.ErrRecord) || p.output.Len() != 0 {
		t.Fatal(err)
	}
}

func TestProxyRejectsMismatchedReplyBeforeStdout(t *testing.T) {
	r := screenwire.Record{Kind: screenwire.Abort, Epoch: 1}
	p := &proxyPort{}
	p.input.Write(response(t, screenwire.Record{Kind: screenwire.Abort, Epoch: 2}))
	var output bytes.Buffer
	if err := serveProxy(bytes.NewReader(encoded(t, r)), &output, p); !errors.Is(err, screenwire.ErrRecord) || output.Len() != 0 {
		t.Fatal(err)
	}
}
