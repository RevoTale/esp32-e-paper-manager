package screenusbhost

import (
	"context"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

var _ screendelivery.RegionSender = (*Sender)(nil)

func (s *Sender) ConfigurePartial(policy refreshpolicy.Policy) error {
	if !s.cadence.Enabled() {
		return refreshpolicy.ErrPolicy
	}
	return s.cadence.ConfigurePartial(policy)
}

func (s *Sender) SendRegion(ctx context.Context, plan screendelivery.RegionPlan, options refreshpolicy.Options) error {
	if s.cadence.PartialPolicy.Normal == 0 {
		return screenclient.ErrUnsupportedRefresh
	}
	return s.send(ctx, options, func() error {
		if s.rebind {
			if err := s.client.Rebind(); err != nil {
				return err
			}
		}
		r, err := plan.Wire(s.client.Baseline(), options, s.cadence.PartialPolicy)
		if err != nil {
			return err
		}
		return s.client.SendRegion(r, plan.Old, plan.New)
	})
}
