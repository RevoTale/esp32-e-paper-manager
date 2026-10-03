package screenusbhost

import (
	"context"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// InspectPanel reads one cached trace without retries or a screen lease. USB
// opening still has the existing DTR/priority effects, as documented by Inspect.
// Unsupported old firmware returns an error; absence is never a zero trace.
func (s *Sender) InspectPanel(ctx context.Context) (screenwire.PanelStatus, error) {
	if err := ctx.Err(); err != nil {
		return screenwire.PanelStatus{}, err
	}
	p, err := s.process()
	if err != nil {
		return screenwire.PanelStatus{}, err
	}
	var result screenwire.Response
	err = operation(ctx, p, func() error { var err error; result, err = screenclient.ReadPanelTrace(p); return err })
	if err != nil {
		return screenwire.PanelStatus{}, translate(err)
	}
	return result.Panel, nil
}
