package engine

import (
	"context"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
)

// ValidateDocument checks the complete scene and embedded asset bounds before
// an atomic authoring edit commits. It does not paint or touch the font cache;
// layout failures still report asynchronously through render diagnostics.
func (r *Renderer) ValidateDocument(ctx context.Context, source []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	doc, err := dom.Parse(source)
	if err != nil {
		return err
	}
	_, err = loadAssets(ctx, doc)
	return err
}
