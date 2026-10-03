//go:build !tinygo

package screenhub

import (
	"context"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

var _ screendelivery.RegionSender = (*Hub)(nil)

func (h *Hub) ConfigurePartial(policy refreshpolicy.Policy) error {
	if !h.cadence.Enabled() {
		return refreshpolicy.ErrPolicy
	}
	return h.cadence.ConfigurePartial(policy)
}

func (h *Hub) SendRegion(ctx context.Context, plan screendelivery.RegionPlan, options refreshpolicy.Options) error {
	if h.cadence.PartialPolicy.Normal == 0 {
		return screenclient.ErrUnsupportedRefresh
	}
	return h.send(ctx, options, func() error {
		r, err := plan.Wire(h.client.Baseline(), options, h.cadence.PartialPolicy)
		if err != nil {
			return err
		}
		return h.client.SendRegion(r, plan.Old, plan.New)
	})
}
