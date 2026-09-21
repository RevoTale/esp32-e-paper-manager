//go:build !darwin

package main

import (
	"errors"
	"io"
	"testing"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type setupFailSerial struct{ fail string }

func (serial *setupFailSerial) Write(value []byte) (int, error) { return len(value), nil }
func (*setupFailSerial) Read([]byte) (int, error)               { return 0, io.EOF }
func (serial *setupFailSerial) SetReadTimeout(time.Duration) error {
	if serial.fail == "timeout" {
		return errors.New("timeout")
	}
	return nil
}
func (serial *setupFailSerial) SetDTR(value bool) error {
	if value && serial.fail == "dtr" {
		return errors.New("dtr")
	}
	return nil
}
func (*setupFailSerial) Close() error { return nil }

func TestFindPortAndConnectFailures(t *testing.T) {
	originalEnumerate, originalOpen := enumeratePorts, openSerial
	defer func() { enumeratePorts, openSerial = originalEnumerate, originalOpen }()
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{{Name: "pico", IsUSB: true, VID: tinyGoVID, PID: tinyGoPID}}, nil
	}
	if port, err := findPort(); err != nil || port != "pico" {
		t.Fatalf("port=%q err=%v", port, err)
	}
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) { return nil, nil }
	if _, err := findPort(); !errors.Is(err, errUSBNotFound) {
		t.Fatalf("missing=%v", err)
	}
	openSerial = func(string, *serial.Mode) (serialDevice, error) { return nil, io.ErrClosedPipe }
	if _, _, err := connect("pico"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("connect=%v", err)
	}
}

func TestConnectAutoAndSetupFailures(t *testing.T) {
	originalEnumerate, originalOpen := enumeratePorts, openSerial
	defer func() { enumeratePorts, openSerial = originalEnumerate, originalOpen }()
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{{Name: "pico", IsUSB: true, VID: tinyGoVID, PID: tinyGoPID}}, nil
	}
	opened := &setupFailSerial{}
	openSerial = func(name string, mode *serial.Mode) (serialDevice, error) {
		if name != "pico" || mode.BaudRate != 115200 {
			t.Fatal("wrong serial configuration")
		}
		return opened, nil
	}
	port, closePort, err := connect("auto")
	if err != nil || port != opened {
		t.Fatalf("port=%v err=%v", port, err)
	}
	closePort()
	for _, failure := range []string{"timeout", "dtr"} {
		openSerial = func(string, *serial.Mode) (serialDevice, error) { return &setupFailSerial{fail: failure}, nil }
		if _, _, err = connect("pico"); err == nil {
			t.Fatalf("failure=%s accepted", failure)
		}
	}
	enumeratePorts = func(...func(string, string) bool) ([]*enumerator.PortDetails, error) {
		return nil, errors.New("enumerate")
	}
	if _, err = findPort(); err == nil {
		t.Fatal("enumeration failure discarded")
	}
}
