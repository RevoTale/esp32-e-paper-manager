package screenwire

import (
	"errors"
	"testing"
)

func TestReplyCorrelationRejectsCrossOperationAndWrongShape(t *testing.T) {
	b := make([]byte, StatusSize+CapabilitiesSize)
	s := Status{Operation: Hello, Boot: [16]byte{1}}
	if err := EncodeStatus(b[:StatusSize], s); err != nil {
		t.Fatal(err)
	}
	if err := EncodeCapabilities(b[StatusSize:], capsFixture()); err != nil {
		t.Fatal(err)
	}
	r := Record{Kind: Reply, Payload: b}
	request := Record{Kind: Hello}
	got, err := ParseReply(r, request)
	if err != nil || got.Capabilities != capsFixture() {
		t.Fatal(got, err)
	}
	for _, mutate := range []func(*Record){
		func(r *Record) { r.Kind = Data }, func(r *Record) { r.Epoch = 1 }, func(r *Record) { r.ID = 1 },
		func(r *Record) { r.Offset = 1 }, func(r *Record) { r.Pass = 1 },
		func(r *Record) { r.Payload = r.Payload[:48] }, func(r *Record) { r.Payload = nil },
	} {
		bad := r
		mutate(&bad)
		if _, err := ParseReply(bad, request); !errors.Is(err, ErrRecord) {
			t.Fatal(bad, err)
		}
	}
	request.Kind = Acquire
	if _, err = ParseReply(r, request); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestReplyNonHelloHasNoCapabilities(t *testing.T) {
	b := make([]byte, StatusSize)
	s := Status{Operation: Bind, Boot: [16]byte{1}, Generation: 2}
	if err := EncodeStatus(b, s); err != nil {
		t.Fatal(err)
	}
	request := Record{Kind: Bind, Epoch: 2, Payload: make([]byte, 32)}
	r := Record{Kind: Reply, Epoch: 2, Payload: b}
	if _, err := ParseReply(r, request); err != nil {
		t.Fatal(err)
	}
	r.Payload = append(b, make([]byte, 40)...)
	if _, err := ParseReply(r, request); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
	r.Payload = b
	b[0] = 0
	if _, err := ParseReply(r, request); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}
