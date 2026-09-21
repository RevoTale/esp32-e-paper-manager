package streamwire

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

type testSink struct{ starts, writes, commits, aborts int }

func (s *testSink) Begin() error                            { s.starts++; return nil }
func (s *testSink) Write(_ uint8, _ uint32, _ []byte) error { s.writes++; return nil }
func (s *testSink) Commit() error                           { s.commits++; return nil }
func (s *testSink) Abort() error                            { s.aborts++; return nil }

func linkFixture(t *testing.T) (*Link, *testSink) {
	t.Helper()
	s := new(testSink)
	l, err := NewLink(streamrx.Config{Epoch: 1, Width: 8, Height: 2, Passes: 2, MaxChunk: 2, Idle: time.Second, Total: 5 * time.Second}, s, 180*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return l, s
}

func exchange(t *testing.T, l *Link, r Record, now time.Duration) Record {
	t.Helper()
	var buf [MaxRecord]byte
	n, err := Encode(buf[:], r)
	if err != nil {
		t.Fatal(err)
	}
	var result Record
	write := func(p []byte) error { var err error; result, err = Decode(append([]byte(nil), p...)); return err }
	for _, b := range buf[:n] {
		if err = l.Push([]byte{b}, now, write); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func TestLinkTwoPassCommitAndReconnectCadence(t *testing.T) {
	l, s := linkFixture(t)
	digest := sha256.Sum256([]byte{128, 1})
	packets := []Record{{Kind: Hello, Epoch: 7}, {Kind: Begin, Epoch: 7, ID: 1, Payload: digest[:]}, {Kind: Data, Epoch: 7, ID: 1, Payload: []byte{128, 1}},
		{Kind: Data, Epoch: 7, ID: 1, Pass: 1, Payload: []byte{128, 1}}, {Kind: Commit, Epoch: 7, ID: 1}, {Kind: Commit, Epoch: 7, ID: 1}, {Kind: Query, Epoch: 7, ID: 1}}
	for _, p := range packets {
		if reply := exchange(t, l, p, 0); reply.Payload[0] != 0 {
			t.Fatal(reply)
		}
	}
	if s.commits != 1 {
		t.Fatal("duplicate refresh")
	}
	if err := l.Disconnect(); err != nil {
		t.Fatal(err)
	}
	exchange(t, l, Record{Kind: Hello, Epoch: 8}, 0)
	r := exchange(t, l, Record{Kind: Begin, Epoch: 8, ID: 1, Payload: digest[:]}, 0)
	if r.Payload[0] != 6 || s.starts != 1 {
		t.Fatal("reconnect bypassed cadence")
	}
}

func TestLinkDisconnectAndCRCRejectWithoutRefresh(t *testing.T) {
	l, s := linkFixture(t)
	digest := sha256.Sum256([]byte{128, 1})
	exchange(t, l, Record{Kind: Hello, Epoch: 7}, 0)
	exchange(t, l, Record{Kind: Begin, Epoch: 7, ID: 1, Payload: digest[:]}, 0)
	var buf [MaxRecord]byte
	n, err := Encode(buf[:], Record{Kind: Data, Epoch: 7, ID: 1, Payload: []byte{128}})
	if err != nil {
		t.Fatal(err)
	}
	buf[n-1] ^= 1
	if err = l.Push(buf[:n], 0, func([]byte) error { return nil }); err == nil {
		t.Fatal("corrupt record")
	}
	if s.aborts != 1 || s.commits != 0 {
		t.Fatal("unsafe corruption recovery")
	}
}
