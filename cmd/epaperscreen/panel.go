package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"io"
)

type panelInspector interface {
	InspectPanel(context.Context) (screenwire.PanelStatus, error)
}

func runPanelStatus(ctx context.Context, args []string, output io.Writer) (result error) {
	if len(args) != 1 {
		return errors.New("usage: epaperscreen --panel-status SERIAL_PORT")
	}
	s, err := openTransport(args[0], display.Size{Width: 800, Height: 480})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, s.Close()) }()
	inspector, ok := s.(panelInspector)
	if !ok {
		return errors.New("panel trace unsupported by transport")
	}
	r, err := inspector.InspectPanel(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "panel_trace version=%d state=%d cycle=%d phase=%d step=%d elapsed_ms=%d\n"+
		"power_on_samples=%d power_on_low=%d refresh_samples=%d refresh_low=%d power_off_samples=%d power_off_low=%d\n"+
		"pixels=unverified (cached boot-local trace; not voltage or optical readback)\n",
		r.Version, r.State, r.Cycle, r.Phase, r.Step, r.ElapsedMS,
		r.Waits[0].Samples, r.Waits[0].LowSamples, r.Waits[1].Samples, r.Waits[1].LowSamples, r.Waits[2].Samples, r.Waits[2].LowSamples)
	return err
}
