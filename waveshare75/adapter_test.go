package waveshare75

import (
	"bytes"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/panel"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestAdapterAdvertisesExactAcceptedProfileWithoutBootIO(t *testing.T) {
	writes := 0
	io := panel.IO{Write: func([]byte) error { writes++; return nil }, SetCS: func(bool) {}, SetDC: func(bool) {},
		SetReset: func(bool) {}, SetPower: func(bool) {}, ReadBusy: func() bool { return true }, Delay: func(time.Duration) {}}
	device, err := New(io, [16]byte{1})
	if err != nil {
		t.Fatal(err)
	}
	var request [screenwire.HeaderSize]byte
	_, _ = screenwire.Encode(request[:], screenwire.Record{Kind: screenwire.Hello})
	var response bytes.Buffer
	if err = device.Open().Push(request[:], 0, func(p []byte) error { _, e := response.Write(p); return e }); err != nil {
		t.Fatal(err)
	}
	record, err := screenwire.Decode(response.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	caps, err := screenwire.DecodeCapabilities(record.Payload[screenwire.StatusSize:])
	want := screenwire.Capabilities{Width: 800, Height: 480, Stride: 100, MaxChunk: 1000,
		Passes: 2, Format: screenwire.Mono1, Features: screenwire.RawFull | screenwire.FeaturePackBits, Profile: 1, ProfileVersion: 1, MinimumFullMS: 180000}
	if err != nil || caps != want {
		t.Fatal(caps, err)
	}
	if writes != 0 {
		t.Fatal("automatic boot display write")
	}
}
