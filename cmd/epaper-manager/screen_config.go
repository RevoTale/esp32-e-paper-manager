package main

import (
	"errors"
	"flag"
	"net"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type screenConfig struct {
	partial             partialConfig
	refreshPolicy       bool
	urgent              time.Duration
	usbWorker, serial   string
	transport, timezone string
	maintenance         time.Duration
	enabled             bool
	size                display.Size
	trusted             bool
	interval            time.Duration
	policy              renderbatch.Policy
}

func screenFlags(f *flag.FlagSet, c *screenConfig) {
	partialFlags(f, &c.partial)
	f.BoolVar(&c.enabled, "screen", false, "use the native Go HTML/inline-CSS screen renderer")
	f.StringVar(&c.transport, "screen-transport", "usb", "screen delivery: usb or wifi (EPN2)")
	f.StringVar(&c.timezone, "timezone", "Europe/Kiev", "USB corner time zone; Wi-Fi uses the USB-provisioned enrollment zone")
	f.DurationVar(&c.maintenance, "maintenance-interval", 10*time.Minute, "periodic full-refresh interval, still bounded by device cooldown")
	f.IntVar(&c.size.Width, "width", 800, "logical display width in pixels")
	f.IntVar(&c.size.Height, "height", 480, "logical display height in pixels")
	f.StringVar(&c.usbWorker, "usb-worker", "", "epaperscreen executable on this USB host (EPS2 serial proxy)")
	f.StringVar(&c.serial, "serial", "", "direct serial port; unified EPS2 firmware required")
	f.BoolVar(&c.trusted, "trusted-html", false, "acknowledge current renderer accepts trusted authors only")
	f.DurationVar(&c.interval, "full-interval", 180*time.Second, "normal refresh interval; shorter than 180s requires refresh-policy operator override")
	f.BoolVar(&c.refreshPolicy, "refresh-policy", false, "enable negotiated cadence override and urgent metadata; requires matching ESP32 firmware")
	f.DurationVar(&c.urgent, "urgent-interval", 30*time.Second, "urgent refresh budget with refresh-policy; not a manufacturer safety guarantee")
	f.DurationVar(&c.policy.Debounce, "debounce-interval", 0, "optional trailing batching interval")
	f.DurationVar(&c.policy.MaxWait, "max-wait", 0, "optional batching ceiling, never overrides full cadence")
}

func validateMode(flags *flag.FlagSet, c config) (config, error) {
	if !c.screen.enabled {
		return validateLegacy(c)
	}
	explicitAddress := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "https-listen" {
			explicitAddress = true
		}
	})
	if !explicitAddress {
		c.httpsAddress = "127.0.0.1:8443"
	}
	host, _, err := net.SplitHostPort(c.httpsAddress)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return config{}, errors.New("experimental screen mode is loopback-only until renderer public-input gates pass")
	}
	if err := validateScreen(c.screen); err != nil {
		return config{}, err
	}
	if c.screen.transport == "wifi" && c.enrollmentFile == "" {
		return config{}, errors.New("Wi-Fi screen delivery requires USB-created enrollment")
	}
	return c, nil
}

func validateLegacy(c config) (config, error) {
	if c.screen.partial.enabled || c.screen.usbWorker != "" || c.screen.serial != "" || c.screen.trusted || c.screen.policy != (renderbatch.Policy{}) {
		return config{}, errors.New("screen options require screen mode; no fallback to legacy delivery")
	}
	if c.enrollmentFile == "" || c.stateFile == "" {
		return config{}, errors.New("legacy mode requires enrollment and state")
	}
	return c, nil
}

func validateScreen(c screenConfig) error {
	if err := validatePartial(c); err != nil {
		return err
	}
	if c.size.Width <= 0 || c.size.Height <= 0 || c.size.Width > 2048 || c.size.Height > 2048 || c.size.Width*c.size.Height > 1_048_576 {
		return errors.New("screen dimensions exceed the bounded display profile")
	}
	if err := validateRefresh(c); err != nil {
		return err
	}
	if err := validateScreenTransport(c); err != nil {
		return err
	}
	_, err := renderbatch.New(c.policy)
	return err
}

func validateRefresh(c screenConfig) error {
	if !c.trusted || c.interval <= 0 || (!c.refreshPolicy && c.interval < 180*time.Second) || c.maintenance <= 0 {
		return errors.New("screen requires trusted-html and positive maintenance; full-interval <180s needs refresh-policy")
	}
	if c.refreshPolicy {
		_, err := screenwire.EncodeRefreshBegin([32]byte{}, refreshpolicy.Options{}, refreshpolicy.Policy{Normal: c.interval, Urgent: c.urgent})
		return err
	}
	return nil
}

func validateScreenTransport(c screenConfig) error {
	switch c.transport {
	case "usb":
		if c.usbWorker == "" || c.serial == "" {
			return errors.New("USB screen delivery requires usb-worker and serial")
		}
	case "wifi":
		if c.usbWorker != "" || c.serial != "" {
			return errors.New("Wi-Fi screen delivery forbids USB transport options")
		}
	default:
		return errors.New("screen-transport must be usb or wifi")
	}
	_, err := time.LoadLocation(c.timezone)
	return err
}
