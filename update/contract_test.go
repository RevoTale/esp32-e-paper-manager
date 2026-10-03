package update

import (
	"errors"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/document"
)

func testRequest(t *testing.T) Request {
	t.Helper()
	source, err := document.NewSource(document.Version1, document.ProfileDashboard, []byte("<p>Ready</p>"))
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}
	id := ID{1, 2, 3, 4}
	request, err := NewRequest(id, 1_788_347_640, "2026-09-02 12:34", "Europe/Kyiv", source)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	return request
}

func TestNewRequestCreatesStableIntent(t *testing.T) {
	request := testRequest(t)
	if request.ContentHash() == (Digest{}) {
		t.Fatal("ContentHash() is zero")
	}
	if !request.SameIntent(request) {
		t.Fatal("SameIntent() rejected identical request")
	}
	changed, err := NewRequest(request.ID(), request.UTC(), "2026-09-02 12:35", request.Timezone(), request.Document())
	if err != nil {
		t.Fatalf("NewRequest(changed) error = %v", err)
	}
	if request.SameIntent(changed) {
		t.Fatal("SameIntent() accepted changed display time")
	}
}

func TestRequestValidateDetectsBorrowedContentMutation(t *testing.T) {
	request := testRequest(t)
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	request.Document().HTML()[0] = 'X'
	if err := request.Validate(); !errors.Is(err, ErrRejected) {
		t.Fatalf("Validate(mutated) error = %v, want ErrRejected", err)
	}
}

func TestNewRequestRejectsInvalidMetadata(t *testing.T) {
	request := testRequest(t)
	tests := []struct {
		name        string
		id          ID
		utc         int64
		displayTime string
		timezone    string
		want        error
	}{
		{name: "zero id", utc: request.UTC(), displayTime: request.DisplayTime(), timezone: request.Timezone(), want: ErrID},
		{name: "utc", id: request.ID(), displayTime: request.DisplayTime(), timezone: request.Timezone(), want: ErrTime},
		{name: "format", id: request.ID(), utc: request.UTC(), displayTime: "02.09.2026 12:34", timezone: request.Timezone(), want: ErrTime},
		{name: "calendar", id: request.ID(), utc: request.UTC(), displayTime: "2026-02-30 12:34", timezone: request.Timezone(), want: ErrTime},
		{name: "month", id: request.ID(), utc: request.UTC(), displayTime: "2026-13-01 12:34", timezone: request.Timezone(), want: ErrTime},
		{name: "day zero", id: request.ID(), utc: request.UTC(), displayTime: "2026-01-00 12:34", timezone: request.Timezone(), want: ErrTime},
		{name: "valid leap day", id: request.ID(), utc: request.UTC(), displayTime: "2024-02-29 12:34", timezone: request.Timezone()},
		{name: "timezone", id: request.ID(), utc: request.UTC(), displayTime: request.DisplayTime(), timezone: "../Kyiv", want: ErrTimezone},
		{name: "empty timezone", id: request.ID(), utc: request.UTC(), displayTime: request.DisplayTime(), want: ErrTimezone},
		{name: "trailing slash", id: request.ID(), utc: request.UTC(), displayTime: request.DisplayTime(), timezone: "Europe/", want: ErrTimezone},
		{name: "double slash", id: request.ID(), utc: request.UTC(), displayTime: request.DisplayTime(), timezone: "Europe//Kyiv", want: ErrTimezone},
		{name: "timezone byte", id: request.ID(), utc: request.UTC(), displayTime: request.DisplayTime(), timezone: "Europe Kyiv", want: ErrTimezone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRequest(test.id, test.utc, test.displayTime, test.timezone, request.Document())
			if !errors.Is(err, test.want) {
				t.Fatalf("NewRequest() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDiagnosticErrorIsTyped(t *testing.T) {
	err := &Error{Diagnostic: Diagnostic{
		Stage:    StageDecode,
		Code:     CodeLimit,
		Limit:    "html_bytes",
		Observed: document.MaxEncodedBytes + 1,
		Maximum:  document.MaxEncodedBytes,
	}}
	if !errors.Is(err, ErrRejected) {
		t.Fatal("errors.Is(error, ErrRejected) = false")
	}
	if got := err.Error(); got != "update: decode limit html_bytes observed=32769 maximum=32768" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestDiagnosticErrorWrapsCauseAndNamesUnknownValues(t *testing.T) {
	cause := errors.New("transport closed")
	err := &Error{Diagnostic: Diagnostic{Stage: StageReceive, Code: CodeCancelled}, Cause: cause}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(error, cause) = false")
	}
	if got := err.Error(); got != "update: receive cancelled" {
		t.Fatalf("Error() = %q", got)
	}
	if got := (*Error)(nil).Error(); got != "update: rejected" {
		t.Fatalf("nil Error() = %q", got)
	}
	if got := Stage(0).String(); got != "unknown" {
		t.Fatalf("Stage(0).String() = %q", got)
	}
	if got := Code(255).String(); got != "unknown" {
		t.Fatalf("Code(255).String() = %q", got)
	}
	if cause := (*Error)(nil).Unwrap(); cause != nil {
		t.Fatalf("nil Unwrap() = %v", cause)
	}
}

func TestResultRejectsInconsistentStates(t *testing.T) {
	request := testRequest(t)
	valid := Result{ID: request.ID(), Status: StatusRefreshed, ContentHash: request.ContentHash()}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	valid.ContentHash = Digest{}
	if err := valid.Validate(); !errors.Is(err, ErrResult) {
		t.Fatalf("Validate(zero hash) error = %v, want ErrResult", err)
	}
	rejected := Result{
		ID: request.ID(), Status: StatusRejected,
		Diagnostic: Diagnostic{Stage: StageParse, Code: CodeLimit, Limit: "nodes", Observed: 257, Maximum: 256},
	}
	if err := rejected.Validate(); err != nil {
		t.Fatalf("Validate(rejected) error = %v", err)
	}
	rejected.Diagnostic.Limit = "bad limit name"
	if err := rejected.Validate(); !errors.Is(err, ErrResult) {
		t.Fatalf("Validate(bad diagnostic) error = %v, want ErrResult", err)
	}
}

func TestDiagnosticAndResultValidationMatrix(t *testing.T) {
	request := testRequest(t)
	tests := []struct {
		name string
		data Diagnostic
		want error
	}{
		{name: "without limit", data: Diagnostic{Stage: StageReceive, Code: CodeCancelled}},
		{name: "bad stage", data: Diagnostic{Code: CodeCancelled}, want: ErrResult},
		{name: "bad code", data: Diagnostic{Stage: StageReceive}, want: ErrResult},
		{name: "values without limit", data: Diagnostic{Stage: StageReceive, Code: CodeCancelled, Observed: 1}, want: ErrResult},
		{name: "negative observed", data: Diagnostic{Stage: StageParse, Code: CodeLimit, Limit: "nodes", Observed: -1, Maximum: 1}, want: ErrResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.data.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
	results := []Result{
		{Status: StatusAccepted, ContentHash: request.ContentHash()},
		{ID: request.ID(), ContentHash: request.ContentHash()},
		{ID: request.ID(), Status: StatusAccepted, ContentHash: request.ContentHash(), Diagnostic: Diagnostic{Stage: StageReceive}},
	}
	for index, result := range results {
		if err := result.Validate(); !errors.Is(err, ErrResult) {
			t.Fatalf("result %d error = %v, want ErrResult", index, err)
		}
	}
}
