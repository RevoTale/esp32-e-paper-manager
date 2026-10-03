package frameio

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

func TestFramePipeRoundTripAndBounds(t *testing.T) {
	frame, _ := display.NewFrame(display.Size{Width: 9, Height: 2}, 2, []byte{128, 128, 0, 0})
	r, err := Reader(frame)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(encoded))
	if err != nil || !bytes.Equal(got.Bytes(), frame.Bytes()) || got.Size() != frame.Size() {
		t.Fatal(got, err)
	}
	checkMalformed(t, encoded)
}

func checkMalformed(t *testing.T, encoded []byte) {
	t.Helper()
	for i := 0; i < len(encoded); i++ {
		if _, err := Read(bytes.NewReader(encoded[:i])); err == nil {
			t.Fatal("truncation", i)
		}
		if _, err := Decode(encoded[:i]); err == nil {
			t.Fatal("decode truncation", i)
		}
	}
	if _, err := Read(bytes.NewReader(append(bytes.Clone(encoded), 0))); err == nil {
		t.Fatal("trailing byte")
	}
	for _, index := range []int{0, 4, 5, 6, 7, 9} {
		bad := bytes.Clone(encoded)
		bad[index] = 255
		if _, err := Read(bytes.NewReader(bad)); err == nil {
			t.Fatal("invalid", index)
		}
		if _, err := Decode(bad); err == nil {
			t.Fatal("decode invalid", index)
		}
	}
}

func TestDecodeBorrowsWorkerOutput(t *testing.T) {
	data := []byte{'B', 'Z', 'M', '1', 8, 0, 1, 0, 128}
	f, err := Decode(data)
	if err != nil || &f.Bytes()[0] != &data[8] {
		t.Fatal(f, err)
	}
}

func TestInvalidFrames(t *testing.T) {
	for _, size := range []display.Size{{}, {Width: 2049, Height: 1}, {Width: 2048, Height: 2048}} {
		frame, _ := display.NewFrame(size, (size.Width+7)/8, make([]byte, (size.Width+7)/8*size.Height))
		if _, err := Reader(frame); !errors.Is(err, ErrFrame) {
			t.Fatal(size, err)
		}
	}
	for _, pixels := range [][]byte{{0, 1}, {0, 0, 0}} {
		frame, _ := display.NewFrame(display.Size{Width: 9, Height: 1}, len(pixels), pixels)
		if _, err := Reader(frame); !errors.Is(err, ErrFrame) {
			t.Fatal(err)
		}
	}
}
