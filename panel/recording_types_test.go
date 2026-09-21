package panel

import (
	"time"
)

type wireEvent struct {
	kind  string
	high  bool
	data  []byte
	delay time.Duration
}

type recordingIO struct {
	events []wireEvent
	busy   []bool
	reads  int
}
