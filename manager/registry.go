// Package manager provides the trusted home control plane.
package manager

import (
	"crypto/sha256"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/hostprovision"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
)

var (
	ErrConfiguration = errors.New("manager: invalid configuration")
	ErrUnknownDevice = errors.New("manager: unknown device")
)

type DeviceRecord struct {
	ID       securetransport.DeviceID
	Key      securetransport.Key
	Timezone string
}

type Registry interface {
	Lookup(securetransport.DeviceID) (DeviceRecord, bool)
}

type StaticRegistry struct {
	records map[securetransport.DeviceID]DeviceRecord
}

func NewStaticRegistry(records []DeviceRecord) (*StaticRegistry, error) {
	registry := &StaticRegistry{records: make(map[securetransport.DeviceID]DeviceRecord, len(records))}
	for _, record := range records {
		if record.ID == (securetransport.DeviceID{}) || record.Key == (securetransport.Key{}) ||
			record.Timezone == "" {
			return nil, ErrConfiguration
		}
		if _, err := time.LoadLocation(record.Timezone); err != nil {
			return nil, ErrConfiguration
		}
		if _, exists := registry.records[record.ID]; exists {
			return nil, ErrConfiguration
		}
		registry.records[record.ID] = record
	}
	return registry, nil
}

func (r *StaticRegistry) Lookup(id securetransport.DeviceID) (DeviceRecord, bool) {
	if r == nil {
		return DeviceRecord{}, false
	}
	record, ok := r.records[id]
	return record, ok
}

func LoadEnrollment(path string) (DeviceRecord, error) {
	enrollment, err := hostprovision.LoadEnrollment(path)
	if err != nil {
		return DeviceRecord{}, ErrConfiguration
	}
	record := DeviceRecord{ID: securetransport.DeviceID(enrollment.DeviceID),
		Key: securetransport.Key(enrollment.DeviceKey), Timezone: enrollment.Timezone}
	registry, err := NewStaticRegistry([]DeviceRecord{record})
	if err != nil {
		return DeviceRecord{}, err
	}
	loaded, _ := registry.Lookup(record.ID)
	return loaded, nil
}

func TokenDigest(token []byte) [sha256.Size]byte { return sha256.Sum256(token) }
