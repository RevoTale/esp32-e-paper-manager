package manager

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

type documentValidator interface {
	ValidateDocument(context.Context, []byte) error
}

var errPatchUnsupported = errors.New("manager: renderer cannot validate authoring edits")

func (a *ScreenAPI) patch(w http.ResponseWriter, r *http.Request, options refreshpolicy.Options) {
	base, ok := a.mutationBase(w, r)
	if !ok {
		return
	}
	body, status, err := screenBody(w, r, "application/json")
	if err != nil {
		writeError(w, status, "invalid_document")
		return
	}
	edits, err := dom.DecodeEdits(body)
	if err != nil {
		writeError(w, 400, "invalid_document")
		return
	}
	revision, err := a.screen.patchOptionsAt(r.Context(), base, edits, a.screen.elapsed, options)
	writeMutation(w, revision, err)
}

func (s *Screen) patchAt(ctx context.Context, base renderbatch.Revision, edits []dom.Edit, now func() time.Duration) (renderbatch.Revision, error) {
	return s.patchOptionsAt(ctx, base, edits, now, refreshpolicy.Options{})
}

func (s *Screen) patchOptionsAt(ctx context.Context, base renderbatch.Revision, edits []dom.Edit, now func() time.Duration, options refreshpolicy.Options) (renderbatch.Revision, error) {
	validator, ok := s.renderer.(documentValidator)
	if !ok {
		return 0, errPatchUnsupported
	}
	s.mu.Lock()
	markup, current := s.markup, s.state.Current
	s.mu.Unlock()
	if base != current {
		return 0, ErrConflict
	}
	updated, err := dom.Apply([]byte(markup), edits)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := validator.ValidateDocument(ctx, updated); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	// Recheck base inside submitAt after parsing outside the lock: a concurrent
	// writer wins or loses as one atomic scene, never as individual DOM edits.
	return s.submitOptionsAt(base, updated, now, options)
}
