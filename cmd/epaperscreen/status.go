package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func runStatus(ctx context.Context, args []string, output io.Writer) (result error) {
	if len(args) != 1 {
		return errors.New("usage: epaperscreen --status SERIAL_PORT")
	}
	s, err := openTransport(args[0], display.Size{Width: 800, Height: 480})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, s.Close()) }()
	r, err := s.Inspect(ctx)
	if err != nil {
		return err
	}
	// Hello/Health carry a zero transaction result, not a live transfer snapshot.
	// Do not present its zeros as measured Idle/no-error/no-image evidence.
	_, err = fmt.Fprintf(output, "screen profile=%d version=%d size=%dx%d passes=%d chunk=%d minimum_full_ms=%d remaining_ms=%d\n"+
		"network state=%d last_failure=%d failures=%d uptime_seconds=%d\n"+
		"transfer_evidence=unavailable (Hello/Health; retained transaction required)\n",
		r.Capabilities.Profile, r.Capabilities.ProfileVersion, r.Capabilities.Width, r.Capabilities.Height, r.Capabilities.Passes, r.Capabilities.MaxChunk,
		r.Capabilities.MinimumFullMS, r.Status.CooldownMS, r.Health.State, r.Health.LastFailure, r.Health.Failures, r.Health.UptimeSeconds)
	return err
}
