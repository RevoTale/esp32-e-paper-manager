package screenwire

import "encoding/binary"

const PanelStatusSize = 36

// BusySamples counts actual reads, not edges or independently measured voltage.
type BusySamples struct{ Samples, LowSamples uint32 }

// PanelStatus is a boot-local, cached last-cycle trace, never pixel proof.
// State: unavailable=0, active=1, completed=2, stopped/error=3. Phase/Step
// belong to the negotiated panel profile. Waits are power-on, refresh, power-off.
type PanelStatus struct {
	Version, State, Phase, Step uint8
	Cycle, ElapsedMS            uint32
	Waits                       [3]BusySamples
}

func (p PanelStatus) valid() bool {
	if p.Version != 1 || p.State > 3 {
		return false
	}
	if p.State == 0 {
		return p == (PanelStatus{Version: 1})
	}
	if p.Cycle == 0 {
		return false
	}
	for _, w := range p.Waits {
		if w.LowSamples > w.Samples {
			return false
		}
	}
	return true
}

func EncodePanelStatus(dst []byte, p PanelStatus) error {
	if len(dst) != PanelStatusSize || !p.valid() {
		return ErrRecord
	}
	dst[0], dst[1], dst[2], dst[3] = p.Version, p.State, p.Phase, p.Step
	binary.LittleEndian.PutUint32(dst[4:8], p.Cycle)
	binary.LittleEndian.PutUint32(dst[8:12], p.ElapsedMS)
	for i, w := range p.Waits {
		binary.LittleEndian.PutUint32(dst[12+i*8:16+i*8], w.Samples)
		binary.LittleEndian.PutUint32(dst[16+i*8:20+i*8], w.LowSamples)
	}
	return nil
}

func DecodePanelStatus(src []byte) (PanelStatus, error) {
	if len(src) != PanelStatusSize {
		return PanelStatus{}, ErrRecord
	}
	p := PanelStatus{Version: src[0], State: src[1], Phase: src[2], Step: src[3],
		Cycle: binary.LittleEndian.Uint32(src[4:8]), ElapsedMS: binary.LittleEndian.Uint32(src[8:12])}
	for i := range p.Waits {
		p.Waits[i] = BusySamples{binary.LittleEndian.Uint32(src[12+i*8 : 16+i*8]), binary.LittleEndian.Uint32(src[16+i*8 : 20+i*8])}
	}
	if !p.valid() {
		return PanelStatus{}, ErrRecord
	}
	return p, nil
}
