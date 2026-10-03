package renderdiag

// Warning is bounded source-free evidence about a successfully rendered scene.
// Rejections use Error instead; callers can never mistake a partial failure for
// a successful target just because it has warnings.
type Warning struct {
	Code    WarningCode `json:"code"`
	Element uint32      `json:"element"`
}

type WarningCode string

const (
	Clipped         WarningCode = "clipped"
	MissingGlyph    WarningCode = "missing-glyph"
	ReservedOverlap WarningCode = "reserved-overlap"
)
