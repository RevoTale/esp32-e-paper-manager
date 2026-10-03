package screenclient

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

// Models only peer replies and failure boundaries; native interop must also
// prove receiver validation and SPI behavior before device enablement.
type regionPeer struct {
	bytes.Buffer
	status  screenwire.Status
	planes  [2][]byte
	commits int
	lose    screenwire.Kind
	bad     screenwire.Kind
	badPass uint8
	mutate  func(*screenwire.Status)
}

func (p *regionPeer) Write(b []byte) (int, error) {
	r, err := screenwire.Decode(b)
	if err != nil {
		return 0, err
	}
	s, err := p.advance(r)
	if err != nil {
		return 0, err
	}
	p.status = s
	if r.Kind == p.lose {
		return 0, io.ErrUnexpectedEOF
	}
	if r.Kind == p.bad && r.Pass == p.badPass {
		s.Code = screenwire.CodeHardware
	}
	if p.mutate != nil {
		p.mutate(&s)
	}
	var body [screenwire.StatusSize]byte
	if err := screenwire.EncodeStatus(body[:], s); err != nil {
		return 0, err
	}
	var out [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(out[:], screenwire.Record{Kind: screenwire.Reply, Epoch: r.Epoch, ID: r.ID, Payload: body[:]})
	if err != nil {
		return 0, err
	}
	_, err = p.Buffer.Write(out[:n])
	return len(b), err
}

func (p *regionPeer) advance(r screenwire.Record) (screenwire.Status, error) {
	s := p.status
	s.Operation = r.Kind
	switch r.Kind {
	case screenwire.BeginRegion:
		s.State = streamrx.Receiving
	case screenwire.Data:
		p.planes[r.Pass] = append(p.planes[r.Pass], r.Payload...)
		s.Pass, s.Offset = r.Pass, r.Offset+uint32(len(r.Payload))
		if s.Offset == 2 {
			s.Pass++
			s.Offset = 0
		}
		if s.Pass == 2 {
			s.State = streamrx.Ready
		}
	case screenwire.Commit:
		p.commits++
		s.State, s.CurrentImage = streamrx.Complete, true
	case screenwire.Query:
	default:
		return s, screenwire.ErrRecord
	}
	return s, nil
}

func regionFixture(t *testing.T) (*Client, *regionPeer, screenwire.RegionBegin, []byte, []byte) {
	t.Helper()
	c, _, _, _, frame := fixture(t)
	if err := c.Send(frame); err != nil {
		t.Fatal(err)
	}
	old, next := []byte{0x80, 0x80}, []byte{0, 0}
	r := screenwire.RegionBegin{Baseline: c.Baseline(), Old: sha256.Sum256(old), New: sha256.Sum256(next), Right: 8, Bottom: 2,
		Policy: refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}}
	p := &regionPeer{status: screenwire.Status{Boot: c.lease.Boot, Generation: c.lease.Generation}}
	c.stream = p
	c.caps.Features |= screenwire.FeatureRegion
	c.maxChunk = 1
	return c, p, r, old, next
}

func TestRegionDeliveryAndLostCommit(t *testing.T) {
	for _, lost := range []bool{false, true} {
		c, p, r, old, next := regionFixture(t)
		if lost {
			p.lose = screenwire.Commit
		}
		err := c.SendRegion(r, old, next)
		if lost {
			if !errors.Is(err, io.ErrUnexpectedEOF) || !c.Pending() || c.Baseline() != [32]byte{} {
				t.Fatal(err)
			}
			// Rebind is covered separately by real transport tests.
			c.bound, c.stream = true, p
			result, queryErr := c.Reconcile()
			if queryErr != nil || !result.Confirmed {
				t.Fatal(result, queryErr)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		assertRegionDelivered(t, c, p, r, old, next)
	}
}

func assertRegionDelivered(t *testing.T, c *Client, p *regionPeer, r screenwire.RegionBegin, old, next []byte) {
	t.Helper()
	b, err := screenwire.EncodeRegionBegin(r)
	if err != nil {
		t.Fatal(err)
	}
	if c.Pending() || c.Baseline() != [32]byte(b[:32]) || p.commits != 1 {
		t.Fatal("unconfirmed or repeated refresh")
	}
	if !bytes.Equal(p.planes[0], old) || !bytes.Equal(p.planes[1], next) {
		t.Fatal(p.planes)
	}
}

func TestRegionFailureRetainsIdentity(t *testing.T) {
	for _, kind := range []screenwire.Kind{screenwire.BeginRegion, screenwire.Data, screenwire.Commit} {
		c, p, r, old, next := regionFixture(t)
		p.bad = kind
		if err := c.SendRegion(r, old, next); err == nil || !c.Pending() || c.Baseline() != [32]byte{} {
			t.Fatal(kind, err)
		}
		if err := c.SendRegion(r, old, next); !errors.Is(err, ErrPending) {
			t.Fatal(err)
		}
	}
	c, _, r, old, next := regionFixture(t)
	c.bound = false
	if err := c.SendRegion(r, old, next); !errors.Is(err, ErrConnection) {
		t.Fatal(err)
	}
}
