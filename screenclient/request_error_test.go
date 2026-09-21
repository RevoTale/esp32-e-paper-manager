package screenclient

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func TestExchangeFailureRetainsOnlyPublicRequestBoundary(t *testing.T) {
	c, _, rw, _, _ := fixture(t)
	rw.before, rw.dropKind = true, screenwire.Data
	request := screenwire.Record{Kind: screenwire.Data, Epoch: 1234567, ID: 9876543,
		Pass: 1, Offset: 480, Payload: []byte("private-pixels")}
	_, err := c.exchange(request)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	for _, want := range []string{"operation=5", "pass=1", "offset=480"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %s in %v", want, err)
		}
	}
	for _, secret := range []string{"1234567", "9876543", "private-pixels"} {
		if strings.Contains(err.Error(), secret) {
			t.Error("request identity or content exposed")
		}
	}
}
