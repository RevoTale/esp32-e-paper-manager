// Package hostprovision validates operator input and creates device enrollment.
package hostprovision

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/provision"
)

var ErrInput = errors.New("host provisioning: invalid input")

type Input struct {
	SSID       string `json:"ssid"`
	Passphrase string `json:"passphrase"`
	Manager    string `json:"manager"`
	Timezone   string `json:"timezone,omitempty"`
}

type Enrollment struct {
	DeviceID  [16]byte `json:"device_id"`
	DeviceKey [32]byte `json:"device_key"`
	Timezone  string   `json:"timezone"`
}

func DecodeInput(reader io.Reader) (Input, error) {
	if reader == nil {
		return Input{}, ErrInput
	}
	decoder := json.NewDecoder(io.LimitReader(reader, 4097))
	decoder.DisallowUnknownFields()
	var input Input
	if err := decoder.Decode(&input); err != nil {
		return Input{}, ErrInput
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return Input{}, ErrInput
	}
	if input.Timezone == "" {
		input.Timezone = "Europe/Kyiv"
	}
	return input, nil
}

func Create(input Input, identity *[16]byte, random io.Reader) (provision.Config, Enrollment, error) {
	return CreateFor(provision.Codec{}, input, identity, random)
}

func CreateFor(codec provision.Codec, input Input, identity *[16]byte, random io.Reader) (provision.Config, Enrollment, error) {
	if random == nil {
		random = rand.Reader
	}
	var enrollment Enrollment
	if identity == nil {
		if _, err := io.ReadFull(random, enrollment.DeviceID[:]); err != nil {
			return provision.Config{}, Enrollment{}, err
		}
	} else {
		enrollment.DeviceID = *identity
	}
	if _, err := io.ReadFull(random, enrollment.DeviceKey[:]); err != nil {
		return provision.Config{}, Enrollment{}, err
	}
	enrollment.Timezone = input.Timezone
	config := provision.Config{SSID: input.SSID, Passphrase: input.Passphrase, Manager: input.Manager,
		Timezone: input.Timezone, DeviceID: enrollment.DeviceID, DeviceKey: enrollment.DeviceKey,
		Auth: codec.AuthMode()}
	if err := config.ValidateFor(codec); err != nil {
		return provision.Config{}, Enrollment{}, ErrInput
	}
	return config, enrollment, nil
}
