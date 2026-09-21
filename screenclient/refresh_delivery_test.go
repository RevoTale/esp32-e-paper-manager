package screenclient

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

// This peer tests client sequencing; Go/C interop separately executes the real
// receiver, SHA validation, plane writes and panel lifecycle with these options.
type refreshPeer struct {
	bytes.Buffer
	status screenwire.Status
	mutate func(*screenwire.Status)
	fail   bool
}

func (p *refreshPeer) Write(b []byte) (int, error) {
	if p.fail {
		return 0, io.ErrUnexpectedEOF
	}
	r, err := screenwire.Decode(b)
	if err != nil {
		return 0, err
	}
	s, err := p.replyStatus(r)
	if err != nil {
		return 0, err
	}
	var body [screenwire.StatusSize]byte
	if err = screenwire.EncodeStatus(body[:], s); err != nil {
		return 0, err
	}
	var reply [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(reply[:], screenwire.Record{Kind: screenwire.Reply, Epoch: r.Epoch, ID: r.ID, Payload: body[:]})
	if err != nil {
		return 0, err
	}
	_, err = p.Buffer.Write(reply[:n])
	return len(b), err
}

func (p *refreshPeer) replyStatus(r screenwire.Record) (screenwire.Status, error) {
	s := p.status
	s.Operation = r.Kind
	s.State = streamrx.Receiving
	switch r.Kind {
	case screenwire.BeginRefresh:
		if _, _, _, err := screenwire.DecodeRefreshBegin(r.Payload); err != nil {
			return s, err
		}
	case screenwire.Data:
		s.Pass = r.Pass
		s.Offset = r.Offset + uint32(len(r.Payload))
		if s.Offset == 27 {
			s.Pass++
			s.Offset = 0
		}
		if s.Pass == 2 {
			s.State = streamrx.Ready
		}
	case screenwire.Commit:
		s.State = streamrx.Complete
		s.Pass = 2
		s.CurrentImage = true
	default:
		return s, screenwire.ErrRecord
	}
	if p.mutate != nil {
		p.mutate(&s)
	}
	return s, nil
}

func TestRefreshClientDeliveryAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fail   bool
		mutate func(*screenwire.Status)
		want   error
	}{
		{"success", false, nil, nil},
		{"lost", true, nil, io.ErrUnexpectedEOF},
		{"bad progress", false, func(s *screenwire.Status) { s.State = streamrx.Ready }, screenwire.ErrRecord},
		{"lease", false, func(s *screenwire.Status) { s.Generation++ }, ErrResync},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _, _, f := fixture(t)
			c.caps.Features |= screenwire.FeatureRefreshPolicy
			c.stream = &refreshPeer{status: screenwire.Status{Boot: c.lease.Boot, Generation: c.lease.Generation}, fail: tc.fail, mutate: tc.mutate}
			err := c.SendWithOptions(f, refreshpolicy.Options{Priority: refreshpolicy.Urgent}, refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second})
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if c.Pending() != (tc.want != nil) {
				t.Fatal("wrong pending identity")
			}
		})
	}
}

func TestRefreshClientRejectsInvalidFrame(t *testing.T) {
	c, _, _, _, _ := fixture(t)
	c.caps.Features |= screenwire.FeatureRefreshPolicy
	if err := c.SendWithOptions(display.Frame{}, refreshpolicy.Options{}, refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); !errors.Is(err, screenwire.ErrRecord) {
		t.Fatal(err)
	}
}
