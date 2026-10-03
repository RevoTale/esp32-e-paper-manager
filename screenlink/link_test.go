package screenlink

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

type sink struct {
	begins, writes, commits, aborts int
	failure                         error
}

func (s *sink) Begin() error                      { s.begins++; return nil }
func (s *sink) Write(uint8, uint32, []byte) error { s.writes++; return nil }
func (s *sink) Commit() error                     { s.commits++; return s.failure }
func (s *sink) Abort() error                      { s.aborts++; return nil }

func wantCalls(t *testing.T, s *sink, want [4]int) {
	t.Helper()
	got := [4]int{s.begins, s.writes, s.commits, s.aborts}
	if got != want {
		t.Fatalf("Begin/Write/Commit/Abort: got %v want %v", got, want)
	}
}

func fixture(t *testing.T) (*Device, *Connection, *sink) {
	t.Helper()
	s := &sink{}
	caps := screenwire.Capabilities{Width: 8, Height: 1, Stride: 1, MaxChunk: 1, Passes: 2, Format: screenwire.Mono1, Features: screenwire.RawFull, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1000}
	d, err := New(caps, [16]byte{1}, s, time.Second, 2*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d, d.Open(), s
}

func exchange(t *testing.T, c *Connection, r screenwire.Record, now time.Duration) screenwire.Status {
	t.Helper()
	b := make([]byte, screenwire.MaxRecord)
	n, err := screenwire.Encode(b, r)
	if err != nil {
		t.Fatal(err)
	}
	var reply []byte
	write := func(p []byte) error { reply = append(reply, p...); return nil }
	for _, v := range b[:n] {
		if err = c.Push([]byte{v}, now, write); err != nil {
			t.Fatal(err)
		}
	}
	got, err := screenwire.Decode(reply)
	if err != nil || got.Kind != screenwire.Reply || got.ID != r.ID || got.Epoch != r.Epoch {
		t.Fatal(got, err)
	}
	response, err := screenwire.ParseReply(got, r)
	if err != nil {
		t.Fatal(response, err)
	}
	return response.Status
}

func bind(t *testing.T, c *Connection, claim byte, acquire bool) {
	t.Helper()
	r := screenwire.Record{Kind: screenwire.Hello}
	hello := exchange(t, c, r, time.Second)
	p := make([]byte, 32)
	copy(p, hello.Boot[:])
	p[16] = claim
	generation := hello.Generation
	if acquire {
		grant := exchange(t, c, screenwire.Record{Kind: screenwire.Acquire, Epoch: generation, Payload: p}, time.Second)
		if grant.Code != screenwire.CodeOK {
			t.Fatal(grant)
		}
		generation = grant.Generation
	}
	status := exchange(t, c, screenwire.Record{Kind: screenwire.Bind, Epoch: generation, Payload: p}, time.Second)
	if status.Code != screenwire.CodeOK {
		t.Fatal(status)
	}
}

func upload(t *testing.T, c *Connection, id uint64) {
	t.Helper()
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: id, Payload: digest[:]}
	if s := exchange(t, c, r, time.Second); s.Code != screenwire.CodeOK {
		t.Fatal(s)
	}
	for pass := uint8(0); pass < 2; pass++ {
		r = screenwire.Record{Kind: screenwire.Data, Epoch: 1, ID: id, Pass: pass, Payload: []byte{128}}
		if s := exchange(t, c, r, time.Second); s.Code != screenwire.CodeOK || s.Pass != pass+1 {
			t.Fatal(s)
		}
	}
}

func TestLostTerminalACKReconcilesWithoutSecondRefresh(t *testing.T) {
	d, c, sink := fixture(t)
	bind(t, c, 2, true)
	upload(t, c, 1)
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]}
	b := make([]byte, screenwire.MaxRecord)
	n, err := screenwire.Encode(b, r)
	if err != nil {
		t.Fatal(err)
	}
	lost := errors.New("lost terminal ACK")
	if err = c.Push(b[:n], time.Second, func([]byte) error { return lost }); !errors.Is(err, lost) {
		t.Fatal(err)
	}
	c = d.Open()
	bind(t, c, 2, false)
	r.Kind = screenwire.Query
	if status := exchange(t, c, r, time.Second); status.Code != screenwire.CodeOK || status.State != streamrx.Complete || !status.CurrentImage {
		t.Fatal(status)
	}
	r.Kind = screenwire.Commit
	if status := exchange(t, c, r, time.Second); status.Code != screenwire.CodeOK || sink.commits != 1 || sink.writes != 2 {
		t.Fatal(status, sink)
	}
}
