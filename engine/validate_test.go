package engine

import (
	"context"
	"testing"
)

func TestDocumentValidation(t *testing.T) {
	r := &Renderer{}
	if err := r.ValidateDocument(context.Background(), []byte(`<p>ok</p>`)); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{`<script>x</script>`, `<img src="data:image/png;base64,YmFk">`} {
		if err := r.ValidateDocument(context.Background(), []byte(source)); err == nil {
			t.Fatal("invalid accepted", source)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.ValidateDocument(ctx, []byte(`<p>ok</p>`)); err != context.Canceled {
		t.Fatal(err)
	}
}
