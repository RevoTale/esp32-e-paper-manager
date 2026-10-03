package screenhub

import "net"

// Reaching the second Accept proves Serve admitted the first socket and
// registered its handshake worker. Dial alone proves neither of those events.
type progressListener struct {
	net.Listener
	secondAccept chan struct{}
	calls        int
}

func (l *progressListener) Accept() (net.Conn, error) {
	l.calls++
	if l.calls == 2 {
		close(l.secondAccept)
	}
	return l.Listener.Accept()
}
