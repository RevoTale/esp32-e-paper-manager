package panel

import (
	"errors"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

type adapterRefresher struct {
	bytes int
	err   error
}

func (r *adapterRefresher) Refresh(frame []byte) error { r.bytes = len(frame); return r.err }

func TestAdapterContractAndDelegation(t *testing.T) {
	refresher := &adapterRefresher{}
	adapter := &Adapter{driver: refresher}
	capabilities := adapter.Capabilities()
	frame, err := display.NewFrame(capabilities.Size, FrameWidth/8, make([]byte, FrameBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err = adapter.Refresh(frame, display.RefreshFull); err != nil || refresher.bytes != FrameBytes {
		t.Fatalf("Refresh() bytes=%d error=%v", refresher.bytes, err)
	}
	refresher.err = ErrBusyTimeout
	if err = adapter.Refresh(frame, display.RefreshFull); !errors.Is(err, ErrBusyTimeout) {
		t.Fatalf("Refresh(driver error) = %v", err)
	}
	if err = adapter.Refresh(frame, display.RefreshPartial); !errors.Is(err, display.ErrRefreshMode) {
		t.Fatalf("Refresh(mode) = %v", err)
	}
	if !capabilities.RefreshAutoSleeps || adapter.Sleep() != nil {
		t.Fatal("adapter lifecycle contract is incorrect")
	}
}

func TestNewAdapterRejectsNil(t *testing.T) {
	if _, err := NewAdapter(nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewAdapter(nil) error=%v", err)
	}
}

func TestNewAdapterAcceptsDriver(t *testing.T) {
	driver, err := New(IO{Write: func([]byte) error { return nil }, SetCS: func(bool) {}, SetDC: func(bool) {},
		SetReset: func(bool) {}, SetPower: func(bool) {}, ReadBusy: func() bool { return false },
		Delay: func(time.Duration) {}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewAdapter(driver); err != nil {
		t.Fatal(err)
	}
}
