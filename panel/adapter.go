package panel

import "github.com/RevoTale/esp32-e-paper-manager/display"

type fullRefresher interface {
	Refresh([]byte) error
}

// Adapter exposes the verified 7.5-inch sequence through the generic display contract.
type Adapter struct {
	driver fullRefresher
}

func NewAdapter(driver *Driver) (*Adapter, error) {
	if driver == nil {
		return nil, ErrConfig
	}
	return &Adapter{driver: driver}, nil
}

func (a *Adapter) Capabilities() display.Capabilities {
	return display.Capabilities{
		Size: display.Size{Width: FrameWidth, Height: FrameHeight}, ColorModel: display.Mono1,
		RefreshModes: display.RefreshFull, WidthAlignment: 8, HeightAlignment: 1,
		StrideAlignment: 1, RefreshAutoSleeps: true,
	}
}

func (a *Adapter) Refresh(frame display.Frame, mode display.RefreshMode) error {
	if err := a.Capabilities().ValidateFrame(frame, mode); err != nil {
		return err
	}
	return a.driver.Refresh(frame.Bytes())
}

// Sleep is idempotent; Driver.Refresh already powers off and enters deep sleep.
func (a *Adapter) Sleep() error { return nil }

var _ display.Device = (*Adapter)(nil)
