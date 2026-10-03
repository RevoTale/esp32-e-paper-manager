package panel

import (
	"time"
)

func (r *recordingIO) io() IO {
	defaultBusyReads := 0
	return IO{
		Write: func(p []byte) error {
			copyOfP := append([]byte(nil), p...)
			r.events = append(r.events, wireEvent{kind: "write", data: copyOfP})
			return nil
		},
		SetCS: func(high bool) { r.events = append(r.events, wireEvent{kind: "cs", high: high}) },
		SetDC: func(high bool) { r.events = append(r.events, wireEvent{kind: "dc", high: high}) },
		SetReset: func(high bool) {
			r.events = append(r.events, wireEvent{kind: "reset", high: high})
		},
		SetPower: func(high bool) {
			r.events = append(r.events, wireEvent{kind: "power", high: high})
		},
		ReadBusy: func() bool {
			if len(r.busy) == 0 {
				value := defaultBusyReads%2 == 1
				defaultBusyReads++
				return value
			}
			if r.reads >= len(r.busy) {
				return true
			}
			value := r.busy[r.reads]
			r.reads++
			return value
		},
		Delay: func(d time.Duration) {
			r.events = append(r.events, wireEvent{kind: "delay", delay: d})
		},
	}
}
