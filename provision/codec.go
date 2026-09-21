package provision

import (
	"encoding/binary"
	"hash/crc32"
)

const (
	RequestSize  = 512
	ResponseSize = 512
)

type Operation uint8

const (
	OperationInspect Operation = iota + 1
	OperationProvision
	OperationRotate
	OperationErase
	OperationDiagnose
)

type State uint8

const (
	StateBlank State = iota
	StateProvisioned
	StateCorrupt
	StateUnknown // The journal could not be read; never infer Blank after I/O failure.
)

type Request struct {
	Operation Operation
	Config    Config
}

type Response struct {
	Operation  Operation
	Code       Code
	State      State
	Generation uint64
	Auth       AuthMode
	DeviceID   [16]byte
	SSID       string
	Manager    string
	Timezone   string
}

func (codec Codec) EncodeRequest(destination []byte, request Request) error {
	if len(destination) < RequestSize || !validOperation(request.Operation) ||
		(needsConfig(request.Operation) && request.Config.ValidateFor(codec) != nil) {
		return ErrInvalidConfig
	}
	fill(destination[:RequestSize], 0)
	copy(destination[:4], "EPCQ")
	destination[4], destination[5] = 2, byte(request.Operation)
	if needsConfig(request.Operation) {
		encodeConfig(destination[:508], request.Config)
	}
	binary.BigEndian.PutUint32(destination[508:512], crc32.ChecksumIEEE(destination[:508]))
	return nil
}

func (codec Codec) DecodeRequest(source []byte) (Request, error) {
	if !validWire(source, "EPCQ") {
		return Request{}, ErrInvalidConfig
	}
	operation := Operation(source[5])
	if !validOperation(operation) {
		return Request{}, ErrInvalidConfig
	}
	request := Request{Operation: operation}
	if needsConfig(operation) {
		config, ok := codec.decodeConfig(source[:508])
		if !ok {
			return Request{}, ErrInvalidConfig
		}
		request.Config = config
	}
	if !requestPadding(source, request) {
		return Request{}, ErrInvalidConfig
	}
	return request, nil
}

func (codec Codec) EncodeResponse(destination []byte, response Response) error {
	if len(destination) < ResponseSize || !codec.validResponse(response) {
		return ErrInvalidConfig
	}
	fill(destination[:ResponseSize], 0)
	copy(destination[:4], "EPCR")
	destination[4], destination[5], destination[6] = 2, byte(response.Operation), byte(response.State)
	destination[7], destination[8] = byte(response.Auth), byte(len(response.SSID))
	destination[9] = byte(len(response.Timezone))
	binary.BigEndian.PutUint16(destination[10:12], uint16(len(response.Manager)))
	destination[12] = byte(response.Code)
	binary.BigEndian.PutUint64(destination[16:24], response.Generation)
	copy(destination[24:40], response.DeviceID[:])
	copy(destination[40:72], response.SSID)
	copy(destination[72:327], response.Manager)
	copy(destination[327:391], response.Timezone)
	binary.BigEndian.PutUint32(destination[508:512], crc32.ChecksumIEEE(destination[:508]))
	return nil
}

func (codec Codec) DecodeResponse(source []byte) (Response, error) {
	if !validWire(source, "EPCR") {
		return Response{}, ErrInvalidConfig
	}
	ssidLength, timezoneLength := int(source[8]), int(source[9])
	managerLength := int(binary.BigEndian.Uint16(source[10:12]))
	if ssidLength > MaxSSID || managerLength > MaxManager || timezoneLength > MaxTimezone {
		return Response{}, ErrInvalidConfig
	}
	response := Response{Operation: Operation(source[5]), Code: Code(source[12]), State: State(source[6]), Auth: AuthMode(source[7]),
		Generation: binary.BigEndian.Uint64(source[16:24]), SSID: string(source[40 : 40+ssidLength]),
		Manager: string(source[72 : 72+managerLength]), Timezone: string(source[327 : 327+timezoneLength])}
	copy(response.DeviceID[:], source[24:40])
	if !codec.validResponse(response) || !responsePadding(source, response) {
		return Response{}, ErrInvalidConfig
	}
	return response, nil
}

func validWire(source []byte, magic string) bool {
	return len(source) == RequestSize && string(source[:4]) == magic && source[4] == 2 &&
		crc32.ChecksumIEEE(source[:508]) == binary.BigEndian.Uint32(source[508:512])
}

func validOperation(operation Operation) bool {
	return operation >= OperationInspect && operation <= OperationDiagnose
}

func needsConfig(operation Operation) bool {
	return operation == OperationProvision || operation == OperationRotate
}

func fill(value []byte, item byte) {
	for index := range value {
		value[index] = item
	}
}
