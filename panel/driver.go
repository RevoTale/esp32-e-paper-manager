package panel

import (
	"sync/atomic"
	"time"
)

const (
	FrameWidth  = 800
	FrameHeight = 480
	FrameBytes  = FrameWidth * FrameHeight / 8
	rowBytes    = FrameWidth / 8

	pollInterval        = 5 * time.Millisecond
	powerOffDelay       = 100 * time.Millisecond
	powerSettleDelay    = 100 * time.Millisecond
	resetHighDelay      = 20 * time.Millisecond
	resetLowDelay       = 2 * time.Millisecond
	powerOnCommandDelay = 100 * time.Millisecond
	refreshCommandDelay = 100 * time.Millisecond
	powerOnBudget       = 10 * time.Second
	refreshBudget       = 30 * time.Second
	powerOffBudget      = 10 * time.Second
	progressBytes       = 12_000
)

type IO struct {
	Write    func([]byte) error
	SetCS    func(high bool)
	SetDC    func(high bool)
	SetReset func(high bool)
	SetPower func(high bool)
	ReadBusy func() bool
	Delay    func(time.Duration)
	Observe  func(Event)
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type Driver struct {
	noCopy      noCopy
	active      uint32
	io          IO
	commandByte [1]byte
	small       [4]byte
	row         [rowBytes]byte
}

func New(io IO) (*Driver, error) {
	if io.Write == nil || io.SetCS == nil || io.SetDC == nil ||
		io.SetReset == nil || io.SetPower == nil || io.ReadBusy == nil ||
		io.Delay == nil {
		return nil, ErrConfig
	}
	return &Driver{io: io}, nil
}

// Refresh performs the Waveshare 7.5-inch V2 full-refresh and deep-sleep
// sequence. Source: https://github.com/waveshareteam/Pico_ePaper_Code/blob/main/c/lib/e-Paper/EPD_7in5_V2.c
func (d *Driver) Refresh(frame []byte) error {
	if len(frame) != FrameBytes {
		return ErrFrameSize
	}
	if !atomic.CompareAndSwapUint32(&d.active, 0, 1) {
		return ErrInUse
	}
	defer atomic.StoreUint32(&d.active, 0)

	return d.refresh(frame)
}

func (d *Driver) refresh(frame []byte) error {
	defer d.releasePower()
	if err := d.start(); err != nil {
		return err
	}
	// Project frames use 0=white and 1=black. Preserve verified plane polarity.
	if err := d.writePlane(PhaseOldPlane, StepOldPlane, 0x10, frame, true); err != nil {
		return err
	}
	if err := d.writePlane(PhaseNewPlane, StepNewPlane, 0x13, frame, false); err != nil {
		return err
	}
	return d.finish()
}

func (d *Driver) releasePower() {
	d.io.SetCS(true)
	d.io.SetPower(false)
}

func (d *Driver) sendCommand(phase Phase, step Step, command byte) error {
	return d.sendData(phase, step, command, nil)
}

func (d *Driver) sendSmall(phase Phase, step Step, command, a, b, c, e byte, size int) error {
	d.small = [4]byte{a, b, c, e}
	return d.sendData(phase, step, command, d.small[:size])
}

func (d *Driver) sendData(phase Phase, step Step, command byte, data []byte) error {
	d.observe(Event{Kind: EventCommand, Phase: phase, Step: step, Command: command, Bytes: uint32(len(data))})
	if err := d.command(command); err != nil {
		return OpError{Phase: phase, Step: step, Command: command, Offset: -1, Cause: err}
	}
	if len(data) == 0 {
		return nil
	}
	if err := d.data(data); err != nil {
		return OpError{Phase: phase, Step: step, Command: command, Offset: 0, Cause: err}
	}
	return nil
}

func (d *Driver) writePlane(phase Phase, step Step, command byte, frame []byte, invert bool) error {
	d.observe(Event{Kind: EventCommand, Phase: phase, Step: step, Command: command})
	if err := d.command(command); err != nil {
		return OpError{Phase: phase, Step: step, Command: command, Offset: -1, Cause: err}
	}
	d.observe(Event{Kind: EventTransferStart, Phase: phase, Step: step, Command: command, Bytes: FrameBytes})
	for offset := 0; offset < len(frame); offset += rowBytes {
		row := frame[offset : offset+rowBytes]
		if invert {
			for i, value := range row {
				d.row[i] = ^value
			}
			row = d.row[:]
		}
		if err := d.data(row); err != nil {
			return OpError{Phase: phase, Step: step, Command: command, Offset: offset, Cause: err}
		}
		completed := offset + rowBytes
		if completed%progressBytes == 0 || completed == len(frame) {
			d.observe(Event{Kind: EventTransferProgress, Phase: phase, Step: step, Command: command, Offset: completed, Bytes: FrameBytes})
		}
	}
	d.observe(Event{Kind: EventTransferDone, Phase: phase, Step: step, Command: command, Offset: FrameBytes, Bytes: FrameBytes})
	return nil
}

func (d *Driver) command(value byte) error {
	d.io.SetDC(false)
	d.io.SetCS(false)
	d.commandByte[0] = value
	err := d.io.Write(d.commandByte[:])
	d.io.SetCS(true)
	return err
}

func (d *Driver) data(data []byte) error {
	d.io.SetDC(true)
	d.io.SetCS(false)
	err := d.io.Write(data)
	d.io.SetCS(true)
	return err
}

func (d *Driver) waitIdle(phase Phase, step Step, command byte, budget time.Duration) error {
	d.observe(Event{Kind: EventBusyWait, Phase: phase, Step: step, Command: command})
	remaining := budget
	var samples, lowSamples uint32
	for {
		busyHigh := d.io.ReadBusy()
		samples++
		if busyHigh {
			d.observe(Event{Kind: EventBusyDone, Phase: phase, Step: step, Command: command, Busy: true, BusyKnown: true, Samples: samples, LowSamples: lowSamples})
			return nil
		}
		lowSamples++
		if remaining == 0 {
			return OpError{Phase: phase, Step: step, Command: command, Offset: -1, Busy: false, BusyKnown: true, Cause: ErrBusyTimeout}
		}
		delay := min(remaining, pollInterval)
		d.io.Delay(delay)
		remaining -= delay
	}
}

func (d *Driver) observe(event Event) {
	if d.io.Observe != nil {
		d.io.Observe(event)
	}
}
