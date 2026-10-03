package screenwire

import "testing"

func TestRegionCapabilityRequiresTwoPasses(t *testing.T) {
	c := Capabilities{Width: 800, Height: 480, Stride: 100, MaxChunk: 1000, Passes: 2,
		Format: Mono1, Features: RawFull | FeatureRegion, Profile: 1, ProfileVersion: 1, MinimumFullMS: 1}
	var b [CapabilitiesSize]byte
	if err := EncodeCapabilities(b[:], c); err != nil {
		t.Fatal(err)
	}
	if got, err := DecodeCapabilities(b[:]); err != nil || got != c {
		t.Fatal(got, err)
	}
	c.Passes = 1
	if err := EncodeCapabilities(b[:], c); err == nil {
		t.Fatal("single-pass region")
	}
}
