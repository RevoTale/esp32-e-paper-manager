package main

import (
	"time"

	"go.bug.st/serial"
)

func connect(requested string) (serialDevice, func(), error) {
	name := requested
	var err error
	if name == "auto" {
		name, err = findPort()
		if err != nil {
			return nil, func() {}, err
		}
	}
	port, err := openSerial(name, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, func() {}, err
	}
	closePort := func() {
		_ = port.SetDTR(false)
		_ = port.Close()
	}
	if err = port.SetReadTimeout(20 * time.Second); err == nil {
		err = port.SetDTR(true)
	}
	if err != nil {
		closePort()
		return nil, func() {}, err
	}
	return port, closePort, nil
}
