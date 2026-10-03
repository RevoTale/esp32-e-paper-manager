package manager

import (
	"context"
	"errors"
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/refreshstamp"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

// ScreenPartialOptions is local adapter/operator policy, never document input.
// Physical support must be negotiated by the RegionSender before transmission.
type ScreenPartialOptions struct {
	Rules          screendelivery.RegionRules
	MaxConsecutive uint16
}

func (s *Screen) configurePartial(options *ScreenPartialOptions) error {
	if options == nil {
		return nil
	}
	if options.MaxConsecutive == 0 {
		return ErrConfiguration
	}
	if err := options.Rules.Validate(s.size); err != nil {
		return err
	}
	copy := *options
	s.cycles.partialOptions = &copy
	return nil
}

func (s *Screen) prepareRegionDelivery(d ScreenDelivery) (ScreenDelivery, error) {
	c := s.cycles
	label, err := c.tracker.ForPartial()
	if err != nil || s.baseline == nil || c.partials >= c.partialOptions.MaxConsecutive {
		return s.regionFallback(d, refreshstamp.ErrUnconfirmed)
	}
	base, err := s.stampedFrame(s.baseline, label)
	if err == nil {
		d.Frame, err = s.stampedFrame(s.candidate.Bytes(), label)
	}
	if err != nil {
		return s.rejectRender(d.Revision, err)
	}
	plan, err := screendelivery.PlanRegion(base, d.Frame, c.partialOptions.Rules)
	if errors.Is(err, screendelivery.ErrRegionBudget) {
		return s.regionFallback(d, err)
	}
	if err != nil {
		return s.rejectRender(d.Revision, err)
	}
	if c.next == math.MaxUint64 {
		return s.rejectRender(d.Revision, refreshstamp.ErrCycle)
	}
	c.next++
	c.partial = true
	c.started = s.elapsed()
	s.state.InFlightCycle = c.next
	d.Cycle, d.Region, d.Options.Mode = c.next, &plan, refreshpolicy.Partial
	return d, nil
}

func (s *Screen) regionFallback(d ScreenDelivery, cause error) (ScreenDelivery, error) {
	if d.Options.Mode == refreshpolicy.Partial {
		return s.rejectPartial(d.Revision, cause)
	}
	return s.prepareFullDelivery(d)
}

func (s *Screen) stampedFrame(pixels []byte, label refreshstamp.Label) (display.Frame, error) {
	f, err := display.NewFrame(s.size, (s.size.Width+7)/8, append([]byte(nil), pixels...))
	if err == nil {
		err = refreshstamp.Paint(f, label)
	}
	return f, err
}

func (s *Screen) resolvePartial(confirmed bool) error {
	var err error
	if confirmed {
		_, err = s.cycleElapsed()
		if err == nil {
			_, err = s.cycles.tracker.ForPartial()
		}
	}
	if !confirmed || err != nil {
		s.invalidateStamp()
		return errors.Join(err, s.resolvePixels(false))
	}
	s.cycles.partials++
	s.state.RefreshTrusted = true
	return s.resolvePixels(true)
}

func (r *screenRecovery) sendRegion(ctx context.Context) error {
	sender, ok := r.pump.sender.(screendelivery.RegionSender)
	if !ok {
		return ErrConfiguration
	}
	return sender.SendRegion(ctx, *r.pending.Region, r.pending.Options)
}
