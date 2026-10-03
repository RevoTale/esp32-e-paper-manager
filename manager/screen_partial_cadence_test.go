package manager

import (
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestPartialCadenceKeepsIndependentLanes(t *testing.T) {
	_, s := newScreenAPI(t)
	clock := time.Duration(0)
	s.elapsedClock = func() time.Duration { return clock }
	p := &ScreenPump{screen: s, interval: 30 * time.Second, urgent: 10 * time.Second,
		partialPolicy: refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}}
	r := screenRecovery{pump: p}
	r.completed()
	clock = time.Second
	checkRecoveryLanes(t, &r, [4]time.Duration{29 * time.Second, 9 * time.Second, time.Second, 0})
	r.postpone(40 * time.Second)
	checkRecoveryLanes(t, &r, [4]time.Duration{40 * time.Second, 40 * time.Second, 40 * time.Second, 40 * time.Second})
	clock += 41 * time.Second
	checkRecoveryLanes(t, &r, [4]time.Duration{})
}

func TestPartialReadinessCannotEraseLongerLocalGuard(t *testing.T) {
	_, s := newScreenAPI(t)
	clock := time.Duration(0)
	s.elapsedClock = func() time.Duration { return clock }
	now := time.Unix(1000, 0)
	p := &ScreenPump{screen: s, interval: 30 * time.Second, urgent: 10 * time.Second,
		partialPolicy: refreshpolicy.Policy{Normal: 5 * time.Second, Urgent: 2 * time.Second},
		now:           func() time.Time { return now.Add(clock) }}
	r := screenRecovery{pump: p}
	r.completed()
	clock = time.Second
	r.postponeReadiness(screendelivery.Readiness{
		NotBefore: now.Add(40 * time.Second), UrgentNotBefore: now.Add(12 * time.Second),
		PartialNotBefore: now.Add(3 * time.Second), PartialUrgentNotBefore: now.Add(8 * time.Second),
	})
	checkRecoveryLanes(t, &r, [4]time.Duration{39 * time.Second, 11 * time.Second, 4 * time.Second, 7 * time.Second})
	// Missing partial permission is conservative, never an immediately ready lane.
	r.postponeReadiness(screendelivery.Readiness{NotBefore: now.Add(time.Minute)})
	checkRecoveryLanes(t, &r, [4]time.Duration{59 * time.Second, 59 * time.Second, 59 * time.Second, 59 * time.Second})
}

func TestUnconfiguredPartialCadenceUsesFullGuard(t *testing.T) {
	_, s := newScreenAPI(t)
	s.elapsedClock = func() time.Duration { return 0 }
	r := screenRecovery{pump: &ScreenPump{screen: s, interval: time.Minute}}
	r.completed()
	checkRecoveryLanes(t, &r, [4]time.Duration{time.Minute, time.Minute, time.Minute, time.Minute})
}

func TestUnknownPartialCadenceRestartsLongerOperatorBudget(t *testing.T) {
	_, s := newScreenAPI(t)
	clock := time.Duration(0)
	s.elapsedClock = func() time.Duration { return clock }
	r := screenRecovery{pump: &ScreenPump{screen: s, interval: 30 * time.Second,
		partialPolicy: refreshpolicy.Policy{Normal: time.Minute, Urgent: 45 * time.Second}}}
	r.completed()
	clock = 40 * time.Second
	r.postpone(30 * time.Second)
	checkRecoveryLanes(t, &r, [4]time.Duration{30 * time.Second, 30 * time.Second, time.Minute, 45 * time.Second})
	// A previously established, longer transport floor survives a later shorter one.
	r.postpone(2 * time.Minute)
	clock += time.Second
	r.postpone(30 * time.Second)
	checkRecoveryLanes(t, &r, [4]time.Duration{119 * time.Second, 119 * time.Second, 119 * time.Second, 119 * time.Second})
}

func checkRecoveryLanes(t *testing.T, r *screenRecovery, want [4]time.Duration) {
	t.Helper()
	options := []refreshpolicy.Options{
		{Mode: refreshpolicy.Full}, {Mode: refreshpolicy.Full, Priority: refreshpolicy.Urgent},
		{Mode: refreshpolicy.Partial}, {Mode: refreshpolicy.Partial, Priority: refreshpolicy.Urgent},
	}
	for i, option := range options {
		if got := r.remainingFor(option); got != want[i] {
			t.Fatalf("mode=%v priority=%v: remaining=%v want=%v", option.Mode, option.Priority, got, want[i])
		}
	}
	if got := r.remainingFor(refreshpolicy.Options{}); got != want[0] {
		t.Fatalf("unresolved Auto used partial deadline: %v, full=%v", got, want[0])
	}
}
