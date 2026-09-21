// Package waveshare75 composes the accepted 800x480 Waveshare 7.5 V2 profile.
// Other panels get separate adapters; generic EPS2/engine code never assumes it.
package waveshare75

import (
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/paneldiag"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

const (
	StagingIdle  = 20 * time.Second
	StagingTotal = 120 * time.Second
)

// New performs no panel I/O. Keep accepted two-plane polarity, 100-byte SPI
// scratch and full-cycle floor; do not infer partial support from panel size.
// https://github.com/waveshareteam/Pico_ePaper_Code/blob/c9bcd84db5adf5f085353649a8a5c31492bc5fb8/c/lib/e-Paper/EPD_7in5_V2.c
func New(io panel.IO, boot [16]byte) (*screenlink.Device, error) {
	trace := paneldiag.NewTrace(time.Now)
	driver, err := panel.New(trace.WrapIO(io))
	if err != nil {
		return nil, err
	}
	physical, err := panel.NewStream(driver)
	if err != nil {
		return nil, err
	}
	logical, err := streamrx.NewChunkedSink(trace.WrapSink(physical), 100)
	if err != nil {
		return nil, err
	}
	caps := screenwire.Capabilities{Width: 800, Height: 480, Stride: 100, MaxChunk: 1000,
		Passes: 2, Format: screenwire.Mono1, Features: screenwire.RawFull | screenwire.FeaturePackBits,
		Profile: 1, ProfileVersion: 1, MinimumFullMS: 180000}
	device, err := screenlink.New(caps, boot, logical, StagingIdle, StagingTotal, paneldiag.Describe)
	if err != nil {
		return nil, err
	}
	device.SetPanelTrace(trace.Snapshot)
	return device, nil
}
