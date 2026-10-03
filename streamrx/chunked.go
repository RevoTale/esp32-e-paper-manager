package streamrx

type ChunkedSink struct {
	sink  Sink
	limit int
}

// NewChunkedSink adapts authenticated logical chunks to a controller's smaller
// write quantum. It borrows slices, allocates no staging buffer and preserves
// pass/absolute offset. Receiver still owns validation, abort and commit order.
func NewChunkedSink(sink Sink, limit int) (*ChunkedSink, error) {
	if sink == nil || limit <= 0 || limit > 4096 {
		return nil, ErrConfig
	}
	return &ChunkedSink{sink: sink, limit: limit}, nil
}

func (s *ChunkedSink) Write(pass uint8, offset uint32, pixels []byte) error {
	if !s.configured() {
		return ErrConfig
	}
	if len(pixels) == 0 || len(pixels) > 4096 || uint64(offset)+uint64(len(pixels)) > uint64(^uint32(0)) {
		return ErrChunk
	}
	for len(pixels) > 0 {
		n := min(s.limit, len(pixels))
		if err := s.sink.Write(pass, offset, pixels[:n]); err != nil {
			return err
		}
		pixels = pixels[n:]
		offset += uint32(n)
	}
	return nil
}

func (s *ChunkedSink) configured() bool {
	return s != nil && s.sink != nil && s.limit > 0 && s.limit <= 4096
}
func (s *ChunkedSink) Begin() error {
	if !s.configured() {
		return ErrConfig
	}
	return s.sink.Begin()
}
func (s *ChunkedSink) Commit() error {
	if !s.configured() {
		return ErrConfig
	}
	return s.sink.Commit()
}
func (s *ChunkedSink) Abort() error {
	if !s.configured() {
		return ErrConfig
	}
	return s.sink.Abort()
}
