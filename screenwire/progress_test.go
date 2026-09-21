package screenwire

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestImpossibleReceiverProgressRejects(t *testing.T) {
	for _, v := range []struct {
		state  streamrx.State
		pass   uint8
		offset uint32
	}{
		{streamrx.Complete, 0, 1}, {streamrx.Ready, 0, 0}, {streamrx.Complete, 1, 1},
		{streamrx.Receiving, 2, 0}, {streamrx.Failed, 2, 1}, {streamrx.Idle, 1, 0}, {streamrx.Idle, 0, 1},
	} {
		s := Status{Operation: Query, State: v.state, Pass: v.pass, Offset: v.offset, Generation: 1, Boot: [16]byte{1}}
		if err := EncodeStatus(make([]byte, StatusSize), s); !errors.Is(err, ErrRecord) {
			t.Fatal(v, err)
		}
	}
}

func TestValidCompleteProofAndUnknownBoot(t *testing.T) {
	s := Status{Operation: Query, State: streamrx.Complete, Pass: 2, CurrentImage: true, Generation: 1, Boot: [16]byte{1}}
	b := make([]byte, StatusSize)
	if err := EncodeStatus(b, s); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeStatus(b)
	if err != nil || got != s {
		t.Fatal(got, err)
	}
	s.Boot = [16]byte{}
	if err = EncodeStatus(b, s); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}
