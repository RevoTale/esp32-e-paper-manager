package paneldiag

import (
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
	"github.com/RevoTale/esp32-e-paper-manager/streamrx"
)

type tracedSink struct {
	trace *Trace
	sink  streamrx.Sink
}

func (s *tracedSink) Begin() error {
	t := s.trace
	if t.status.State != 1 {
		t.status = screenwire.PanelStatus{Version: 1, State: 1, Cycle: increment(t.status.Cycle)}
		t.started, t.wait = t.now(), 0
	}
	err := s.sink.Begin()
	if err != nil {
		t.end(3)
	}
	return err
}

func (s *tracedSink) Write(pass uint8, offset uint32, data []byte) error {
	err := s.sink.Write(pass, offset, data)
	if err != nil {
		s.trace.end(3)
	}
	return err
}

func (s *tracedSink) Commit() error {
	err := s.sink.Commit()
	if err != nil {
		s.trace.end(3)
	} else {
		s.trace.end(2)
	}
	return err
}

func (s *tracedSink) Abort() error {
	err := s.sink.Abort()
	s.trace.end(3)
	return err
}
