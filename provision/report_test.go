package provision

import (
	"errors"
	"testing"
)

func TestReportUnknownStorageNeverInventsBlankIdentity(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	service, _ := NewService(store)
	flash.readError = errors.New("private flash detail")
	report, config := service.Report(Request{Operation: OperationInspect})
	if report.Code != CodeStorage || report.State != StateUnknown || config != (Config{}) {
		t.Fatal(report, config)
	}
	var raw [ResponseSize]byte
	if err := EncodeResponse(raw[:], report); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeResponse(raw[:])
	if err != nil || decoded != report || raw[12] != 3 || raw[6] != 3 || raw[4] != 2 {
		t.Fatal(decoded, err)
	}
}

func TestReportReturnsAuthoritativeStateAfterFailedOperation(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	service, _ := NewService(store)
	want := validConfig()
	_, _ = store.Save(want)
	flash.eraseError = errors.New("erase failed")
	report, config := service.Report(Request{Operation: OperationErase})
	if report.Code != CodeStorage || report.State != StateProvisioned || report.Generation != 1 || config != want {
		t.Fatal(report, config)
	}
	report, _ = service.Report(Request{Operation: OperationProvision, Config: want})
	if report.Code != CodeState || report.State != StateProvisioned {
		t.Fatal(report)
	}
}
