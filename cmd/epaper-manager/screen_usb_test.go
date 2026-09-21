package main

import (
	"context"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
)

func TestUSBCompositionUsesRetainedEPS2ClientWithoutOpeningPort(t *testing.T) {
	c, err := parseConfig(screenArgs())
	if err != nil {
		t.Fatal(err)
	}
	transport, zone, err := screenDelivery(c)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := transport.close(); err != nil {
			t.Error(err)
		}
	}()
	if _, ok := transport.sender.(*screenusbhost.Sender); !ok {
		t.Fatal("manager still uses legacy EPS1 sender")
	}
	if zone.String() != "Europe/Kiev" {
		t.Fatal(zone)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := transport.run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestUSBCompositionRejectsInvalidLocalConfigurationBeforeWorker(t *testing.T) {
	c, err := parseConfig(screenArgs())
	if err != nil {
		t.Fatal(err)
	}
	c.screen.timezone = "unknown/zone"
	if _, _, err := screenDelivery(c); err == nil {
		t.Fatal("invalid timezone")
	}
	c.screen.timezone = "UTC"
	c.screen.usbWorker = ""
	if _, _, err := screenDelivery(c); err == nil {
		t.Fatal("invalid worker")
	}
}
