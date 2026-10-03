package main

import (
	"errors"
	"flag"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

type partialConfig struct {
	enabled              bool
	policy               refreshpolicy.Policy
	maxUpdates, maxBytes int
}

func partialFlags(f *flag.FlagSet, p *partialConfig) {
	f.BoolVar(&p.enabled, "experimental-partial", false, "opt into unqualified 7.5 V2 partial candidate; requires region-capable firmware")
	f.DurationVar(&p.policy.Normal, "partial-interval", time.Second, "normal partial budget; not a panel safety guarantee")
	f.DurationVar(&p.policy.Urgent, "partial-urgent-interval", time.Second, "urgent partial budget; cannot bypass BUSY")
	f.IntVar(&p.maxUpdates, "partial-max-updates", 5, "operator limit of consecutive partials before full")
	f.IntVar(&p.maxBytes, "partial-max-bytes", 12000, "operator limit per packed old/new region plane")
}

func validatePartial(c screenConfig) error {
	p := c.partial
	if !p.enabled {
		return nil
	}
	if !c.refreshPolicy || c.size.Width != 800 || c.size.Height != 480 || p.maxUpdates < 1 || p.maxUpdates > 65535 || p.maxBytes < 4 || p.maxBytes > 48000 {
		return errors.New("partial candidate requires refresh-policy, 800x480, 1..65535 updates and 4..48000 bytes")
	}
	var cadence screendelivery.Cadence
	return cadence.ConfigurePartial(p.policy)
}

func (p partialConfig) options() *manager.ScreenPartialOptions {
	if !p.enabled {
		return nil
	}
	// Hardware geometry: V2 specification R90h; see docs/esp32-partial-research.md.
	return &manager.ScreenPartialOptions{
		Rules:          screendelivery.RegionRules{MinWidth: 16, MinHeight: 2, MaxBytes: p.maxBytes},
		MaxConsecutive: uint16(p.maxUpdates),
	}
}
