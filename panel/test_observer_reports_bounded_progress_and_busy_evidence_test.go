package panel

import (
	"testing"
)

func TestObserverReportsBoundedProgressAndBusyEvidence(t *testing.T) {
	io := noOpIO()
	var events []Event
	io.Observe = func(event Event) { events = append(events, event) }
	driver, err := New(io)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Refresh(make([]byte, FrameBytes)); err != nil {
		t.Fatal(err)
	}

	progress, busyDone, complete := inspectProgress(t, events)
	if progress != 8 || busyDone != 3 || complete != 1 {
		t.Fatalf("events: progress=%d busy_done=%d complete=%d", progress, busyDone, complete)
	}
	if len(events) > 40 {
		t.Fatalf("observer emitted %d events for one refresh", len(events))
	}
}
