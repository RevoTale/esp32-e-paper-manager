package provisionclient

import (
	"errors"
	"fmt"
	"math"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

// SetManager inspects once, then changes only the endpoint if needed. An ambiguous
// acknowledgement is never retried. Reopen USB and inspect before another attempt.
func (c *Client) SetManager(endpoint string) (provision.Response, error) {
	if c == nil || !c.correlated || !provision.ValidESP32Endpoint(endpoint) {
		return provision.Response{}, ErrConfiguration
	}
	before, err := c.Execute(provision.Request{Operation: provision.OperationInspectManager})
	if err != nil {
		return provision.Response{}, fmt.Errorf("ESP32 set-manager preflight: check USB and firmware support: %w", err)
	}
	if before.State != provision.StateProvisioned {
		return before, errors.New("set-manager requires a provisioned ESP32")
	}
	if before.Manager == endpoint {
		return before, nil
	}
	if before.Generation == math.MaxUint64 {
		return before, errors.New("device configuration generation exhausted")
	}
	change := provision.ManagerChange{Endpoint: endpoint, DeviceID: before.DeviceID, Generation: before.Generation}
	after, err := c.Execute(provision.Request{Operation: provision.OperationSetManager, Change: change})
	if err != nil {
		return after, fmt.Errorf("set-manager: reopen USB and inspect the stored endpoint before retrying: %w", err)
	}
	expected := before
	expected.Operation, expected.Manager, expected.Generation = provision.OperationSetManager, endpoint, before.Generation+1
	if after != expected {
		c.stream = nil
		return provision.Response{}, errors.New("set-manager acknowledgement mismatch: reopen USB and inspect")
	}
	return after, nil
}
