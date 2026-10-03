package screenwire

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

const RegionBeginSize = 148

// RegionBegin binds an old/new pixel pair to a confirmed transaction identity.
// Coordinates are half-open. Panel-specific minimum sizes remain adapter rules.
// See docs/eps2-region.md; this codec does not grant partial-refresh capability.
type RegionBegin struct {
	Baseline, Old, New       [32]byte
	Left, Top, Right, Bottom uint16
	Priority                 refreshpolicy.Priority
	Policy                   refreshpolicy.Policy
}

func (r RegionBegin) valid(width, height uint16) bool {
	return r.Baseline != [32]byte{} && r.Priority <= refreshpolicy.Urgent &&
		wireInterval(r.Policy.Normal) && wireInterval(r.Policy.Urgent) && r.geometry(width, height)
}

func (r RegionBegin) geometry(width, height uint16) bool {
	return r.Left < r.Right && r.Top < r.Bottom && r.Right <= width &&
		r.Bottom <= height && r.Left%8 == 0 && r.Right%8 == 0
}

func EncodeRegionBegin(r RegionBegin) ([RegionBeginSize]byte, error) {
	var b [RegionBeginSize]byte
	if !r.valid(65535, 65535) {
		return b, ErrRecord
	}
	copy(b[32:64], r.Baseline[:])
	copy(b[64:96], r.Old[:])
	copy(b[96:128], r.New[:])
	for i, v := range [...]uint16{r.Left, r.Top, r.Right, r.Bottom} {
		binary.LittleEndian.PutUint16(b[128+i*2:], v)
	}
	b[136] = byte(r.Priority)
	binary.LittleEndian.PutUint32(b[140:144], uint32(r.Policy.Normal/time.Millisecond))
	binary.LittleEndian.PutUint32(b[144:148], uint32(r.Policy.Urgent/time.Millisecond))
	digest := regionDigest(b[32:])
	copy(b[:32], digest[:])
	return b, nil
}

func DecodeRegionBegin(b []byte, width, height uint16) (RegionBegin, error) {
	if len(b) != RegionBeginSize || b[137] != 0 || b[138] != 0 || b[139] != 0 {
		return RegionBegin{}, ErrRecord
	}
	r := RegionBegin{Left: binary.LittleEndian.Uint16(b[128:130]), Top: binary.LittleEndian.Uint16(b[130:132]),
		Right: binary.LittleEndian.Uint16(b[132:134]), Bottom: binary.LittleEndian.Uint16(b[134:136]),
		Priority: refreshpolicy.Priority(b[136]),
		Policy: refreshpolicy.Policy{Normal: time.Duration(binary.LittleEndian.Uint32(b[140:144])) * time.Millisecond,
			Urgent: time.Duration(binary.LittleEndian.Uint32(b[144:148])) * time.Millisecond}}
	copy(r.Baseline[:], b[32:64])
	copy(r.Old[:], b[64:96])
	copy(r.New[:], b[96:128])
	digest := regionDigest(b[32:])
	if !r.valid(width, height) || !bytes.Equal(b[:32], digest[:]) {
		return RegionBegin{}, ErrRecord
	}
	return r, nil
}

func regionDigest(body []byte) [32]byte {
	const domain = "EPS2-region-v1\x00"
	var input [len(domain) + RegionBeginSize - 32]byte
	copy(input[:], domain)
	copy(input[len(domain):], body)
	return sha256.Sum256(input[:])
}
