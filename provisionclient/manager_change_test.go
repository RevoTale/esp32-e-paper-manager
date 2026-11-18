package provisionclient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"math"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

const newManager = "tcp://screen.home.arpa:9757"

type changingPeer struct {
	bytes.Buffer
	before provision.Response
	after  provision.Response
	writes int
	drop   int
}

func (p *changingPeer) Write(request []byte) (int, error) {
	p.writes++
	if p.writes == p.drop {
		return 0, io.ErrClosedPipe
	}
	res := p.before
	if request[5] == byte(provision.OperationSetManager) {
		res = p.after
	}
	// The legacy encoder builds the common public metadata. The v3 peer echoes
	// the received op and correlation ID and recomputes its own CRC.
	res.Operation = provision.OperationInspect
	var reply [512]byte
	if err := provision.ESP32Codec().EncodeResponse(reply[:], res); err != nil {
		return 0, err
	}
	reply[4], reply[5] = 3, request[5]
	copy(reply[392:408], request[486:502])
	binary.BigEndian.PutUint32(reply[508:], crc32.ChecksumIEEE(reply[:508]))
	p.Buffer.Write(reply[:])
	return len(request), nil
}

func newChangingPeer() *changingPeer {
	before := provision.Response{State: provision.StateProvisioned, Auth: provision.AuthWPA2PSK,
		SSID: "test", Timezone: "Europe/Kyiv", Manager: "tcp://old:9757", DeviceID: [16]byte{9}, Generation: 1}
	after := before
	after.Manager, after.Generation = newManager, 2
	return &changingPeer{before: before, after: after}
}

func TestSetManagerOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		alter  func(*changingPeer)
		writes int
		ok     bool
	}{
		{"updated", func(*changingPeer) {}, 2, true},
		{"already configured", func(p *changingPeer) { p.before.Manager = newManager }, 1, true},
		{"blank", func(p *changingPeer) { p.before = provision.Response{} }, 1, false},
		{"generation exhausted", func(p *changingPeer) { p.before.Generation = math.MaxUint64 }, 1, false},
		{"inspect lost", func(p *changingPeer) { p.drop = 1 }, 1, false},
		{"mutation lost", func(p *changingPeer) { p.drop = 2 }, 2, false},
		{"stale snapshot", func(p *changingPeer) { p.after.Code = provision.CodeState }, 2, false},
		{"wrong endpoint", func(p *changingPeer) { p.after.Manager = p.before.Manager }, 2, false},
		{"wrong generation", func(p *changingPeer) { p.after.Generation++ }, 2, false},
		{"changed identity", func(p *changingPeer) { p.after.DeviceID[0]++ }, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := newChangingPeer()
			tc.alter(peer)
			client, err := NewCorrelated(peer)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.SetManager(newManager)
			if (err == nil) != tc.ok || peer.writes != tc.writes || (tc.ok && got.Manager != newManager) {
				t.Fatal(got, err, peer.writes)
			}
		})
	}
}

func TestSetManagerRejectsInvalidClientAndAddress(t *testing.T) {
	peer := newChangingPeer()
	c, _ := NewFor(peer, provision.ESP32Codec())
	if _, err := c.SetManager(newManager); !errors.Is(err, ErrConfiguration) {
		t.Fatal(err)
	}
	c, _ = NewCorrelated(peer)
	if _, err := c.SetManager("invalid"); !errors.Is(err, ErrConfiguration) || peer.writes != 0 {
		t.Fatal(err, peer.writes)
	}
}
