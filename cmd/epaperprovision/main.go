package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/provision"
	"github.com/RevoTale/esp32-e-paper-manager/provisionclient"
	"go.bug.st/serial"
)

const (
	tinyGoVID = "2E8A"
	tinyGoPID = "000A"
)

var errUSBNotFound = errors.New("TinyGo Pico USB serial port 2E8A:000A not found")

type serialDevice interface {
	io.ReadWriter
	SetReadTimeout(time.Duration) error
	SetDTR(bool) error
	Close() error
}

var openSerial = func(name string, mode *serial.Mode) (serialDevice, error) { return serial.Open(name, mode) }

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "epaperprovision:", err)
		os.Exit(1)
	}
}

func run(arguments []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("epaperprovision", flag.ContinueOnError)
	portName := flags.String("port", "auto", "serial port; auto is unavailable on macOS")
	registry := flags.String("registry", "", "private manager enrollment file")
	confirmErase := flags.Bool("confirm-erase", false, "confirm irreversible credential erase")
	target := flags.String("target", "pico", "pico (WPA3) or esp32 (WPA2/CCMP)")
	correlated := flags.Bool("usb-v3", false, "require ESP32 correlated USB replies; needs v3 firmware")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: epaperprovision [flags] inspect|provision|rotate|erase|diagnose")
	}
	operation, err := parseOperation(flags.Arg(0), *confirmErase)
	if err != nil {
		return err
	}
	codec, err := boardCodec(*target)
	if err != nil {
		return err
	}
	if *correlated && *target != "esp32" {
		return errors.New("-usb-v3 requires -target esp32")
	}
	port, closePort, err := connectFor(*portName, codec)
	if err != nil {
		return err
	}
	defer closePort()
	client, err := provisioningClient(port, codec, *correlated)
	if err != nil {
		return err
	}
	request, pending, err := buildRequest(client, operation, *registry, input)
	if err != nil {
		return err
	}
	return executeRequest(client, request, pending, output)
}

func executeRequest(client *provisionclient.Client, request provision.Request,
	pending *hostprovision.PendingEnrollment, output io.Writer,
) error {
	if pending != nil {
		defer func() { _ = pending.Close() }()
	}
	response, err := client.Execute(request)
	if response.Operation == request.Operation {
		printResponse(output, response)
	}
	if err != nil {
		if pending != nil {
			return fmt.Errorf("%w: %w", hostprovision.ErrAcknowledgement, err)
		}
		return err
	}
	if pending != nil {
		return pending.ConfirmFor(client.Codec(), request, response)
	}
	return nil
}

func buildRequest(client *provisionclient.Client, operation provision.Operation, registry string,
	input io.Reader,
) (provision.Request, *hostprovision.PendingEnrollment, error) {
	if operation != provision.OperationProvision && operation != provision.OperationRotate {
		return provision.Request{Operation: operation}, nil, nil
	}
	if registry == "" {
		return provision.Request{}, nil, errors.New("provision and rotate require -registry")
	}
	operatorInput, err := hostprovision.DecodeInput(input)
	if err != nil {
		return provision.Request{}, nil, err
	}
	var identity *[16]byte
	if operation == provision.OperationRotate {
		current, inspectErr := client.Execute(provision.Request{Operation: provision.OperationInspect})
		if inspectErr != nil || current.State != provision.StateProvisioned {
			return provision.Request{}, nil, errors.New("rotate requires a provisioned device")
		}
		identity = &current.DeviceID
	}
	config, enrollment, err := hostprovision.CreateFor(client.Codec(), operatorInput, identity, nil)
	if err != nil {
		return provision.Request{}, nil, err
	}
	pending, err := hostprovision.StageEnrollment(registry, enrollment)
	if err != nil {
		return provision.Request{}, nil, err
	}
	return provision.Request{Operation: operation, Config: config}, pending, nil
}

func parseOperation(value string, confirmed bool) (provision.Operation, error) {
	operations := map[string]provision.Operation{"inspect": provision.OperationInspect,
		"provision": provision.OperationProvision, "rotate": provision.OperationRotate,
		"erase": provision.OperationErase, "diagnose": provision.OperationDiagnose}
	operation, ok := operations[value]
	if !ok {
		return 0, errors.New("unknown operation")
	}
	if operation == provision.OperationErase && !confirmed {
		return 0, errors.New("erase requires -confirm-erase")
	}
	return operation, nil
}

func printResponse(output io.Writer, response provision.Response) {
	_, _ = fmt.Fprintf(output, "code=%d state=%d generation=%d device=%x auth=%d ssid=%q manager=%q timezone=%q\n",
		response.Code, response.State, response.Generation, response.DeviceID, response.Auth,
		response.SSID, response.Manager, response.Timezone)
}
