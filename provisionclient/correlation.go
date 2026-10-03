package provisionclient

import (
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

var ErrResponseSync = errors.New("provision client: correlated response missing within stream budget")

// NewCorrelated creates an ESP32 v3 client. The transport must bound each Read;
// the serial CLI uses 500ms polls. There is no legacy fallback or mutation retry.
func NewCorrelated(stream io.ReadWriter) (*Client, error) {
	c, err := NewFor(stream, provision.ESP32Codec())
	if err != nil {
		return nil, err
	}
	c.correlated = true
	return c, nil
}

func (c *Client) executeCorrelated(request provision.Request) (provision.Response, error) {
	var id provision.RequestID
	if _, err := io.ReadFull(rand.Reader, id[:]); err != nil {
		return provision.Response{}, err
	}
	if err := c.codec.EncodeCorrelatedRequest(c.tx[:], request, id); err != nil {
		return provision.Response{}, err
	}
	if err := writeFull(c.stream, c.tx[:]); err != nil {
		c.stream = nil
		return provision.Response{}, err
	}
	response, err := c.readCorrelated(id)
	if err != nil || response.Operation != request.Operation {
		c.stream = nil
		if err == nil {
			err = provision.ErrInvalidConfig
		}
		return provision.Response{}, err
	}
	if response.Code != provision.CodeOK {
		return response, &RemoteError{Code: response.Code}
	}
	return response, nil
}

func (c *Client) readCorrelated(id provision.RequestID) (provision.Response, error) {
	var input [4096]byte
	defer clear(input[:])
	used, scanned, empty := 0, 0, 0
	deadline := time.Now().Add(10 * time.Second)
	for used < len(input) && empty < 20 && time.Now().Before(deadline) {
		n, err := c.stream.Read(input[used:])
		if n < 0 || n > len(input)-used {
			return provision.Response{}, io.ErrUnexpectedEOF
		}
		used += n
		for scanned+provision.ResponseSize <= used {
			response, decodeErr := c.codec.DecodeCorrelatedResponse(input[scanned:scanned+provision.ResponseSize], id)
			if decodeErr == nil {
				return response, nil
			}
			scanned++
		}
		if err != nil {
			return provision.Response{}, err
		}
		if n == 0 {
			empty++
		}
	}
	return provision.Response{}, ErrResponseSync
}
