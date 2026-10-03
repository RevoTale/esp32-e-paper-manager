package interop

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

type nativePeer struct {
	t        *testing.T
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	input    io.WriteCloser
	output   io.ReadCloser
	started  bool
	finished bool
}

func startNativePeer(t *testing.T) *nativePeer {
	return startNativePeerNamed(t, "EP_CRYPTO_CLI")
}

func startNativePeerNamed(t *testing.T, variable string, args ...string) *nativePeer {
	t.Helper()
	path := os.Getenv(variable)
	if path == "" {
		t.Fatal(variable + " must name the compiled native peer")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	peer := &nativePeer{t: t, cmd: exec.CommandContext(ctx, path, args...), cancel: cancel}
	peer.cmd.WaitDelay = time.Second
	peer.cmd.Stderr = os.Stderr
	// Register before pipe creation or handshake assertions can fail.
	t.Cleanup(peer.finish)
	var err error
	peer.input, err = peer.cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	peer.output, err = peer.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = peer.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	peer.started = true
	return peer
}

func (p *nativePeer) finish() {
	p.t.Helper()
	if p.finished {
		return
	}
	p.finished = true
	defer p.cancel()
	if p.t.Failed() {
		p.cancel() // Assertions must not leave the child waiting for more input.
	}
	p.closePipe(p.input)
	if p.started {
		if err := p.cmd.Wait(); err != nil {
			p.t.Errorf("native peer exit: %v", err)
		}
	}
	p.closePipe(p.output)
}

func (p *nativePeer) closePipe(pipe io.Closer) {
	p.t.Helper()
	if pipe != nil {
		if err := pipe.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			p.t.Errorf("native peer pipe close: %v", err)
		}
	}
}
