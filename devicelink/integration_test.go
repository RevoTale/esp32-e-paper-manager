package devicelink

import (
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/document"
	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

func TestEncryptedUpdateRoundTripAndCompletion(t *testing.T) {
	now := time.Unix(100, 0)
	record := manager.DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := manager.NewStaticRegistry([]manager.DeviceRecord{record})
	store := manager.NewStore()
	request := linkRequest(t, now)
	if _, _, err := store.Submit(record.ID, request, now, time.Hour); err != nil {
		t.Fatal(err)
	}
	device, deviceSide, managerDone := startLink(t, record, registry, store, now, 3, 4)
	mustReport(t, device, nil, 0)
	received := mustReceive(t, device, true)
	if received.ContentHash() != request.ContentHash() {
		t.Fatalf("request=%+v", received)
	}
	result := update.Result{ID: received.ID(), Status: update.StatusAccepted, ContentHash: received.ContentHash()}
	mustSendResult(t, device, result)
	mustManagerDone(t, managerDone)
	if !store.Status(record.ID, now).Pending {
		t.Fatal("manager cleared update before physical refresh")
	}
	_ = deviceSide.Close()
	device, _, managerDone = startLink(t, record, registry, store, now, 5, 6)
	result.Status = update.StatusRefreshed
	mustReport(t, device, &result, 1)
	_ = mustReceive(t, device, false)
	mustManagerDone(t, managerDone)
	if store.Status(record.ID, now).Pending {
		t.Fatal("manager retained physically refreshed update")
	}
}

func mustReport(t *testing.T, session *DeviceSession, result *update.Result, generation uint64) {
	t.Helper()
	if err := session.Report(result, generation); err != nil {
		t.Fatal(err)
	}
}

func mustReceive(t *testing.T, session *DeviceSession, expected bool) update.Request {
	t.Helper()
	request, available, err := session.Receive()
	if err != nil || available != expected {
		t.Fatalf("available=%v expected=%v err=%v", available, expected, err)
	}
	return request
}

func mustSendResult(t *testing.T, session *DeviceSession, result update.Result) {
	t.Helper()
	if err := session.SendResult(result); err != nil {
		t.Fatal(err)
	}
}

func mustManagerDone(t *testing.T, done <-chan error) {
	t.Helper()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func startLink(t *testing.T, record manager.DeviceRecord, registry manager.Registry, store *manager.Store,
	now time.Time, serverByte, clientByte byte,
) (*DeviceSession, net.Conn, <-chan error) {
	t.Helper()
	managerSide, deviceSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeManager(managerSide, ManagerConfig{Registry: registry, Store: store,
			Random: bytes.NewReader(bytes.Repeat([]byte{serverByte}, 16)), Now: func() time.Time { return now }})
	}()
	device, err := ConnectDevice(deviceSide, DeviceConfig{ID: record.ID, Key: record.Key},
		bytes.NewReader(bytes.Repeat([]byte{clientByte}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return device, deviceSide, done
}

func TestIdleSessionAndWrongKeyFailClosed(t *testing.T) {
	now := time.Unix(100, 0)
	record := manager.DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := manager.NewStaticRegistry([]manager.DeviceRecord{record})
	managerSide, deviceSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeManager(managerSide, ManagerConfig{Registry: registry, Store: manager.NewStore(),
			Random: bytes.NewReader(bytes.Repeat([]byte{3}, 16)), Now: func() time.Time { return now }})
	}()
	device, err := ConnectDevice(deviceSide, DeviceConfig{ID: record.ID, Key: record.Key},
		bytes.NewReader(bytes.Repeat([]byte{4}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if err = device.Report(nil, 0); err != nil {
		t.Fatal(err)
	}
	_, available, err := device.Receive()
	if err != nil || available {
		t.Fatalf("idle available=%v err=%v", available, err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}

	badManager, badDevice := net.Pipe()
	go func() {
		done <- ServeManager(badManager, ManagerConfig{Registry: registry, Store: manager.NewStore(),
			Random: bytes.NewReader(bytes.Repeat([]byte{3}, 16)), Now: func() time.Time { return now }})
	}()
	wrong := record.Key
	wrong[0] ^= 1
	if _, err = ConnectDevice(badDevice, DeviceConfig{ID: record.ID, Key: wrong},
		bytes.NewReader(bytes.Repeat([]byte{4}, 32))); err == nil {
		t.Fatal("wrong key authenticated")
	}
	_ = badDevice.Close()
	if err = <-done; err == nil {
		t.Fatal("manager accepted wrong key")
	}
}

func TestUnknownDeviceFailsBeforeAuthentication(t *testing.T) {
	registry, _ := manager.NewStaticRegistry(nil)
	managerSide, deviceSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeManager(managerSide, ManagerConfig{Registry: registry, Store: manager.NewStore(),
			Random: bytes.NewReader(bytes.Repeat([]byte{3}, 16)), Now: time.Now})
	}()
	id := securetransport.DeviceID{9}
	_, _ = deviceSide.Write(id[:])
	_ = deviceSide.Close()
	if err := <-done; !errors.Is(err, manager.ErrUnknownDevice) {
		t.Fatalf("unknown=%v", err)
	}
}

type linkRuntime struct {
	report     update.Result
	generation uint64
	available  bool
	applied    update.Request
}

func (runtime *linkRuntime) ApplyNetwork(request update.Request) update.Result {
	runtime.applied = request
	return update.Result{ID: request.ID(), Status: update.StatusAccepted, ContentHash: request.ContentHash()}
}

func (runtime *linkRuntime) RecordNetwork(generation uint64, result update.Result) {
	runtime.generation, runtime.report, runtime.available = generation, result, true
}

func (runtime *linkRuntime) NetworkReport() (update.Result, uint64, bool) {
	return runtime.report, runtime.generation, runtime.available
}

func TestExchangeOwnsRetainedReportAndDelivery(t *testing.T) {
	now := time.Unix(100, 0)
	record := manager.DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := manager.NewStaticRegistry([]manager.DeviceRecord{record})
	store := manager.NewStore()
	request := linkRequest(t, now)
	_, _, _ = store.Submit(record.ID, request, now, time.Hour)
	managerSide, deviceSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeManager(managerSide, ManagerConfig{Registry: registry, Store: store,
			Random: bytes.NewReader(bytes.Repeat([]byte{7}, 16)), Now: func() time.Time { return now }})
	}()
	runtime := &linkRuntime{}
	err := Exchange(deviceSide, DeviceConfig{ID: record.ID, Key: record.Key},
		bytes.NewReader(bytes.Repeat([]byte{8}, 32)), runtime)
	if err != nil || runtime.applied.ID() != request.ID() || runtime.generation != 1 {
		t.Fatalf("applied=%+v generation=%d err=%v", runtime.applied, runtime.generation, err)
	}
	mustManagerDone(t, done)
}

func TestExchangeRejectsMissingRuntime(t *testing.T) {
	if err := Exchange(nil, DeviceConfig{}, nil, nil); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("error=%v", err)
	}
}

func TestExchangeReportsRetainedTerminalResult(t *testing.T) {
	now := time.Unix(200, 0)
	record := manager.DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := manager.NewStaticRegistry([]manager.DeviceRecord{record})
	store := manager.NewStore()
	request := linkRequest(t, now)
	_, _, _ = store.Submit(record.ID, request, now, time.Hour)
	result := update.Result{ID: request.ID(), Status: update.StatusRefreshed, ContentHash: request.ContentHash()}
	_ = store.Complete(record.ID, 1, result, now)
	managerSide, deviceSide := net.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeManager(managerSide, ManagerConfig{Registry: registry, Store: store,
			Random: bytes.NewReader(bytes.Repeat([]byte{9}, 16)), Now: func() time.Time { return now }})
	}()
	runtime := &linkRuntime{report: result, generation: 1, available: true}
	if err := Exchange(deviceSide, DeviceConfig{ID: record.ID, Key: record.Key},
		bytes.NewReader(bytes.Repeat([]byte{10}, 32)), runtime); err != nil {
		t.Fatal(err)
	}
	mustManagerDone(t, done)
}

func linkRequest(t *testing.T, now time.Time) update.Request {
	t.Helper()
	source, err := document.NewSource(document.Version1, document.ProfileDashboard, []byte("<p>encrypted</p>"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := update.NewRequest(update.ID{1}, now.Unix(), "2026-09-02 12:00", "Europe/Kyiv", source)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
