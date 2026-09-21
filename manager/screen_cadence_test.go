package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

type cadenceSender struct {
	options refreshpolicy.Options
	policy  refreshpolicy.Policy
}

func (*cadenceSender) Send(context.Context, display.Frame) error { return nil }
func (s *cadenceSender) ConfigureRefresh(p refreshpolicy.Policy) error {
	if _, err := p.Wait(refreshpolicy.Normal, 0); err != nil {
		return err
	}
	s.policy = p
	return nil
}
func (s *cadenceSender) SendWithOptions(_ context.Context, _ display.Frame, o refreshpolicy.Options) error {
	s.options = o
	return nil
}

func TestPumpUrgentCannotBypassUnknownCompletion(t *testing.T) {
	_, s := newScreenAPI(t)
	clock := time.Duration(0)
	s.elapsedClock = func() time.Duration { return clock }
	p, err := NewScreenPumpWithPolicy(s, &cadenceSender{}, refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	s.options.Priority = refreshpolicy.Urgent
	r := screenRecovery{pump: p, cooldown: time.Minute, urgentCooldown: time.Minute}
	clock = time.Second
	if r.remaining() != 59*time.Second {
		t.Fatal("startup bypass")
	}
	r.completed()
	clock += time.Second
	if r.remaining() != 0 {
		t.Fatal("urgent not ready")
	}
	s.options.Priority = refreshpolicy.Normal
	if r.remaining() != 59*time.Second {
		t.Fatal("normal inherited urgency")
	}
	s.options.Priority = refreshpolicy.Urgent
	r.postpone(time.Minute)
	if r.remaining() != time.Minute {
		t.Fatal("unknown ACK bypass")
	}
	r.postponeReadiness(screendelivery.Readiness{NotBefore: p.now().Add(2 * time.Minute)})
	if r.remaining() < 119*time.Second {
		t.Fatal("transport guard bypass")
	}
}

func TestPumpPolicyValidation(t *testing.T) {
	_, s := newScreenAPI(t)
	for _, policy := range []refreshpolicy.Policy{{}, {Normal: time.Second}} {
		if _, err := NewScreenPumpWithPolicy(s, &cadenceSender{}, policy); err == nil {
			t.Fatal(policy)
		}
	}
	if _, err := NewScreenPumpWithPolicy(nil, &cadenceSender{}, refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
}

func TestPumpDeliversLeasedMetadata(t *testing.T) {
	_, s := newScreenAPI(t)
	sender := &cadenceSender{}
	p, err := NewScreenPumpWithPolicy(s, sender, refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	o := refreshpolicy.Options{Priority: refreshpolicy.Urgent, Mode: refreshpolicy.Full}
	if _, err := s.submitOptionsAt(0, []byte("urgent"), s.elapsed, o); err != nil {
		t.Fatal(err)
	}
	r := screenRecovery{pump: p}
	if err := r.deliver(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.options != o || s.Status().Delivered != 1 || r.urgentCooldown != time.Second {
		t.Fatal(sender, s.Status(), r)
	}
}
