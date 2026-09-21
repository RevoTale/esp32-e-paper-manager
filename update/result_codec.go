package update

import (
	"encoding/binary"
	"hash/crc32"
)

const ResultEncodedSize = 104

var resultMagic = [4]byte{'E', 'P', 'R', '2'}

func EncodeResult(destination []byte, result Result) (int, error) {
	if len(destination) < ResultEncodedSize {
		return 0, ErrBuffer
	}
	if err := result.Validate(); err != nil {
		return 0, err
	}
	wire := destination[:ResultEncodedSize]
	clear(wire)
	copy(wire[:4], resultMagic[:])
	wire[4] = Version
	wire[5] = byte(result.Status)
	wire[6] = byte(result.Diagnostic.Stage)
	wire[7] = byte(result.Diagnostic.Code)
	copy(wire[8:24], result.ID[:])
	copy(wire[24:56], result.ContentHash[:])
	binary.BigEndian.PutUint32(wire[56:60], uint32(result.Diagnostic.Observed))
	binary.BigEndian.PutUint32(wire[60:64], uint32(result.Diagnostic.Maximum))
	binary.BigEndian.PutUint16(wire[64:66], result.Diagnostic.Node)
	wire[66] = byte(len(result.Diagnostic.Limit))
	copy(wire[68:100], result.Diagnostic.Limit)
	binary.BigEndian.PutUint32(wire[100:104], crc32.ChecksumIEEE(wire[:100]))
	return ResultEncodedSize, nil
}

func DecodeResult(wire []byte) (Result, error) {
	if !validResultWire(wire) {
		return Result{}, ErrCodec
	}
	var result Result
	result.Status = Status(wire[5])
	result.Diagnostic.Stage = Stage(wire[6])
	result.Diagnostic.Code = Code(wire[7])
	copy(result.ID[:], wire[8:24])
	copy(result.ContentHash[:], wire[24:56])
	result.Diagnostic.Observed = int(binary.BigEndian.Uint32(wire[56:60]))
	result.Diagnostic.Maximum = int(binary.BigEndian.Uint32(wire[60:64]))
	result.Diagnostic.Node = binary.BigEndian.Uint16(wire[64:66])
	result.Diagnostic.Limit = string(wire[68 : 68+wire[66]])
	if err := result.Validate(); err != nil {
		return Result{}, ErrCodec
	}
	return result, nil
}

func validResultWire(wire []byte) bool {
	if !validResultIdentity(wire) || wire[66] > maxLimitNameLength || wire[67] != 0 {
		return false
	}
	return zeroResultPadding(wire) && crc32.ChecksumIEEE(wire[:100]) == binary.BigEndian.Uint32(wire[100:104])
}

func validResultIdentity(wire []byte) bool {
	return len(wire) == ResultEncodedSize && wire[0] == resultMagic[0] && wire[1] == resultMagic[1] &&
		wire[2] == resultMagic[2] && wire[3] == resultMagic[3] && wire[4] == Version
}

func zeroResultPadding(wire []byte) bool {
	for _, value := range wire[68+int(wire[66]) : 100] {
		if value != 0 {
			return false
		}
	}
	return true
}
