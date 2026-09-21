package screenwire

import (
	"encoding/binary"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

// BeginRefresh extends Begin; Commit and Query keep the original digest identity.
// The receiver rejects consumed IDs regardless of options. Cadence is selected
// before staging and cannot be changed by replaying Begin or Commit.
func EncodeRefreshBegin(digest [32]byte, options refreshpolicy.Options, policy refreshpolicy.Policy) ([44]byte, error) {
	var b [44]byte
	if options.Priority > refreshpolicy.Urgent || options.Mode > refreshpolicy.Full ||
		!wireInterval(policy.Normal) || !wireInterval(policy.Urgent) {
		return b, ErrRecord
	}
	copy(b[:32], digest[:])
	b[32], b[33] = byte(options.Priority), byte(options.Mode)
	binary.LittleEndian.PutUint32(b[36:40], uint32(policy.Normal/time.Millisecond))
	binary.LittleEndian.PutUint32(b[40:44], uint32(policy.Urgent/time.Millisecond))
	return b, nil
}

func wireInterval(d time.Duration) bool {
	return d >= time.Millisecond && d%time.Millisecond == 0 && d/time.Millisecond <= 1<<32-1
}

func DecodeRefreshBegin(b []byte) ([32]byte, refreshpolicy.Options, refreshpolicy.Policy, error) {
	var digest [32]byte
	var options refreshpolicy.Options
	var policy refreshpolicy.Policy
	if len(b) != 44 || b[34] != 0 || b[35] != 0 {
		return digest, options, policy, ErrRecord
	}
	copy(digest[:], b[:32])
	options = refreshpolicy.Options{Priority: refreshpolicy.Priority(b[32]), Mode: refreshpolicy.Mode(b[33])}
	policy = refreshpolicy.Policy{Normal: time.Duration(binary.LittleEndian.Uint32(b[36:40])) * time.Millisecond,
		Urgent: time.Duration(binary.LittleEndian.Uint32(b[40:44])) * time.Millisecond}
	_, err := EncodeRefreshBegin(digest, options, policy)
	return digest, options, policy, err
}
