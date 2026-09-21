//go:build !tinygo

package screenpeer

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

type Lookup func(securetransport.DeviceID) (securetransport.Key, error)

// Manager selects a key by the public preface, then verifies the HMAC binding
// before sending authentication. A successful return, not the untrusted ID,
// grants permission to open a screen session. No EPS1/legacy downgrade exists.
func Manager(socket Socket, lookup Lookup, now time.Time) (*securetransport.RecordStream, securetransport.DeviceID, error) {
	return accept(socket, lookup, now, rand.Reader)
}

func accept(socket Socket, lookup Lookup, now time.Time, random io.Reader,
) (stream *securetransport.RecordStream, id securetransport.DeviceID, err error) {
	if socket == nil {
		return nil, id, securetransport.ErrConfig
	}
	defer func() {
		finish(socket, &stream, &err)
		if err != nil {
			id = securetransport.DeviceID{}
		}
	}()
	if lookup == nil || random == nil {
		return nil, id, securetransport.ErrConfig
	}
	if err = socket.SetDeadline(now.Add(HandshakeTimeout)); err != nil {
		return nil, id, err
	}
	bounded := progress{socket}
	id, key, err := readIdentity(bounded, lookup)
	if err != nil {
		return nil, id, err
	}
	var nonce securetransport.ClientNonce
	if _, err = io.ReadFull(random, nonce[:]); err != nil {
		return nil, id, err
	}
	stream, err = securetransport.HostHandshake(bounded, key, id, nonce)
	return stream, id, err
}

func readIdentity(reader io.Reader, lookup Lookup) (securetransport.DeviceID, securetransport.Key, error) {
	var id securetransport.DeviceID
	var preface [20]byte
	if _, err := io.ReadFull(reader, preface[:]); err != nil {
		return id, securetransport.Key{}, err
	}
	copy(id[:], preface[4:])
	if string(preface[:4]) != "EPN2" || id == (securetransport.DeviceID{}) {
		return id, securetransport.Key{}, securetransport.ErrAuthentication
	}
	key, err := lookup(id)
	if err != nil || key == (securetransport.Key{}) {
		return id, securetransport.Key{}, securetransport.ErrAuthentication
	}
	return id, key, nil
}
