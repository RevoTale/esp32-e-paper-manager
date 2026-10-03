// Package provisionclient implements the host side of physical USB provisioning.
package provisionclient

import (
	"errors"
	"io"
	"strconv"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

var ErrConfiguration = errors.New("provision client: invalid configuration")

// RemoteError reports an acknowledged operation failure. Inspect the returned
// response State before deciding what to do; a failed write may have committed.
type RemoteError struct{ Code provision.Code }

func (e *RemoteError) Error() string {
	return "provision client: device rejected operation, code=" + strconv.Itoa(int(e.Code))
}

type Client struct {
	stream     io.ReadWriter
	codec      provision.Codec
	correlated bool
	tx         [provision.RequestSize]byte
	rx         [provision.ResponseSize]byte
}

func New(stream io.ReadWriter) (*Client, error) {
	return NewFor(stream, provision.Codec{})
}

func NewFor(stream io.ReadWriter, codec provision.Codec) (*Client, error) {
	if stream == nil {
		return nil, ErrConfiguration
	}
	return &Client{stream: stream, codec: codec}, nil
}

// Codec retains the Pico default for offline request preparation without a client.
func (c *Client) Codec() provision.Codec {
	if c == nil {
		return provision.Codec{}
	}
	return c.codec
}

func (c *Client) Execute(request provision.Request) (provision.Response, error) {
	if c == nil || c.stream == nil {
		return provision.Response{}, ErrConfiguration
	}
	if c.correlated {
		return c.executeCorrelated(request)
	}
	if err := c.codec.EncodeRequest(c.tx[:], request); err != nil {
		return provision.Response{}, err
	}
	if err := writeFull(c.stream, c.tx[:]); err != nil {
		c.stream = nil
		return provision.Response{}, err
	}
	if err := readFull(c.stream, c.rx[:]); err != nil {
		c.stream = nil
		return provision.Response{}, err
	}
	response, err := c.codec.DecodeResponse(c.rx[:])
	if err != nil || response.Operation != request.Operation {
		c.stream = nil
		return provision.Response{}, provision.ErrInvalidConfig
	}
	if response.Code != provision.CodeOK {
		return response, &RemoteError{Code: response.Code}
	}
	return response, nil
}

func readFull(reader io.Reader, value []byte) error {
	for len(value) > 0 {
		count, err := reader.Read(value)
		if count < 0 || count > len(value) {
			return io.ErrUnexpectedEOF
		}
		value = value[count:]
		if len(value) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func writeFull(writer io.Writer, value []byte) error {
	for len(value) > 0 {
		count, err := writer.Write(value)
		if err != nil {
			return err
		}
		if count <= 0 || count > len(value) {
			return io.ErrShortWrite
		}
		value = value[count:]
	}
	return nil
}
