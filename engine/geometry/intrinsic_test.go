package geometry

import "testing"

func TestIntrinsicContributionsIncludeEdgesAndLimits(t *testing.T) {
	for _, tc := range []struct {
		css  string
		want Intrinsic
	}{
		{"", Intrinsic{20, 100}}, {"width:60px;padding:3px;border:1px solid", Intrinsic{68, 68}},
		{"width:60px;box-sizing:border-box;padding:3px;border:1px solid", Intrinsic{60, 60}},
		{"width:50%;padding:10%;max-width:50%", Intrinsic{20, 100}},
		{"min-width:40px;max-width:80px", Intrinsic{40, 80}},
		{"display:inline;width:60px;min-width:60px;padding:3px", Intrinsic{26, 106}},
		{"box-sizing:border-box;min-width:60px;max-width:80px;padding:3px", Intrinsic{60, 80}},
		{"margin-left:-50px", Intrinsic{0, 50}},
	} {
		got, err := IntrinsicContribution(boxStyle(t, tc.css), Reference{Width: 300}, Intrinsic{20, 100})
		if err != nil || got != tc.want {
			t.Errorf("%s: %+v %v, want %+v", tc.css, got, err, tc.want)
		}
	}
}

func TestIntrinsicRejectsUnboundedContributions(t *testing.T) {
	for _, css := range []string{"padding:8192em", "width:8192em", "min-width:8192em", "min-width:90px;max-width:80px"} {
		if _, err := IntrinsicContribution(boxStyle(t, css), Reference{Width: 200}, Intrinsic{20, 100}); err == nil {
			t.Errorf("accepted %s", css)
		}
	}
	if _, err := IntrinsicContribution(boxStyle(t, "padding:8000px"), Reference{}, Intrinsic{20000, 25000}); err == nil {
		t.Fatal("unbounded final contribution")
	}
}
