package engine

import (
	"context"
	"errors"
	"image/color"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func TestRendererRejectsBeforeReturningPartialPixels(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, html := range []string{
		`<script>secret</script>`, `<div style="bad-property:1">secret</div>`,
		`<img src="data:image/png;base64,AAAA">`, `<div style="padding:8192em"></div>`,
		`<div style="height:8192em"></div>`, `<div><span style="font-size:1px"></span></div>`,
		`<div><div style="min-width:10px;max-width:1px"></div></div>`,
		`<div style="position:relative;left:8192em"></div>`,
		`<div style="position:relative;right:8192em"></div>`,
		`<div style="position:relative;top:8192em"></div>`,
		`<div style="position:relative;bottom:8192em"></div>`,
		`<div style="position:absolute;left:8192em"></div>`,
		`<div style="position:absolute;top:8192em"></div>`,
		`<div style="height:10px;border-radius:8192em"></div>`,
		`<div style="overflow:hidden;height:10px;border-radius:8192em"><div style="height:10px;background:red"></div></div>`,
	} {
		result, err := r.RenderRGBA(context.Background(), display.Size{Width: 100, Height: 100}, []byte(html))
		if !errors.Is(err, renderdiag.ErrRejected) || result.Image != nil {
			t.Errorf("%s: got pixels=%t, %v", html, result.Image != nil, err)
		}
	}
	if _, err := r.RenderRGBA(context.Background(), display.Size{}, []byte("")); !errors.Is(err, display.ErrFrameGeometry) {
		t.Fatalf("invalid viewport: %v", err)
	}
}

func TestRoundedClipAndRelativeTrailingInsets(t *testing.T) {
	img := rendered(t, 80, 80, `<div style="width:40px;height:40px;overflow:hidden;border-radius:20px"><div style="position:relative;right:5px;bottom:5px;width:80px;height:80px;background:black"></div></div>`)
	pixel(t, img, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	pixel(t, img, 20, 20, color.RGBA{A: 255})
	pixel(t, img, 45, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

func TestSceneResourceBudgetsRejectRatherThanDegrade(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		size display.Size
		html string
	}{
		{display.Size{Width: 100, Height: 100}, strings.Repeat(`<div style="opacity:.5">`, 9) + "text" + strings.Repeat("</div>", 9)},
		{display.Size{Width: 2048, Height: 512}, strings.Repeat(`<div style="position:absolute;left:0;right:0;top:0;bottom:0;background:black"></div>`, 35)},
		{display.Size{Width: 2048, Height: 512}, strings.Repeat(`<div style="opacity:.5">`, 8) + `<div style="width:2048px;height:512px;background:black"></div>` + strings.Repeat("</div>", 8)},
	} {
		result, err := r.RenderRGBA(context.Background(), tc.size, []byte(tc.html))
		var diagnostic *renderdiag.Error
		if !errors.As(err, &diagnostic) || diagnostic.Code != renderdiag.InputLimit || result.Image != nil {
			t.Fatalf("budget: %v, pixels=%t", err, result.Image != nil)
		}
	}
}

func TestRendererCancellationReleasesAdmission(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, busy := range []bool{false, true} {
		if busy {
			r.gate <- struct{}{}
		}
		if _, err := r.RenderRGBA(ctx, display.Size{Width: 20, Height: 20}, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: %v", err)
		}
		if busy {
			<-r.gate
		}
	}
	if _, err := r.RenderRGBA(context.Background(), display.Size{Width: 20, Height: 20}, []byte("ok")); err != nil {
		t.Fatal(err)
	}
}
