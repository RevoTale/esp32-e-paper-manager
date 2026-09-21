package network

import (
	"errors"
	"testing"
	"time"
)

func TestRetryUntilConnected(t *testing.T) {
	attempts := 0
	var waits []time.Duration
	err := RetryUntilConnected(func() error {
		attempts++
		if attempts < 4 {
			return errors.New("offline")
		}
		return nil
	}, func(wait time.Duration) { waits = append(waits, wait) })
	if err != nil || attempts != 4 {
		t.Fatalf("result err=%v attempts=%d", err, attempts)
	}
	want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	for index := range want {
		if waits[index] != want[index] {
			t.Fatalf("wait %d: got %v want %v", index, waits[index], want[index])
		}
	}
}

func TestRetryUntilConnectedRejectsMissingHooks(t *testing.T) {
	if err := RetryUntilConnected(nil, time.Sleep); !errors.Is(err, ErrPolling) {
		t.Fatalf("nil attempt: %v", err)
	}
	if err := RetryUntilConnected(func() error { return nil }, nil); !errors.Is(err, ErrPolling) {
		t.Fatalf("nil sleep: %v", err)
	}
}

func TestEnsureConnectedRestoresLinkBeforeConfiguration(t *testing.T) {
	var calls []string
	err := EnsureConnected(
		func() bool { calls = append(calls, "status"); return false },
		func() error { calls = append(calls, "join"); return nil },
		func() error { calls = append(calls, "configure"); return nil },
		func(time.Duration) {},
	)
	if err != nil || len(calls) != 3 || calls[0] != "status" || calls[1] != "join" || calls[2] != "configure" {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

func TestEnsureConnectedLeavesHealthyLinkUntouched(t *testing.T) {
	called := false
	err := EnsureConnected(func() bool { return true }, func() error { called = true; return nil },
		func() error { called = true; return nil }, func(time.Duration) { called = true })
	if err != nil || called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}

func TestEnsureConnectedReturnsConfigurationAndHookErrors(t *testing.T) {
	want := errors.New("dhcp")
	err := EnsureConnected(func() bool { return false }, func() error { return nil },
		func() error { return want }, func(time.Duration) {})
	if !errors.Is(err, want) {
		t.Fatalf("configure=%v", err)
	}
	if err = EnsureConnected(nil, func() error { return nil }, func() error { return nil }, time.Sleep); !errors.Is(err, ErrPolling) {
		t.Fatalf("nil status=%v", err)
	}
	if err = EnsureConnected(func() bool { return false }, nil, func() error { return nil }, time.Sleep); !errors.Is(err, ErrPolling) {
		t.Fatalf("nil join=%v", err)
	}
}
