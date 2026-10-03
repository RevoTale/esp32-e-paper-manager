package manager

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

func TestStoreRejectsInvalidSubmitAndCompletionIdentity(t *testing.T) {
	store := NewStore()
	now := time.Now()
	if _, _, err := store.Submit(securetransport.DeviceID{}, update.Request{}, now, 0); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("invalid submit=%v", err)
	}
	id := securetransport.DeviceID{1}
	request := testRequest(t, update.ID{1}, "<p>x</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Hour)
	result := update.Result{ID: update.ID{2}, Status: update.StatusRefreshed, ContentHash: request.ContentHash()}
	if err := store.Complete(id, 2, result, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("mismatched completion=%v", err)
	}
}

func TestCompletionRollsBackPersistenceFailure(t *testing.T) {
	store := NewStore()
	id, now := securetransport.DeviceID{1}, time.Now()
	request := testRequest(t, update.ID{3}, "<p>rollback</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Hour)
	store.path = filepath.Join(t.TempDir(), "missing", "state.json")
	result := update.Result{ID: request.ID(), Status: update.StatusRefreshed, ContentHash: request.ContentHash()}
	if err := store.Complete(id, 1, result, now); err == nil {
		t.Fatal("persistence failure accepted")
	}
	if !store.Status(id, now).Pending {
		t.Fatal("failed completion was not rolled back")
	}
	if _, err := NewPersistentStore(""); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("empty path=%v", err)
	}
}
