package screenusbhost

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

type packedSink struct {
	testSink
	written int
	invalid bool
}

func (s *packedSink) Write(_ uint8, _ uint32, data []byte) error {
	s.written += len(data)
	s.invalid = s.invalid || !bytes.Equal(data, make([]byte, 16))
	return nil
}

func TestProxyDataPackedReachesTwoPlaneSessionUnchanged(t *testing.T) {
	sink := &packedSink{}
	caps := screenwire.Capabilities{Width: 128, Height: 1, Stride: 16, MaxChunk: 16, Passes: 2, Format: screenwire.Mono1,
		Features: screenwire.RawFull | screenwire.FeaturePackBits, Profile: 99, ProfileVersion: 2, MinimumFullMS: 1000}
	d, err := screenlink.New(caps, [16]byte{1}, sink, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := &testWorker{link: d.Open(), done: make(chan struct{})}
	defer func() { _ = p.Close() }()
	var claim [32]byte
	claim[0], claim[16] = 1, 2
	digest := sha256.Sum256(make([]byte, 16))
	records := []screenwire.Record{
		{Kind: screenwire.Hello},
		{Kind: screenwire.Acquire, Payload: claim[:]},
		{Kind: screenwire.Bind, Epoch: 1, Payload: claim[:]},
		{Kind: screenwire.Begin, Epoch: 1, ID: 1, Payload: digest[:]},
		{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Payload: []byte{16, 0, 241, 0}},
		{Kind: screenwire.DataPacked, Epoch: 1, ID: 1, Pass: 1, Payload: []byte{16, 0, 241, 0}},
		{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: digest[:]},
	}
	var input, output bytes.Buffer
	for _, r := range records {
		input.Write(encoded(t, r))
	}
	if err = serveProxy(&input, &output, p); err != nil {
		t.Fatal(err)
	}
	assertPackedReplies(t, &output, records)
	if sink.commits != 1 || sink.written != 32 || sink.invalid {
		t.Fatal(sink)
	}
}

func assertPackedReplies(t *testing.T, output *bytes.Buffer, requests []screenwire.Record) {
	t.Helper()
	var data [screenwire.MaxRecord]byte
	for _, request := range requests {
		n, err := readRecord(output, data[:])
		if err != nil {
			t.Fatal(err)
		}
		r, err := screenwire.Decode(data[:n])
		if err != nil {
			t.Fatal(err)
		}
		reply, err := screenwire.ParseReply(r, request)
		if err != nil || reply.Status.Code != screenwire.CodeOK {
			t.Fatal(reply, err)
		}
	}
	if output.Len() != 0 {
		t.Fatal("extra reply bytes")
	}
}
