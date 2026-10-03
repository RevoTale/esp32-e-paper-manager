package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

// Cancel at the post-acquisition Err check, not before select: two ready cases
// are chosen randomly (https://go.dev/ref/spec#Select_statements).
// This context models cancellation at that boundary without sleeps or races.
type cancelAfterAdmission struct {
	context.Context
	cancel context.CancelFunc
	gate   chan struct{}
	seen   bool
}

func (c *cancelAfterAdmission) Err() error {
	if len(c.gate) == 1 {
		c.seen = true
		c.cancel()
	}
	return c.Context.Err()
}

func TestRendererCancellationAfterAdmission(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelAfterAdmission{Context: base, cancel: cancel, gate: r.gate}
	// Cancellation must win before parsing this otherwise rejected document.
	result, err := r.RenderRGBA(ctx, display.Size{Width: 20, Height: 20}, []byte("<script>bad</script>"))
	if !ctx.seen || !errors.Is(err, context.Canceled) || result.Image != nil {
		t.Fatalf("post-admission cancellation: seen=%t, pixels=%t, err=%v", ctx.seen, result.Image != nil, err)
	}
	if len(r.gate) != 0 {
		t.Fatal("cancelled renderer retained admission")
	}
	if _, err := r.RenderRGBA(context.Background(), display.Size{Width: 20, Height: 20}, []byte("ok")); err != nil {
		t.Fatal(err)
	}
}
