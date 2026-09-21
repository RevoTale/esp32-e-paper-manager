package provision

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

const RecordSize = 512

var (
	ErrBlank   = errors.New("provision: blank")
	ErrCorrupt = errors.New("provision: corrupt")
	ErrStorage = errors.New("provision: invalid storage")
)

type BlockDevice interface {
	io.ReaderAt
	io.WriterAt
	Size() int64
	WriteBlockSize() int64
	EraseBlockSize() int64
	EraseBlocks(start, length int64) error
}

type Store struct {
	device BlockDevice
	start  int64
	erase  int64
	page   [RecordSize]byte
}

func NewStore(device BlockDevice, start int64) (*Store, error) {
	if device == nil {
		return nil, ErrStorage
	}
	erase, write := device.EraseBlockSize(), device.WriteBlockSize()
	if start < 0 || erase < RecordSize || write <= 0 || RecordSize%write != 0 ||
		start%erase != 0 || start+2*erase > device.Size() {
		return nil, ErrStorage
	}
	return &Store{device: device, start: start, erase: erase}, nil
}

func (s *Store) Load() (Config, uint64, error) {
	left, leftGeneration, leftState, err := s.read(0)
	if err != nil {
		return Config{}, 0, err
	}
	right, rightGeneration, rightState, err := s.read(1)
	if err != nil {
		return Config{}, 0, err
	}
	if leftState == slotValid && (rightState != slotValid || leftGeneration >= rightGeneration) {
		return left, leftGeneration, nil
	}
	if rightState == slotValid {
		return right, rightGeneration, nil
	}
	if leftState == slotBlank && rightState == slotBlank {
		return Config{}, 0, ErrBlank
	}
	return Config{}, 0, ErrCorrupt
}

func (s *Store) Save(config Config) (uint64, error) {
	if err := config.Validate(); err != nil {
		return 0, err
	}
	next, err := s.nextGeneration()
	if err != nil {
		return 0, err
	}
	target := int(next & 1)
	if err := s.device.EraseBlocks((s.start+int64(target)*s.erase)/s.erase, 1); err != nil {
		return 0, err
	}
	return s.writeAndVerify(config, next, target)
}

func (s *Store) nextGeneration() (uint64, error) {
	_, generation, err := s.Load()
	if err != nil && !errors.Is(err, ErrBlank) && !errors.Is(err, ErrCorrupt) {
		return 0, err
	}
	if generation == ^uint64(0) {
		return 0, ErrStorage
	}
	return generation + 1, nil
}

func (s *Store) writeAndVerify(config Config, next uint64, target int) (uint64, error) {
	encodeRecord(s.page[:], config, next)
	offset := s.start + int64(target)*s.erase
	if count, err := s.device.WriteAt(s.page[:], offset); err != nil || count != RecordSize {
		if err == nil {
			err = io.ErrShortWrite
		}
		return 0, err
	}
	verified, verifiedGeneration, state, err := s.read(target)
	if err != nil || state != slotValid || verifiedGeneration != next || verified != config {
		return 0, ErrCorrupt
	}
	return next, nil
}

func (s *Store) FactoryReset() error {
	return s.device.EraseBlocks(s.start/s.erase, 2)
}

type slotState uint8

const (
	slotBlank slotState = iota
	slotValid
	slotInvalid
)

func (s *Store) read(slot int) (Config, uint64, slotState, error) {
	offset := s.start + int64(slot)*s.erase
	if _, err := s.device.ReadAt(s.page[:], offset); err != nil {
		return Config{}, 0, slotInvalid, err
	}
	if erased(s.page[:]) {
		return Config{}, 0, slotBlank, nil
	}
	config, generation, ok := decodeRecord(s.page[:])
	if !ok {
		return Config{}, 0, slotInvalid, nil
	}
	return config, generation, slotValid, nil
}

func encodeRecord(destination []byte, config Config, generation uint64) {
	for index := range destination {
		destination[index] = 0xff
	}
	copy(destination[:4], "EPC2")
	destination[4] = 1
	binary.BigEndian.PutUint64(destination[8:16], generation)
	encodeConfig(destination, config)
	for index := 486; index < 508; index++ {
		destination[index] = 0
	}
	binary.BigEndian.PutUint32(destination[508:512], crc32.ChecksumIEEE(destination[:508]))
}

func decodeRecord(source []byte) (Config, uint64, bool) {
	if len(source) != RecordSize || string(source[:4]) != "EPC2" || source[4] != 1 ||
		crc32.ChecksumIEEE(source[:508]) != binary.BigEndian.Uint32(source[508:512]) {
		return Config{}, 0, false
	}
	config, valid := decodeConfig(source)
	generation := binary.BigEndian.Uint64(source[8:16])
	return config, generation, generation > 0 && valid
}

func erased(source []byte) bool {
	for _, value := range source {
		if value != 0xff {
			return false
		}
	}
	return true
}
