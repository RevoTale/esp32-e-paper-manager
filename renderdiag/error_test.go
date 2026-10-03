package renderdiag

import (
	"errors"
	"testing"
)

func TestDiagnosticIsStableAndSourceFree(t *testing.T) {
	d := &Error{Code: InvalidValue, Source: Inline, Element: 5, Start: 6, End: 12}
	if !errors.Is(d, ErrRejected) {
		t.Fatal("missing rejection classification")
	}
	if d.Error() != "render: rejected inline code=8 element=5 bytes=6..12" {
		t.Fatal(d)
	}
}
