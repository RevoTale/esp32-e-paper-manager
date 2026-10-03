package manager

import (
	"context"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestNativeWarningsAreExposedAndIndependentlyOwned(t *testing.T) {
	_, screen, _ := nativeScreen(t, display.Size{Width: 100, Height: 40})
	_, err := screen.Submit(0, []byte(`<div style="width:30px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">long text</div>`), 0)
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := screen.RenderNext(context.Background(), time.Second, true)
	if err != nil {
		t.Fatal(err)
	}
	status := screen.Status()
	if len(status.Warnings) == 0 || status.Warnings[0].Code != renderdiag.Clipped {
		t.Fatalf("warnings %+v", status.Warnings)
	}
	status.Warnings[0].Code = "mutated"
	if screen.Status().Warnings[0].Code != renderdiag.Clipped {
		t.Fatal("caller changed retained warning")
	}
	if err = screen.Resolve(delivery.Revision, true); err != nil {
		t.Fatal(err)
	}
	if _, err = screen.Submit(1, []byte("new"), 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if len(screen.Status().Warnings) != 0 {
		t.Fatal("new revision kept stale warnings")
	}
}
