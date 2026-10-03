package panel

import (
	"reflect"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

func TestThousandByteRecordsPreserveAcceptedPanelWire(t *testing.T) {
	frame := make([]byte, FrameBytes)
	for i := range frame {
		frame[i] = byte(i*29 + 11)
	}
	wantIO, gotIO := &recordingIO{}, &recordingIO{}
	want, _ := New(wantIO.io())
	if err := want.Refresh(frame); err != nil {
		t.Fatal(err)
	}
	driver, _ := New(gotIO.io())
	stream, _ := NewStream(driver)
	chunked, err := streamrx.NewChunkedSink(stream, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err = chunked.Begin(); err != nil {
		t.Fatal(err)
	}
	for pass := uint8(0); pass < 2; pass++ {
		for offset := 0; offset < len(frame); offset += 1000 {
			if err = chunked.Write(pass, uint32(offset), frame[offset:min(offset+1000, len(frame))]); err != nil {
				t.Fatal(err)
			}
		}
	}
	assertNoRefresh(t, gotIO)
	if err = chunked.Commit(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantIO.events, gotIO.events) {
		t.Fatal("larger protocol chunks changed accepted panel wire")
	}
}
