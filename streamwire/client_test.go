package streamwire

import (
	"bytes"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

type loopback struct {
	link  *Link
	reply bytes.Buffer
}

func (l *loopback) Write(p []byte) (int, error) {
	err := l.link.Push(p, 0, func(b []byte) error { _, err := l.reply.Write(b); return err })
	return len(p), err
}
func (l *loopback) Read(p []byte) (int, error) { return l.reply.Read(p) }

func TestClientUsesRealLink(t *testing.T) {
	l, s := linkFixture(t)
	f, err := display.NewFrame(display.Size{Width: 8, Height: 2}, 1, []byte{128, 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = Send(&loopback{link: l}, 7, f); err != nil {
		t.Fatal(err)
	}
	if s.writes != 2 || s.commits != 1 {
		t.Fatal("incomplete client")
	}
}
