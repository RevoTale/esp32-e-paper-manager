package paneldiag

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func newTraceFixture(sink streamrx.Sink, now func() time.Time) (*Trace, streamrx.Sink) {
	trace := NewTrace(now)
	return trace, trace.WrapSink(sink)
}

type traceSink struct {
	calls         []string
	failOperation string
	failure       error
	pass          uint8
	offset        uint32
	data          []byte
	onBegin       func()
}

func (s *traceSink) called(operation string) error {
	s.calls = append(s.calls, operation)
	if operation == s.failOperation {
		return s.failure
	}
	return nil
}

func (s *traceSink) Begin() error {
	if s.onBegin != nil {
		s.onBegin()
	}
	return s.called("begin")
}
func (s *traceSink) Commit() error { return s.called("commit") }
func (s *traceSink) Abort() error  { return s.called("abort") }
func (s *traceSink) Write(pass uint8, offset uint32, data []byte) error {
	s.pass, s.offset, s.data = pass, offset, bytes.Clone(data)
	return s.called("write")
}

func callTraceOperation(sink streamrx.Sink, operation string) error {
	switch operation {
	case "begin":
		return sink.Begin()
	case "write":
		return sink.Write(1, 200, []byte{7})
	case "commit":
		return sink.Commit()
	default:
		return sink.Abort()
	}
}

func traceWaits() [3]panel.Event {
	return [3]panel.Event{
		{Kind: panel.EventBusyWait, Phase: panel.PhasePowerOn, Step: panel.StepControllerPowerOn},
		{Kind: panel.EventBusyWait, Phase: panel.PhaseRefresh, Step: panel.StepDisplayRefresh},
		{Kind: panel.EventBusyWait, Phase: panel.PhasePowerOff, Step: panel.StepControllerPowerOff},
	}
}

func traceSampleWait(t *testing.T, wrapped panel.IO, event panel.Event, immediateHigh bool) {
	t.Helper()
	wrapped.Observe(event)
	if !immediateHigh && wrapped.ReadBusy() {
		t.Fatal("fixture should first read LOW")
	}
	if !wrapped.ReadBusy() {
		t.Fatal("fixture should finish HIGH")
	}
	event.Kind, event.Samples = panel.EventBusyDone, 1
	if !immediateHigh {
		event.Samples, event.LowSamples = 2, 1
	}
	wrapped.Observe(event)
}

func TestTracePreservesSinkArgumentsAndIOCallbacks(t *testing.T) {
	sink := &traceSink{}
	trace, wrappedSink := newTraceFixture(sink, func() time.Time { return time.Unix(1000, 0) })
	var calls []string
	failure := errors.New("SPI transport failure")
	io := panel.IO{
		Write:    func([]byte) error { calls = append(calls, "write"); return failure },
		SetCS:    func(bool) { calls = append(calls, "cs") },
		SetDC:    func(bool) { calls = append(calls, "dc") },
		SetReset: func(bool) { calls = append(calls, "reset") },
		SetPower: func(bool) { calls = append(calls, "power") },
		ReadBusy: func() bool { calls = append(calls, "busy"); return true },
		Delay:    func(time.Duration) { calls = append(calls, "delay") },
		Observe:  func(panel.Event) { calls = append(calls, "observe") },
	}
	wrapped := trace.WrapIO(io)
	if len(calls) != 0 {
		t.Fatal("wrapping performed IO")
	}
	if err := wrapped.Write([]byte{1}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	wrapped.SetCS(true)
	wrapped.SetDC(false)
	wrapped.SetReset(true)
	wrapped.SetPower(false)
	if !wrapped.ReadBusy() {
		t.Fatal("BUSY value changed")
	}
	wrapped.Delay(time.Millisecond)
	wrapped.Observe(traceWaits()[0])
	if !reflect.DeepEqual(calls, []string{"write", "cs", "dc", "reset", "power", "busy", "delay", "observe"}) {
		t.Fatalf("callback forwarding=%v", calls)
	}
	assertTraceSinkForwarding(t, wrappedSink, sink)
}

func assertTraceSinkForwarding(t *testing.T, wrappedSink streamrx.Sink, sink *traceSink) {
	t.Helper()
	if err := wrappedSink.Begin(); err != nil {
		t.Fatal(err)
	}
	payload := []byte{1, 2, 3}
	if err := wrappedSink.Write(1, 123, payload); err != nil {
		t.Fatal(err)
	}
	if err := wrappedSink.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := wrappedSink.Abort(); err != nil {
		t.Fatal(err)
	}
	if sink.pass != 1 || sink.offset != 123 || !bytes.Equal(sink.data, payload) ||
		!reflect.DeepEqual(sink.calls, []string{"begin", "write", "commit", "abort"}) {
		t.Fatalf("sink forwarding=%+v", sink)
	}
}
