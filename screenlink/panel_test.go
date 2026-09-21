package screenlink

import (
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"testing"
	"time"
)

func TestPanelTraceIsCachedAndDoesNotTickOrAcquire(t *testing.T) {
	d, owner, sink := fixture(t)
	bind(t, owner, 2, true)
	upload(t, owner, 1)
	want := screenwire.PanelStatus{Version: 1, State: 2, Cycle: 1, Waits: [3]screenwire.BusySamples{{Samples: 1}, {Samples: 3, LowSamples: 2}, {Samples: 1}}}
	calls := 0
	d.SetPanelTrace(func() screenwire.PanelStatus { calls++; return want })
	before := owner.snapshot(owner.tx)
	r := panelExchange(t, d.Open())
	if r.Panel != want || r.Status.Code != screenwire.CodeOK || calls != 1 {
		t.Fatal(r, calls)
	}
	if owner.snapshot(owner.tx) != before || d.active != owner {
		t.Fatal("trace changed ownership")
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 0})
}

func TestPanelTraceMissingProvider(t *testing.T) {
	_, c, sink := fixture(t)
	r := panelExchange(t, c)
	if r.Status.Code != screenwire.CodeConfig || r.Panel != (screenwire.PanelStatus{Version: 1}) {
		t.Fatal(r)
	}
	wantCalls(t, sink, [4]int{})
}

func panelExchange(t *testing.T, c *Connection) screenwire.Response {
	t.Helper()
	request := screenwire.Record{Kind: screenwire.PanelTrace}
	var buf [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(buf[:], request)
	if err != nil {
		t.Fatal(err)
	}
	var out screenwire.Response
	err = c.Push(buf[:n], 3*time.Second, func(data []byte) error {
		r, err := screenwire.Decode(data)
		if err != nil {
			return err
		}
		out, err = screenwire.ParseReply(r, request)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
