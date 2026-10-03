package screenclient

import (
	"fmt"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

func (c *Client) exchange(request screenwire.Record) (screenwire.Response, error) {
	if c.stream == nil {
		return screenwire.Response{}, ErrConnection
	}
	r, err := c.exchangeRecord(request)
	if err != nil {
		c.stream = nil
		c.bound = false
		// Request coordinates identify an unconfirmed boundary, not an ACK.
		// Do not retain or format identity, lease, digest, or payload here.
		err = fmt.Errorf("screen exchange: operation=%d pass=%d offset=%d: %w",
			request.Kind, request.Pass, request.Offset, err)
	}
	return r, err
}

func (c *Client) exchangeRecord(request screenwire.Record) (screenwire.Response, error) {
	n, err := screenwire.Encode(c.buffer[:], request)
	if err != nil {
		return screenwire.Response{}, err
	}
	if err = writeAll(c.stream, c.buffer[:n]); err != nil {
		return screenwire.Response{}, err
	}
	if _, err = io.ReadFull(progressReader{c.stream}, c.buffer[:screenwire.HeaderSize]); err != nil {
		return screenwire.Response{}, err
	}
	n, err = screenwire.Size(c.buffer[:screenwire.HeaderSize])
	if err != nil {
		return screenwire.Response{}, err
	}
	if _, err = io.ReadFull(progressReader{c.stream}, c.buffer[screenwire.HeaderSize:n]); err != nil {
		return screenwire.Response{}, err
	}
	r, err := screenwire.Decode(c.buffer[:n])
	if err != nil {
		return screenwire.Response{}, err
	}
	return screenwire.ParseReply(r, request)
}

type progressReader struct{ io.Reader }

func (r progressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n == 0 && err == nil && len(p) > 0 {
		return 0, io.ErrNoProgress
	}
	return n, err
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(p) {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
