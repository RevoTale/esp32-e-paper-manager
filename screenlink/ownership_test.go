package screenlink

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestDiscoveryDoesNotPreemptAndExplicitPreemptionDoes(t *testing.T) {
	d, a, sink := fixture(t)
	bind(t, a, 2, true)
	upload(t, a, 1)
	b := d.Open()
	hello := exchange(t, b, screenwire.Record{Kind: screenwire.Hello}, time.Second)
	if hello.Generation != 1 || sink.aborts != 0 {
		t.Fatal(hello, sink)
	}
	p := make([]byte, 32)
	p[0] = 1
	p[16] = 3
	r := screenwire.Record{Kind: screenwire.Acquire, Epoch: 1, Payload: p}
	if s := exchange(t, b, r, time.Second); s.Code != screenwire.CodeBusy {
		t.Fatal(s)
	}
	if err := d.Preempt(); err != nil {
		t.Fatal(err)
	}
	if s := exchange(t, b, r, time.Second); s.Code != screenwire.CodeOK || s.Generation != 2 {
		t.Fatal(s)
	}
	r.Kind = screenwire.Bind
	r.Epoch = 2
	if s := exchange(t, b, r, time.Second); s.Code != screenwire.CodeOK {
		t.Fatal(s)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if d.active != b {
		t.Fatal("wrong active binding")
	}
	wantCalls(t, sink, [4]int{1, 2, 0, 1})
}

func TestBindRetryIsLocalAndAnotherConnectionCannotAdopt(t *testing.T) {
	d, a, _ := fixture(t)
	bind(t, a, 2, true)
	p := make([]byte, 32)
	p[0] = 1
	p[16] = 2
	r := screenwire.Record{Kind: screenwire.Bind, Epoch: 1, Payload: p}
	if s := exchange(t, a, r, time.Second); s.Code != screenwire.CodeOK {
		t.Fatal(s)
	}
	b := d.Open()
	if s := exchange(t, b, r, time.Second); s.Code != screenwire.CodeBusy {
		t.Fatal(s)
	}
	p[16] = 3
	if s := exchange(t, a, r, time.Second); s.Code != screenwire.CodeLease {
		t.Fatal(s)
	}
	if s := exchange(t, b, r, time.Second); s.Code != screenwire.CodeLease {
		t.Fatal(s)
	}
}

func TestAbortAndConsumedIDRemainAcrossDisconnect(t *testing.T) {
	d, c, sink := fixture(t)
	bind(t, c, 2, true)
	upload(t, c, 1)
	s := exchange(t, c, screenwire.Record{Kind: screenwire.Abort, Epoch: 1}, time.Second)
	if s.Code != screenwire.CodeOK || s.State != streamrx.Closed || sink.aborts != 1 {
		t.Fatal(s, sink)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
	c = d.Open()
	bind(t, c, 2, false)
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}
	if s = exchange(t, c, r, time.Second); s.Code != screenwire.CodeStale {
		t.Fatal(s)
	}
	if err := d.Preempt(); err != nil {
		t.Fatal(err)
	}
	if err := d.Preempt(); err != nil {
		t.Fatal(err)
	}
}

func TestUnboundCannotStageAndCooldownIsReported(t *testing.T) {
	_, c, sink := fixture(t)
	digest := sha256.Sum256([]byte{128})
	r := screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}
	if s := exchange(t, c, r, 0); s.Code != screenwire.CodeLease || s.CooldownMS != 1000 {
		t.Fatal(s)
	}
	if sink.begins != 0 {
		t.Fatal(sink)
	}
}
