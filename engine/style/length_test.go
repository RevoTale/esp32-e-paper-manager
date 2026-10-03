package style

import (
	"math"
	"testing"
)

func TestLengthsUsePropertySpecificReference(t *testing.T) {
	ctx := Metrics{Containing: 400, Font: 20, RootFont: 16, ViewportWidth: 800, ViewportHeight: 480}
	for _, tc := range []struct {
		input string
		want  float64
	}{
		{"0", 0}, {"12px", 12}, {"-2.5px", -2.5}, {"50%", 200},
		{"1.5em", 30}, {"2rem", 32}, {"10vw", 80}, {"25vh", 120},
		{"1e1px", 10}, {"2PX", 2},
	} {
		t.Run(tc.input, func(t *testing.T) {
			length, err := ParseLength(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := length.Resolve(ctx)
			if err != nil || got != tc.want {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestLengthRejectsUnboundedOrUnsupportedInput(t *testing.T) {
	for _, input := range []string{"", "1", "NaNpx", "Infpx", "1e99px", "8193px", "-8193%", "1pt", "calc(1px)", "1 px", "0x1px", "1_0px"} {
		if _, err := ParseLength(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
	for _, value := range []Length{{Unit: Auto}, {Unit: None}, {Unit: 99}, {Unit: Px, Value: math.Inf(1)}} {
		if _, err := value.Resolve(Metrics{}); err == nil {
			t.Errorf("resolved invalid %+v", value)
		}
	}
	value, err := ParseLength("8192em")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.Resolve(Metrics{Font: 128}); err == nil {
		t.Fatal("accepted unbounded resolved coordinate")
	}
}

func TestAutoAndNoneRemainTyped(t *testing.T) {
	for _, tc := range []struct {
		input string
		unit  Unit
	}{{"auto", Auto}, {"none", None}} {
		value, err := ParseLength(tc.input)
		if err != nil || value.Unit != tc.unit {
			t.Fatalf("%s: %+v %v", tc.input, value, err)
		}
	}
}
