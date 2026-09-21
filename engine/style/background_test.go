package style

import "testing"

func TestBackgroundShorthandIsAnAtomicReset(t *testing.T) {
	block, err := Parse("background-size:cover;background-position:right bottom;background-color:blue;background:white", 1)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := block.Get("background-position-x")
	size, _ := block.Get("background-size-x")
	bg, _ := block.Get("background-color")
	if x.Length.Value != 0 || size.Length.Unit != Auto || bg.Color.R != 255 {
		t.Fatalf("incomplete reset %+v %+v %+v", x, size, bg)
	}
	if _, err := Parse(`background:rgba(0, 0, 0, .5) url("data:image/png;base64,AAAA") no-repeat center / cover`, 1); err != nil {
		t.Fatal(err)
	}
}

func TestBackgroundRejectsDuplicateAndMultilayerValues(t *testing.T) {
	for _, raw := range []string{"red blue", "none url(data:image/png;base64,AAAA)", "red,blue", "center / cover / auto", "linear-gradient(black,white)", "0 0 0", "center / -1px", "no-repeat repeat"} {
		if _, err := Parse("background:"+raw, 1); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestParserResourceBudgetsAndEmptyStyle(t *testing.T) {
	if block, err := Parse("", 0); err != nil || block.Count() != 0 {
		t.Fatalf("empty: %+v %v", block, err)
	}
	for _, raw := range []string{
		repeat(";", 32769),
		repeat("width:0;", MaxDeclarations+1),
		"width:" + repeat("0 ", 300),
	} {
		if _, err := Parse(raw, 1); err == nil {
			t.Fatal("accepted over-budget style")
		}
	}
}

func repeat(value string, count int) string {
	result := make([]byte, 0, len(value)*count)
	for range count {
		result = append(result, value...)
	}
	return string(result)
}
