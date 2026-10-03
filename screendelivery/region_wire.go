package screendelivery

import (
	"crypto/sha256"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/screenwire"
)

// Wire binds final crops to the transport's confirmed transaction identity.
// Validate signed coordinates before narrowing them to the EPS2 u16 fields.
// Peer viewport/capability validation remains the screen client's responsibility.
func (p RegionPlan) Wire(base [32]byte, options refreshpolicy.Options, policy refreshpolicy.Policy) (screenwire.RegionBegin, error) {
	b := p.Bounds
	if options.Mode != refreshpolicy.Partial || !p.validWireShape() {
		return screenwire.RegionBegin{}, ErrRegionInput
	}
	r := screenwire.RegionBegin{Baseline: base, Old: sha256.Sum256(p.Old), New: sha256.Sum256(p.New),
		Left: uint16(b.Min.X), Top: uint16(b.Min.Y), Right: uint16(b.Max.X), Bottom: uint16(b.Max.Y),
		Priority: options.Priority, Policy: policy}
	_, err := screenwire.EncodeRegionBegin(r)
	return r, err
}

func (p RegionPlan) validWireShape() bool {
	b := p.Bounds
	return !b.Empty() && b.Min.X >= 0 && b.Min.Y >= 0 && b.Max.X <= 65535 && b.Max.Y <= 65535 &&
		b.Min.X%8 == 0 && b.Max.X%8 == 0 && len(p.Old) == b.Dx()/8*b.Dy() && len(p.New) == len(p.Old)
}
