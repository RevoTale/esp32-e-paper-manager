package manager

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

var ErrSuperseded = errors.New("manager: render superseded by newer scene")

// ScreenRenderer must return canonical packed 1bpp with independently owned
// storage, never modifying it after return. engine.Renderer satisfies this.
type ScreenRenderer interface {
	Render(context.Context, display.Size, []byte) (display.Frame, error)
}

type detailedScreenRenderer interface {
	RenderDetailed(context.Context, display.Size, []byte) (display.Frame, []renderdiag.Warning, error)
}

type ScreenDelivery struct {
	Options  refreshpolicy.Options
	Revision renderbatch.Revision
	Cycle    refreshstamp.CycleID
	Frame    display.Frame
}

type ScreenStatus struct {
	Current        renderbatch.Revision `json:"current"`
	InFlight       renderbatch.Revision `json:"in_flight"`
	Confirmed      renderbatch.Revision `json:"confirmed"`
	Delivered      renderbatch.Revision `json:"delivered"`
	Failure        *ScreenFailure       `json:"failure,omitempty"`
	Warnings       []renderdiag.Warning `json:"warnings,omitempty"`
	InFlightCycle  refreshstamp.CycleID `json:"in_flight_cycle,omitempty"`
	FullRefresh    *ScreenRefresh       `json:"full_refresh,omitempty"`
	RefreshTrusted bool                 `json:"refresh_trusted"`
}

type ScreenFailure struct {
	Revision   renderbatch.Revision `json:"revision"`
	Diagnostic renderdiag.Error     `json:"diagnostic"`
}

// Screen coordinates complete scenes, including atomic by-ID authoring edits. The caller
// authenticates submissions and maps their base revision to HTTP If-Match:
// https://www.rfc-editor.org/rfc/rfc9110.html#name-if-match
// This server-only coordinator does not change the legacy manager HTTP API.
type Screen struct {
	options, flightOptions refreshpolicy.Options
	refreshEnabled         bool // Fixed by pump construction before serving requests.
	mu                     sync.Mutex
	pumping                atomic.Bool
	started                time.Time
	wake                   chan struct{}
	renderer               ScreenRenderer
	size                   display.Size
	queue                  *renderbatch.Queue
	markup                 string
	state                  ScreenStatus
	prepared               bool
	candidate              display.Frame
	baseline               []byte
	cycles                 *screenCycles
	elapsedClock           func() time.Duration
}

func NewScreen(renderer ScreenRenderer, size display.Size, policy renderbatch.Policy) (*Screen, error) {
	if renderer == nil || size.Width <= 0 || size.Height <= 0 || size.Width > 2048 || size.Height > 2048 || size.Width*size.Height > 1_048_576 {
		return nil, ErrConfiguration
	}
	q, err := renderbatch.New(policy)
	if err != nil {
		return nil, err
	}
	return &Screen{renderer: renderer, size: size, queue: q, started: time.Now(), wake: make(chan struct{}, 1)}, nil
}

// Submit copies bounded UTF-8 input. An old base never overwrites a newer scene.
func (s *Screen) Submit(base renderbatch.Revision, markup []byte, now time.Duration) (renderbatch.Revision, error) {
	return s.submitAt(base, markup, func() time.Duration { return now })
}

// The live API samples elapsed time inside the same lock as queue mutation:
// concurrent callers must not submit an earlier sampled clock observation late.
func (s *Screen) submitAt(base renderbatch.Revision, markup []byte, now func() time.Duration) (renderbatch.Revision, error) {
	return s.submitOptionsAt(base, markup, now, refreshpolicy.Options{})
}

