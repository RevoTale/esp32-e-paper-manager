package screenusbhost

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screenclient"
	"github.com/RevoTale/esp32-e-paper-manager/screenlink"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
	"github.com/RevoTale/esp32-e-paper-manager/waveshare75"
)

// Characterization of software evidence, not a simulation of visible pixels.
// Exercises the direct client/codec/device/driver path, without Sender, proxy,
// DTR, operating-system serial I/O or USB electrical behavior.
// Immediate HIGH is accepted by the existing driver; do not invent a required
// prior LOW edge merely to reproduce an optical failure in a host test.
func TestFreshEPS2ClientsRefreshDifferentFramesOnPersistentPanel(t *testing.T) {
	for _, alwaysHigh := range []bool{false, true} {
		name := "busy_low_then_high"
		if alwaysHigh {
			name = "busy_permanently_high"
		}
		t.Run(name, func(t *testing.T) { characterizeRepeatedPanel(t, alwaysHigh) })
	}
}

func characterizeRepeatedPanel(t *testing.T, alwaysHigh bool) {
	t.Helper()
	recorder := &repeatPanelIO{alwaysHigh: alwaysHigh}
	device, err := waveshare75.New(recorder.io(), [16]byte{7})
	if err != nil {
		t.Fatal(err)
	}
	frames := [2]display.Frame{repeatPanelFrame(t, 0), repeatPanelFrame(t, 0xff)}
	if bytes.Equal(frames[0].Bytes(), frames[1].Bytes()) {
		t.Fatal("repetition fixture must change every pixel byte")
	}
	for index, frame := range frames {
		sendRepeatPanelFrame(t, device, recorder, index, frame, frames[0])
	}
	if recorder.alwaysHigh && recorder.lowReads != 0 || !recorder.alwaysHigh && recorder.lowReads == 0 {
		t.Fatalf("BUSY fixture not exercised: high-only=%t low reads=%d", recorder.alwaysHigh, recorder.lowReads)
	}
}

func sendRepeatPanelFrame(t *testing.T, device *screenlink.Device, recorder *repeatPanelIO,
	index int, frame, first display.Frame) {
	t.Helper()
	port := &repeatPanelLink{connection: device.Open(), now: &recorder.now}
	client, err := screenclient.NewWithMaxChunk(bytes.NewReader(bytes.Repeat([]byte{byte(index + 1)}, 16)), 480)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := client.Connect(port)
	if err != nil || caps.MinimumFullMS != 180000 {
		t.Fatalf("connection %d: capabilities=%+v error=%v", index, caps, err)
	}
	assertRepeatPanelProof(t, port, uint64(index+1), first, screenwire.CodeOK, streamrx.Idle, false)
	recorder.now += time.Duration(caps.MinimumFullMS) * time.Millisecond
	if err = client.Send(frame); err != nil {
		t.Fatalf("connection %d transfer: %v", index, err)
	}
	assertRepeatPanelProof(t, port, uint64(index+1), frame, screenwire.CodeOK, streamrx.Complete, true)
	assertRepeatPanelCycle(t, recorder, index, frame)
	assertRepeatTrace(t, port, index, recorder.alwaysHigh)
	if err = port.connection.Disconnect(); err != nil {
		t.Fatal(err)
	}
}

func assertRepeatTrace(t *testing.T, port *repeatPanelLink, index int, highOnly bool) {
	t.Helper()
	r, err := screenclient.ReadPanelTrace(port)
	if err != nil || r.Panel.State != 2 || r.Panel.Cycle != uint32(index+1) {
		t.Fatal(r, err)
	}
	for _, wait := range r.Panel.Waits {
		if wait.Samples == 0 || (wait.LowSamples == 0) != highOnly {
			t.Fatal("incorrect BUSY trace", r.Panel)
		}
	}
}

func repeatPanelFrame(t *testing.T, invert byte) display.Frame {
	t.Helper()
	pixels := make([]byte, 48000)
	for index := range pixels {
		pixels[index] = byte(index*37+index/251) ^ invert
	}
	frame, err := display.NewFrame(display.Size{Width: 800, Height: 480}, 100, pixels)
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

func assertRepeatPanelProof(t *testing.T, port *repeatPanelLink, epoch uint64, frame display.Frame,
	code screenwire.Code, state streamrx.State, current bool) {
	t.Helper()
	digest := sha256.Sum256(frame.Bytes())
	request := screenwire.Record{Kind: screenwire.Query, Epoch: epoch, ID: 1, Payload: digest[:]}
	if _, err := port.Write(encoded(t, request)); err != nil {
		t.Fatal(err)
	}
	record, err := screenwire.Decode(port.output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	reply, err := screenwire.ParseReply(record, request)
	port.output.Reset()
	if err != nil || reply.Status.Code != code || reply.Status.State != state || reply.Status.CurrentImage != current {
		t.Fatalf("epoch %d transaction 1: status=%+v error=%v", epoch, reply.Status, err)
	}
}
