package screenwire

import (
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func TestRefreshBeginContract(t *testing.T) {
	p := refreshpolicy.Policy{Normal: 180 * time.Second, Urgent: 30 * time.Second}
	o := refreshpolicy.Options{Priority: refreshpolicy.Urgent, Mode: refreshpolicy.Full}
	b, err := EncodeRefreshBegin([32]byte{9}, o, p)
	if err != nil || b[32] != 1 || b[33] != 2 {
		t.Fatal(b, err)
	}
	digest, options, policy, err := DecodeRefreshBegin(b[:])
	if err != nil || digest != ([32]byte{9}) || options != o || policy != p {
		t.Fatal(digest, options, policy, err)
	}
	var wire [MaxRecord]byte
	n, err := Encode(wire[:], Record{Kind: BeginRefresh, Epoch: 1, ID: 1, Payload: b[:]})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(wire[:n]); err != nil {
		t.Fatal(err)
	}
	checkMalformedRefresh(t, b)
}

func checkMalformedRefresh(t *testing.T, b [44]byte) {
	t.Helper()
	for _, index := range []int{32, 33, 34, 35} {
		bad := b
		bad[index] = 255
		if _, _, _, err := DecodeRefreshBegin(bad[:]); err == nil {
			t.Fatal(index)
		}
	}
	if _, _, _, err := DecodeRefreshBegin(b[:43]); err == nil {
		t.Fatal("short")
	}
}

func TestRefreshBeginRejectsInvalidPolicy(t *testing.T) {
	for _, p := range []refreshpolicy.Policy{
		{}, {Normal: time.Second}, {Normal: time.Second, Urgent: time.Nanosecond},
		{Normal: (1 << 32) * time.Millisecond, Urgent: time.Second},
		{Normal: -time.Second, Urgent: time.Second},
	} {
		if _, err := EncodeRefreshBegin([32]byte{}, refreshpolicy.Options{}, p); err == nil {
			t.Fatal(p)
		}
	}
	p := refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}
	for _, o := range []refreshpolicy.Options{{Priority: 2}, {Mode: 3}} {
		if _, err := EncodeRefreshBegin([32]byte{}, o, p); err == nil {
			t.Fatal(o)
		}
	}
}
