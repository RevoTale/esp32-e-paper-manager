// epaperscreen renders bounded HTML in pure Go and delivers EPS2 over local USB.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
)

var ErrUnconfirmed = errors.New("screen outcome unconfirmed; no automatic replay was attempted")

type transport interface {
	screendelivery.RecoveringSender
	Close() error
	Inspect(context.Context) (screenusbhost.Snapshot, error)
}

var openTransport = func(port string, size display.Size) (transport, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return screenusbhost.New(executable, port, size)
}
var proxy = screenusbhost.SerialProxy
var render = renderHTML

func main() { os.Exit(mainExit()) }

func mainExit() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		if len(os.Args) > 1 && os.Args[1] == "--serial-proxy" {
			return screenusbhost.ProxyExitCode(err)
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) > 0 && args[0] == "--serial-proxy" {
		return runProxy(args[1:], output)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if len(args) > 0 && args[0] == "--status" {
		return runStatus(ctx, args[1:], output)
	}
	if len(args) > 0 && args[0] == "--panel-status" {
		return runPanelStatus(ctx, args[1:], output)
	}
	c, err := renderOptions(args)
	if err != nil {
		return err
	}
	s, err := openTransport(c.port, c.size)
	if err != nil {
		return err
	}
	return runDelivery(ctx, s, c, output)
}

func runProxy(args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: epaperscreen --serial-proxy SERIAL_PORT (supervised child only)")
	}
	return proxy(args[0], os.Stdin, output)
}

func runDelivery(ctx context.Context, s transport, c renderConfig, output io.Writer) (result error) {
	defer func() { result = errors.Join(result, s.Close()) }()
	if err := awaitReady(ctx, s); err != nil {
		return err
	}
	source, err := readHTML(c.path)
	if err != nil {
		return err
	}
	scene, err := render(ctx, c, source)
	if err != nil {
		return err
	}
	for _, warning := range scene.warnings {
		if _, err := fmt.Fprintf(output, "warning=%s element=%d\n", warning.Code, warning.Element); err != nil {
			return err
		}
	}
	if err = sendOnce(ctx, s, scene.frame); err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "screen=confirmed (verify visible pixels separately)")
	return err
}

func sendOnce(ctx context.Context, s transport, frame display.Frame) error {
	err := s.Send(ctx, frame)
	if !errors.Is(err, screendelivery.ErrTransportLost) {
		return err
	}
	// Keep the parent Client alive and query the exact pending identity. Never
	// pass this frame to Send again, even if the controller staging was aborted.
	r, reconcileErr := s.WaitReady(ctx)
	if reconcileErr != nil {
		return errors.Join(ErrUnconfirmed, err, reconcileErr)
	}
	if r.Pending != screendelivery.PendingConfirmed {
		return errors.Join(ErrUnconfirmed, err)
	}
	return nil
}

func awaitReady(ctx context.Context, s transport) error {
	for {
		r, err := s.WaitReady(ctx)
		if err != nil {
			return err
		}
		if r.Pending != screendelivery.NoPending {
			return ErrUnconfirmed
		}
		delay := time.Until(r.NotBefore)
		if delay <= 0 {
			return nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
