package document

import (
	"errors"
	"testing"
)

func TestNewSourceBorrowsBoundedUTF8(t *testing.T) {
	html := []byte("<h1>Стан</h1>")
	source, err := NewSource(Version1, ProfileDashboard, html)
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}
	if source.Version() != Version1 || source.Profile() != ProfileDashboard {
		t.Fatalf("source contract = version %d profile %d", source.Version(), source.Profile())
	}
	html[4] = 'X'
	if source.HTML()[4] != 'X' {
		t.Fatal("NewSource() copied caller bytes")
	}
}

func TestNewSourceRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		profile Profile
		html    []byte
		want    error
	}{
		{name: "version", version: 9, profile: ProfileDashboard, html: []byte("x"), want: ErrVersion},
		{name: "profile", version: Version1, profile: 9, html: []byte("x"), want: ErrProfile},
		{name: "empty", version: Version1, profile: ProfileDashboard, want: ErrDocument},
		{name: "too large", version: Version1, profile: ProfileDashboard, html: make([]byte, MaxEncodedBytes+1), want: ErrDocument},
		{name: "utf8", version: Version1, profile: ProfileDashboard, html: []byte{0xff}, want: ErrEncoding},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewSource(test.version, test.profile, test.html)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewSource() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLimitsValidate(t *testing.T) {
	limits := DashboardLimits()
	if err := limits.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	limits.MaxNodes = 0
	if err := limits.Validate(); !errors.Is(err, ErrLimits) {
		t.Fatalf("Validate() error = %v, want ErrLimits", err)
	}
	limits = DashboardLimits()
	limits.MaxWork++
	if err := limits.Validate(); !errors.Is(err, ErrLimits) {
		t.Fatalf("Validate(over maximum) error = %v, want ErrLimits", err)
	}
}
