package panel

import (
	"time"
)

func noOpIO() IO {
	busyReads := 0
	return IO{
		Write:    func([]byte) error { return nil },
		SetCS:    func(bool) {},
		SetDC:    func(bool) {},
		SetReset: func(bool) {},
		SetPower: func(bool) {},
		ReadBusy: func() bool {
			value := busyReads%2 == 1
			busyReads++
			return value
		},
		Delay: func(time.Duration) {},
	}
}
