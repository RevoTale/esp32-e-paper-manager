package streamrx

import (
	"errors"
	"testing"
	"time"
)

func TestInvalidateAbortsOnceAndRetainsAbortFailure(t *testing.T) {
	r, s, intent := fixture(t)
	if err := r.Begin(intent, 0); err != nil {
		t.Fatal(err)
	}
	broken := errors.New("power cleanup failed")
	s.errAbort = broken
	err := r.Invalidate(nil)
	if !errors.Is(err, ErrState) || !errors.Is(err, broken) {
		t.Fatal(err)
	}
	if r.Status().State != Failed || !errors.Is(r.Status().Failure, broken) {
		t.Fatal(r.Status())
	}
	if err = r.Invalidate(ErrChunk); !errors.Is(err, ErrChunk) {
		t.Fatal(err)
	}
	if s.aborts != 1 || s.commits != 0 {
		t.Fatal("incorrect abort/commit count")
	}
}

func TestExpiredOperationsAbortBeforeIO(t *testing.T) {
	for _, operation := range []string{"begin", "write", "commit"} {
		r, s, intent := fixture(t)
		if err := r.Begin(intent, 0); err != nil {
			t.Fatal(err)
		}
		var err error
		switch operation {
		case "begin":
			err = r.Begin(intent, time.Second)
		case "write":
			err = r.Write(intent, 0, 0, []byte{128, 1}, time.Second)
		case "commit":
			err = r.Commit(intent, time.Second)
		}
		if !errors.Is(err, ErrTimeout) {
			t.Fatal(err)
		}
		if [4]int{s.aborts, s.begin, s.writes, s.commits} != [4]int{1, 1, 0, 0} {
			t.Fatal("expired operation reached sink")
		}
	}
}
