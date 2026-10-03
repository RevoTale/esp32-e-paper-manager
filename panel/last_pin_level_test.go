package panel

func lastPinLevel(events []wireEvent, kind string) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].kind == kind {
			return events[i].high
		}
	}
	return false
}
