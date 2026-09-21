package renderbatch

import (
	"errors"
	"math"
	"testing"
	"time"
)

func queue(t *testing.T, p Policy) *Queue {
	t.Helper()
	q, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPolicyAndClockRejections(t *testing.T) {
	for _, p := range []Policy{{Debounce: -1}, {MaxWait: -1}} {
		if _, err := New(p); !errors.Is(err, ErrPolicy) {
			t.Fatal(err)
		}
	}
	q := queue(t, Policy{})
	must(t, q.Submit(1, 10))
	if err := q.Submit(2, 9); !errors.Is(err, ErrClock) {
		t.Fatal(err)
	}
	if _, _, err := q.Wait(-1); !errors.Is(err, ErrClock) {
		t.Fatal(err)
	}
	if _, err := q.Take(9, true); !errors.Is(err, ErrClock) {
		t.Fatal(err)
	}
	if rev, err := q.Take(10, true); err != nil || rev != 1 {
		t.Fatal(rev, err)
	}
}

func TestStaleRevisionAndReleaseCannotLoseWork(t *testing.T) {
	q := queue(t, Policy{})
	if err := q.Release(0); !errors.Is(err, ErrRevision) {
		t.Fatal(err)
	}
	must(t, q.Submit(2, 0))
	for _, rev := range []Revision{0, 1, 2} {
		if err := q.Submit(rev, 0); !errors.Is(err, ErrRevision) {
			t.Fatal(err)
		}
	}
	if rev, err := q.Take(0, true); err != nil || rev != 2 {
		t.Fatal(rev, err)
	}
	if err := q.Release(1); !errors.Is(err, ErrRevision) {
		t.Fatal(err)
	}
	must(t, q.Release(2))
	if _, pending, err := q.Wait(0); err != nil || pending {
		t.Fatal(pending, err)
	}
}

func TestDebounceOnlyAndNewWindow(t *testing.T) {
	q := queue(t, Policy{Debounce: 10})
	must(t, q.Submit(1, 0))
	must(t, q.Submit(2, 9))
	if wait, _, err := q.Wait(10); err != nil || wait != 9 {
		t.Fatal(wait, err)
	}
	if rev, err := q.Take(19, true); err != nil || rev != 2 {
		t.Fatal(rev, err)
	}
	must(t, q.Submit(3, 20))
	must(t, q.Release(2))
	if wait, _, err := q.Wait(20); err != nil || wait != 10 {
		t.Fatal(wait, err)
	}
}

func TestMaxWaitShorterThanDebounceAndUnavailable(t *testing.T) {
	q := queue(t, Policy{Debounce: 10, MaxWait: 2})
	must(t, q.Submit(1, 0))
	if rev, err := q.Take(2, false); err != nil || rev != 0 {
		t.Fatal(rev, err)
	}
	must(t, q.Submit(2, 3))
	if rev, err := q.Take(3, true); err != nil || rev != 2 {
		t.Fatal(rev, err)
	}
}

func TestDurationsDoNotOverflow(t *testing.T) {
	q := queue(t, Policy{Debounce: time.Duration(math.MaxInt64), MaxWait: time.Duration(math.MaxInt64)})
	must(t, q.Submit(1, math.MaxInt64-1))
	if wait, _, err := q.Wait(math.MaxInt64); err != nil || wait != math.MaxInt64-1 {
		t.Fatal(wait, err)
	}
}

func TestSuccessfulCyclesAllocateNothing(t *testing.T) {
	q := queue(t, Policy{})
	var revision Revision
	allocs := testing.AllocsPerRun(100, func() {
		revision++
		must(t, q.Submit(revision, 0))
		got, err := q.Take(0, true)
		must(t, err)
		must(t, q.Release(got))
	})
	if allocs != 0 {
		t.Fatal(allocs)
	}
}
