package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/engine/dom"
	"github.com/RevoTale/esp32-e-paper-manager/renderdiag"
)

func assetURL(t testing.TB, width, height int) string {
	t.Helper()
	source := image.NewNRGBA(image.Rect(0, 0, width, height))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func TestAssetsKeepAlphaAndShareImmutableDecodedPixels(t *testing.T) {
	url := assetURL(t, 2, 2)
	doc, err := dom.Parse([]byte(`<div id="a" style="background-image:url(` + url + `)"><img id="b" src="` + url + `"></div>`))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := loadAssets(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	a, b := assets.backgrounds[doc.ByID["a"]], assets.images[doc.ByID["b"]]
	if a != b || assets.pixels != 8 || color.NRGBAModel.Convert(a.At(0, 0)) != (color.NRGBA{R: 255, A: 128}) {
		t.Fatal("lost alpha/cache identity/scene accounting")
	}
}

func TestAssetSceneBoundsRejectBeforePainting(t *testing.T) {
	for _, source := range []string{
		`<img>`, `<img src="data:image/png;base64,AAAA">`,
		strings.Repeat(`<img src="`+assetURL(t, 1, 1)+`">`, 33),
		strings.Repeat(`<img src="`+assetURL(t, 768, 768)+`">`, 2),
	} {
		doc, err := dom.Parse([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadAssets(context.Background(), doc); !errors.Is(err, renderdiag.ErrRejected) {
			t.Fatalf("expected scene rejection, got %v", err)
		}
	}
}

func TestAssetLoadingHonorsCancelledScene(t *testing.T) {
	doc, err := dom.Parse([]byte("<div></div>"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := loadAssets(ctx, doc); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
