package engine

import (
	"context"
	"image"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

// Renderer owns its embedded fonts and serializes their mutable shaping caches.
// Documents cannot read host fonts/files or issue network/subprocess requests.
type Renderer struct {
	fonts fonts
	gate  chan struct{}
}

type Result struct {
	Image    *image.RGBA
	Warnings []renderdiag.Warning
}

func New() (*Renderer, error) {
	f, err := loadFonts()
	if err != nil {
		return nil, err
	}
	return &Renderer{fonts: f, gate: make(chan struct{}, 1)}, nil
}

// RenderRGBA returns independently owned pixels, composited before any device
// quantization. Admission is context-aware; callers cannot race the font cache.
func (r *Renderer) RenderRGBA(ctx context.Context, size display.Size, source []byte) (Result, error) {
	return r.RenderRGBAWithOptions(ctx, size, source, Options{})
}

// RenderRGBAWithOptions reserves only explicitly configured device metadata
// pixels. CSS viewport dimensions remain the complete logical display size.
func (r *Renderer) RenderRGBAWithOptions(ctx context.Context, size display.Size, source []byte, options Options) (Result, error) {
	if err := options.validate(size); err != nil {
		return Result{}, err
	}
	select {
	case r.gate <- struct{}{}:
		defer func() { <-r.gate }()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return r.renderDocument(ctx, size, source, options)
}

func (r *Renderer) renderDocument(ctx context.Context, size display.Size, source []byte, options Options) (Result, error) {
	target, err := newSurface(size)
	if err != nil {
		return Result{}, err
	}
	doc, err := dom.Parse(source)
	if err != nil {
		return Result{}, err
	}
	a, err := loadAssets(ctx, doc)
	if err != nil {
		return Result{}, err
	}
	s, err := layoutScene(ctx, size, doc, a, r.fonts)
	if err != nil {
		return Result{}, err
	}
	p := &painter{scene: s, target: target.image}
	if err := p.paintContext(s.root); err != nil {
		return Result{}, err
	}
	result := Result{Image: target.image, Warnings: s.warnings}
	if err := result.reserve(ctx, options.Reserved); err != nil {
		return Result{}, err
	}
	return result, nil
}
