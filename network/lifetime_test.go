package network

import (
	"errors"
	"testing"
)

func TestDurableLifetimeHasExactNonRepeatingIdentity(t *testing.T) {
	lifetime := lifetimeFixture(t)
	for i := uint64(1); i <= 3; i++ {
		epoch, session, err := lifetime.NextSession()
		if err != nil || epoch != 7 || session != i {
			t.Fatal(epoch, session, err)
		}
	}
}

func lifetimeFixture(t *testing.T) *Lifetime {
	t.Helper()
	lifetime, err := NewLifetime([8]byte{9}, 7)
	if err != nil {
		t.Fatal(err)
	}
	wantBoot := [16]byte{9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7}
	if lifetime.BootID() != wantBoot {
		t.Fatal(lifetime.BootID())
	}
	return lifetime
}

func TestLifetimeRejectsMissingIdentityAndExhaustion(t *testing.T) {
	for _, input := range []struct {
		uid   [8]byte
		epoch uint64
	}{{[8]byte{}, 1}, {[8]byte{1}, 0}} {
		if _, err := NewLifetime(input.uid, input.epoch); !errors.Is(err, ErrEntropy) {
			t.Fatal(err)
		}
	}
	lifetime, _ := NewLifetime([8]byte{1}, 1)
	lifetime.counter = ^uint64(0)
	if _, _, err := lifetime.NextSession(); !errors.Is(err, ErrEntropy) {
		t.Fatal(err)
	}
	var absent *Lifetime
	if _, _, err := absent.NextSession(); !errors.Is(err, ErrEntropy) {
		t.Fatal(err)
	}
}
