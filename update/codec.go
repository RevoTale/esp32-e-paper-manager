package update

import (
	"encoding/binary"
	"errors"
	"hash/crc32"

	"github.com/RevoTale/esp32-e-paper-manager/document"
)

const (
	HeaderSize     = 160
	TrailerSize    = 4
	MaxEncodedSize = HeaderSize + document.MaxEncodedBytes + TrailerSize
)

var (
	ErrBuffer = errors.New("update: output buffer too small")
	ErrCodec  = errors.New("update: invalid encoded request")
)

var requestMagic = [4]byte{'E', 'P', 'U', '2'}

func EncodedSize(request Request) int {
	return HeaderSize + len(request.document.HTML()) + TrailerSize
}

func Encode(destination []byte, request Request) (int, error) {
	if err := request.Validate(); err != nil {
		return 0, err
	}
	size := EncodedSize(request)
	if len(destination) < size {
		return 0, ErrBuffer
	}
	header := destination[:HeaderSize]
	clear(header)
	copy(header[:4], requestMagic[:])
	header[4] = Version
	header[5] = byte(request.document.Profile())
	header[6] = byte(request.document.Version())
	binary.BigEndian.PutUint32(header[8:12], uint32(len(request.document.HTML())))
	binary.BigEndian.PutUint64(header[12:20], uint64(request.utc))
	header[20] = DisplayTimeLength
	header[21] = request.timezoneLen
	copy(header[24:40], request.id[:])
	copy(header[40:72], request.contentHash[:])
	copy(header[72:92], request.displayTime[:])
	copy(header[92:156], request.timezone[:])
	binary.BigEndian.PutUint32(header[156:160], crc32.ChecksumIEEE(header[:156]))
	payloadEnd := HeaderSize + len(request.document.HTML())
	copy(destination[HeaderSize:payloadEnd], request.document.HTML())
	binary.BigEndian.PutUint32(destination[payloadEnd:size], crc32.ChecksumIEEE(destination[:payloadEnd]))
	return size, nil
}

func Decode(wire []byte) (Request, error) {
	payloadEnd, ok := encodedBounds(wire)
	if !ok {
		return Request{}, ErrCodec
	}
	header := wire[:HeaderSize]
	if !validHeader(header) || !validTrailer(wire, payloadEnd) {
		return Request{}, ErrCodec
	}
	source, err := document.NewSource(document.Version(header[6]), document.Profile(header[5]), wire[HeaderSize:payloadEnd])
	if err != nil {
		return Request{}, ErrCodec
	}
	return decodeRequest(header, source)
}

// EncodedSizeFromHeader returns the complete record size after the fixed
// header is available. Integrity and semantic validation still happen in Decode.
func EncodedSizeFromHeader(header []byte) (int, error) {
	if len(header) < HeaderSize || !validHeaderIdentity(header[:HeaderSize]) {
		return 0, ErrCodec
	}
	payloadLength := int(binary.BigEndian.Uint32(header[8:12]))
	if payloadLength <= 0 || payloadLength > document.MaxEncodedBytes {
		return 0, ErrCodec
	}
	return HeaderSize + payloadLength + TrailerSize, nil
}

func encodedBounds(wire []byte) (int, bool) {
	if len(wire) < HeaderSize+TrailerSize || len(wire) > MaxEncodedSize {
		return 0, false
	}
	payloadLength := int(binary.BigEndian.Uint32(wire[8:12]))
	payloadEnd := HeaderSize + payloadLength
	valid := payloadLength > 0 && payloadLength <= document.MaxEncodedBytes &&
		len(wire) == payloadEnd+TrailerSize
	return payloadEnd, valid
}

func validTrailer(wire []byte, payloadEnd int) bool {
	want := binary.BigEndian.Uint32(wire[payloadEnd : payloadEnd+TrailerSize])
	return crc32.ChecksumIEEE(wire[:payloadEnd]) == want
}

func decodeRequest(header []byte, source document.Source) (Request, error) {
	var id ID
	copy(id[:], header[24:40])
	displayTime := string(header[72 : 72+header[20]])
	timezone := string(header[92 : 92+header[21]])
	request, err := NewRequest(id, int64(binary.BigEndian.Uint64(header[12:20])), displayTime, timezone, source)
	if err != nil {
		return Request{}, ErrCodec
	}
	if !equalDigest(request.contentHash, header[40:72]) {
		return Request{}, ErrCodec
	}
	return request, nil
}

func validHeader(header []byte) bool {
	if len(header) != HeaderSize || !validHeaderIdentity(header) || !validHeaderLengths(header) {
		return false
	}
	return allZero(header[22:24]) && allZero(header[88:92]) &&
		allZero(header[92+int(header[21]):156]) &&
		crc32.ChecksumIEEE(header[:156]) == binary.BigEndian.Uint32(header[156:160])
}

func validHeaderIdentity(header []byte) bool {
	return header[0] == requestMagic[0] && header[1] == requestMagic[1] &&
		header[2] == requestMagic[2] && header[3] == requestMagic[3] &&
		header[4] == Version && header[7] == 0
}

func validHeaderLengths(header []byte) bool {
	return header[20] == DisplayTimeLength && header[21] > 0 && header[21] <= MaxTimezoneLength
}

func equalDigest(digest Digest, wire []byte) bool {
	var difference byte
	for index := range digest {
		difference |= digest[index] ^ wire[index]
	}
	return difference == 0
}

func allZero(wire []byte) bool {
	var combined byte
	for _, value := range wire {
		combined |= value
	}
	return combined == 0
}
