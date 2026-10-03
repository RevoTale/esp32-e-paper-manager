package streamwire

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

// Link is owned by one USB loop and retained across disconnects so reconnect
// cannot bypass refresh cadence. Call Tick even without incoming bytes.
type Link struct {
	config              streamrx.Config
	sink                streamrx.Sink
	rx                  *streamrx.Receiver
	interval, lastStart time.Duration
	hasStart            bool
	buffer              [MaxRecord]byte
	used, expected      int
	reply               [MaxRecord]byte
	body                [20]byte
}

func NewLink(config streamrx.Config, sink streamrx.Sink, interval time.Duration) (*Link, error) {
	if interval <= 0 || config.MaxChunk > MaxPayload {
		return nil, streamrx.ErrConfig
	}
	// Validate all local capabilities without retaining an active session.
	if _, err := streamrx.New(config, sink); err != nil {
		return nil, err
	}
	return &Link{config: config, sink: sink, interval: interval}, nil
}

func (l *Link) Disconnect() error {
	l.used, l.expected = 0, 0
	if l.rx == nil {
		return nil
	}
	err := l.rx.Close()
	l.rx = nil
	return err
}

func (l *Link) Tick(now time.Duration) error {
	if l.rx == nil {
		return nil
	}
	return l.rx.Tick(now)
}

// Push accepts arbitrary fragmentation but sends one reply per complete record.
// A malformed frame closes the session; caller must drop DTR/reopen to resync.
func (l *Link) Push(src []byte, now time.Duration, write func([]byte) error) error {
	for len(src) > 0 {
		limit := l.expected
		if limit == 0 {
			limit = HeaderSize
		}
		n := copy(l.buffer[l.used:limit], src)
		l.used += n
		src = src[n:]
		if l.used < limit {
			continue
		}
		if l.expected == 0 {
			size, err := Size(l.buffer[:HeaderSize])
			if err != nil {
				return errors.Join(err, l.Disconnect())
			}
			l.expected = size
			if size > HeaderSize {
				continue
			}
		}
		r, err := Decode(l.buffer[:l.expected])
		l.used, l.expected = 0, 0
		if err != nil {
			return errors.Join(err, l.Disconnect())
		}
		err = l.dispatch(r, now)
		if outputErr := l.respond(r, err, write); outputErr != nil {
			return errors.Join(outputErr, l.Disconnect())
		}
	}
	return nil
}

func (l *Link) dispatch(r Record, now time.Duration) error {
	err := l.handle(r, now)
	if err != nil && l.rx != nil {
		return l.rx.Invalidate(err)
	}
	return err
}

func (l *Link) handle(r Record, now time.Duration) error {
	if r.Kind == Hello {
		return l.hello(r)
	}
	if l.rx == nil || r.Epoch != l.config.Epoch {
		return streamrx.ErrState
	}
	intent := l.rx.Status().Intent
	if r.Kind == Begin {
		if len(r.Payload) != 32 || r.Pass != 0 || r.Offset != 0 {
			return ErrRecord
		}
		intent = streamrx.Intent{Epoch: r.Epoch, ID: r.ID}
		copy(intent.Digest[:], r.Payload)
		return l.begin(intent, now)
	}
	if r.ID != intent.ID {
		return streamrx.ErrState
	}
	return l.apply(r, intent, now)
}

func (l *Link) hello(r Record) error {
	if r.Epoch == 0 || r.ID != 0 || r.Pass != 0 || r.Offset != 0 || len(r.Payload) != 0 {
		return ErrRecord
	}
	if l.rx != nil {
		if r.Epoch == l.config.Epoch {
			return nil
		}
		return streamrx.ErrState
	}
	l.config.Epoch = r.Epoch
	rx, err := streamrx.New(l.config, l.sink)
	l.rx = rx
	return err
}

func (l *Link) begin(intent streamrx.Intent, now time.Duration) error {
	if l.rx.Status().State == streamrx.Complete && l.rx.Status().Intent == intent {
		return l.rx.Begin(intent, now)
	}
	if l.hasStart && (now < l.lastStart || now-l.lastStart < l.interval) {
		return errDeferred
	}
	return l.rx.Begin(intent, now)
}

func (l *Link) apply(r Record, intent streamrx.Intent, now time.Duration) error {
	if r.Kind == Data {
		return l.rx.Write(intent, r.Pass, r.Offset, r.Payload, now)
	}
	if len(r.Payload) != 0 || r.Pass != 0 || r.Offset != 0 {
		return ErrRecord
	}
	switch r.Kind {
	case Query:
		return l.rx.Status().Failure
	case Abort:
		return l.rx.Close()
	case Commit:
		if l.rx.Status().State == streamrx.Ready {
			l.hasStart = true
			l.lastStart = now
		}
		return l.rx.Commit(intent, now)
	default:
		return ErrRecord
	}
}

var errDeferred = errors.New("stream: refresh deferred")

func (l *Link) respond(request Record, err error, write func([]byte) error) error {
	clear(l.body[:])
	l.body[0] = errorCode(err)
	if l.rx != nil {
		status := l.rx.Status()
		l.body[1], l.body[2] = byte(status.State), status.Pass
		binary.LittleEndian.PutUint32(l.body[4:8], status.Offset)
	}
	binary.LittleEndian.PutUint16(l.body[8:10], l.config.Width)
	binary.LittleEndian.PutUint16(l.body[10:12], l.config.Height)
	binary.LittleEndian.PutUint16(l.body[12:14], uint16(l.config.MaxChunk))
	l.body[14] = l.config.Passes
	binary.LittleEndian.PutUint32(l.body[16:20], uint32(l.interval/time.Second))
	n, encodeErr := Encode(l.reply[:], Record{Kind: Reply, Epoch: request.Epoch, ID: request.ID, Payload: l.body[:]})
	if encodeErr != nil {
		return encodeErr
	}
	return write(l.reply[:n])
}

func errorCode(err error) byte {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrRecord):
		return 1
	case errors.Is(err, streamrx.ErrState):
		return 2
	case errors.Is(err, streamrx.ErrChunk):
		return 3
	case errors.Is(err, streamrx.ErrDigest):
		return 4
	case errors.Is(err, streamrx.ErrTimeout):
		return 5
	case errors.Is(err, errDeferred):
		return 6
	default:
		return 7
	}
}
