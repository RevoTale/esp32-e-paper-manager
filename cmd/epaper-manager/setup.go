package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/RevoTale/esp32-e-paper-manager/internal/managersetup"
)

func runSetup(arguments []string) error {
	flags := flag.NewFlagSet("epaper-manager setup", flag.ContinueOnError)
	configuration := managersetup.Config{}
	flags.StringVar(&configuration.Enrollment, "enrollment", "", "existing private USB enrollment file")
	flags.StringVar(&configuration.Directory, "output", "", "new credential directory; parent must exist")
	flags.IntVar(&configuration.UID, "uid", -1, "credential owner UID (-1: current user)")
	flags.IntVar(&configuration.GID, "gid", -1, "credential owner GID (-1: current group)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || configuration.Enrollment == "" || configuration.Directory == "" {
		return errors.New("setup requires -enrollment and -output; positional arguments forbidden")
	}
	if err := managersetup.Run(configuration); err != nil {
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, "Manager credentials ready. Existing credentials preserved; no device changes.")
	return err
}
