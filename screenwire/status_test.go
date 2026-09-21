package screenwire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func statusFixture() Status {
	return Status{Operation: Commit, Code: CodeHardware, State: streamrx.Failed, Pass: 1, Offset: 1234,
		Diagnostic: Diagnostic{Domain: DomainPanel, Code: 2, Phase: 5, Step: 14, Command: 0x12, Offset: -1, BusyKnown: true},
		CooldownMS: 30000, Generation: 7, Boot: [16]byte{3}}
}

func TestStatusOffsetsAndRoundtrip(t *testing.T) {
	want := statusFixture()
	b := make([]byte, 48)
	if err := EncodeStatus(b, want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b[:16], []byte{6, 11, 4, 1, 0xd2, 4, 0, 0, 0, 1, 2, 5, 14, 0x12, 1, 0}) {
		t.Fatal(b)
	}
	if binary.LittleEndian.Uint32(b[4:8]) != 1234 || binary.LittleEndian.Uint32(b[16:20]) != 0xffffffff || binary.LittleEndian.Uint64(b[24:32]) != 7 || b[32] != 3 {
		t.Fatal(b)
	}
	got, err := DecodeStatus(b)
	if err != nil || got != want {
		t.Fatal(got, err)
	}
}

func TestStatusRejectsUnknownOrInconsistentFields(t *testing.T) {
	b := make([]byte, StatusSize)
	if err := EncodeStatus(b, statusFixture()); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct {
		offset int
		value  byte
	}{
		{0, 0}, {0, byte(Reply)}, {1, 255}, {2, 255}, {3, 3}, {8, 2}, {9, 255}, {14, 2}, {14, 4}, {15, 1}, {8, 1},
	} {
		bad := append([]byte(nil), b...)
		bad[change.offset] = change.value
		if _, err := DecodeStatus(bad); !errors.Is(err, ErrRecord) {
			t.Fatal(change, err)
		}
	}
	for n := 0; n < StatusSize; n++ {
		if _, err := DecodeStatus(b[:n]); !errors.Is(err, ErrRecord) {
			t.Fatal(n, err)
		}
	}
	if err := EncodeStatus(b[:47], statusFixture()); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
	bad := statusFixture()
	bad.Diagnostic.Busy = true
	bad.Diagnostic.BusyKnown = false
	if err := EncodeStatus(b, bad); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func capsFixture() Capabilities {
	return Capabilities{Width: 17, Height: 9, Stride: 3, MaxChunk: 1000, Passes: 2, Format: Mono1,
		Features: RawFull, Profile: 1, ProfileVersion: 1, MinimumFullMS: 30000, DeviceID: [16]byte{4}}
}

func TestCapabilitiesExactOffsetsAndPadding(t *testing.T) {
	want := capsFixture()
	b := make([]byte, 40)
	if err := EncodeCapabilities(b, want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b[:10], []byte{17, 0, 9, 0, 3, 0, 0xe8, 3, 2, 1}) || b[24] != 4 {
		t.Fatal(b)
	}
	got, err := DecodeCapabilities(b)
	if err != nil || got != want {
		t.Fatal(got, err)
	}
	for _, offset := range []int{18, 19} {
		b[offset] = 1
		if _, err = DecodeCapabilities(b); !errors.Is(err, ErrRecord) {
			t.Fatal(offset, err)
		}
		b[offset] = 0
	}
}

func TestInvalidCapabilitiesReject(t *testing.T) {
	for _, mutate := range []func(*Capabilities){
		func(c *Capabilities) { c.Width = 0 }, func(c *Capabilities) { c.Height = 0 },
		func(c *Capabilities) { c.Stride = 2 }, func(c *Capabilities) { c.MaxChunk = 0 }, func(c *Capabilities) { c.MaxChunk = 1025 },
		func(c *Capabilities) { c.Passes = 0 }, func(c *Capabilities) { c.Passes = 3 },
		func(c *Capabilities) { c.Format = 2 }, func(c *Capabilities) { c.Features = 0 }, func(c *Capabilities) { c.Features |= 16 },
		func(c *Capabilities) { c.Profile = 0 }, func(c *Capabilities) { c.ProfileVersion = 0 }, func(c *Capabilities) { c.MinimumFullMS = 0 },
	} {
		c := capsFixture()
		mutate(&c)
		if err := EncodeCapabilities(make([]byte, 40), c); !errors.Is(err, ErrRecord) {
			t.Fatal(c, err)
		}
	}
	for n := 0; n < 40; n++ {
		if _, err := DecodeCapabilities(make([]byte, n)); !errors.Is(err, ErrRecord) {
			t.Fatal(n, err)
		}
	}
	if err := EncodeCapabilities(make([]byte, 39), capsFixture()); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}
