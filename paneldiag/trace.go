package paneldiag

import (
	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"time"
)

// Trace observes one serialized panel owner. It retains no pixels, errors,
// tokens or transaction identities. Snapshot never calls hardware or the clock.
type Trace struct {
	status  screenwire.PanelStatus
	now     func() time.Time
	started time.Time
	wait    uint8
}

func NewTrace(now func() time.Time) *Trace {
	return &Trace{now: now, status: screenwire.PanelStatus{Version: 1}}
}

func (t *Trace) Snapshot() screenwire.PanelStatus { return t.status }

// WrapIO preserves every pin/write/delay callback and forwards each BUSY read
// exactly once. Only reads inside existing driver waits are counted; a zero LOW
// count says nothing about the pin during command delays or visible pixels.
// Exact sequence: https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c
func (t *Trace) WrapIO(io panel.IO) panel.IO {
	read, observe := io.ReadBusy, io.Observe
	if read != nil {
		io.ReadBusy = func() bool {
			high := read()
			if t.status.State == 1 && t.wait != 0 {
				w := &t.status.Waits[t.wait-1]
				w.Samples = increment(w.Samples)
				if !high {
					w.LowSamples = increment(w.LowSamples)
				}
			}
			return high
		}
	}
	io.Observe = func(e panel.Event) {
		t.observe(e)
		if observe != nil {
			observe(e)
		}
	}
	return io
}

func (t *Trace) observe(e panel.Event) {
	if t.status.State != 1 {
		return
	}
	t.status.Phase, t.status.Step = uint8(e.Phase), uint8(e.Step)
	if e.Kind == panel.EventBusyDone {
		t.wait = 0
	}
	if e.Kind == panel.EventBusyWait {
		switch e.Phase {
		case panel.PhasePowerOn:
			t.wait = 1
		case panel.PhaseRefresh:
			t.wait = 2
		case panel.PhasePowerOff:
			t.wait = 3
		default:
			t.wait = 0
		}
	}
}

// WrapSink shares this recorder with WrapIO; configure both before owner starts.
// sink and clock are required, trusted composition dependencies.
func (t *Trace) WrapSink(sink streamrx.Sink) streamrx.Sink { return &tracedSink{trace: t, sink: sink} }

func increment(n uint32) uint32 {
	if n == ^uint32(0) {
		return n
	}
	return n + 1
}

func (t *Trace) end(state uint8) {
	if t.status.State != 1 {
		return
	}
	t.status.State, t.wait = state, 0
	ms := max(int64(0), t.now().Sub(t.started).Milliseconds())
	t.status.ElapsedMS = uint32(min(ms, int64(^uint32(0))))
}
