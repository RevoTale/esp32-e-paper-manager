package securetransport

import (
	"io"
)

type RecordStream struct {
	stream   io.ReadWriter
	session  *Session
	envelope [MaxEnvelope]byte
	plain    [MaxPlaintext]byte
	start    int
	end      int
}

func NewRecordStream(stream io.ReadWriter, session *Session) *RecordStream {
	return &RecordStream{stream: stream, session: session}
}

func HostHandshake(stream io.ReadWriter, key Key, device DeviceID, nonce ClientNonce) (*RecordStream, error) {
	var challenge Challenge
	if _, err := io.ReadFull(stream, challenge[:]); err != nil {
		return nil, err
	}
	auth, session, err := NewHostSession(key, device, challenge[:], nonce)
	if err != nil {
		return nil, err
	}
	if err := writeFull(stream, auth[:]); err != nil {
		return nil, err
	}
	return NewRecordStream(stream, session), nil
}

func ServerHandshake(stream io.ReadWriter, key Key, device DeviceID, epoch, sessionNumber uint64) (*RecordStream, error) {
	challenge, err := NewChallenge(key, device, epoch, sessionNumber)
	if err != nil {
		return nil, err
	}
	if err := writeFull(stream, challenge[:]); err != nil {
		return nil, err
	}
	var auth Auth
	if _, err := io.ReadFull(stream, auth[:]); err != nil {
		return nil, err
	}
	session, err := NewDeviceSession(key, device, challenge[:], auth[:])
	if err != nil {
		return nil, err
	}
	return NewRecordStream(stream, session), nil
}

func (s *RecordStream) Write(record []byte) (int, error) {
	if s == nil || s.stream == nil || s.session == nil || len(record) == 0 || len(record) > MaxPlaintext {
		return 0, ErrBounds
	}
	n, err := s.session.Seal(s.envelope[:], record)
	if err != nil {
		return 0, err
	}
	if err := writeFull(s.stream, s.envelope[:n]); err != nil {
		return 0, err
	}
	return len(record), nil
}

func (s *RecordStream) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if s == nil || s.stream == nil || s.session == nil {
		return 0, ErrConfig
	}
	if s.start == s.end {
		n, err := s.readEnvelope(ReadLimits{})
		if err != nil {
			return 0, err
		}
		s.start, s.end = 0, n
	}
	n := copy(dst, s.plain[s.start:s.end])
	s.start += n
	return n, nil
}

func writeFull(writer io.Writer, value []byte) error {
	for len(value) > 0 {
		n, err := writer.Write(value)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(value) {
			return io.ErrShortWrite
		}
		value = value[n:]
	}
	return nil
}
