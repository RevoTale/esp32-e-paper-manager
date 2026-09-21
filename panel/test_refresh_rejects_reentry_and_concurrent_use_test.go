package panel

import (
	"errors"
	"testing"
)

func TestRefreshRejectsReentryAndConcurrentUse(t *testing.T) {
	frame := make([]byte, FrameBytes)
	t.Run("callback reentry", func(t *testing.T) {
		io := noOpIO()
		var driver *Driver
		var nested error
		called := false
		io.Write = func([]byte) error {
			if !called {
				called = true
				nested = driver.Refresh(frame)
			}
			return nil
		}
		var err error
		driver, err = New(io)
		if err != nil {
			t.Fatal(err)
		}
		if err := driver.Refresh(frame); err != nil {
			t.Fatal(err)
		}
		if !errors.Is(nested, ErrInUse) {
			t.Fatalf("nested error = %v", nested)
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		entered := make(chan struct{})
		release := make(chan struct{})
		io := noOpIO()
		firstWrite := true
		io.Write = func([]byte) error {
			if firstWrite {
				firstWrite = false
				close(entered)
				<-release
			}
			return nil
		}
		driver, err := New(io)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- driver.Refresh(frame) }()
		<-entered
		if err := driver.Refresh(frame); !errors.Is(err, ErrInUse) {
			t.Fatalf("concurrent error = %v", err)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}
