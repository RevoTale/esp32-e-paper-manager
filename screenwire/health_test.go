package screenwire

import (
	"bytes"
	"errors"
	"testing"
)

func TestHealthLiteralBytesAndLegacyOperationValues(t *testing.T) {
	if got := [10]Kind{Hello, Acquire, Bind, Begin, Data, Commit, Query, Abort, Reply, Health}; got != [10]Kind{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		t.Fatal(got)
	}
	r := Record{Kind: Health}
	b := make([]byte, HeaderSize)
	if n, err := Encode(b, r); err != nil || n != HeaderSize || !bytes.Equal(b, literal(r)) {
		t.Fatal(n, err, b)
	}
	h := HealthStatus{Version: 1, State: 6, LastFailure: 3, Failures: 255, UptimeSeconds: 0x08070605}
	data := make([]byte, HealthSize)
	if err := EncodeHealth(data, h); err != nil || !bytes.Equal(data, []byte{1, 6, 3, 255, 5, 6, 7, 8}) {
		t.Fatal(data, err)
	}
	if got, err := DecodeHealth(data); err != nil || got != h {
		t.Fatal(got, err)
	}
}

func TestHealthRejectsUnknownVersionStateAndWrongLengths(t *testing.T) {
	for _, value := range []HealthStatus{{}, {Version: 2}, {Version: 1, State: 8}, {Version: 1, LastFailure: 8}} {
		if err := EncodeHealth(make([]byte, HealthSize), value); !errors.Is(err, ErrRecord) {
			t.Fatal(value, err)
		}
	}
	for _, data := range [][]byte{nil, make([]byte, 7), make([]byte, 9), {2, 0, 0, 0, 0, 0, 0, 0}, {1, 8, 0, 0, 0, 0, 0, 0}, {1, 0, 8, 0, 0, 0, 0, 0}} {
		if _, err := DecodeHealth(data); !errors.Is(err, ErrRecord) {
			t.Fatal(data, err)
		}
	}
	if err := EncodeHealth(make([]byte, HealthSize-1), HealthStatus{Version: 1}); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestHealthRequestRequiresEmptyZeroIdentity(t *testing.T) {
	for _, r := range []Record{{Kind: Health, Epoch: 1}, {Kind: Health, ID: 1}, {Kind: Health, Pass: 1}, {Kind: Health, Offset: 1}, {Kind: Health, Payload: []byte{0}}, {Kind: 11}} {
		if _, err := Decode(literal(r)); !errors.Is(err, ErrRecord) {
			t.Fatal(r, err)
		}
	}
}

func TestHealthReplyRequiresExactOperationAndBody(t *testing.T) {
	body := make([]byte, StatusSize+HealthSize)
	status := Status{Operation: Health, Boot: [16]byte{1}}
	if err := EncodeStatus(body[:StatusSize], status); err != nil {
		t.Fatal(err)
	}
	h := HealthStatus{Version: 1, State: 5, UptimeSeconds: 12}
	if err := EncodeHealth(body[StatusSize:], h); err != nil {
		t.Fatal(err)
	}
	r := Record{Kind: Reply, Payload: body}
	if got, err := ParseReply(r, Record{Kind: Health}); err != nil || got.Health != h || got.Status != status {
		t.Fatal(got, err)
	}
	for _, request := range []Record{{Kind: Hello}, {Kind: Reply}, {Kind: 11}, {Kind: Health, ID: 1}} {
		if _, err := ParseReply(r, request); !errors.Is(err, ErrRecord) {
			t.Fatal(request, err)
		}
	}
	r.Payload = body[:StatusSize]
	if _, err := ParseReply(r, Record{Kind: Health}); !errors.Is(err, ErrRecord) {
		t.Fatal(err)
	}
}

func TestHealthDoesNotMakeReplyAValidStatusOperation(t *testing.T) {
	for _, operation := range []Kind{0, Reply, 14, 255} {
		if err := EncodeStatus(make([]byte, StatusSize), Status{Operation: operation, Boot: [16]byte{1}}); !errors.Is(err, ErrRecord) {
			t.Fatal(operation, err)
		}
	}
	body := make([]byte, StatusSize+HealthSize)
	if err := EncodeStatus(body[:StatusSize], Status{Operation: Health, Boot: [16]byte{1}}); err != nil {
		t.Fatal(err)
	}
	body[StatusSize] = 99
	if _, err := ParseReply(Record{Kind: Reply, Payload: body}, Record{Kind: Health}); !errors.Is(err, ErrRecord) {
		t.Fatal("invalid health version accepted", err)
	}
}
