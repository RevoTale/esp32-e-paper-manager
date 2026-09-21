package screenlink

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func healthExchange(t *testing.T, c *Connection, now time.Duration) screenwire.Response {
	t.Helper()
	request := screenwire.Record{Kind: screenwire.Health}
	var record [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(record[:], request)
	if err != nil {
		t.Fatal(err)
	}
	var response screenwire.Response
	if err = c.Push(record[:n], now, func(data []byte) error {
		r, err := screenwire.Decode(data)
		if err != nil {
			return err
		}
		response, err = screenwire.ParseReply(r, request)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestHealthDoesNotAcquireTickOrTouchExpiredStaging(t *testing.T) {
	d, owner, sink := fixture(t)
	bind(t, owner, 2, true)
	upload(t, owner, 1)
	want := screenwire.HealthStatus{Version: 1, State: 6, LastFailure: 2, Failures: 1, UptimeSeconds: 3}
	calls := 0
	d.SetHealth(func() screenwire.HealthStatus { calls++; return want })
	before := owner.snapshot(owner.tx)
	generation := d.session.Epoch()
	observer := d.Open()
	response := healthExchange(t, observer, 3*time.Second)
	if response.Health != want || response.Status.Code != screenwire.CodeOK || calls != 1 {
		t.Fatal(response, calls)
	}
	if after := owner.snapshot(owner.tx); after != before || d.session.Epoch() != generation || d.active != owner {
		t.Fatal("health mutated active ownership/staging", before, after)
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 0})
	if err := d.Tick(3 * time.Second); !errors.Is(err, streamrx.ErrTimeout) {
		t.Fatal("fixture was not actually expired", err)
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 1})
}

func TestHealthWithoutProviderReturnsExplicitConfigurationError(t *testing.T) {
	d, c, sink := fixture(t)
	r := healthExchange(t, c, 0)
	if r.Status.Code != screenwire.CodeConfig || r.Health.Version != screenwire.HealthVersion || d.session.Epoch().Generation != 0 {
		t.Fatal(r)
	}
	wantCalls(t, sink, [4]int{})
}

func TestInvalidHealthProviderFailsClosedWithoutPanelIO(t *testing.T) {
	d, c, sink := fixture(t)
	d.SetHealth(func() screenwire.HealthStatus { return screenwire.HealthStatus{Version: 99} })
	var record [screenwire.HeaderSize]byte
	n, err := screenwire.Encode(record[:], screenwire.Record{Kind: screenwire.Health})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Push(record[:n], 0, func([]byte) error { t.Fatal("invalid health encoded"); return nil }); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
	wantCalls(t, sink, [4]int{})
}
