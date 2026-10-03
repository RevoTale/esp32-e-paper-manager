package renderbatch

import (
	"testing"
	"time"
)

func TestOptionalDelayPolicy(t *testing.T) {
	for _, policy := range []Policy{{}, {MaxWait: time.Hour}} {
		q, err := New(policy)
		if err != nil {
			t.Fatal(err)
		}
		if err = q.Submit(1, 0); err != nil {
			t.Fatal(err)
		}
		rev, err := q.Take(0, true)
		if err != nil || rev != 1 {
			t.Fatalf("rev=%d err=%v", rev, err)
		}
	}
}

func TestDebounceBoundedByFirstArrival(t *testing.T) {
	q, err := New(Policy{Debounce: 4 * time.Second, MaxWait: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		if err = q.Submit(Revision(i), time.Duration(i-1)*time.Second); err != nil {
			t.Fatal(err)
		}
		if rev, err := q.Take(time.Duration(i-1)*time.Second, true); err != nil || rev != 0 {
			t.Fatal(rev, err)
		}
	}
	if rev, err := q.Take(5*time.Second, true); err != nil || rev != 5 {
		t.Fatal(rev, err)
	}
}

func TestOneInFlightAndNewestPending(t *testing.T) {
	q := queue(t, Policy{})
	must(t, q.Submit(1, 0))
	if rev, err := q.Take(0, true); err != nil || rev != 1 {
		t.Fatal(rev, err)
	}
	for i := 2; i < 10; i++ {
		must(t, q.Submit(Revision(i), 0))
	}
	if rev, err := q.Take(0, true); err != nil || rev != 0 {
		t.Fatal(rev, err)
	}
	must(t, q.Release(1))
	if rev, err := q.Take(0, false); err != nil || rev != 0 {
		t.Fatal(rev, err)
	}
	if rev, err := q.Take(0, true); err != nil || rev != 9 {
		t.Fatal(rev, err)
	}
}
