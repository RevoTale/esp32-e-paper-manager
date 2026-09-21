package refreshpolicy

import (
	"testing"
	"time"
)

func TestMetadataDefaultsAndExplicitModes(t *testing.T) {
	for _, priority := range []string{"", "normal", "urgent"} {
		for _, mode := range []string{"", "auto", "full", "partial"} {
			o, err := Parse(priority, mode)
			if err != nil {
				t.Fatal(err)
			}
			if (o.Priority == Urgent) != (priority == "urgent") {
				t.Fatal(o)
			}
			if (o.Mode == Partial) != (mode == "partial") {
				t.Fatal(o)
			}
		}
	}
	for _, input := range []string{"URGENT", "normal,urgent", " urgent", "unknown"} {
		if _, err := Parse(input, "auto"); err == nil {
			t.Fatal(input)
		}
		if _, err := Parse("normal", input); err == nil {
			t.Fatal(input)
		}
	}
}

func TestUrgencyHasSeparateBoundedCadence(t *testing.T) {
	p := Policy{Normal: 180 * time.Second, Urgent: 5 * time.Second}
	for _, tc := range []struct {
		priority      Priority
		elapsed, want time.Duration
	}{
		{Normal, time.Second, 179 * time.Second},
		{Urgent, time.Second, 4 * time.Second},
		{Urgent, 5 * time.Second, 0},
		{Normal, 200 * time.Second, 0},
	} {
		got, err := p.Wait(tc.priority, tc.elapsed)
		if err != nil || got != tc.want {
			t.Fatal(got, err, tc)
		}
	}
	for _, p := range []Policy{{}, {Normal: time.Second}, {Normal: time.Second, Urgent: -1}} {
		if _, err := p.Wait(Normal, 0); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
	if _, err := p.Wait(Priority(9), 0); err == nil {
		t.Fatal("invalid priority")
	}
	if _, err := p.Wait(Urgent, -1); err == nil {
		t.Fatal("negative elapsed")
	}
}
