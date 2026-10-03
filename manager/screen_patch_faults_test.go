package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
)

type validatingRenderer struct {
	apiRenderer
	validate func(context.Context, []byte) error
}

func (r validatingRenderer) ValidateDocument(ctx context.Context, source []byte) error {
	return r.validate(ctx, source)
}

func TestPatchAssetsAndConcurrentCAS(t *testing.T) {
	a, s := newScreenAPI(t)
	s.renderer = &engine.Renderer{}
	if _, err := s.Submit(0, []byte(`<div id="a">old</div>`), 0); err != nil {
		t.Fatal(err)
	}
	bad := `{"edits":[{"id":"a","html":"<img src=\"data:image/png;base64,YmFk\">"}]}`
	if w := patchRequest(a, a.tag(1), bad, "application/json"); w.Code != 400 || s.Status().Current != 1 {
		t.Fatal(w, s.Status())
	}
	s.renderer = validatingRenderer{validate: func(context.Context, []byte) error {
		_, err := s.submitAt(1, []byte(`<p>concurrent winner</p>`), s.elapsed)
		return err
	}}
	good := `{"edits":[{"id":"a","text":"loser"}]}`
	if w := patchRequest(a, a.tag(1), good, "application/json"); w.Code != 412 || s.Status().Current != 2 {
		t.Fatal(w, s.Status())
	}
	if s.markup != `<p>concurrent winner</p>` {
		t.Fatal(s.markup)
	}
}

func TestPatchCancellation(t *testing.T) {
	_, s := newScreenAPI(t)
	if _, err := s.Submit(0, []byte(`<p id="a">old</p>`), 0); err != nil {
		t.Fatal(err)
	}
	text := "new"
	edits := []dom.Edit{{ID: "a", Text: &text}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.patchAt(ctx, 1, edits, s.elapsed); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.patchAt(context.Background(), 0, edits, s.elapsed); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if s.Status().Current != 1 {
		t.Fatal(s.Status())
	}
}

func TestPatchRequiresAssetValidator(t *testing.T) {
	a, s := newScreenAPI(t)
	s.renderer = screenRenderer(blackRenderer)
	if _, err := s.Submit(0, []byte(`<p id="a">old</p>`), 0); err != nil {
		t.Fatal(err)
	}
	patch := `{"edits":[{"id":"a","text":"new"}]}`
	if w := patchRequest(a, a.tag(1), patch, "application/json"); w.Code != 501 || s.Status().Current != 1 {
		t.Fatal(w, s.Status())
	}
}
