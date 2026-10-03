package screenlink

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamsession"
)

// Push accepts arbitrary fragmentation. write must consume borrowed ACK bytes
// before returning and impose a transport deadline; an ACK error disconnects.
// Framing failure is fatal for this transport: caller must close/reopen it.
func (c *Connection) Push(src []byte, now time.Duration, write func([]byte) error) error {
	if !c.current() {
		return errors.Join(streamsession.ErrLease, c.Disconnect())
	}
	if write == nil {
		return errors.Join(screenwire.ErrRecord, c.Disconnect())
	}
	for len(src) > 0 {
		limit := c.expected
		if limit == 0 {
			limit = screenwire.HeaderSize
		}
		n := copy(c.buffer[c.used:limit], src)
		c.used += n
		src = src[n:]
		if c.used < limit {
			continue
		}
		if c.expected == 0 {
			size, err := screenwire.Size(c.buffer[:screenwire.HeaderSize])
			if err != nil {
				return errors.Join(err, c.Disconnect())
			}
			c.expected = size
			if size > screenwire.HeaderSize {
				continue
			}
		}
		if err := c.process(now, write); err != nil {
			return errors.Join(err, c.Disconnect())
		}
	}
	return nil
}

func (c *Connection) current() bool {
	return !c.device.disabled && c.authentication == c.device.authentication
}

func (c *Connection) process(now time.Duration, write func([]byte) error) error {
	r, err := screenwire.Decode(c.buffer[:c.expected])
	c.used, c.expected = 0, 0
	if err != nil {
		return err
	}
	result, operationErr := c.dispatch(r, now)
	return c.respond(r, result, operationErr, write)
}
