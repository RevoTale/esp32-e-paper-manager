package screenusbhost

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestUSBRegionDeliveryAndLostCommit(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "confirmed", true: "lost_ack"}[lost], func(t *testing.T) {
			s, peer, now := regionSender(t)
			r, err := s.WaitReady(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			*now = r.NotBefore
			frame, err := display.NewFrame(display.Size{Width: 16, Height: 4}, 2, make([]byte, 8))
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Send(context.Background(), frame); err != nil {
				t.Fatal(err)
			}
			plan := screendelivery.RegionPlan{Bounds: image.Rect(0, 0, 16, 2), Old: make([]byte, 4), New: []byte{1, 2, 3, 4}}
			o := refreshpolicy.Options{Mode: refreshpolicy.Partial}
			if err = s.SendRegion(context.Background(), plan, o); !errors.Is(err, ErrCooldown) {
				t.Fatal(err)
			}
			*now = now.Add(2 * time.Second)
			peer.dropCommit = lost
			err = s.SendRegion(context.Background(), plan, o)
			checkRegionOutcome(t, s, lost, err)
			if peer.commits != 2 || peer.region.Policy.Normal != 2*time.Second || peer.region.Policy.Urgent != time.Second {
				t.Fatal(peer.commits, peer.region)
			}
		})
	}
}

func checkRegionOutcome(t *testing.T, s *Sender, lost bool, err error) {
	t.Helper()
	if !lost {
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if !errors.Is(err, screendelivery.ErrTransportLost) {
		t.Fatal(err)
	}
	r, err := s.WaitReady(context.Background())
	if err != nil || r.Pending != screendelivery.PendingConfirmed {
		t.Fatal(r, err)
	}
	if r.PartialNotBefore.Before(r.NotBefore) {
		t.Fatal("reconcile bypass", r)
	}
}

func regionSender(t *testing.T) (*Sender, *regionPeerState, *time.Time) {
	t.Helper()
	s, err := New("worker", "port", display.Size{Width: 16, Height: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	state := &regionPeerState{}
	s.start = func() (worker, error) { return &regionWorker{state: state, done: make(chan struct{})}, nil }
	now := time.Unix(100, 0)
	s.now = func() time.Time { return now }
	if err = s.ConfigureRefresh(refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if err = s.ConfigurePartial(refreshpolicy.Policy{Normal: 2 * time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	return s, state, &now
}

func TestUSBRegionInvalidPlanCannotRefresh(t *testing.T) {
	s, peer, now := regionSender(t)
	r, err := s.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*now = r.NotBefore
	if err = s.SendRegion(context.Background(), screendelivery.RegionPlan{}, refreshpolicy.Options{Mode: refreshpolicy.Partial}); !errors.Is(err, screendelivery.ErrRegionInput) {
		t.Fatal(err)
	}
	if peer.commits != 0 || s.client.Pending() {
		t.Fatal("invalid plan reached commit")
	}
}
