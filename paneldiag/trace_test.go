package paneldiag

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestTraceSnapshotStartsUnavailableAndDoesNotPoll(t *testing.T) {
	clockCalls, reads := 0, 0
	sink := &traceSink{}
	trace, _ := newTraceFixture(sink, func() time.Time { clockCalls++; return time.Unix(1000, 0) })
	wrapped := trace.WrapIO(panel.IO{ReadBusy: func() bool { reads++; return true }})
	for range 3 {
		if got := trace.Snapshot(); got != (screenwire.PanelStatus{Version: 1}) {
			t.Fatalf("initial snapshot=%+v", got)
		}
	}
	if clockCalls != 0 || reads != 0 || len(sink.calls) != 0 {
		t.Fatalf("snapshot performed work: clock=%d reads=%d calls=%v", clockCalls, reads, sink.calls)
	}
	wrapped.Observe(panel.Event{Kind: panel.EventBusyWait, Phase: panel.PhaseRefresh, Step: panel.StepDisplayRefresh})
	if !wrapped.ReadBusy() || reads != 1 || trace.Snapshot() != (screenwire.PanelStatus{Version: 1}) {
		t.Fatal("inactive observation changed cached evidence or failed to forward read")
	}
}

func TestTraceRetainsCompletedCycleAndClearsNextCycle(t *testing.T) {
	now := time.Unix(1000, 0)
	sink := &traceSink{}
	trace, wrappedSink := newTraceFixture(sink, func() time.Time { return now })
	low := false
	wrapped := trace.WrapIO(panel.IO{ReadBusy: func() bool { low = !low; return !low }})
	if err := wrappedSink.Begin(); err != nil {
		t.Fatal(err)
	}
	for _, wait := range traceWaits() {
		traceSampleWait(t, wrapped, wait, false)
	}
	now = now.Add(1234 * time.Millisecond)
	if err := wrappedSink.Commit(); err != nil {
		t.Fatal(err)
	}
	completed := trace.Snapshot()
	assertTraceCompletedWaits(t, completed)
	now = now.Add(time.Hour)
	if err := wrappedSink.Abort(); err != nil || trace.Snapshot() != completed {
		t.Fatalf("completed evidence changed on cleanup: error=%v snapshot=%+v", err, trace.Snapshot())
	}
	if err := wrappedSink.Begin(); err != nil {
		t.Fatal(err)
	}
	want := screenwire.PanelStatus{Version: 1, State: 1, Cycle: 2}
	if got := trace.Snapshot(); got != want {
		t.Fatalf("next cycle inherited evidence: %+v", got)
	}
}

func TestTraceCapturesSynchronousBeginFailure(t *testing.T) {
	sink := &traceSink{failure: panel.ErrBusyTimeout, failOperation: "begin"}
	trace, wrappedSink := newTraceFixture(sink, func() time.Time { return time.Unix(1000, 0) })
	wrapped := trace.WrapIO(panel.IO{ReadBusy: func() bool { return false }})
	sink.onBegin = func() {
		wrapped.Observe(traceWaits()[0])
		_ = wrapped.ReadBusy()
	}
	if err := wrappedSink.Begin(); !errors.Is(err, panel.ErrBusyTimeout) {
		t.Fatalf("begin error=%v", err)
	}
	got := trace.Snapshot()
	if got.State != 3 || got.Cycle != 1 || got.Waits[0] != (screenwire.BusySamples{Samples: 1, LowSamples: 1}) {
		t.Fatalf("synchronous initialization failure not captured: %+v", got)
	}
}

func TestTraceImmediateHighCountsOnceAndStopsAtBusyDone(t *testing.T) {
	trace, wrappedSink := newTraceFixture(&traceSink{}, func() time.Time { return time.Unix(1000, 0) })
	reads := 0
	wrapped := trace.WrapIO(panel.IO{ReadBusy: func() bool { reads++; return true }})
	if err := wrappedSink.Begin(); err != nil {
		t.Fatal(err)
	}
	for _, wait := range traceWaits() {
		traceSampleWait(t, wrapped, wait, true)
		_ = wrapped.ReadBusy() // Outside a wait: forwarded, but not counted.
	}
	if reads != 6 {
		t.Fatalf("read forwarding count=%d", reads)
	}
	for slot, got := range trace.Snapshot().Waits {
		if got != (screenwire.BusySamples{Samples: 1}) {
			t.Fatalf("immediate-HIGH wait %d=%+v", slot, got)
		}
	}
}

func TestTraceErrorsRetainLowSamplesWithoutBusyDone(t *testing.T) {
	for _, operation := range []string{"begin", "write", "commit", "abort"} {
		t.Run(operation, func(t *testing.T) {
			now := time.Unix(1000, 0)
			sink := &traceSink{}
			trace, wrappedSink := newTraceFixture(sink, func() time.Time { return now })
			wrapped := trace.WrapIO(panel.IO{ReadBusy: func() bool { return false }})
			if err := wrappedSink.Begin(); err != nil {
				t.Fatal(err)
			}
			wrapped.Observe(traceWaits()[1])
			for range 3 {
				_ = wrapped.ReadBusy()
			}
			now = now.Add(17 * time.Millisecond)
			sink.failure, sink.failOperation = errors.New("private device detail"), operation
			if err := callTraceOperation(wrappedSink, operation); !errors.Is(err, sink.failure) {
				t.Fatalf("delegated error=%v", err)
			}
			got := trace.Snapshot()
			if got.State != 3 || got.Cycle != 1 || got.ElapsedMS != 17 ||
				got.Waits[1] != (screenwire.BusySamples{Samples: 3, LowSamples: 3}) {
				t.Fatalf("lost failed-cycle evidence: %+v", got)
			}
		})
	}
}

func TestTraceElapsedClampsRegressionAndSaturatesWithoutSnapshotClockReads(t *testing.T) {
	for _, tc := range []struct {
		name    string
		elapsed time.Duration
		want    uint32
	}{
		{"regression", -time.Second, 0},
		{"saturation", (time.Duration(math.MaxUint32) + 100) * time.Millisecond, math.MaxUint32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, calls := time.Unix(1000, 0), 0
			trace, wrappedSink := newTraceFixture(&traceSink{}, func() time.Time { calls++; return now })
			if err := wrappedSink.Begin(); err != nil {
				t.Fatal(err)
			}
			now = now.Add(tc.elapsed)
			if err := wrappedSink.Abort(); err != nil {
				t.Fatal(err)
			}
			before := calls
			for range 3 {
				if got := trace.Snapshot(); got.State != 3 || got.ElapsedMS != tc.want {
					t.Fatalf("elapsed snapshot=%+v", got)
				}
			}
			if calls != before {
				t.Fatal("snapshot called the clock")
			}
		})
	}
}

func assertTraceCompletedWaits(t *testing.T, completed screenwire.PanelStatus) {
	t.Helper()
	if completed.State != 2 || completed.Cycle != 1 || completed.ElapsedMS != 1234 {
		t.Fatalf("completion=%+v", completed)
	}
	if completed.Phase != uint8(panel.PhasePowerOff) || completed.Step != uint8(panel.StepControllerPowerOff) {
		t.Fatalf("last observed phase/step lost: %+v", completed)
	}
	for slot, got := range completed.Waits {
		if got != (screenwire.BusySamples{Samples: 2, LowSamples: 1}) {
			t.Fatalf("wait %d=%+v", slot, got)
		}
	}
}
