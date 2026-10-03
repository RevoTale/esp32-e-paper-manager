package screenlink

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestCorruptOrTruncatedFrameNeverRefreshes(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		d, c, sink := fixture(t)
		bind(t, c, 2, true)
		digest := sha256.Sum256([]byte{128})
		exchange(t, c, screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}, time.Second)
		b := make([]byte, screenwire.MaxRecord)
		n, err := screenwire.Encode(b, screenwire.Record{Kind: screenwire.Data, Epoch: 1, ID: 1, Payload: []byte{128}})
		if err != nil {
			t.Fatal(err)
		}
		if corrupt {
			b[n-1] ^= 1
		} else {
			n--
		}
		err = c.Push(b[:n], time.Second, func([]byte) error { return nil })
		if corrupt && !errors.Is(err, screenwire.ErrRecord) {
			t.Fatal(err)
		}
		if !corrupt {
			if err = d.Tick(2 * time.Second); !errors.Is(err, streamrx.ErrTimeout) {
				t.Fatal(err)
			}
		}
		wantCalls(t, sink, [4]int{1, 0, 0, 1})
	}
}

func TestInvalidHeaderAndMissingWriterCloseStaging(t *testing.T) {
	for _, missing := range []bool{false, true} {
		_, c, sink := fixture(t)
		bind(t, c, 2, true)
		upload(t, c, 1)
		write := func([]byte) error { return nil }
		if missing {
			write = nil
		}
		if err := c.Push(make([]byte, 32), time.Second, write); !errors.Is(err, screenwire.ErrRecord) {
			t.Fatal(err)
		}
		if sink.aborts != 1 || sink.commits != 0 {
			t.Fatal(sink)
		}
	}
}

func TestWrongTransactionCannotBorrowActiveDigest(t *testing.T) {
	_, c, sink := fixture(t)
	bind(t, c, 2, true)
	digest := sha256.Sum256([]byte{128})
	exchange(t, c, screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}, time.Second)
	r := screenwire.Record{Kind: screenwire.Data, Epoch: 1, ID: 2, Payload: []byte{128}}
	if s := exchange(t, c, r, time.Second); s.Code != screenwire.CodeState {
		t.Fatal(s)
	}
	r.ID = 1
	r.Epoch = 2
	if s := exchange(t, c, r, time.Second); s.Code != screenwire.CodeLease {
		t.Fatal(s)
	}
	if sink.writes != 0 || sink.commits != 0 {
		t.Fatal(sink)
	}
}

func TestIdleTickIsVisibleInTerminalStatus(t *testing.T) {
	_, c, sink := fixture(t)
	bind(t, c, 2, true)
	upload(t, c, 1)
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Query, Epoch: 1, ID: 1, Payload: digest[:]}
	s := exchange(t, c, r, 2*time.Second)
	if s.Code != screenwire.CodeTimeout || s.State != streamrx.Failed || sink.aborts != 1 {
		t.Fatal(s, sink)
	}
	s = exchange(t, c, r, 2*time.Second)
	if s.Code != screenwire.CodeTimeout || sink.aborts != 1 {
		t.Fatal(s, sink)
	}
}

func TestHardwareDiagnosticDoesNotLeakErrorText(t *testing.T) {
	_, c, sink := fixture(t)
	bind(t, c, 2, true)
	upload(t, c, 1)
	sink.failure = errors.New("secret password and source HTML")
	diag := screenwire.Diagnostic{Domain: screenwire.DomainPanel, Code: 2, Phase: 5, Step: 14, Command: 0x12, Offset: -1, BusyKnown: true}
	c.device.diagnose = func(error) screenwire.Diagnostic { return diag }
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]}
	s := exchange(t, c, r, time.Second)
	if s.Code != screenwire.CodeHardware || s.Diagnostic != diag || s.CurrentImage {
		t.Fatal(s)
	}
	if bytes.Contains(c.reply[:], []byte("secret")) {
		t.Fatal("error text on wire")
	}
}

func TestErrorCodesAndConstructorBounds(t *testing.T) {
	for _, entry := range codes {
		if got := errorCode(errors.Join(errors.New("wrapper"), entry.err)); got != entry.code {
			t.Fatal(got, entry)
		}
	}
	d, _, _ := fixture(t)
	caps := d.caps
	caps.Width = 0
	if _, err := New(caps, [16]byte{1}, &sink{}, time.Second, time.Second, nil); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	if _, err := New(d.caps, [16]byte{}, &sink{}, time.Second, time.Second, nil); !errors.Is(err, streamrx.ErrConfig) {
		t.Fatal(err)
	}
}
