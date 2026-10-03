package screenusbhost

import "time"

func (s *Sender) watch() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer close(s.watched)
	s.watchTicks(ticker.C)
}

// The fixed watcher only coalesces readiness notifications. All serial/client
// operations remain in the delivery owner. This discovers an idle unplug/reboot
// without an author edit and without timer-triggered uploads or session claims.
func (s *Sender) watchTicks(ticks <-chan time.Time) {
	for {
		select {
		case <-s.done:
			return
		case <-ticks:
			s.signal()
		}
	}
}
