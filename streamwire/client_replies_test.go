package streamwire

import (
	"bytes"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

type changedReply struct {
	link   *Link
	reply  bytes.Buffer
	change func(Record, []byte) []byte
}

func (s *changedReply) Read(p []byte) (int, error) { return s.reply.Read(p) }
func (s *changedReply) Write(p []byte) (int, error) {
	request, err := Decode(p)
	if err != nil {
		return 0, err
	}
	err = s.link.Push(p, 0, func(reply []byte) error {
		_, err := s.reply.Write(s.change(request, reply))
		return err
	})
	return len(p), err
}

func TestClientRejectsInvalidReplies(t *testing.T) {
	cases := []struct {
		name         string
		kind         Kind
		field, value int
	}{
		{"width", Hello, 8, 9}, {"height", Hello, 10, 9},
		{"zero chunk", Hello, 12, 0}, {"large chunk", Hello, 12, 101},
		{"passes", Hello, 14, 1}, {"begin rejected", Begin, 0, 2},
		{"data rejected", Data, 0, 3}, {"wrong offset", Data, 4, 99},
		{"wrong pass", Data, 2, 99}, {"commit rejected", Commit, 0, 7},
		{"commit incomplete", Commit, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, _ := linkFixture(t)
			stream := &changedReply{link: l, change: func(request Record, raw []byte) []byte {
				if request.Kind != tc.kind {
					return raw
				}
				r, err := Decode(raw)
				if err != nil {
					t.Fatal(err)
				}
				r.Payload[tc.field] = byte(tc.value)
				out := make([]byte, MaxRecord)
				n, err := Encode(out, r)
				if err != nil {
					t.Fatal(err)
				}
				return out[:n]
			}}
			if err := Send(stream, 7, clientFrame(t)); err == nil {
				t.Fatal("invalid reply accepted")
			}
		})
	}
}

func clientFrame(t *testing.T) display.Frame {
	t.Helper()
	f, err := display.NewFrame(display.Size{Width: 8, Height: 2}, 1, []byte{0, 255})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestClientRejectsDamagedReplyEncoding(t *testing.T) {
	for _, mode := range []string{"magic", "crc", "body", "epoch", "kind", "short payload"} {
		t.Run(mode, func(t *testing.T) {
			l, _ := linkFixture(t)
			s := &changedReply{link: l, change: func(_ Record, raw []byte) []byte {
				out := append([]byte(nil), raw...)
				switch mode {
				case "magic":
					out[0] ^= 1
				case "crc":
					out[28] ^= 1
				case "body":
					return out[:HeaderSize+1]
				default:
					return alteredRecord(t, out, mode)
				}
				return out
			}}
			if err := Send(s, 7, clientFrame(t)); err == nil {
				t.Fatal("damaged reply accepted")
			}
		})
	}
}

func alteredRecord(t *testing.T, raw []byte, mode string) []byte {
	t.Helper()
	r, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	switch mode {
	case "epoch":
		r.Epoch++
	case "kind":
		r.Kind = Hello
	case "short payload":
		r.Payload = r.Payload[:1]
	}
	out := make([]byte, MaxRecord)
	n, err := Encode(out, r)
	if err != nil {
		t.Fatal(err)
	}
	return out[:n]
}
