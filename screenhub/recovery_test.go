package screenhub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestAuthenticatedSendCooldownAndQuietReconnect(t *testing.T) {
	f := deviceTest(t, 1)
	done := f.offer(t, 0)
	ready, err := f.hub.WaitReady(context.Background())
	if err != nil || ready.Pending != screendelivery.NoPending || !ready.NotBefore.Equal(f.now.Add(time.Second)) {
		t.Fatal(ready, err)
	}
	if err = f.hub.Send(context.Background(), f.frame); !errors.Is(err, ErrCooldown) {
		t.Fatal(err)
	}
	f.now = ready.NotBefore
	if err = f.hub.Send(context.Background(), f.frame); err != nil {
		t.Fatal(err)
	}
	floor := f.hub.floor
	f.hub.disconnect()
	awaitPeer(t, done)
	done = f.offer(t, 0)
	ready, err = f.hub.WaitReady(context.Background())
	if err != nil || ready.Pending != screendelivery.NoPending || !ready.NotBefore.Equal(floor) {
		t.Fatal(ready, err)
	}
	if f.sink.commits.Load() != 1 {
		t.Fatal("reconnect caused physical IO")
	}
	f.hub.disconnect()
	awaitPeer(t, done)
}

func TestLostReplyReconcilesWithoutRefreshRetry(t *testing.T) {
	for _, kind := range []screenwire.Kind{screenwire.Commit, screenwire.Data} {
		t.Run(kindName(kind), func(t *testing.T) {
			f := deviceTest(t, 1)
			done := f.offer(t, kind)
			ready, err := f.hub.WaitReady(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			f.now = ready.NotBefore
			if err = f.hub.Send(context.Background(), f.frame); !errors.Is(err, screendelivery.ErrTransportLost) {
				t.Fatal(err)
			}
			awaitPeer(t, done)
			done = f.offer(t, 0)
			ready, err = f.hub.WaitReady(context.Background())
			want, commits := screendelivery.PendingUnconfirmed, int32(0)
			if kind == screenwire.Commit {
				want, commits = screendelivery.PendingConfirmed, 1
			}
			if err != nil || ready.Pending != want || f.sink.commits.Load() != commits {
				t.Fatal(ready, err, f.sink.commits.Load())
			}
			f.hub.disconnect()
			awaitPeer(t, done)
		})
	}
}

func kindName(kind screenwire.Kind) string {
	if kind == screenwire.Commit {
		return "commit"
	}
	return "data"
}

func TestRebootRequiresExplicitBaselineInvalidation(t *testing.T) {
	f := deviceTest(t, 1)
	done := f.offer(t, 0)
	if _, err := f.hub.WaitReady(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.hub.disconnect()
	awaitPeer(t, done)
	other := deviceTest(t, 2)
	other.hub = f.hub
	done = other.offer(t, 0)
	if _, err := f.hub.WaitReady(context.Background()); !errors.Is(err, screendelivery.ErrTransportResync) {
		t.Fatal(err)
	}
	awaitPeer(t, done)
	f.hub.ResetSession()
	done = other.offer(t, 0)
	if ready, err := f.hub.WaitReady(context.Background()); err != nil || ready.Pending != screendelivery.NoPending {
		t.Fatal(ready, err)
	}
	f.hub.disconnect()
	awaitPeer(t, done)
}
