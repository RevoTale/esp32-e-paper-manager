package screenhub

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestCommitCutRejectsMissingReply(t *testing.T) {
	p := &ackCut{ReadWriteCloser: &cutMemory{}, armed: true}
	var data [screenwire.MaxRecord]byte
	n, err := screenwire.Encode(data[:], screenwire.Record{Kind: screenwire.Commit, Epoch: 1, ID: 1, Payload: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Write(data[:n]); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Read(data[:]); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if p.cut {
		t.Fatal("missing ACK was reported as injected loss")
	}
}

type cutMemory struct{ bytes.Buffer }

func (*cutMemory) Close() error                { return nil }
func (*cutMemory) Write(p []byte) (int, error) { return len(p), nil }

// Test-only fault injection: consume and validate the authenticated terminal
// Commit response, then hide it from the client. Never cut during panel work.
type ackCut struct {
	io.ReadWriteCloser
	armed, cut       bool
	request          screenwire.Record
	commits, queries int
	target           screenwire.Kind
}

func (p *ackCut) cutKind() screenwire.Kind {
	if p.target == screenwire.Data {
		return screenwire.Data
	}
	return screenwire.Commit
}

func (p *ackCut) Write(data []byte) (int, error) {
	r, err := screenwire.Decode(data)
	if err != nil {
		return 0, err
	}
	if r.Kind == screenwire.Commit {
		p.commits++
	}
	if r.Kind == p.cutKind() {
		p.request = r
	}
	if r.Kind == screenwire.Query {
		p.queries++
	}
	return p.ReadWriteCloser.Write(data)
}

func (p *ackCut) Read(data []byte) (int, error) {
	if !p.armed || p.request.Kind != p.cutKind() {
		return p.ReadWriteCloser.Read(data)
	}
	var wire [screenwire.MaxRecord]byte
	if _, err := io.ReadFull(p.ReadWriteCloser, wire[:screenwire.HeaderSize]); err != nil {
		return 0, err
	}
	n, err := screenwire.Size(wire[:screenwire.HeaderSize])
	if err != nil {
		return 0, err
	}
	if _, err = io.ReadFull(p.ReadWriteCloser, wire[screenwire.HeaderSize:n]); err != nil {
		return 0, err
	}
	r, err := screenwire.Decode(wire[:n])
	if err != nil {
		return 0, err
	}
	reply, err := screenwire.ParseReply(r, p.request)
	if err != nil {
		return 0, err
	}
	if reply.Status.Code != screenwire.CodeOK || reply.Status.State != p.cutState() {
		return 0, errors.New("fault probe: unexpected cut-point state; stop, do not replay")
	}
	p.cut = true
	return 0, errors.Join(io.ErrUnexpectedEOF, p.Close())
}

func (p *ackCut) cutState() streamrx.State {
	if p.cutKind() == screenwire.Data {
		return streamrx.Receiving
	}
	return streamrx.Complete
}
