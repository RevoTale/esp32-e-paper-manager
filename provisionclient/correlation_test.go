package provisionclient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

type correlatedPeer struct {
	bytes.Buffer
	writes         int
	prefix         []byte
	staleOnly      bool
	stalePrefix    bool
	wrongOperation bool
	fragment       int
	code           provision.Code
}

func (p *correlatedPeer) Write(request []byte) (int, error) {
	p.writes++
	var reply [512]byte
	codec := provision.ESP32Codec()
	operation := provision.Operation(request[5])
	if p.wrongOperation {
		operation = provision.OperationDiagnose
	}
	err := codec.EncodeResponse(reply[:], provision.Response{Operation: operation, State: provision.StateBlank, Code: p.code})
	if err != nil {
		return 0, err
	}
	reply[4] = 3
	copy(reply[392:408], request[486:502])
	if p.staleOnly {
		reply[392] ^= 1
	}
	binary.BigEndian.PutUint32(reply[508:], crc32.ChecksumIEEE(reply[:508]))
	p.Buffer.Write(p.prefix)
	if p.stalePrefix {
		stale := reply
		stale[392] ^= 1
		binary.BigEndian.PutUint32(stale[508:], crc32.ChecksumIEEE(stale[:508]))
		p.Buffer.Write(stale[:])
	}
	p.Buffer.Write(reply[:])
	return len(request), nil
}

func (p *correlatedPeer) Read(out []byte) (int, error) {
	if p.fragment > 0 && len(out) > p.fragment {
		out = out[:p.fragment]
	}
	return p.Buffer.Read(out)
}

func TestCorrelatedClientSkipsPrefixAndStaleReply(t *testing.T) {
	for _, fragment := range []int{1, 17, 4096} {
		p := &correlatedPeer{prefix: bytes.Repeat([]byte{0xa5}, 256), fragment: fragment, stalePrefix: true}
		c, err := NewCorrelated(p)
		if err != nil {
			t.Fatal(err)
		}
		r, err := c.Execute(provision.Request{Operation: provision.OperationInspect})
		if err != nil || r.State != provision.StateBlank || p.writes != 1 {
			t.Fatal(r, err, p.writes)
		}
	}
}

func TestCorrelatedClientRejectsStaleAndPoisonsConnection(t *testing.T) {
	p := &correlatedPeer{staleOnly: true}
	c, err := NewCorrelated(p)
	if err != nil {
		t.Fatal(err)
	}
	request := provision.Request{Operation: provision.OperationErase}
	if _, err = c.Execute(request); err == nil {
		t.Fatal("stale accepted")
	}
	if _, err = c.Execute(request); !errors.Is(err, ErrConfiguration) || p.writes != 1 {
		t.Fatal("retried ambiguous operation", err)
	}
}

func TestCorrelatedClientBoundsNoiseAndChecksOperation(t *testing.T) {
	for _, p := range []*correlatedPeer{{prefix: bytes.Repeat([]byte{0xa5}, 4096)}, {wrongOperation: true}} {
		c, err := NewCorrelated(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Execute(provision.Request{Operation: provision.OperationInspect}); err == nil {
			t.Fatal("invalid response accepted")
		}
		if p.writes != 1 {
			t.Fatal("request retried")
		}
	}
	if _, err := NewCorrelated(nil); err == nil {
		t.Fatal("nil stream accepted")
	}
}

var _ io.ReadWriter = (*correlatedPeer)(nil)
