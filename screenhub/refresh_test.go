package screenhub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
)

func TestWiFiPolicyRejectsLegacyPeer(t *testing.T) {
	f := deviceTest(t, 1)
	if err := f.hub.ConfigureRefresh(refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	done := f.offer(t, 0)
	if _, err := f.hub.WaitReady(context.Background()); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	awaitPeer(t, done)
	if f.sink.commits.Load() != 0 || f.hub.current != nil {
		t.Fatal("unsupported peer enabled")
	}
}

func TestWiFiDoesNotSilentlyIgnoreOptions(t *testing.T) {
	f := deviceTest(t, 1)
	done := f.offer(t, 0)
	r, err := f.hub.WaitReady(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.now = r.NotBefore
	if err := f.hub.SendWithOptions(context.Background(), f.frame, refreshpolicy.Options{Priority: refreshpolicy.Urgent}); !errors.Is(err, screenclient.ErrUnsupportedRefresh) {
		t.Fatal(err)
	}
	f.hub.disconnect()
	awaitPeer(t, done)
}
