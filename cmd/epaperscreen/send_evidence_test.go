package main

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
)

func TestUnconfirmedSendRetainsOriginalFailure(t *testing.T) {
	for _, reconcileErr := range []error{nil, context.DeadlineExceeded} {
		sends := 0
		s := &fakeTransport{
			send: func(display.Frame) error {
				sends++
				return errors.Join(screendelivery.ErrTransportLost, io.ErrUnexpectedEOF)
			},
			ready: func() (screendelivery.Readiness, error) {
				return screendelivery.Readiness{Pending: screendelivery.PendingUnconfirmed}, reconcileErr
			},
		}
		err := sendOnce(context.Background(), s, display.Frame{})
		if !errors.Is(err, ErrUnconfirmed) || !errors.Is(err, io.ErrUnexpectedEOF) || sends != 1 {
			t.Fatalf("original failure lost or replayed: %v, sends=%d", err, sends)
		}
		if reconcileErr != nil && !errors.Is(err, reconcileErr) {
			t.Fatal("reconciliation failure lost:", err)
		}
	}
}
