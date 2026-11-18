package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

func setManager(args []string, target, portName string, output io.Writer) error {
	if target != "esp32" {
		return errors.New("set-manager requires -target esp32")
	}
	endpoint, err := managerArgument(args)
	if err != nil {
		return err
	}
	codec := provision.ESP32Codec()
	port, closePort, err := connectFor(portName, codec)
	if err != nil {
		return err
	}
	defer closePort()
	client, err := provisioningClient(port, codec, true)
	if err != nil {
		return err
	}
	response, err := client.SetManager(endpoint)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "manager=%q generation=%d changed=%t\n", response.Manager,
		response.Generation, response.Operation == provision.OperationSetManager)
	return err
}

func managerArgument(args []string) (string, error) {
	if len(args) != 2 && len(args) != 3 {
		return "", errors.New("usage: set-manager HOST or tcp://HOST:PORT")
	}
	endpoint := args[1]
	if len(args) == 3 {
		var err error
		endpoint, err = mobileEndpoint(endpoint, args[2])
		if err != nil {
			return "", err
		}
	} else if !strings.Contains(endpoint, ":") {
		endpoint = "tcp://" + endpoint + ":9757"
	}
	if !provision.ValidESP32Endpoint(endpoint) {
		return "", errors.New("expected an IPv4 address, DNS name, or tcp://HOST:PORT")
	}
	return endpoint, nil
}
