package manager

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/document"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

func TestStoreIdempotency(t *testing.T) {
	store := NewStore()
	id := securetransport.DeviceID{1}
	now := time.Unix(100, 0)
	request := testRequest(t, update.ID{1}, "<p>one</p>", now)
	generation, created, err := store.Submit(id, request, now, time.Minute)
	if err != nil || generation != 1 || !created {
		t.Fatalf("submit generation=%d created=%v err=%v", generation, created, err)
	}
	if repeated, createdAgain, err := store.Submit(id, request, now, time.Minute); err != nil || repeated != 1 || createdAgain {
		t.Fatalf("repeat generation=%d created=%v err=%v", repeated, createdAgain, err)
	}
	conflict := testRequest(t, update.ID{1}, "<p>two</p>", now)
	if _, _, err = store.Submit(id, conflict, now, time.Minute); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict=%v", err)
	}
	_, _, err = store.Lease(id, now)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPersistentStoreRejectsPublicOrCorruptState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"devices":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// WriteFile applies the process umask; make this negative fixture public
	// explicitly so the security assertion also runs under release umask 077.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentStore(path); err == nil {
		t.Fatal("public state accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"unknown":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentStore(path); err == nil {
		t.Fatal("corrupt state accepted")
	}
}

func TestPersistentStoreRollsBackFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "state.json")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(300, 0)
	request := testRequest(t, update.ID{8}, "<p>rollback</p>", now)
	if _, _, err = store.Submit(securetransport.DeviceID{1}, request, now, time.Hour); err == nil {
		t.Fatal("unwritable persistence accepted")
	}
	if store.Status(securetransport.DeviceID{1}, now).Pending {
		t.Fatal("failed persistence mutated memory state")
	}
}

func TestStoreAcceptedWaitsForPhysicalCompletion(t *testing.T) {
	store := NewStore()
	id := securetransport.DeviceID{1}
	now := time.Unix(100, 0)
	request := testRequest(t, update.ID{1}, "<p>one</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Minute)
	pending, leasedGeneration, err := store.Lease(id, now)
	if err != nil || pending.Request.ID() != request.ID() || leasedGeneration != 1 {
		t.Fatalf("pending=%+v generation=%d err=%v", pending, leasedGeneration, err)
	}
	result := update.Result{ID: request.ID(), Status: update.StatusAccepted, ContentHash: request.ContentHash()}
	if err = store.Complete(id, 1, result, now); err != nil {
		t.Fatal(err)
	}
	if !store.Status(id, now).Pending {
		t.Fatal("accepted update was cleared before physical refresh")
	}
	if _, _, err = store.Lease(id, now); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("accepted update redelivered: %v", err)
	}
	if err = store.ReopenAccepted(id, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.Lease(id, now); err != nil {
		t.Fatalf("rebooted update not reopened: %v", err)
	}
}

func TestStoreTerminalCompletionIsIdempotent(t *testing.T) {
	store := NewStore()
	id, now := securetransport.DeviceID{1}, time.Unix(100, 0)
	request := testRequest(t, update.ID{1}, "<p>one</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Minute)
	result := update.Result{ID: request.ID(), Status: update.StatusRefreshed, ContentHash: request.ContentHash()}
	if err := store.Complete(id, 1, result, now); err != nil {
		t.Fatal(err)
	}
	if store.Status(id, now).Pending {
		t.Fatal("terminal update stayed pending")
	}
	if err := store.Complete(id, 1, result, now); err != nil {
		t.Fatalf("duplicate terminal report not idempotent: %v", err)
	}
}

func TestStoreExpiresPendingUpdate(t *testing.T) {
	store := NewStore()
	id := securetransport.DeviceID{1}
	now := time.Unix(100, 0)
	expiring := testRequest(t, update.ID{2}, "<p>three</p>", now)
	_, _, _ = store.Submit(id, expiring, now, time.Second)
	if _, _, err := store.Lease(id, now.Add(time.Second)); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("expired lease=%v", err)
	}
}

func TestPersistentStoreSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	id := securetransport.DeviceID{3}
	now := time.Unix(200, 0)
	request := testRequest(t, update.ID{4}, "<p>restart</p>", now)
	if _, _, err = store.Submit(id, request, now, time.Hour); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	pending, generation, err := reloaded.Lease(id, now)
	if err != nil || generation != 1 || pending.Request.ContentHash() != request.ContentHash() {
		t.Fatalf("generation=%d pending=%+v err=%v", generation, pending, err)
	}
}

func TestPersistentStoreRestoresCompletedResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, _ := NewPersistentStore(path)
	id, now := securetransport.DeviceID{5}, time.Unix(400, 0)
	request := testRequest(t, update.ID{6}, "<p>done</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Hour)
	result := update.Result{ID: request.ID(), Status: update.StatusRefreshed, ContentHash: request.ContentHash()}
	if err := store.Complete(id, 1, result, now); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	status := reloaded.Status(id, now)
	if status.Pending || status.LastResult != result {
		t.Fatalf("status=%+v", status)
	}
}

func TestPersistentStoreRetainsAcceptedStateAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, _ := NewPersistentStore(path)
	id, now := securetransport.DeviceID{7}, time.Unix(500, 0)
	request := testRequest(t, update.ID{8}, "<p>accepted</p>", now)
	_, _, _ = store.Submit(id, request, now, time.Hour)
	accepted := update.Result{ID: request.ID(), Status: update.StatusAccepted, ContentHash: request.ContentHash()}
	if err := store.Complete(id, 1, accepted, now); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewPersistentStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = reloaded.Lease(id, now); !errors.Is(err, ErrNoUpdate) {
		t.Fatalf("accepted state redelivered=%v", err)
	}
	if err = reloaded.ReopenAccepted(id, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err = reloaded.Lease(id, now); err != nil {
		t.Fatalf("reopened=%v", err)
	}
}

func TestDisconnectAndInvalidCompletion(t *testing.T) {
	store := NewStore()
	id, now := securetransport.DeviceID{1}, time.Unix(600, 0)
	store.Disconnect(id, now)
	status := store.Status(id, now)
	if status.Connected || status.LastSeen != now {
		t.Fatalf("status=%+v", status)
	}
	if err := store.Complete(id, 0, update.Result{}, now); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("invalid result=%v", err)
	}
	valid := update.Result{ID: update.ID{1}, Status: update.StatusRefreshed, ContentHash: update.Digest{1}}
	if err := store.Complete(id, 1, valid, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("missing pending=%v", err)
	}
}

func TestPersistentStoreRejectsInvalidAcceptedShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	data := `{"devices":[{"id":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"generation":1,"accepted":true}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentStore(path); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("invalid accepted state=%v", err)
	}
}

func TestPersistentStoreRejectsInvalidDeviceRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	records := []string{
		`{"devices":[{"id":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"generation":1}]}`,
		`{"devices":[{"id":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"generation":0}]}`,
		`{"devices":[{"id":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"generation":1,"pending":"AA=="}]}`,
		`{"devices":[{"id":[1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"generation":1,"result":"AA=="}]}`,
	}
	for _, data := range records {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewPersistentStore(path); !errors.Is(err, ErrConfiguration) {
			t.Fatalf("accepted record=%s err=%v", data, err)
		}
	}
}

func TestDiskStateRejectsInvalidInMemoryRecords(t *testing.T) {
	id := securetransport.DeviceID{1}
	store := NewStore()
	store.devices[id] = &deviceState{generation: 1, pending: &Pending{Request: update.Request{}, ExpiresAt: time.Now()}}
	if _, err := store.diskState(); err == nil {
		t.Fatal("invalid pending serialized")
	}
	store.devices[id] = &deviceState{generation: 1, result: update.Result{ID: update.ID{1}}}
	if _, err := store.diskState(); err == nil {
		t.Fatal("invalid result serialized")
	}
}

func TestReopenRollbackOnPersistenceFailure(t *testing.T) {
	id := securetransport.DeviceID{1}
	store := NewStore()
	store.path = filepath.Join(t.TempDir(), "missing", "state.json")
	request := testRequest(t, update.ID{2}, "<p>rollback</p>", time.Now())
	store.devices[id] = &deviceState{generation: 1,
		pending: &Pending{Request: request, Accepted: true, ExpiresAt: time.Now().Add(time.Hour)}}
	if err := store.ReopenAccepted(id, time.Now()); err == nil || !store.devices[id].pending.Accepted {
		t.Fatalf("error=%v accepted=%v", err, store.devices[id].pending.Accepted)
	}
}

func testRequest(t *testing.T, id update.ID, markup string, now time.Time) update.Request {
	t.Helper()
	source, err := document.NewSource(document.Version1, document.ProfileDashboard, []byte(markup))
	if err != nil {
		t.Fatal(err)
	}
	request, err := update.NewRequest(id, now.Unix(), "2026-09-02 12:00", "Europe/Kyiv", source)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
