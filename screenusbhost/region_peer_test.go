package screenusbhost

import (
	"bytes"
	"crypto/sha256"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

// Reply peer exercises the real USB adapter/client without GPIO. Native C
// interop separately verifies receiver validation and panel command sequences.
type regionPeerState struct {
	status           screenwire.Status
	region           screenwire.RegionBegin
	baseline, digest [32]byte
	planes           [2][]byte
	length, commits  int
	dropCommit       bool
}

type regionWorker struct {
	bytes.Buffer
	state *regionPeerState
	done  chan struct{}
}

func (w *regionWorker) Done() <-chan struct{} { return w.done }
func (w *regionWorker) Close() error {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
	return nil
}

func (w *regionWorker) Write(data []byte) (int, error) {
	r, err := screenwire.Decode(data)
	if err != nil {
		return 0, err
	}
	if err = w.state.advance(r); err != nil {
		return 0, err
	}
	if r.Kind == screenwire.Commit && w.state.dropCommit {
		w.state.dropCommit = false
		return 0, io.ErrUnexpectedEOF
	}
	body := make([]byte, screenwire.StatusSize)
	if err = screenwire.EncodeStatus(body, w.state.status); err != nil {
		return 0, err
	}
	if r.Kind == screenwire.Hello {
		caps := screenwire.Capabilities{Width: 16, Height: 4, Stride: 2, MaxChunk: 8, Passes: 2, Format: screenwire.Mono1,
			Features: screenwire.RawFull | screenwire.FeatureRefreshPolicy | screenwire.FeatureRegion,
			Profile:  1, ProfileVersion: 1, MinimumFullMS: 1}
		b := make([]byte, screenwire.CapabilitiesSize)
		if err = screenwire.EncodeCapabilities(b, caps); err != nil {
			return 0, err
		}
		body = append(body, b...)
	}
	var reply [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(reply[:], screenwire.Record{Kind: screenwire.Reply, Epoch: r.Epoch, ID: r.ID, Payload: body})
	if err != nil {
		return 0, err
	}
	_, err = w.Buffer.Write(reply[:n])
	return len(data), err
}

func (s *regionPeerState) advance(r screenwire.Record) error {
	s.status.Operation, s.status.Boot = r.Kind, [16]byte{1}
	switch r.Kind {
	case screenwire.Acquire:
		s.status.Generation++
	case screenwire.Hello, screenwire.Bind, screenwire.Query:
	case screenwire.BeginRefresh, screenwire.BeginRegion:
		return s.begin(r)
	case screenwire.Data:
		s.planes[r.Pass] = append(s.planes[r.Pass], r.Payload...)
		s.status.Pass, s.status.Offset = r.Pass, r.Offset+uint32(len(r.Payload))
		if len(s.planes[r.Pass]) == s.length {
			s.status.Pass++
			s.status.Offset = 0
		}
		if s.status.Pass == 2 {
			s.status.State = streamrx.Ready
		}
	case screenwire.Commit:
		return s.commit()
	default:
		return screenwire.ErrRecord
	}
	return nil
}

func (s *regionPeerState) commit() error {
	if sha256.Sum256(s.planes[0]) != s.region.Old || sha256.Sum256(s.planes[1]) != s.region.New {
		return screenwire.ErrRecord
	}
	s.commits++
	s.baseline = s.digest
	s.status.State, s.status.CurrentImage = streamrx.Complete, true
	return nil
}

func (s *regionPeerState) begin(r screenwire.Record) error {
	if r.Kind == screenwire.BeginRegion {
		region, err := screenwire.DecodeRegionBegin(r.Payload, 16, 4)
		if err != nil {
			return err
		}
		if region.Baseline != s.baseline {
			return screenwire.ErrRecord
		}
		s.region, s.length = region, int(region.Right-region.Left)/8*int(region.Bottom-region.Top)
	} else {
		digest, _, _, err := screenwire.DecodeRefreshBegin(r.Payload)
		if err != nil {
			return err
		}
		s.region, s.length = screenwire.RegionBegin{Old: digest, New: digest}, 8
	}
	s.digest = [32]byte(r.Payload[:32])
	s.planes = [2][]byte{}
	s.status.State, s.status.CurrentImage, s.status.Pass, s.status.Offset = streamrx.Receiving, false, 0, 0
	return nil
}
