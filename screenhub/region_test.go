package screenhub

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestWiFiRegionRequiresNegotiatedSupport(t *testing.T) {
	f := deviceTest(t, 1)
	if err := f.hub.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); !errors.Is(err, refreshpolicy.ErrPolicy) {
		t.Fatal("partial without full policy", err)
	}
	if err := f.hub.ConfigureRefresh(refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := f.hub.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	done := f.offer(t, 0)
	if _, err := f.hub.WaitReady(context.Background()); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	awaitPeer(t, done)
	if f.sink.commits.Load() != 0 || f.hub.current != nil {
		t.Fatal("unsupported partial peer admitted")
	}
}

func TestWiFiRegionDoesNotInventPolicy(t *testing.T) {
	f := deviceTest(t, 1)
	if err := f.hub.SendRegion(context.Background(), screendelivery.RegionPlan{}, refreshpolicy.Options{Mode: refreshpolicy.Partial}); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
}

func TestWiFiRegionValidationAndUnsupportedClientPreserveFullBaseline(t *testing.T) {
	for _, valid := range []bool{false, true} {
		f := deviceTest(t, 1)
		done := f.offer(t, 0)
		r, err := f.hub.WaitReady(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		f.now = r.NotBefore
		if err = f.hub.Send(context.Background(), f.frame); err != nil {
			t.Fatal(err)
		}
		base := f.hub.client.Baseline()
		// Configure after the legacy full send to exercise the dispatch's own
		// client-side guard independently of connect's negotiation rejection.
		f.hub.cadence.PartialPolicy = refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
		f.now = f.hub.floor
		plan := screendelivery.RegionPlan{}
		want := screendelivery.ErrRegionInput
		if valid {
			plan = screendelivery.RegionPlan{Bounds: image.Rect(0, 0, 8, 1), Old: []byte{0x80}, New: []byte{0}}
			want = screenclient.ErrUnsupportedRefresh
		}
		err = f.hub.SendRegion(context.Background(), plan, refreshpolicy.Options{Mode: refreshpolicy.Partial})
		if !errors.Is(err, want) || f.hub.client.Pending() || f.hub.client.Baseline() != base {
			t.Fatal(err)
		}
		awaitPeer(t, done)
		if f.sink.commits.Load() != 1 {
			t.Fatal("invalid request refreshed")
		}
	}
}
