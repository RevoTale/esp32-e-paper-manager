package interop

import (
	"bytes"
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenhub"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

func TestEncryptedHubRegionToNativePanel(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "confirmed", true: "lost_commit"}[lost], func(t *testing.T) {
			testHubRegion(t, lost)
		})
	}
}

func testHubRegion(t *testing.T, lost bool) {
	t.Helper()
	peer := startNativePeerNamed(t, "EP_RECEIVER_CLI", "--region-fast")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	hub, config, lifetime := newRegionHub(t)
	bridge := offerNativeRegion(t, hub, config, lifetime, peer, lost)
	ready, err := hub.WaitReady(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waitRegionDeadline(t, ready.NotBefore)
	if err = hub.Send(ctx, testFrame(t, 0xa5)); err != nil {
		t.Fatal(err)
	}
	ready, err = hub.WaitReady(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waitRegionDeadline(t, ready.PartialNotBefore)
	plan := screendelivery.RegionPlan{Bounds: image.Rect(240, 254, 256, 256),
		Old: bytes.Repeat([]byte{0xa5}, 4), New: bytes.Repeat([]byte{0x5a}, 4)}
	err = hub.SendRegion(ctx, plan, refreshpolicy.Options{Mode: refreshpolicy.Partial})
	if lost {
		if !errors.Is(err, screendelivery.ErrTransportLost) {
			t.Fatal(err)
		}
		bridge.wait(t)
		bridge = offerNativeRegion(t, hub, config, lifetime, peer, false)
		ready, err = hub.WaitReady(ctx)
		checkHubReconciliation(t, ready, err)
	} else if err != nil {
		t.Fatal(err)
	}
	if err = hub.Close(); err != nil {
		t.Fatal(err)
	}
	bridge.wait(t)
	// Native assertions check exact pixels, window and two physical refreshes.
	peer.finish()
}

func checkHubReconciliation(t *testing.T, ready screendelivery.Readiness, err error) {
	t.Helper()
	if err != nil || ready.Pending != screendelivery.PendingConfirmed {
		t.Fatal(ready, err)
	}
	if ready.PartialNotBefore.Before(ready.NotBefore) {
		t.Fatal("uncertainty shortened guard")
	}
}

func newRegionHub(t *testing.T) (*screenhub.Hub, screenhub.Config, *network.Lifetime) {
	t.Helper()
	config := screenhub.Config{ID: securetransport.DeviceID{2}, Key: securetransport.Key{3},
		Size: display.Size{Width: 800, Height: 480}, Profile: 1, ProfileVersion: 1}
	hub, err := screenhub.New(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = hub.Close() })
	if err = hub.ConfigureRefresh(refreshpolicy.Policy{Normal: 2 * time.Millisecond, Urgent: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if err = hub.ConfigurePartial(refreshpolicy.Policy{Normal: time.Millisecond, Urgent: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	lifetime, err := network.NewLifetime([8]byte{1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return hub, config, lifetime
}

func waitRegionDeadline(t *testing.T, deadline time.Time) {
	t.Helper()
	if delay := time.Until(deadline); delay > time.Second {
		t.Fatal("unexpected fixture cadence", delay)
	}
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	<-ctx.Done()
}
