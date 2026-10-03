// Package screenlink joins EPS2 framing to the retained screen-session core.
// Callers authenticate network input, serialize ALL connections and poll Tick.
// One Device owns the panel; opening a connection alone never grants ownership.
package screenlink

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

type Device struct {
	session        *streamsession.Session
	caps           screenwire.Capabilities
	diagnose       func(error) screenwire.Diagnostic
	health         func() screenwire.HealthStatus
	panelTrace     func() screenwire.PanelStatus
	active         *Connection
	authentication uint64
	disabled       bool
	decoded        [screenwire.MaxPayload]byte // Shared only by the serialized owner.
}

func New(caps screenwire.Capabilities, boot [16]byte, sink streamrx.Sink, idle, total time.Duration, diagnose func(error) screenwire.Diagnostic) (*Device, error) {
	var scratch [screenwire.CapabilitiesSize]byte
	if err := screenwire.EncodeCapabilities(scratch[:], caps); err != nil {
		return nil, err
	}
	s, err := streamsession.New(boot, streamrx.Config{Width: caps.Width, Height: caps.Height, Passes: caps.Passes,
		MaxChunk: int(caps.MaxChunk), Idle: idle, Total: total}, sink, time.Duration(caps.MinimumFullMS)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	return &Device{session: s, caps: caps, diagnose: diagnose, authentication: 1}, nil
}

func (d *Device) Tick(now time.Duration) error { return d.session.Tick(now) }

// SetHealth is configured once before owner/network tasks start. The callback
// reads bounded cached state only: never radio/stack calls or hardware probes.
func (d *Device) SetHealth(provider func() screenwire.HealthStatus) { d.health = provider }

// SetPanelTrace installs an owner-serialized cached observer before tasks start.
// The provider must not read pins, advance timers, or perform any panel I/O.
func (d *Device) SetPanelTrace(provider func() screenwire.PanelStatus) { d.panelTrace = provider }

// Open follows transport authentication. USB priority is an explicit owner-loop
// decision through Preempt, never a consequence of receiving a Hello packet.
func (d *Device) Open() *Connection { return &Connection{device: d, authentication: d.authentication} }

// Reconfigure revokes every connection, including authenticated but unbound
// ones. Call before credential mutation (zero ID) and after authoritative Load.
// The owner must also stop old network workers and discard their queued events;
// a worker must never open a fresh connection with stale authentication.
// It retains refresh cadence; storage generations are not screen generations.
func (d *Device) Reconfigure(id [16]byte) error {
	d.caps.DeviceID = [16]byte{}
	if d.disabled || d.authentication == ^uint64(0) {
		d.disabled = true
		return streamsession.ErrLease
	}
	d.authentication++
	err := errors.Join(d.Preempt(), d.session.Fence())
	if err != nil {
		d.disabled = true
		return err
	}
	d.caps.DeviceID = id
	return nil
}

// Preempt aborts incomplete staging. Commit is synchronous; the serialized
// owner cannot enter here until refresh/BUSY/power-off finishes.
func (d *Device) Preempt() error {
	if d.active == nil {
		return nil
	}
	return d.active.Disconnect()
}

type Connection struct {
	device         *Device
	authentication uint64
	binding        streamsession.Binding
	tx             streamsession.Transaction
	buffer         [screenwire.MaxRecord]byte
	reply          [screenwire.MaxRecord]byte
	body           [screenwire.StatusSize + screenwire.CapabilitiesSize]byte
	used, expected int
}

// Revoked reports authentication-lifetime invalidation, not pixel ownership.
// A physical control callback can revoke even while only inspecting storage.
func (c *Connection) Revoked() bool { return !c.current() }

func (c *Connection) Disconnect() error {
	c.used, c.expected = 0, 0
	var err error
	if c.binding.Serial != 0 {
		err = c.device.session.Disconnect(c.binding)
	}
	c.binding = streamsession.Binding{}
	c.tx = streamsession.Transaction{}
	if c.device.active == c {
		c.device.active = nil
	}
	return err
}
