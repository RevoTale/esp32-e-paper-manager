package engine

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
)

//go:embed testdata/dashboard.html
var dashboard []byte

func BenchmarkDashboardWarm(b *testing.B) {
	r, err := New()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := r.Render(context.Background(), display.Size{Width: 800, Height: 480}, dashboard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDashboardCold(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		r, err := New()
		if err != nil {
			b.Fatal(err)
		}
		if _, err = r.Render(context.Background(), display.Size{Width: 800, Height: 480}, dashboard); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzRenderer(f *testing.F) {
	for _, source := range []string{`<p>Привіт</p>`, `<div style="width:1e300px">x`, `<span style="padding:2px">abc</span>`, `<div style="opacity:0.5"><img src="x"></div>`, `<pre>a\tb\ncc</pre>`} {
		f.Add([]byte(source), uint8(64), uint8(48))
	}
	r, err := New()
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, source []byte, width, height uint8) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		size := display.Size{Width: 1 + int(width), Height: 1 + int(height)}
		frame, err := r.Render(ctx, size, source)
		if err == nil && (frame.Size() != size || frame.Stride() != (size.Width+7)/8) {
			t.Fatal("invalid successful frame")
		}
	})
}
