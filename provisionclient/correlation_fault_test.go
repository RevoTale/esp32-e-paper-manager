package provisionclient

import (
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

type correlationFault struct {
	reads, writes int
	readCount     int
	writeError    error
}

func (f *correlationFault) Read([]byte) (int, error) {
	f.reads++
	return f.readCount, nil
}

func (f *correlationFault) Write(data []byte) (int, error) {
	f.writes++
	return len(data), f.writeError
}

func TestCorrelatedTransportFailureNeverRetriesErase(t *testing.T) {
	for _, f := range []*correlationFault{{}, {readCount: -1}, {readCount: 4097}, {writeError: io.ErrClosedPipe}} {
		client, err := NewCorrelated(f)
		if err != nil {
			t.Fatal(err)
		}
		request := provision.Request{Operation: provision.OperationErase}
		if _, err = client.Execute(request); err == nil {
			t.Fatal("failure accepted")
		}
		if f.reads > 20 {
			t.Fatal("unbounded reads")
		}
		if _, err = client.Execute(request); !errors.Is(err, ErrConfiguration) || f.writes != 1 {
			t.Fatal("ambiguous erase retried", err, f.writes)
		}
	}
}

func TestCorrelatedInvalidRequestDoesNotWriteOrPoison(t *testing.T) {
	peer := &correlatedPeer{}
	client, err := NewCorrelated(peer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Execute(provision.Request{}); err == nil || peer.writes != 0 {
		t.Fatal("invalid request written", err)
	}
	if _, err = client.Execute(provision.Request{Operation: provision.OperationInspect}); err != nil {
		t.Fatal(err)
	}
}

func TestCorrelatedRemoteErrorIsAcknowledged(t *testing.T) {
	peer := &correlatedPeer{code: provision.Code(1)}
	client, err := NewCorrelated(peer)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Execute(provision.Request{Operation: provision.OperationInspect})
	var remote *RemoteError
	if !errors.As(err, &remote) || remote.Code != peer.code || response.Code != peer.code {
		t.Fatal("lost remote failure", err, response)
	}
	peer.code = provision.CodeOK
	if _, err = client.Execute(provision.Request{Operation: provision.OperationInspect}); err != nil {
		t.Fatal("acknowledged failure poisoned connection", err)
	}
}
