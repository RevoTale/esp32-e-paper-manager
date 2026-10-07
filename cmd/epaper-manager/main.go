package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/devicelink"
	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"golang.org/x/net/netutil"
)

type config struct {
	screen         screenConfig
	httpsAddress   string
	deviceAddress  string
	certificate    string
	certificateKey string
	tokenFile      string
	enrollmentFile string
	stateFile      string
}

var (
	listenTCP = net.Listen
	serveTLS  = serveHTTPS
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "epaper-manager:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) > 0 && arguments[0] == "setup" {
		return runSetup(arguments[1:])
	}
	return runManager(arguments)
}

func runManager(arguments []string) error {
	configuration, err := parseConfig(arguments)
	if err != nil {
		return err
	}
	token, err := readSecret(configuration.tokenFile)
	if err != nil {
		return err
	}
	defer clear(token)
	if configuration.screen.enabled {
		return runScreen(configuration, token)
	}
	record, err := manager.LoadEnrollment(configuration.enrollmentFile)
	if err != nil {
		return err
	}
	registry, err := manager.NewStaticRegistry([]manager.DeviceRecord{record})
	if err != nil {
		return err
	}
	store, err := manager.NewPersistentStore(configuration.stateFile)
	if err != nil {
		return err
	}
	api, err := manager.NewServer(store, registry, token, time.Now, 24*time.Hour)
	clear(token)
	if err != nil {
		return err
	}
	listener, err := listenTCP("tcp", configuration.deviceAddress)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	go serveDevices(listener, registry, store)
	return serveTLS(httpServer(configuration.httpsAddress, api), configuration.certificate, configuration.certificateKey)
}

func parseConfig(arguments []string) (config, error) {
	flags := flag.NewFlagSet("epaper-manager", flag.ContinueOnError)
	configuration := config{}
	flags.StringVar(&configuration.httpsAddress, "https-listen", ":8443", "HTTPS listen address")
	flags.StringVar(&configuration.deviceAddress, "device-listen", "[::]:9757", "encrypted device-link listen address")
	flags.StringVar(&configuration.certificate, "tls-cert", "", "TLS certificate file")
	flags.StringVar(&configuration.certificateKey, "tls-key", "", "TLS private key file")
	flags.StringVar(&configuration.tokenFile, "token-file", "", "mode-0600 API token file")
	flags.StringVar(&configuration.enrollmentFile, "enrollment", "", "mode-0600 device enrollment file")
	flags.StringVar(&configuration.stateFile, "state", "", "mode-0600 manager state file")
	screenFlags(flags, &configuration.screen)
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 || configuration.certificate == "" || configuration.certificateKey == "" || configuration.tokenFile == "" {
		return config{}, errors.New("tls-cert, tls-key, token-file are required; positional arguments forbidden")
	}
	return validateMode(flags, configuration)
}

func readSecret(path string) ([]byte, error) {
	value, err := hostprovision.ReadPrivateFile(path, 4096)
	if err != nil {
		return nil, errors.New("secret file must exist, be private, and be bounded")
	}
	value = bytes.TrimSpace(value)
	if len(value) < 32 {
		clear(value)
		return nil, errors.New("API token must contain at least 32 bytes")
	}
	return value, nil
}

func serveDevices(listener net.Listener, registry manager.Registry, store *manager.Store) {
	listener = netutil.LimitListener(listener, 4)
	defer func() { _ = listener.Close() }()
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		go serveDevice(connection, registry, store)
	}
}

func serveDevice(connection net.Conn, registry manager.Registry, store *manager.Store) {
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(45 * time.Second))
	err := devicelink.ServeManager(connection, devicelink.ManagerConfig{
		Registry: registry, Store: store, Random: rand.Reader, Now: time.Now,
	})
	if err != nil {
		slog.Warn("device link ended", "error", err)
	}
}
