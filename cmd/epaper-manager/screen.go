package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func runScreen(c config, token []byte) error {
	renderer, err := engine.New()
	if err != nil {
		return err
	}
	transport, zone, err := screenDelivery(c)
	if err != nil {
		return err
	}
	defer func() { _ = transport.close() }()
	screen, err := manager.NewScreenWithOptions(renderer, c.screen.size, c.screen.policy,
		manager.ScreenOptions{Zone: zone, MaintenanceInterval: c.screen.maintenance, Partial: c.screen.partial.options()})
	if err != nil {
		return err
	}
	var epoch [16]byte
	if _, err = rand.Read(epoch[:]); err != nil {
		return err
	}
	api, err := manager.NewScreenAPI(screen, token, epoch, time.Now)
	if err != nil {
		return err
	}
	pump, err := manager.NewScreenPump(screen, transport.sender, c.screen.interval)
	if c.screen.refreshPolicy {
		pump, err = manager.NewScreenPumpWithPolicy(screen, transport.sender, refreshpolicy.Policy{Normal: c.screen.interval, Urgent: c.screen.urgent})
	}
	if err != nil {
		return err
	}
	server := httpServer(c.httpsAddress, api)
	if c.screen.partial.enabled {
		if err := pump.ConfigurePartial(c.screen.partial.policy); err != nil {
			return err
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return serveScreen(ctx, server, func() error { return serveTLS(server, c.certificate, c.certificateKey) }, pump.Run, transport.run)
}

// Stop both owners on the first failure or signal. In particular, a lost USB
// completion stops further submissions/delivery; no silent restart or replay.
func serveScreen(parent context.Context, server *http.Server, serve func() error, pump func(context.Context) error, transports ...func(context.Context) error) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan error, 2+len(transports))
	go func() { done <- pump(ctx) }()
	go func() { done <- serve() }()
	for _, run := range transports {
		go func() { done <- run(ctx) }()
	}
	first := <-done
	cancel()
	closeErr := server.Close()
	result := errors.Join(screenStopError(first), closeErr)
	for range 1 + len(transports) {
		result = errors.Join(result, screenStopError(<-done))
	}
	return result
}

func screenStopError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
