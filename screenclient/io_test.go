package screenclient

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type readWriter struct {
	io.Reader
	io.Writer
}
type noProgress struct{}

func (noProgress) Read([]byte) (int, error) { return 0, nil }

type shortWriter struct {
	n       int
	failure error
}

func (w shortWriter) Write(p []byte) (int, error) {
	if w.n < 0 {
		return len(p) + 1, w.failure
	}
	return min(w.n, len(p)), w.failure
}

func helloBytes(t *testing.T, code screenwire.Code) []byte {
	t.Helper()
	c, _, _, _, _ := fixture(t)
	body := make([]byte, screenwire.StatusSize+screenwire.CapabilitiesSize)
	if err := screenwire.EncodeStatus(body[:screenwire.StatusSize], screenwire.Status{Operation: screenwire.Hello, Code: code, Boot: [16]byte{1}}); err != nil {
		t.Fatal(err)
	}
	if err := screenwire.EncodeCapabilities(body[screenwire.StatusSize:], c.caps); err != nil {
		t.Fatal(err)
	}
	out := make([]byte, 32+len(body))
	if _, err := screenwire.Encode(out, screenwire.Record{Kind: screenwire.Reply, Payload: body}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMalformedRepliesAndNoProgressDisconnect(t *testing.T) {
	valid := helloBytes(t, screenwire.CodeOK)
	for _, mutate := range []func([]byte) []byte{
		func(p []byte) []byte { return p[:3] }, func(p []byte) []byte { return p[:40] },
		func(p []byte) []byte { p[0] = '!'; return p }, func(p []byte) []byte { p[32] ^= 1; return p },
	} {
		p := mutate(append([]byte(nil), valid...))
		c, err := New(bytes.NewReader(make([]byte, 16)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Connect(readWriter{bytes.NewReader(p), io.Discard}); err == nil || c.stream != nil {
			t.Fatal(err)
		}
	}
	c, err := New(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Connect(readWriter{noProgress{}, io.Discard}); !errors.Is(err, io.ErrNoProgress) {
		t.Fatal(err)
	}
}

func TestMissingRandomnessAndDiscoveryError(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("nil RNG")
	}
	for _, rng := range []io.Reader{bytes.NewReader(nil), bytes.NewReader(make([]byte, 16)), noProgress{}} {
		c, err := New(rng)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Connect(readWriter{bytes.NewReader(helloBytes(t, screenwire.CodeOK)), io.Discard}); err == nil {
			t.Fatal("bad RNG accepted")
		}
	}
	c, err := New(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Connect(nil); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
	var remote RemoteError
	if _, err = c.Connect(readWriter{bytes.NewReader(helloBytes(t, screenwire.CodeBusy)), io.Discard}); !errors.As(err, &remote) {
		t.Fatal(err)
	}
}

func TestBoundedReadWriteContract(t *testing.T) {
	for _, w := range []shortWriter{{n: 0}, {n: -1}, {n: 1, failure: io.ErrClosedPipe}} {
		if err := writeAll(w, []byte{1, 2, 3}); err == nil {
			t.Fatal(w)
		}
	}
	if err := writeAll(shortWriter{n: 1}, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	c, err := New(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	c.stream = readWriter{bytes.NewReader(nil), io.Discard}
	if _, err = c.exchange(screenwire.Record{Kind: 0}); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	c.stream = readWriter{bytes.NewReader(nil), shortWriter{n: 0}}
	if _, err = c.exchange(screenwire.Record{Kind: screenwire.Hello}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}
