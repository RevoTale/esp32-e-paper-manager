package provision

import (
	"errors"
	"testing"
)

func TestServiceProvision(t *testing.T) {
	device := newFakeFlash()
	store, err := NewStore(device, 0)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	config := validConfig()
	created, err := service.Execute(Request{Operation: OperationProvision, Config: config})
	if err != nil || created.State != StateProvisioned || created.Generation != 1 {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	if created.SSID != config.SSID || created.Manager != config.Manager || created.Timezone != config.Timezone {
		t.Fatal("public fields missing")
	}
	if _, err = service.Execute(Request{Operation: OperationProvision, Config: config}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("duplicate provision: %v", err)
	}
}

func TestServiceBlankInspection(t *testing.T) {
	store, _ := NewStore(newFakeFlash(), 0)
	service, _ := NewService(store)
	blank, err := service.Execute(Request{Operation: OperationInspect})
	if err != nil || blank.State != StateBlank {
		t.Fatalf("blank=%+v err=%v", blank, err)
	}
}

func TestServiceReset(t *testing.T) {
	store, _ := NewStore(newFakeFlash(), 0)
	service, _ := NewService(store)
	_, _ = service.Execute(Request{Operation: OperationProvision, Config: validConfig()})
	reset, err := service.Execute(Request{Operation: OperationErase})
	if err != nil || reset.State != StateBlank {
		t.Fatalf("reset=%+v err=%v", reset, err)
	}
}

func TestServiceRotationPreservesIdentity(t *testing.T) {
	store, _ := NewStore(newFakeFlash(), 0)
	service, _ := NewService(store)
	config := validConfig()
	_, _ = service.Execute(Request{Operation: OperationProvision, Config: config})
	rotatedConfig := config
	rotatedConfig.DeviceKey[0] ^= 0xff
	rotated, err := service.Execute(Request{Operation: OperationRotate, Config: rotatedConfig})
	if err != nil || rotated.Generation != 2 {
		t.Fatalf("rotated=%+v err=%v", rotated, err)
	}
	wrongIdentity := rotatedConfig
	wrongIdentity.DeviceID[0] ^= 1
	if _, err = service.Execute(Request{Operation: OperationRotate, Config: wrongIdentity}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("identity replacement accepted: %v", err)
	}
}

func TestInspectAndDiagnoseProvisionedDevice(t *testing.T) {
	store, _ := NewStore(newFakeFlash(), 0)
	service, _ := NewService(store)
	config := validConfig()
	_, _ = service.Execute(Request{Operation: OperationProvision, Config: config})
	for _, operation := range []Operation{OperationInspect, OperationDiagnose} {
		response, err := service.Execute(Request{Operation: operation})
		if err != nil || response.State != StateProvisioned || response.DeviceID != config.DeviceID {
			t.Fatalf("operation=%d response=%+v err=%v", operation, response, err)
		}
	}
}

func TestServicePropagatesStorageFailures(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	service, _ := NewService(store)
	flash.eraseError = errors.New("erase")
	if _, err := service.Execute(Request{Operation: OperationProvision, Config: validConfig()}); err == nil {
		t.Fatal("provision storage failure accepted")
	}
	if _, err := service.Execute(Request{Operation: OperationErase}); err == nil {
		t.Fatal("erase storage failure accepted")
	}
}

func TestServiceRejectsInvalidTransitionsAndReportsCorruption(t *testing.T) {
	if _, err := NewService(nil); !errors.Is(err, ErrStorage) {
		t.Fatalf("nil=%v", err)
	}
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	service, _ := NewService(store)
	if _, err := service.Execute(Request{Operation: 99}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("operation=%v", err)
	}
	flash.data[0] = 0
	response, err := service.Execute(Request{Operation: OperationDiagnose})
	if err != nil || response.State != StateCorrupt {
		t.Fatalf("response=%+v err=%v", response, err)
	}
	if _, err = service.Execute(Request{Operation: OperationRotate, Config: validConfig()}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("rotate blank=%v", err)
	}
}
