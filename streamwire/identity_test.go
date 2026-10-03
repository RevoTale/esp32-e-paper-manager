package streamwire

import (
	"crypto/sha256"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestWrongIdentityInvalidatesReadyTransaction(t *testing.T) {
	l, s := linkFixture(t)
	d := sha256.Sum256([]byte{128, 1})
	for _, r := range []Record{{Kind: Hello, Epoch: 7}, {Kind: Begin, Epoch: 7, ID: 1, Payload: d[:]}, {Kind: Data, Epoch: 7, ID: 1, Payload: []byte{128, 1}}, {Kind: Data, Epoch: 7, ID: 1, Pass: 1, Payload: []byte{128, 1}}} {
		exchange(t, l, r, 0)
	}
	exchange(t, l, Record{Kind: Data, Epoch: 7, ID: 99, Payload: []byte{128}}, 0)
	r := exchange(t, l, Record{Kind: Commit, Epoch: 7, ID: 1}, 0)
	if r.Payload[0] == 0 || streamrx.State(r.Payload[1]) != streamrx.Failed || s.commits != 0 || s.aborts != 1 {
		t.Fatal("identity error left staged frame usable")
	}
}
