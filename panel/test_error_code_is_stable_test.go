package panel

import (
	"errors"
	"testing"
)

func TestErrorCodeIsStable(t *testing.T) {
	if got := ErrorCode(nil); got != "OK" {
		t.Fatalf("nil code = %q", got)
	}
	if got := ErrorCode(ErrFrameSize); got != "E_FRAME_SIZE" {
		t.Fatalf("frame code = %q", got)
	}
	if got := ErrorCode(OpError{Cause: ErrBusyTimeout}); got != "E_BUSY_TIMEOUT" {
		t.Fatalf("BUSY code = %q", got)
	}
	if got := ErrorCode(OpError{Cause: errors.New("spi")}); got != "E_SPI_WRITE" {
		t.Fatalf("SPI code = %q", got)
	}
}
