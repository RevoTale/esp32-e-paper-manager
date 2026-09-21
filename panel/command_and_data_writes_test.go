package panel

import (
	"testing"
)

func commandAndDataWrites(t *testing.T, events []wireEvent) ([]byte, map[byte][]byte) {
	t.Helper()
	var commands []byte
	data := make(map[byte][]byte)
	dc := false
	var command byte
	for _, event := range events {
		switch event.kind {
		case "dc":
			dc = event.high
		case "write":
			if !dc {
				if len(event.data) != 1 {
					t.Fatalf("command write length = %d", len(event.data))
				}
				command = event.data[0]
				commands = append(commands, command)
			} else {
				data[command] = append(data[command], event.data...)
			}
		}
	}
	return commands, data
}
