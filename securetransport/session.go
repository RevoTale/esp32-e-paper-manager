package securetransport

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
)

const (
	ChallengeSize = 56
	AuthSize      = 72
	HeaderSize    = 16
	TagSize       = 16
	// One bounded EPS2 header + 1024-byte payload. Independent of the legacy
	// buffered-display protocol; no silent fragmentation inside RecordStream.
	MaxPlaintext = 1056
	MaxEnvelope  = HeaderSize + MaxPlaintext + TagSize
)

var (
	ErrAuthentication = errors.New("secure transport: authentication failed")
	ErrBounds         = errors.New("secure transport: invalid bounds")
	ErrConfig         = errors.New("secure transport: invalid configuration")
	ErrSequence       = errors.New("secure transport: invalid sequence")
)

type Key [32]byte
type DeviceID [16]byte
type ClientNonce [32]byte
type Challenge [ChallengeSize]byte
type Auth [AuthSize]byte

type Session struct {
	send       cipher.AEAD
	receive    cipher.AEAD
	sendPrefix [4]byte
	recvPrefix [4]byte
	sendSeq    uint64
	recvSeq    uint64
}

func NewChallenge(key Key, device DeviceID, epoch, session uint64) (Challenge, error) {
	var challenge Challenge
	if zero(key[:]) || zero(device[:]) || epoch == 0 || session == 0 {
		return challenge, ErrConfig
	}
	copy(challenge[0:4], "EPWA")
	challenge[4] = 1
	binary.BigEndian.PutUint64(challenge[8:16], epoch)
	binary.BigEndian.PutUint64(challenge[16:24], session)
	proof := mac(key[:], []byte("epaper/server/v1"), device[:], challenge[:24])
	copy(challenge[24:], proof[:])
	return challenge, nil
}

func NewHostSession(key Key, device DeviceID, wire []byte, nonce ClientNonce) (Auth, *Session, error) {
	var auth Auth
	if zero(key[:]) || zero(device[:]) || zero(nonce[:]) || !verifyChallenge(key, device, wire) {
		return auth, nil, ErrAuthentication
	}
	copy(auth[0:4], "EPWP")
	auth[4] = 1
	copy(auth[8:40], nonce[:])
	proof := mac(key[:], []byte("epaper/client/v1"), device[:], wire, nonce[:])
	copy(auth[40:], proof[:])
	session, err := deriveSession(key, device, wire, nonce, true)
	if err != nil {
		return Auth{}, nil, err
	}
	return auth, session, nil
}

func NewDeviceSession(key Key, device DeviceID, challengeWire, authWire []byte) (*Session, error) {
	if zero(key[:]) || zero(device[:]) || !verifyChallenge(key, device, challengeWire) ||
		len(authWire) != AuthSize || string(authWire[0:4]) != "EPWP" || authWire[4] != 1 ||
		!zero(authWire[5:8]) || zero(authWire[8:40]) {
		return nil, ErrAuthentication
	}
	want := mac(key[:], []byte("epaper/client/v1"), device[:], challengeWire, authWire[8:40])
	if !hmac.Equal(authWire[40:], want[:]) {
		return nil, ErrAuthentication
	}
	var nonce ClientNonce
	copy(nonce[:], authWire[8:40])
	return deriveSession(key, device, challengeWire, nonce, false)
}

func (s *Session) Seal(dst, plaintext []byte) (int, error) {
	if len(plaintext) > MaxPlaintext || len(dst) < HeaderSize+len(plaintext)+TagSize {
		return 0, ErrBounds
	}
	if s == nil || s.send == nil || s.sendSeq == math.MaxUint64 {
		return 0, ErrSequence
	}
	header := dst[:HeaderSize]
	clear(header)
	binary.BigEndian.PutUint64(header[0:8], s.sendSeq)
	binary.BigEndian.PutUint16(header[8:10], uint16(len(plaintext)))
	nonce := envelopeNonce(s.sendPrefix, s.sendSeq)
	sealed := s.send.Seal(dst[HeaderSize:HeaderSize], nonce[:], plaintext, header)
	s.sendSeq++
	return HeaderSize + len(sealed), nil
}

func (s *Session) Open(dst, envelope []byte) (int, error) {
	if s == nil || s.receive == nil || len(envelope) < HeaderSize+TagSize || len(envelope) > MaxEnvelope {
		return 0, ErrBounds
	}
	header := envelope[:HeaderSize]
	sequence := binary.BigEndian.Uint64(header[0:8])
	if sequence != s.recvSeq || sequence == math.MaxUint64 {
		return 0, ErrSequence
	}
	if !validEnvelopePayload(len(dst), envelope) {
		return 0, ErrBounds
	}
	nonce := envelopeNonce(s.recvPrefix, sequence)
	var scratch [MaxPlaintext]byte
	plain, err := s.receive.Open(scratch[:0], nonce[:], envelope[HeaderSize:], header)
	if err != nil {
		return 0, ErrAuthentication
	}
	copy(dst, plain)
	s.recvSeq++
	return len(plain), nil
}

// The caller has already checked envelope header/tag size and sequence.
func validEnvelopePayload(destinationSize int, envelope []byte) bool {
	length := int(binary.BigEndian.Uint16(envelope[8:10]))
	return length <= MaxPlaintext && destinationSize >= length && zero(envelope[10:16]) && len(envelope) == HeaderSize+length+TagSize
}

func verifyChallenge(key Key, device DeviceID, wire []byte) bool {
	if len(wire) != ChallengeSize || string(wire[0:4]) != "EPWA" || wire[4] != 1 ||
		!zero(wire[5:8]) || binary.BigEndian.Uint64(wire[8:16]) == 0 ||
		binary.BigEndian.Uint64(wire[16:24]) == 0 {
		return false
	}
	want := mac(key[:], []byte("epaper/server/v1"), device[:], wire[:24])
	return hmac.Equal(wire[24:], want[:])
}

func deriveSession(key Key, device DeviceID, challenge []byte, nonce ClientNonce, host bool) (*Session, error) {
	master := mac(key[:], []byte("epaper/session/v1"), device[:], challenge, nonce[:])
	hostKey := mac(master[:], []byte("epaper/host-to-device/v1"))
	deviceKey := mac(master[:], []byte("epaper/device-to-host/v1"))
	hostAEAD, err := newAEAD(hostKey[:])
	if err != nil {
		return nil, err
	}
	deviceAEAD, err := newAEAD(deviceKey[:])
	if err != nil {
		return nil, err
	}
	session := &Session{}
	if host {
		session.send, session.receive = hostAEAD, deviceAEAD
		copy(session.sendPrefix[:], "H2D1")
		copy(session.recvPrefix[:], "D2H1")
	} else {
		session.send, session.receive = deviceAEAD, hostAEAD
		copy(session.sendPrefix[:], "D2H1")
		copy(session.recvPrefix[:], "H2D1")
	}
	return session, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func envelopeNonce(prefix [4]byte, sequence uint64) [12]byte {
	var nonce [12]byte
	copy(nonce[:4], prefix[:])
	binary.BigEndian.PutUint64(nonce[4:], sequence)
	return nonce
}

func mac(key []byte, parts ...[]byte) [32]byte {
	hash := hmac.New(sha256.New, key)
	for _, part := range parts {
		_, _ = hash.Write(part)
	}
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func zero(value []byte) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}
