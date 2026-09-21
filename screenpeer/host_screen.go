//go:build !tinygo

package screenpeer

import (
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

// HostScreen adapts complete EPN2 reply envelopes to screenclient's USB-style
// io.ReadWriter without losing message boundaries. One request/one reply only.
type HostScreen struct {
	socket     Socket
	records    *securetransport.RecordStream
	buffer     [securetransport.MaxPlaintext]byte
	start, end int
	failed     bool
}

func NewHostScreen(socket Socket, records *securetransport.RecordStream) (*HostScreen, error) {
	if socket == nil || records == nil {
		return nil, securetransport.ErrConfig
	}
	return &HostScreen{socket: socket, records: records}, nil
}

func (s *HostScreen) Read(dst []byte) (int, error) {
	if s.failed {
		return 0, securetransport.ErrConfig
	}
	if len(dst) == 0 {
		return 0, nil
	}
	if s.start == s.end {
		err := s.readReply()
		if err != nil {
			return 0, s.fail(err)
		}
	}
	n := copy(dst, s.buffer[s.start:s.end])
	s.start += n
	return n, nil
}

func (s *HostScreen) readReply() error {
	// Waiting for physical Commit is longer than receiving its first ciphertext
	// byte. No deadline renewal for slow fragments after that first byte.
	limits := securetransport.ReadLimits{Idle: 180 * time.Second, Active: 20 * time.Second, SetDeadline: s.socket.SetDeadline}
	n, err := s.records.ReadRecord(s.buffer[:], limits)
	if err != nil {
		return err
	}
	reply, err := screenwire.Decode(s.buffer[:n])
	if err != nil || reply.Kind != screenwire.Reply {
		return screenwire.ErrRecord
	}
	s.start, s.end = 0, n
	return nil
}

func (s *HostScreen) Write(p []byte) (int, error) {
	if s.failed {
		return 0, securetransport.ErrConfig
	}
	if s.start != s.end {
		return 0, s.fail(screenwire.ErrRecord)
	}
	r, err := screenwire.Decode(p)
	if err != nil || r.Kind == screenwire.Reply {
		return 0, s.fail(screenwire.ErrRecord)
	}
	if err = s.socket.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, s.fail(err)
	}
	n, err := s.records.Write(p)
	if err != nil {
		return 0, s.fail(err)
	}
	return n, nil
}

func (s *HostScreen) fail(err error) error {
	s.failed = true
	clear(s.buffer[:])
	return errors.Join(err, s.socket.Close())
}
