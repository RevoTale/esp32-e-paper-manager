package screenusbhost

import (
	"bytes"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
)

type repeatPanelLink struct {
	connection *screenlink.Connection
	output     bytes.Buffer
	now        *time.Duration
}

func (p *repeatPanelLink) Read(data []byte) (int, error) { return p.output.Read(data) }

func (p *repeatPanelLink) Write(data []byte) (int, error) {
	err := p.connection.Push(data, *p.now, func(reply []byte) error {
		_, err := p.output.Write(reply)
		return err
	})
	return len(data), err
}

type repeatPanelCycle struct {
	planes   [2][]byte
	commands []byte
	refresh  time.Duration
}

type repeatPanelIO struct {
	cycles                     []repeatPanelCycle
	now                        time.Duration
	command                    byte
	dc, powered, alwaysHigh    bool
	reads, lowReads, refreshes int
}

func (r *repeatPanelIO) io() panel.IO {
	return panel.IO{
		Write: r.write, SetCS: func(bool) {}, SetDC: func(high bool) { r.dc = high },
		SetReset: func(bool) {}, ReadBusy: r.readBusy,
		SetPower: func(high bool) {
			r.powered = high
			if high {
				r.cycles = append(r.cycles, repeatPanelCycle{})
			}
		},
		Delay: func(duration time.Duration) { r.now += duration },
	}
}

func (r *repeatPanelIO) readBusy() bool {
	r.reads++
	high := r.alwaysHigh || r.reads%2 == 0
	if !high {
		r.lowReads++
	}
	return high
}

func (r *repeatPanelIO) write(data []byte) error {
	cycle := &r.cycles[len(r.cycles)-1]
	if !r.dc {
		r.command = data[0]
		cycle.commands = append(cycle.commands, r.command)
		if r.command == 0x12 {
			r.refreshes++
			cycle.refresh = r.now
		}
		return nil
	}
	switch r.command {
	case 0x10:
		cycle.planes[0] = append(cycle.planes[0], data...)
	case 0x13:
		cycle.planes[1] = append(cycle.planes[1], data...)
	}
	return nil
}

func assertRepeatPanelCycle(t *testing.T, recorder *repeatPanelIO, index int, frame display.Frame) {
	t.Helper()
	if len(recorder.cycles) != index+1 || recorder.refreshes != index+1 || recorder.powered {
		t.Fatalf("transfer %d: power cycles=%d refreshes=%d powered=%t", index,
			len(recorder.cycles), recorder.refreshes, recorder.powered)
	}
	cycle := recorder.cycles[index]
	wantCommands := []byte{0x01, 0x06, 0x04, 0x00, 0x61, 0x15, 0x50, 0x60, 0x10, 0x13, 0x12, 0x50, 0x02, 0x07}
	if !bytes.Equal(cycle.commands, wantCommands) {
		t.Fatalf("transfer %d commands=%x", index, cycle.commands)
	}
	assertRepeatPanelPlanes(t, cycle, index, frame)
	if index > 0 && cycle.refresh-recorder.cycles[index-1].refresh < 180*time.Second {
		t.Fatal("second refresh bypassed the accepted cadence")
	}
}

func assertRepeatPanelPlanes(t *testing.T, cycle repeatPanelCycle, index int, frame display.Frame) {
	t.Helper()
	for pass, actual := range cycle.planes {
		want := bytes.Clone(frame.Bytes())
		if pass == 0 {
			for offset := range want {
				want[offset] = ^want[offset]
			}
		}
		if !bytes.Equal(actual, want) {
			t.Fatalf("transfer %d plane 0x%02x differs: got %d bytes, want %d", index,
				[]byte{0x10, 0x13}[pass], len(actual), len(want))
		}
	}
}
