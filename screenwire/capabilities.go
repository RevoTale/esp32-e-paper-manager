package screenwire

import "encoding/binary"

const Mono1 uint8 = 1
const RawFull uint16 = 3                   // bit 0 raw mono1 encoding; bit 1 full refresh
const FeaturePackBits uint16 = 1 << 2      // Optional bounded PackBits DataPacked.
const FeatureRefreshPolicy uint16 = 1 << 3 // BeginRefresh with explicit cadence.
const FeatureRegion uint16 = 1 << 4        // BeginRegion with old/new pixel passes.

// Capabilities are physical adapter properties, not remotely editable settings.
// New dimensions are valid only with a matching local panel implementation.
type Capabilities struct {
	Width, Height, Stride, MaxChunk uint16
	Passes, Format                  uint8
	Features                        uint16
	Profile                         uint32
	ProfileVersion                  uint16
	MinimumFullMS                   uint32
	DeviceID                        [16]byte
}

func (c Capabilities) valid() bool {
	return c.validGeometry() && c.validProfile()
}

func (c Capabilities) validGeometry() bool {
	return c.Width != 0 && c.Height != 0 && uint32(c.Stride) == (uint32(c.Width)+7)/8 && c.MaxChunk >= 1 && c.MaxChunk <= MaxPayload
}

func (c Capabilities) validProfile() bool {
	return c.Passes >= 1 && c.Passes <= 2 && c.Format == Mono1 && c.validFeatures() && c.Profile != 0 && c.ProfileVersion != 0 && c.MinimumFullMS != 0
}

func (c Capabilities) validFeatures() bool {
	return c.Features&RawFull == RawFull && c.Features&^(RawFull|FeaturePackBits|FeatureRefreshPolicy|FeatureRegion) == 0 &&
		(c.Features&FeatureRegion == 0 || c.Passes == 2)
}

func EncodeCapabilities(dst []byte, c Capabilities) error {
	if len(dst) != CapabilitiesSize || !c.valid() {
		return ErrRecord
	}
	clear(dst)
	binary.LittleEndian.PutUint16(dst[0:2], c.Width)
	binary.LittleEndian.PutUint16(dst[2:4], c.Height)
	binary.LittleEndian.PutUint16(dst[4:6], c.Stride)
	binary.LittleEndian.PutUint16(dst[6:8], c.MaxChunk)
	dst[8], dst[9] = c.Passes, c.Format
	binary.LittleEndian.PutUint16(dst[10:12], c.Features)
	binary.LittleEndian.PutUint32(dst[12:16], c.Profile)
	binary.LittleEndian.PutUint16(dst[16:18], c.ProfileVersion)
	binary.LittleEndian.PutUint32(dst[20:24], c.MinimumFullMS)
	copy(dst[24:40], c.DeviceID[:])
	return nil
}

func DecodeCapabilities(src []byte) (Capabilities, error) {
	if len(src) != CapabilitiesSize {
		return Capabilities{}, ErrRecord
	}
	if src[18] != 0 || src[19] != 0 {
		return Capabilities{}, ErrRecord
	}
	c := Capabilities{Width: binary.LittleEndian.Uint16(src[0:2]), Height: binary.LittleEndian.Uint16(src[2:4]),
		Stride: binary.LittleEndian.Uint16(src[4:6]), MaxChunk: binary.LittleEndian.Uint16(src[6:8]), Passes: src[8], Format: src[9],
		Features: binary.LittleEndian.Uint16(src[10:12]), Profile: binary.LittleEndian.Uint32(src[12:16]),
		ProfileVersion: binary.LittleEndian.Uint16(src[16:18]), MinimumFullMS: binary.LittleEndian.Uint32(src[20:24])}
	copy(c.DeviceID[:], src[24:40])
	if !c.valid() {
		return Capabilities{}, ErrRecord
	}
	return c, nil
}
