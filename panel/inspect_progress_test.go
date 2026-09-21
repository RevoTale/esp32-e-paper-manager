package panel

import (
	"testing"
)

func inspectProgress(t *testing.T, events []Event) (progress, busyDone, complete int) {
	t.Helper()
	for _, event := range events {
		switch event.Kind {
		case EventTransferProgress:
			progress++
			assertProgressEvent(t, event)
		case EventBusyDone:
			busyDone++
			if !event.BusyKnown || !event.Busy || event.Samples == 0 {
				t.Fatalf("invalid BUSY evidence: %#v", event)
			}
		case EventComplete:
			complete++
		}
	}

	return progress, busyDone, complete
}

func assertProgressEvent(t *testing.T, event Event) {
	t.Helper()
	if event.Offset <= 0 || event.Offset > FrameBytes || event.Bytes != FrameBytes {
		t.Fatalf("invalid progress event: %#v", event)
	}
}
