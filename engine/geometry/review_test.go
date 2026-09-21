package geometry

import "testing"

func TestInlineBlockAutoMarginsDoNotAbsorbFreeSpace(t *testing.T) {
	box, err := ResolveWidth(boxStyle(t, "display:inline-block;width:100px;margin:auto"), Reference{Width: 400}, Intrinsic{})
	if err != nil || box.Margin != (Edges{}) {
		t.Fatalf("inline-block auto margins: %+v %v", box.Margin, err)
	}
}

func TestGeometryResourceCeilingIsNotACSSMaximum(t *testing.T) {
	for _, source := range []string{
		"margin:0 -2048em", "width:0;margin-left:auto;margin-right:-2048em",
		"box-sizing:border-box;padding:10px;min-width:10px;max-width:5px",
	} {
		if _, err := ResolveWidth(boxStyle(t, source), Reference{Width: 400}, Intrinsic{}); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
	c := boxStyle(t, "box-sizing:border-box;padding:10px;min-height:10px;max-height:5px")
	box, err := ResolveWidth(c, Reference{Width: 400}, Intrinsic{})
	if err != nil {
		t.Fatal(err)
	}
	if err := box.ResolveHeight(c, 0); err == nil {
		t.Fatal("floored away incompatible height limits")
	}
}

func TestExplicitCSSMaximumCanConstrainLargeAutomaticWidth(t *testing.T) {
	box, err := ResolveWidth(boxStyle(t, "margin:0 -2048em;max-width:200px"), Reference{Width: 400}, Intrinsic{})
	if err != nil || box.ContentWidth != 200 {
		t.Fatalf("genuine CSS max should constrain: %+v %v", box, err)
	}
}
