package screenusbhost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestUSBRegionRequiresNegotiatedSupport(t *testing.T) {
	s, _, sink, _ := usbFixture(t)
	if err := s.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); !errors.Is(err, refreshpolicy.ErrPolicy) {
		t.Fatal("partial without full policy", err)
	}
	if err := s.ConfigureRefresh(refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WaitReady(context.Background()); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	if sink.commits != 0 || s.bound {
		t.Fatal("unsupported partial peer admitted")
	}
}

func TestUSBRegionDoesNotInventPolicy(t *testing.T) {
	s, _, _, _ := usbFixture(t)
	if err := s.SendRegion(context.Background(), screendelivery.RegionPlan{}, refreshpolicy.Options{Mode: refreshpolicy.Partial}); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
}
