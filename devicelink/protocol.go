// Package devicelink transports update records over authenticated encryption.
package devicelink

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

const headerSize = 20

var (
	ErrConfiguration = errors.New("device link: invalid configuration")
	ErrProtocol      = errors.New("device link: invalid protocol")
)

type messageType uint8

const (
	messageIdle messageType = iota + 1
	messageUpdate
	messageResult
)

type messageHeader struct {
	typeID     messageType
	length     uint32
	generation uint64
}

func writeMessage(stream *securetransport.RecordStream, kind messageType, generation uint64,
	payload []byte,
) error {
	if stream == nil || len(payload) > update.MaxEncodedSize ||
		(kind != messageIdle && len(payload) == 0) || kind < messageIdle || kind > messageResult {
		return ErrConfiguration
	}
	var header [headerSize]byte
	copy(header[:4], "EPD2")
	header[4] = byte(kind)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(payload)))
	binary.BigEndian.PutUint64(header[12:20], generation)
	if err := writeChunked(stream, header[:]); err != nil {
		return err
	}
	return writeChunked(stream, payload)
}

func readHeader(stream io.Reader) (messageHeader, error) {
	var wire [headerSize]byte
	if _, err := io.ReadFull(stream, wire[:]); err != nil {
		return messageHeader{}, err
	}
	header := messageHeader{typeID: messageType(wire[4]), length: binary.BigEndian.Uint32(wire[8:12]),
		generation: binary.BigEndian.Uint64(wire[12:20])}
	if string(wire[:4]) != "EPD2" || header.typeID < messageIdle || header.typeID > messageResult ||
		header.length > update.MaxEncodedSize || !allZero(wire[5:8]) {
		return messageHeader{}, ErrProtocol
	}
	return header, nil
}

func writeChunked(stream io.Writer, payload []byte) error {
	for len(payload) > 0 {
		limit := min(len(payload), securetransport.MaxPlaintext)
		count, err := stream.Write(payload[:limit])
		if err != nil {
			return err
		}
		if count != limit {
			return io.ErrShortWrite
		}
		payload = payload[limit:]
	}
	return nil
}

func allZero(value []byte) bool {
	var combined byte
	for _, item := range value {
		combined |= item
	}
	return combined == 0
}
