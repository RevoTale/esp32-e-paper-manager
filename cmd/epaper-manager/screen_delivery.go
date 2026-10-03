package main

import (
	"context"
	"errors"
	"time"
	_ "time/tzdata" // Standalone manager binaries must not depend on host zoneinfo.

	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/screendelivery"
	"github.com/RevoTale/esp32-e-paper-manager/screenhub"
	"github.com/RevoTale/esp32-e-paper-manager/screenusbhost"
)

type screenTransport struct {
	sender screendelivery.Sender
	run    func(context.Context) error
	close  func() error
}

func screenDelivery(c config) (screenTransport, *time.Location, error) {
	if c.screen.transport == "wifi" {
		return screenNetwork(c)
	}
	zone, err := time.LoadLocation(c.screen.timezone)
	if err != nil {
		return screenTransport{}, nil, err
	}
	sender, err := screenusbhost.New(c.screen.usbWorker, c.screen.serial, c.screen.size)
	if err != nil {
		return screenTransport{}, nil, err
	}
	return screenTransport{sender: sender,
		run: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, close: sender.Close}, zone, nil
}

func screenNetwork(c config) (screenTransport, *time.Location, error) {
	record, err := manager.LoadEnrollment(c.enrollmentFile)
	if err != nil {
		return screenTransport{}, nil, err
	}
	zone, err := time.LoadLocation(record.Timezone)
	if err != nil {
		return screenTransport{}, nil, err
	}
	hub, err := screenhub.New(screenhub.Config{ID: record.ID, Key: record.Key, Size: c.screen.size, Profile: 1, ProfileVersion: 1})
	if err != nil {
		return screenTransport{}, nil, err
	}
	listener, err := listenTCP("tcp", c.deviceAddress)
	if err != nil {
		return screenTransport{}, nil, errors.Join(err, hub.Close())
	}
	return screenTransport{sender: hub, run: func(ctx context.Context) error { return hub.Serve(ctx, listener) },
		close: func() error { return errors.Join(hub.Close(), listener.Close()) }}, zone, nil
}
