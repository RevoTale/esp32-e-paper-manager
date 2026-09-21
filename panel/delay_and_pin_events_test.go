package panel

import (
	"time"
)

func delayAndPinEvents(events []wireEvent) ([]time.Duration, []bool, []bool) {
	var delays []time.Duration
	var power, reset []bool
	for _, event := range events {
		switch event.kind {
		case "delay":
			delays = append(delays, event.delay)
		case "power":
			power = append(power, event.high)
		case "reset":
			reset = append(reset, event.high)
		}
	}

	return delays, power, reset
}
