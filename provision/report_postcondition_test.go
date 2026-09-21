package provision

import "testing"

type ignoredErase struct {
	*fakeFlash
	ignore bool
}

func (f *ignoredErase) EraseBlocks(start, length int64) error {
	if f.ignore {
		return nil
	}
	return f.fakeFlash.EraseBlocks(start, length)
}

func TestSuccessfulEraseMustActuallyReloadBlank(t *testing.T) {
	flash := &ignoredErase{fakeFlash: newFakeFlash()}
	store, _ := NewStore(flash, 0)
	want := validConfig()
	_, _ = store.Save(want)
	flash.ignore = true
	service, _ := NewService(store)
	report, config := service.Report(Request{Operation: OperationErase})
	if report.Code != CodeStorage || report.State != StateProvisioned || config != want {
		t.Fatal("erase reported success without erasing", report)
	}
}

func TestMutationSuccessHasOperationSpecificPostcondition(t *testing.T) {
	for _, response := range []Response{
		{Operation: OperationProvision},
		{Operation: OperationRotate, State: StateCorrupt},
		publicResponse(OperationErase, validConfig(), 1),
	} {
		var p [ResponseSize]byte
		if err := EncodeResponse(p[:], response); err == nil {
			t.Fatal("impossible success accepted", response)
		}
	}
}
