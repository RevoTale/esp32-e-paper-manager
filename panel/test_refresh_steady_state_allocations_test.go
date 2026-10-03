package panel

import (
	"testing"
)

func TestRefreshSteadyStateAllocations(t *testing.T) {
	io := noOpIO()
	driver, err := New(io)
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, FrameBytes)
	allocs := testing.AllocsPerRun(100, func() {
		if err := driver.Refresh(frame); err != nil {
			panic(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("allocations = %v, want 0", allocs)
	}
}
