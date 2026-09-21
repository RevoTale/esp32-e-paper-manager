//go:build !tinygo

package devicelink

import (
	"encoding/binary"
	"io"

	"github.com/RevoTale/esp32-e-paper-manager/manager"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

type ManagerConfig struct {
	Registry manager.Registry
	Store    *manager.Store
	Random   io.Reader
	Now      manager.Clock
}

func ServeManager(connection io.ReadWriter, config ManagerConfig) error {
	if connection == nil || config.Registry == nil || config.Store == nil || config.Random == nil || config.Now == nil {
		return ErrConfiguration
	}
	id, secured, err := managerHandshake(connection, config)
	if err != nil {
		return err
	}
	if err = receiveRetained(secured, id, config); err != nil {
		return err
	}
	return servePending(secured, id, config)
}

func receiveRetained(secured *securetransport.RecordStream, id securetransport.DeviceID,
	config ManagerConfig,
) error {
	header, err := readHeader(secured)
	if err != nil {
		return err
	}
	if header.typeID == messageIdle && header.length == 0 && header.generation == 0 {
		return config.Store.ReopenAccepted(id, config.Now())
	}
	if header.typeID != messageResult || header.generation == 0 ||
		header.length != update.ResultEncodedSize {
		return ErrProtocol
	}
	result, err := readResult(secured)
	if err != nil {
		return err
	}
	return config.Store.Complete(id, header.generation, result, config.Now())
}

func managerHandshake(connection io.ReadWriter, config ManagerConfig) (securetransport.DeviceID,
	*securetransport.RecordStream, error,
) {
	var id securetransport.DeviceID
	if _, err := io.ReadFull(connection, id[:]); err != nil {
		return id, nil, err
	}
	record, ok := config.Registry.Lookup(id)
	if !ok {
		return id, nil, manager.ErrUnknownDevice
	}
	epoch, session, err := randomCounters(config.Random)
	if err != nil {
		return id, nil, err
	}
	secured, err := securetransport.ServerHandshake(connection, record.Key, id, epoch, session)
	if err != nil {
		return id, nil, err
	}
	return id, secured, nil
}

func servePending(secured *securetransport.RecordStream, id securetransport.DeviceID,
	config ManagerConfig,
) error {
	pending, generation, err := config.Store.Lease(id, config.Now())
	if err == manager.ErrNoUpdate {
		return writeMessage(secured, messageIdle, generation, nil)
	}
	if err != nil {
		return err
	}
	wire := make([]byte, update.EncodedSize(pending.Request))
	if _, err = update.Encode(wire, pending.Request); err != nil {
		return err
	}
	if err = writeMessage(secured, messageUpdate, generation, wire); err != nil {
		return err
	}
	return receiveResult(secured, id, generation, config)
}

func receiveResult(secured *securetransport.RecordStream, id securetransport.DeviceID,
	generation uint64, config ManagerConfig,
) error {
	header, err := readHeader(secured)
	if err != nil || header.typeID != messageResult || header.generation != generation ||
		header.length != update.ResultEncodedSize {
		return ErrProtocol
	}
	result, err := readResult(secured)
	if err != nil {
		return err
	}
	return config.Store.Complete(id, generation, result, config.Now())
}

func readResult(reader io.Reader) (update.Result, error) {
	var resultWire [update.ResultEncodedSize]byte
	if _, err := io.ReadFull(reader, resultWire[:]); err != nil {
		return update.Result{}, err
	}
	return update.DecodeResult(resultWire[:])
}

func randomCounters(random io.Reader) (uint64, uint64, error) {
	var wire [16]byte
	if _, err := io.ReadFull(random, wire[:]); err != nil {
		return 0, 0, err
	}
	epoch, session := binary.BigEndian.Uint64(wire[:8]), binary.BigEndian.Uint64(wire[8:])
	if epoch == 0 || session == 0 {
		return 0, 0, ErrConfiguration
	}
	return epoch, session, nil
}
