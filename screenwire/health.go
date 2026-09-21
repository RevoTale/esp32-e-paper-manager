package screenwire

import "encoding/binary"

const HealthSize = 8
const HealthVersion uint8 = 1

// HealthStatus is a source-free network snapshot, not a panel diagnostic. State
// and LastFailure use the frozen 0..7 values documented in docs/eps2-wire.md.
// Failures counts unexpected failed attempts in this boot, saturating at 255.
type HealthStatus struct {
	Version, State, LastFailure, Failures uint8
	UptimeSeconds                         uint32
}

func (h HealthStatus) valid() bool {
	return h.Version == HealthVersion && h.State <= 7 && h.LastFailure <= 7
}

func EncodeHealth(dst []byte, h HealthStatus) error {
	if len(dst) != HealthSize || !h.valid() {
		return ErrRecord
	}
	dst[0], dst[1], dst[2], dst[3] = h.Version, h.State, h.LastFailure, h.Failures
	binary.LittleEndian.PutUint32(dst[4:8], h.UptimeSeconds)
	return nil
}

func DecodeHealth(src []byte) (HealthStatus, error) {
	if len(src) != HealthSize {
		return HealthStatus{}, ErrRecord
	}
	h := HealthStatus{Version: src[0], State: src[1], LastFailure: src[2], Failures: src[3], UptimeSeconds: binary.LittleEndian.Uint32(src[4:8])}
	if !h.valid() {
		return HealthStatus{}, ErrRecord
	}
	return h, nil
}
