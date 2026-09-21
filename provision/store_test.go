package provision

import (
	"errors"
	"io"
	"testing"
)

const (
	testErase = 4096
	testWrite = 256
)

type fakeFlash struct {
	data       []byte
	torn       int
	writeError error
	eraseError error
	readError  error
	corrupt    bool
}

func newFakeFlash() *fakeFlash {
	data := make([]byte, 2*testErase)
	for index := range data {
		data[index] = 0xff
	}
	return &fakeFlash{data: data}
}

func (f *fakeFlash) ReadAt(destination []byte, offset int64) (int, error) {
	if f.readError != nil {
		return 0, f.readError
	}
	if offset < 0 || int(offset)+len(destination) > len(f.data) {
		return 0, io.EOF
	}
	return copy(destination, f.data[int(offset):]), nil
}

func (f *fakeFlash) WriteAt(source []byte, offset int64) (int, error) {
	if f.writeError != nil {
		return 0, f.writeError
	}
	count := len(source)
	if f.torn > 0 && f.torn < count {
		count, f.torn = f.torn, 0
	}
	for index := 0; index < count; index++ {
		f.data[int(offset)+index] &= source[index]
	}
	if f.corrupt && count > 0 {
		f.data[int(offset)] ^= 1
	}
	if count != len(source) {
		return count, io.ErrUnexpectedEOF
	}
	return count, nil
}

func (f *fakeFlash) Size() int64           { return int64(len(f.data)) }
func (f *fakeFlash) WriteBlockSize() int64 { return testWrite }
func (f *fakeFlash) EraseBlockSize() int64 { return testErase }
func (f *fakeFlash) EraseBlocks(start, length int64) error {
	if f.eraseError != nil {
		return f.eraseError
	}
	for index := int(start * testErase); index < int((start+length)*testErase); index++ {
		f.data[index] = 0xff
	}
	return nil
}

func TestStoreAtomicRotationAndFactoryReset(t *testing.T) {
	flash := newFakeFlash()
	store, err := NewStore(flash, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.Load(); !errors.Is(err, ErrBlank) {
		t.Fatalf("Load(blank) error=%v", err)
	}
	first := validConfig()
	if generation, err := store.Save(first); err != nil || generation != 1 {
		t.Fatalf("Save(first) generation=%d error=%v", generation, err)
	}
	second := first
	second.SSID = "rotated-wpa3"
	if generation, err := store.Save(second); err != nil || generation != 2 {
		t.Fatalf("Save(second) generation=%d error=%v", generation, err)
	}
	loaded, generation, err := store.Load()
	if err != nil || loaded != second || generation != 2 {
		t.Fatalf("Load() config=%#v generation=%d error=%v", loaded, generation, err)
	}
}

func TestFactoryResetReturnsStoreToBlank(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	_, _ = store.Save(validConfig())
	if err := store.FactoryReset(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(); !errors.Is(err, ErrBlank) {
		t.Fatalf("Load(reset) error=%v", err)
	}
}

func TestTornRotationPreservesPreviousGeneration(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	first := validConfig()
	_, _ = store.Save(first)
	flash.torn = 100
	second := first
	second.SSID = "new-network"
	if _, err := store.Save(second); err == nil {
		t.Fatal("Save(torn) succeeded")
	}
	loaded, generation, err := store.Load()
	if err != nil || loaded != first || generation != 1 {
		t.Fatalf("Load() config=%#v generation=%d error=%v", loaded, generation, err)
	}
}

func TestStoreCorruptionAndStorageFailures(t *testing.T) {
	if _, err := NewStore(nil, 0); !errors.Is(err, ErrStorage) {
		t.Fatalf("NewStore(nil) error=%v", err)
	}
	flash := newFakeFlash()
	if _, err := NewStore(flash, 1); !errors.Is(err, ErrStorage) {
		t.Fatalf("NewStore(alignment) error=%v", err)
	}
	store, _ := NewStore(flash, 0)
	flash.data[0], flash.data[testErase] = 0, 0
	if _, _, err := store.Load(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Load(corrupt) error=%v", err)
	}
	flash.eraseError = errors.New("erase")
	if _, err := store.Save(validConfig()); err == nil {
		t.Fatal("Save(erase failure) succeeded")
	}
}

func TestStoreReadAndVerifyFailures(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	flash.readError = errors.New("read")
	if _, _, err := store.Load(); err == nil {
		t.Fatal("read failure accepted")
	}
	flash.readError = nil
	flash.corrupt = true
	if _, err := store.Save(validConfig()); err == nil {
		t.Fatal("corrupt write verified")
	}
}

func TestGenerationOverflowIsRejected(t *testing.T) {
	flash := newFakeFlash()
	store, _ := NewStore(flash, 0)
	record := make([]byte, RecordSize)
	encodeRecord(record, validConfig(), ^uint64(0))
	copy(flash.data, record)
	if _, err := store.Save(validConfig()); !errors.Is(err, ErrStorage) {
		t.Fatalf("overflow=%v", err)
	}
}
