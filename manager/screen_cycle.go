package manager

import (
	"context"
	"errors"
	"image"
	"math"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

const DefaultMaintenanceInterval = 600 * time.Second

// ScreenOptions enables full-cycle stamps and maintenance. Now supplies trusted
// wall time only; scheduling uses elapsed monotonic time, never calendar time.
type ScreenOptions struct {
	Partial             *ScreenPartialOptions // Nil preserves full-only behavior; copied at construction.
	Zone                *time.Location
	MaintenanceInterval time.Duration // Zero selects 600 seconds; negative rejects.
	Now                 Clock         // Nil selects time.Now; the caller establishes clock trust.
}

// ScreenRefresh is historical matching-terminal-ACK evidence, not optical proof.
// RefreshTrusted becomes false after ambiguity, reset, or unusable completion time.
type ScreenRefresh struct {
	Cycle     refreshstamp.CycleID `json:"cycle"`
	Started   time.Time            `json:"started"`
	Completed time.Time            `json:"completed"`
}

type reservedScreenRenderer interface {
	RenderReserved(context.Context, display.Size, []byte, image.Rectangle) (display.Frame, []renderdiag.Warning, error)
}

type screenCycles struct {
	partialOptions          *ScreenPartialOptions
	partial                 bool
	partials                uint16
	tracker                 *refreshstamp.Tracker
	renderer                reservedScreenRenderer
	area                    image.Rectangle
	now                     Clock
	interval                time.Duration
	completed, started      time.Duration
	next                    refreshstamp.CycleID
	forced, refresh, resync bool
	rejected                renderbatch.Revision
}

// NewScreenWithOptions requires a renderer that explicitly reserves the corner
// before quantization. Legacy NewScreen remains unstamped for existing callers.
func NewScreenWithOptions(renderer ScreenRenderer, size display.Size, policy renderbatch.Policy, options ScreenOptions) (*Screen, error) {
	s, err := NewScreen(renderer, size, policy)
	if err != nil {
		return nil, err
	}
	reserved, ok := renderer.(reservedScreenRenderer)
	if !ok || options.MaintenanceInterval < 0 {
		return nil, ErrConfiguration
	}
	tracker, err := refreshstamp.New(options.Zone)
	if err != nil {
		return nil, err
	}
	area, err := refreshstamp.Bounds(size)
	if err != nil {
		return nil, err
	}
	if options.MaintenanceInterval == 0 {
		options.MaintenanceInterval = DefaultMaintenanceInterval
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	s.cycles = &screenCycles{tracker: tracker, renderer: reserved, area: area, now: options.Now, interval: options.MaintenanceInterval}
	if err := s.configurePartial(options.Partial); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Screen) prepareDelivery() (ScreenDelivery, error) {
	d := ScreenDelivery{Revision: s.state.InFlight, Frame: s.candidate, Options: s.flightOptions}
	if s.cycles == nil {
		return d, nil
	}
	if s.cycles.partialOptions != nil && d.Options.Mode != refreshpolicy.Full {
		return s.prepareRegionDelivery(d)
	}
	return s.prepareFullDelivery(d)
}

func (s *Screen) prepareFullDelivery(d ScreenDelivery) (ScreenDelivery, error) {
	d.Options.Mode = refreshpolicy.Full
	c := s.cycles
	if c.next == math.MaxUint64 {
		return s.rejectRender(d.Revision, refreshstamp.ErrCycle)
	}
	c.next++
	label, err := c.tracker.Begin(c.next, c.now())
	if err != nil {
		return s.rejectRender(d.Revision, err)
	}
	s.state.InFlightCycle, c.started = c.next, s.elapsed()
	d.Cycle = c.next
	d.Frame, err = display.NewFrame(s.size, s.candidate.Stride(), append([]byte(nil), s.candidate.Bytes()...))
	if err == nil {
		err = refreshstamp.Paint(d.Frame, label)
	}
	if err != nil {
		c.tracker.Invalidate()
		s.state.RefreshTrusted = false
		return s.rejectRender(d.Revision, err)
	}
	return d, nil
}

// ResolveCycle accepts only this physical lease, even when maintenance repeats
// the same HTML revision. A true result must be a matching terminal protocol ACK.
// A clock error after true means the physical cycle DID receive success, but its
// timestamp proof is unusable; no historical confirmation is advanced.
func (s *Screen) ResolveCycle(id refreshstamp.CycleID, confirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles == nil || !s.prepared || id == 0 || id != s.state.InFlightCycle {
		return ErrConflict
	}
	c := s.cycles
	if c.partial {
		return s.resolvePartial(confirmed)
	}
	var err error
	var completed time.Duration
	if confirmed {
		completed, err = s.completeCycle(id)
	}
	if !confirmed || err != nil {
		s.invalidateStamp()
		return errors.Join(err, s.resolvePixels(false))
	}
	proof := c.tracker.Confirmed()
	s.state.FullRefresh = &ScreenRefresh{Cycle: proof.Cycle, Started: proof.Started, Completed: proof.Completed}
	s.state.RefreshTrusted, c.completed = true, completed
	c.partials = 0
	return s.resolvePixels(true)
}

func (s *Screen) completeCycle(id refreshstamp.CycleID) (time.Duration, error) {
	completed, err := s.cycleElapsed()
	if err != nil {
		return 0, err
	}
	return completed, s.cycles.tracker.Complete(id, s.cycles.now())
}

func (s *Screen) cycleElapsed() (time.Duration, error) {
	completed := s.elapsed()
	if _, _, err := s.queue.Wait(completed); err != nil {
		return 0, err
	}
	if completed < s.cycles.started {
		return 0, renderbatch.ErrClock
	}
	return completed, nil
}

func (s *Screen) invalidateStamp() {
	if s.cycles != nil {
		s.cycles.tracker.Invalidate()
		s.state.RefreshTrusted = false
	}
}

func (s *Screen) clearRejected() {
	if s.cycles != nil {
		s.cycles.rejected = 0
	}
}

// A maintenance fallback to old pixels requires an explicit source-free scene
// diagnostic. A broken renderer contract is fatal, not an invented rejection.
func nativeRenderError(err error) error {
	if !errors.Is(err, renderdiag.ErrRejected) {
		return err
	}
	var diagnostic *renderdiag.Error
	if !errors.As(err, &diagnostic) || diagnostic == nil {
		return ErrConfiguration
	}
	return err
}
