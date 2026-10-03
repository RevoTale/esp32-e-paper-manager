package panel

import "errors"

var (
	ErrFrameSize   = errors.New("panel: wrong frame size")
	ErrConfig      = errors.New("panel: invalid IO configuration")
	ErrInUse       = errors.New("panel: driver is already in use")
	ErrBusyTimeout = errors.New("panel: BUSY remained LOW until deadline")
)

type Phase uint8

const (
	PhaseInitialize Phase = iota + 1
	PhasePowerOn
	PhaseOldPlane
	PhaseNewPlane
	PhaseRefresh
	PhasePowerOff
)

func (p Phase) String() string {
	switch p {
	case PhaseInitialize:
		return "initialize"
	case PhasePowerOn:
		return "power_on"
	case PhaseOldPlane:
		return "old_plane"
	case PhaseNewPlane:
		return "new_plane"
	case PhaseRefresh:
		return "refresh"
	case PhasePowerOff:
		return "power_off"
	default:
		return "unknown"
	}
}

type Step uint8

const (
	StepPowerOff Step = iota + 1
	StepPowerOn
	StepReset
	StepPowerSettings
	StepBoosterSoftStart
	StepControllerPowerOn
	StepPanelSettings
	StepResolution
	StepDualSPI
	StepVCOM
	StepTCON
	StepOldPlane
	StepNewPlane
	StepDisplayRefresh
	StepVCOMOff
	StepControllerPowerOff
	StepDeepSleep
)

func (s Step) String() string {
	names := [...]string{
		StepPowerOff:           "hat_power_off",
		StepPowerOn:            "hat_power_on",
		StepReset:              "reset",
		StepPowerSettings:      "power_settings",
		StepBoosterSoftStart:   "booster_soft_start",
		StepControllerPowerOn:  "controller_power_on",
		StepPanelSettings:      "panel_settings",
		StepResolution:         "resolution",
		StepDualSPI:            "dual_spi",
		StepVCOM:               "vcom",
		StepTCON:               "tcon",
		StepOldPlane:           "old_plane",
		StepNewPlane:           "new_plane",
		StepDisplayRefresh:     "display_refresh",
		StepVCOMOff:            "vcom_off",
		StepControllerPowerOff: "controller_power_off",
		StepDeepSleep:          "deep_sleep",
	}
	if int(s) >= len(names) || names[s] == "" {
		return "unknown"
	}
	return names[s]
}

type EventKind uint8

const (
	EventStep EventKind = iota + 1
	EventCommand
	EventTransferStart
	EventTransferProgress
	EventTransferDone
	EventBusyWait
	EventBusyDone
	EventComplete
)

func (k EventKind) String() string {
	switch k {
	case EventStep:
		return "step"
	case EventCommand:
		return "command"
	case EventTransferStart:
		return "transfer_start"
	case EventTransferProgress:
		return "transfer_progress"
	case EventTransferDone:
		return "transfer_done"
	case EventBusyWait:
		return "busy_wait"
	case EventBusyDone:
		return "busy_done"
	case EventComplete:
		return "complete"
	default:
		return "unknown"
	}
}

type Event struct {
	Kind       EventKind
	Phase      Phase
	Step       Step
	Command    byte
	Offset     int
	Bytes      uint32
	Busy       bool
	BusyKnown  bool
	Samples    uint32
	LowSamples uint32
}

type OpError struct {
	Phase     Phase
	Step      Step
	Command   byte
	Offset    int
	Busy      bool
	BusyKnown bool
	Cause     error
}

func (e OpError) Error() string {
	return "panel: " + e.Phase.String() + "/" + e.Step.String() + " failed"
}

func (e OpError) Unwrap() error { return e.Cause }

func ErrorCode(err error) string {
	switch {
	case err == nil:
		return "OK"
	case errors.Is(err, ErrFrameSize):
		return "E_FRAME_SIZE"
	case errors.Is(err, ErrConfig):
		return "E_PANEL_CONFIG"
	case errors.Is(err, ErrInUse):
		return "E_PANEL_IN_USE"
	case errors.Is(err, ErrBusyTimeout):
		return "E_BUSY_TIMEOUT"
	default:
		var op OpError
		if errors.As(err, &op) {
			return "E_SPI_WRITE"
		}
		return "E_PANEL_UNKNOWN"
	}
}
