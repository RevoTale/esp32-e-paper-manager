package devicelink

import (
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

type DeviceConfig struct {
	ID  securetransport.DeviceID
	Key securetransport.Key
}

type DeviceSession struct {
	stream     *securetransport.RecordStream
	generation uint64
	wire       [update.MaxEncodedSize]byte
	result     [update.ResultEncodedSize]byte
}

func ConnectDevice(connection io.ReadWriter, config DeviceConfig, random io.Reader) (*DeviceSession, error) {
	if connection == nil || random == nil || config.ID == (securetransport.DeviceID{}) ||
		config.Key == (securetransport.Key{}) {
		return nil, ErrConfiguration
	}
	if err := writeFull(connection, config.ID[:]); err != nil {
		return nil, err
	}
	var nonce securetransport.ClientNonce
	if _, err := io.ReadFull(random, nonce[:]); err != nil {
		return nil, err
	}
	stream, err := securetransport.HostHandshake(connection, config.Key, config.ID, nonce)
	if err != nil {
		return nil, err
	}
	return &DeviceSession{stream: stream}, nil
}

func (s *DeviceSession) Receive() (update.Request, bool, error) {
	if s == nil || s.stream == nil {
		return update.Request{}, false, ErrConfiguration
	}
	header, err := readHeader(s.stream)
	if err != nil {
		return update.Request{}, false, err
	}
	if header.typeID == messageIdle && header.length == 0 {
		return update.Request{}, false, nil
	}
	if header.typeID != messageUpdate || header.length < update.HeaderSize+update.TrailerSize {
		return update.Request{}, false, ErrProtocol
	}
	if _, err = io.ReadFull(s.stream, s.wire[:header.length]); err != nil {
		return update.Request{}, false, err
	}
	request, err := update.Decode(s.wire[:header.length])
	if err != nil {
		return update.Request{}, false, err
	}
	s.generation = header.generation
	return request, true, nil
}

// Report sends retained state from a previous exchange before the manager
// decides whether a request needs delivery. A nil result means a clean boot
// with no retained request state.
func (s *DeviceSession) Report(result *update.Result, generation uint64) error {
	if s == nil || s.stream == nil {
		return ErrConfiguration
	}
	if result == nil {
		return writeMessage(s.stream, messageIdle, 0, nil)
	}
	if generation == 0 || result.Validate() != nil {
		return ErrConfiguration
	}
	size, err := update.EncodeResult(s.result[:], *result)
	if err != nil {
		return err
	}
	return writeMessage(s.stream, messageResult, generation, s.result[:size])
}

func (s *DeviceSession) Generation() uint64 {
	if s == nil {
		return 0
	}
	return s.generation
}

func (s *DeviceSession) SendResult(result update.Result) error {
	if s == nil || s.stream == nil || s.generation == 0 || result.Validate() != nil {
		return ErrConfiguration
	}
	size, err := update.EncodeResult(s.result[:], result)
	if err != nil {
		return err
	}
	return writeMessage(s.stream, messageResult, s.generation, s.result[:size])
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
