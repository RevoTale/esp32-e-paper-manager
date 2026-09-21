package engine

func (p *painter) paintInlineBox(e *element, clips []clip) error {
	width, height, offset := 0.0, 0.0, 0.0
	for _, area := range e.fragments {
		width += area.Width
		height = max(height, area.Height)
	}
	for _, fragment := range e.fragments {
		// Paint an unbroken horizontal strip through each fragment's own clip.
		// This slices side borders, radii and background positioning together,
		// without cloning decorations at every line break or allocating a strip.
		// https://www.w3.org/TR/css-break-3/#joining-boxes
		area := fragment
		area.X -= offset
		if !inlineStartsLeft(e) {
			area.X = fragment.Right() - width + offset
		}
		area.Width = width
		if err := p.paintDecoration(e, area, height, append(clips, clip{area: fragment})); err != nil {
			return err
		}
		offset += fragment.Width
	}
	return nil
}
