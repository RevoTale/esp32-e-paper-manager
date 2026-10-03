package screenusbhost

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type process struct {
	input, output *os.File
	done          chan struct{}
	cancel        context.CancelFunc
	once          sync.Once
	exitCode      int // Published by closing done; read only after receiving done.
}

// Parent-owned os.Pipes avoid Cmd.Wait closing StdoutPipe while the delivery
// owner is reading. The child never spawns descendants. Cancellation kills the
// direct process even if a macOS serial write does not respond to port.Close.
// https://pkg.go.dev/os/exec#Cmd.Wait and https://pkg.go.dev/os/exec#CommandContext
func startProcess(executable, port string, changed chan<- struct{}) (worker, error) {
	inRead, inWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(ErrWorker, err)
	}
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(ErrWorker, err, inRead.Close(), inWrite.Close())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, executable, "--serial-proxy", port)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = inRead, outWrite, io.Discard
	cmd.WaitDelay = time.Second
	if err = cmd.Start(); err != nil {
		cancel()
		return nil, errors.Join(ErrWorker, err, inRead.Close(), inWrite.Close(), outRead.Close(), outWrite.Close())
	}
	// The child owns duplicate descriptors after Start; close parent duplicates.
	_ = inRead.Close()
	_ = outWrite.Close()
	p := &process{input: inWrite, output: outRead, done: make(chan struct{}), cancel: cancel}
	go p.reap(cmd, changed)
	return p, nil
}

func (p *process) reap(cmd *exec.Cmd, changed chan<- struct{}) {
	// Exit errors never contain trusted delivery evidence. Pipe EOF triggers
	// reconciliation; do not print child diagnostics or private record bytes.
	_ = cmd.Wait()
	p.exitCode = cmd.ProcessState.ExitCode()
	close(p.done)
	select {
	case changed <- struct{}{}:
	default:
	}
}

func (p *process) Read(data []byte) (int, error) {
	n, err := p.output.Read(data)
	if !errors.Is(err, io.EOF) {
		return n, err
	}
	// EOF may precede Wait's publication. Bound collection if a broken worker
	// closes stdout without exiting; the outer operation still owns cancellation.
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		if p.exitCode != 0 {
			err = errors.Join(err, errors.New(proxyStage(p.exitCode)))
		}
	case <-timer.C:
	}
	return n, err
}
func (p *process) Write(data []byte) (int, error) { return p.input.Write(data) }
func (p *process) Done() <-chan struct{}          { return p.done }

func (p *process) Close() error {
	p.once.Do(func() {
		p.cancel()
		// Closing parent pipes also wakes any blocked operation immediately.
		_ = p.input.Close()
		_ = p.output.Close()
	})
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		return nil
	case <-timer.C:
		return ErrWorker
	}
}
