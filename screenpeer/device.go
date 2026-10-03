// Package screenpeer authenticates EPN2 connections before EPS2 ownership.
// TCP connection direction does not change the cryptographic device/host roles.
package screenpeer

import (
	"io"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/network"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

const HandshakeTimeout = 15 * time.Second

// Socket deadlines must interrupt pending I/O. Its owner must close on every
// subsequent record error and set per-exchange deadlines after authentication.
// A RecordStream is half-duplex: concurrent Read/Write share bounded scratch.
type Socket interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
}

// Device consumes a durable-epoch session number BEFORE any bytes are written.
// The manager supplies the random nonce; Pico needs uniqueness, not a weak RNG.
// https://www.rfc-editor.org/rfc/rfc5116.html#section-3.1
func Device(socket Socket, key securetransport.Key, id securetransport.DeviceID,
	lifetime *network.Lifetime, now time.Time,
) (stream *securetransport.RecordStream, err error) {
	if socket == nil {
		return nil, securetransport.ErrConfig
	}
	defer func() { finish(socket, &stream, &err) }()
	if key == (securetransport.Key{}) || id == (securetransport.DeviceID{}) {
		return nil, securetransport.ErrConfig
	}
	epoch, session, err := lifetime.NextSession()
	if err != nil {
		return nil, err
	}
	if err = socket.SetDeadline(now.Add(HandshakeTimeout)); err != nil {
		return nil, err
	}
	var preface [20]byte
	copy(preface[:4], "EPN2")
	copy(preface[4:], id[:])
	bounded := progress{socket}
	if err = writeFull(bounded, preface[:]); err != nil {
		return nil, err
	}
	return securetransport.ServerHandshake(bounded, key, id, epoch, session)
}

func finish(socket Socket, stream **securetransport.RecordStream, err *error) {
	if *err == nil {
		*err = socket.SetDeadline(time.Time{})
	}
	if *err != nil {
		*stream = nil
		_ = socket.Close()
	}
}

// Reject broken adapters instead of letting io.ReadFull spin on (0, nil).
type progress struct{ io.ReadWriter }

func (p progress) Read(dst []byte) (int, error) {
	n, err := p.ReadWriter.Read(dst)
	if n < 0 || n > len(dst) || (n == 0 && err == nil && len(dst) != 0) {
		return 0, io.ErrNoProgress
	}
	return n, err
}

func writeFull(writer io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := writer.Write(p)
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
