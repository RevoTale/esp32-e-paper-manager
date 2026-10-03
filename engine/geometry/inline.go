package geometry

import "github.com/RevoTale/esp32-e-paper-manager/engine/style"

// ResolveInline resolves decorations, not width/height, on non-replaced inline
// boxes. Vertical decorations do not enlarge the line-height strut.
// https://www.w3.org/TR/CSS22/visudet.html#inline-non-replaced
func ResolveInline(c style.Computed, reference Reference) (Box, error) {
	if !nonnegative(reference.Width) || !nonnegative(reference.Height) {
		return Box{}, style.ErrValue
	}
	b := Box{reference: reference}
	if err := b.resolveEdges(c); err != nil {
		return Box{}, err
	}
	return b, nil
}
