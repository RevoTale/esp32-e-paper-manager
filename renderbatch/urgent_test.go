package renderbatch

import (
	"testing"
	"time"
)

func TestUrgentBypassesGroupingButNotActiveLease(t *testing.T) {
	q, err := New(Policy{Debounce: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if err = q.SubmitUrgent(1, 0); err != nil {
		t.Fatal(err)
	}
	if rev, err := q.Take(0, false); rev != 0 || err != nil {
		t.Fatal(rev, err)
	}
	if rev, err := q.Take(0, true); rev != 1 || err != nil {
		t.Fatal(rev, err)
	}
	checkUrgentDuringFlight(t, q)
}

func checkUrgentDuringFlight(t *testing.T, q *Queue) {
	t.Helper()
	if err := q.SubmitUrgent(2, 0); err != nil {
		t.Fatal(err)
	}
	if rev, err := q.Take(0, true); rev != 0 || err != nil {
		t.Fatal(rev, err)
	}
	if err := q.Release(1); err != nil {
		t.Fatal(err)
	}
	if rev, err := q.Take(0, true); rev != 2 || err != nil {
		t.Fatal(rev, err)
	}
}

func TestNormalReplacementDoesNotInheritUrgency(t *testing.T) {
	q, err := New(Policy{Debounce: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if err = q.SubmitUrgent(1, 0); err != nil {
		t.Fatal(err)
	}
	if err = q.SubmitUrgent(1, 0); err == nil {
		t.Fatal("stale accepted")
	}
	if err = q.Submit(2, 0); err != nil {
		t.Fatal(err)
	}
	if wait, pending, err := q.Wait(0); wait != time.Minute || !pending || err != nil {
		t.Fatal(wait, pending, err)
	}
}
