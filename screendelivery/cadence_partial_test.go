package screendelivery

import (
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestCadenceRequiresAllConfiguredCapabilities(t *testing.T) {
	p := refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
	for _, tc := range []struct {
		cadence  Cadence
		features uint16
		want     bool
	}{
		{Cadence{}, screenwire.RawFull, true},
		{Cadence{Policy: p}, screenwire.RawFull, false},
		{Cadence{Policy: p}, screenwire.RawFull | screenwire.FeatureRefreshPolicy, true},
		{Cadence{Policy: p, PartialPolicy: p}, screenwire.RawFull | screenwire.FeatureRefreshPolicy, false},
		{Cadence{Policy: p, PartialPolicy: p}, screenwire.RawFull | screenwire.FeatureRegion, false},
		{Cadence{Policy: p, PartialPolicy: p}, screenwire.RawFull | screenwire.FeatureRefreshPolicy | screenwire.FeatureRegion, true},
	} {
		if got := tc.cadence.Supports(screenwire.Capabilities{Features: tc.features}); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}

func TestPartialCadenceSeparatesKnownCompletionDeadlines(t *testing.T) {
	var c Cadence
	if err := c.Configure(refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfigurePartial(refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	full := c.Complete(now)
	for _, tc := range []struct {
		mode     refreshpolicy.Mode
		priority refreshpolicy.Priority
		wait     time.Duration
	}{
		{refreshpolicy.Full, refreshpolicy.Normal, 30 * time.Second},
		{refreshpolicy.Full, refreshpolicy.Urgent, 5 * time.Second},
		{refreshpolicy.Partial, refreshpolicy.Normal, 2 * time.Second},
		{refreshpolicy.Partial, refreshpolicy.Urgent, time.Second},
		{refreshpolicy.Auto, refreshpolicy.Normal, 30 * time.Second},
	} {
		if got := c.Deadline(full, refreshpolicy.Options{Mode: tc.mode, Priority: tc.priority}); got != now.Add(tc.wait) {
			t.Fatalf("mode=%v priority=%v deadline=%v", tc.mode, tc.priority, got)
		}
	}
	r := c.Ready(full)
	if r.PartialNotBefore != now.Add(2*time.Second) || r.PartialUrgentNotBefore != now.Add(time.Second) {
		t.Fatal(r)
	}
}

func TestPartialCadenceUnknownNeverOpensFastLane(t *testing.T) {
	c := Cadence{Policy: refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: time.Second}}
	if err := c.ConfigurePartial(refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	full := c.Unknown(now, now.Add(time.Minute), 10*time.Second)
	r := c.Ready(full)
	if r.PartialNotBefore != full || r.PartialUrgentNotBefore != full {
		t.Fatal("uncertainty bypass", r)
	}
	if got := c.Deadline(full, refreshpolicy.Options{Mode: refreshpolicy.Partial, Priority: refreshpolicy.Urgent}); got != full {
		t.Fatal(got)
	}
}

func TestPartialCadenceConfigurationIsAtomicAndHonorsLongerBudget(t *testing.T) {
	var c Cadence
	p := refreshpolicy.Policy{Normal: time.Minute, Urgent: 2 * time.Minute}
	if err := c.ConfigurePartial(p); err != nil {
		t.Fatal(err)
	}
	if c.ConfigurePartial(refreshpolicy.Policy{}) == nil || c.PartialPolicy != p {
		t.Fatal("invalid configuration changed policy")
	}
	now := time.Unix(100, 0)
	full := c.Unknown(now, time.Time{}, time.Second)
	r := c.Ready(full)
	if r.PartialNotBefore != now.Add(time.Minute) || r.PartialUrgentNotBefore != now.Add(2*time.Minute) {
		t.Fatal(r)
	}
}

func TestPartialCadenceResetRevokesKnownDeadlines(t *testing.T) {
	c := Cadence{Policy: refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 5 * time.Second}}
	p := refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}
	if err := c.ConfigurePartial(p); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0)
	c.Complete(now)
	c.Reset()
	r := c.Ready(now.Add(time.Minute))
	if !r.PartialNotBefore.IsZero() || !r.PartialUrgentNotBefore.IsZero() || !c.Urgent.IsZero() || c.PartialPolicy != p {
		t.Fatal("reset retained permission or discarded configuration", r, c)
	}
	if got := c.Deadline(r.NotBefore, refreshpolicy.Options{Mode: refreshpolicy.Partial}); got != r.NotBefore {
		t.Fatal("reset bypass", got)
	}
	c.Unknown(now, r.NotBefore, time.Second)
	if got := c.Ready(r.NotBefore).PartialNotBefore; got != r.NotBefore {
		t.Fatal(got)
	}
}

func TestPartialCadenceUnknownRetainsLongerPreviousGuard(t *testing.T) {
	c := Cadence{PartialPolicy: refreshpolicy.Policy{Normal: time.Minute, Urgent: 2 * time.Minute}}
	now := time.Unix(100, 0)
	c.Complete(now)
	// A later observation with shorter configuration cannot erase a deadline
	// already granted to a prior physical transaction.
	c.PartialPolicy = refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
	c.Unknown(now.Add(time.Second), time.Time{}, time.Second)
	r := c.Ready(time.Time{})
	if r.PartialNotBefore != now.Add(time.Minute) || r.PartialUrgentNotBefore != now.Add(2*time.Minute) {
		t.Fatal(r)
	}
}
