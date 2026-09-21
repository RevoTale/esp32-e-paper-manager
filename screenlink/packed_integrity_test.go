package screenlink

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestPackedDataPreservesDigestAndOffsetFailures(t *testing.T) {
	for _, tc := range []struct {
		pixels byte
		offset uint32
		code   screenwire.Code
	}{
		{129, 0, screenwire.CodeDigest},
		{128, 1, screenwire.CodeChunk},
	} {
		d, c, sink := fixture(t)
		d.caps.Features |= screenwire.FeaturePackBits
		bind(t, c, 1, true)
		digest := sha256.Sum256([]byte{128})
		r := screenwire.Record{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]}
		if got := exchange(t, c, r, time.Second); got.Code != screenwire.CodeOK {
			t.Fatal(got)
		}
		r = screenwire.Record{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Offset: tc.offset, Payload: []byte{1, 0, 0, tc.pixels}}
		if got := exchange(t, c, r, time.Second); got.Code != tc.code {
			t.Fatal(tc, got)
		}
		if sink.commits != 0 || sink.aborts != 1 {
			t.Fatal(sink)
		}
	}
}