func (s *Screen) submitOptionsAt(base renderbatch.Revision, markup []byte, now func() time.Duration, options refreshpolicy.Options) (renderbatch.Revision, error) {
	if len(markup) > 32768 || !utf8.Valid(markup) {
		return 0, ErrConfiguration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if base != s.state.Current || base == math.MaxUint64 {
		return 0, ErrConflict
	}
	next := base + 1
	submit := s.queue.Submit
	if options.Priority == refreshpolicy.Urgent {
		submit = s.queue.SubmitUrgent
	}
	if err := submit(next, now()); err != nil {
		return 0, err
	}
	s.markup, s.state.Current = string(markup), next
	s.options = options
	s.state.Failure = nil
	s.state.Warnings = nil
	s.clearRejected()
	s.notify()
	return next, nil
}

// RenderNext leases one ready target, then renders without blocking new edits.
// A result superseded during rendering is discarded before transport. Zero
// revision means not ready/available. Borrow the returned frame read-only until
// Resolve; no second consumer or premature success may resolve this lease.
func (s *Screen) RenderNext(ctx context.Context, now time.Duration, available bool) (ScreenDelivery, error) {
	return s.renderAt(ctx, func() time.Duration { return now }, available)
}

func (s *Screen) renderAt(ctx context.Context, now func() time.Duration, available bool) (ScreenDelivery, error) {
	s.mu.Lock()
	rev, reuse, err := s.takeScene(now(), available)
	if err != nil || rev == 0 {
		s.mu.Unlock()
		return ScreenDelivery{}, err
	}
	s.state.InFlight = rev
	s.flightOptions = s.options
	if s.refreshLease() {
		s.flightOptions = refreshpolicy.Options{Mode: refreshpolicy.Full}
	}
	if reuse {
		defer s.mu.Unlock()
		return s.prepareBaseline()
	}
	markup := s.markup
	s.mu.Unlock()
	frame, warnings, err := s.renderFrame(ctx, []byte(markup))
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finishRender(rev, frame, warnings, err)
}

func (s *Screen) renderFrame(ctx context.Context, markup []byte) (display.Frame, []renderdiag.Warning, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if s.cycles != nil {
		frame, warnings, err := s.cycles.renderer.RenderReserved(ctx, s.size, markup, s.cycles.area)
		return frame, warnings, nativeRenderError(err)
	}
	if detailed, ok := s.renderer.(detailedScreenRenderer); ok {
		return detailed.RenderDetailed(ctx, s.size, markup)
	}
	frame, err := s.renderer.Render(ctx, s.size, markup)
	return frame, nil, err
}

// Called with mu held: validate the result before classifying supersession.
func (s *Screen) finishRender(rev renderbatch.Revision, frame display.Frame, warnings []renderdiag.Warning, err error) (ScreenDelivery, error) {
	if err == nil && !s.validFrame(frame) {
		err = display.ErrFrameGeometry
	}
	if rev != s.state.Current && (err == nil || errors.Is(err, renderdiag.ErrRejected)) {
		err = errors.Join(err, ErrSuperseded)
	}
	if err != nil {
		return s.rejectRender(rev, err)
	}
	s.prepared = true
	s.state.Warnings = append([]renderdiag.Warning(nil), warnings...)
	if s.samePixels(frame) && !s.refreshLease() {
		s.state.Confirmed = rev
		return ScreenDelivery{}, s.release()
	}
	s.candidate = frame
	return s.prepareDelivery()
}

func (s *Screen) rejectRender(rev renderbatch.Revision, err error) (ScreenDelivery, error) {
	if s.cycles != nil && errors.Is(err, renderdiag.ErrRejected) && rev == s.state.Current {
		s.cycles.rejected = rev
	}
	if releaseErr := s.release(); releaseErr != nil {
		// Queue invariants take precedence over a recoverable document error.
		return ScreenDelivery{}, releaseErr
	}
	var diagnostic *renderdiag.Error
	if rev == s.state.Current && errors.As(err, &diagnostic) {
		s.state.Failure = &ScreenFailure{Revision: rev, Diagnostic: *diagnostic}
	}
	return ScreenDelivery{}, err
}

// Resolve records protocol completion, not visible acceptance. Unknown/failure
// invalidates the confirmed baseline. No implicit retry; the caller must submit
// a complete current scene again after resynchronizing transport. Renderer
// errors similarly preserve current scene but require an explicit new submit.
func (s *Screen) Resolve(rev renderbatch.Revision, confirmed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cycles != nil || !s.prepared || rev != s.state.InFlight {
		return ErrConflict
	}
	return s.resolvePixels(confirmed)
}

func (s *Screen) resolvePixels(confirmed bool) error {
	if confirmed {
		s.state.Confirmed = s.state.InFlight
		s.state.Delivered = s.state.InFlight
		s.baseline = append(s.baseline[:0], s.candidate.Bytes()...)
	} else {
		s.state.Confirmed = 0
		s.baseline = nil
	}
	return s.release()
}

func (s *Screen) Status() ScreenStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.state
	status.Warnings = append([]renderdiag.Warning(nil), status.Warnings...)
	if status.Failure != nil {
		failure := *status.Failure
		status.Failure = &failure
	}
	if status.FullRefresh != nil {
		refresh := *status.FullRefresh
		status.FullRefresh = &refresh
	}
	return status
}

func (s *Screen) release() error {
	var err error
	if s.cycles == nil || !s.cycles.forced {
		err = s.queue.Release(s.state.InFlight)
	}
	if s.cycles != nil {
		s.cycles.forced, s.cycles.refresh = false, false
	}
	s.state.InFlightCycle = 0
	s.state.InFlight, s.prepared = 0, false
	s.candidate = display.Frame{}
	s.notify()
	return err
}

func (s *Screen) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Screen) validFrame(f display.Frame) bool {
	if f.Size() != s.size || f.Stride() != (s.size.Width+7)/8 {
		return false
	}
	if s.size.Width%8 == 0 {
		return true
	}
	mask := byte(0xff >> uint(s.size.Width%8))
	for y := 0; y < s.size.Height; y++ {
		if f.Bytes()[(y+1)*f.Stride()-1]&mask != 0 {
			return false
		}
	}
	return true
}
