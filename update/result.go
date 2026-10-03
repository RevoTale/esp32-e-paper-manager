package update

import (
	"errors"
	"fmt"
)

const maxLimitNameLength = 32

var (
	ErrRejected = errors.New("update: rejected")
	ErrResult   = errors.New("update: invalid result")
)

type Stage uint8

const (
	StageReceive Stage = 1 + iota
	StageAuthenticate
	StageDecode
	StageParse
	StageStyle
	StageLayout
	StageRaster
	StageRefresh
)

type Code uint8

const (
	CodeInvalid Code = 1 + iota
	CodeUnsupported
	CodeLimit
	CodeConflict
	CodeCancelled
	CodeStalled
	CodeUnavailable
	CodeInternal
)

type Diagnostic struct {
	Stage    Stage
	Code     Code
	Limit    string
	Observed int
	Maximum  int
	Node     uint16
}

func (d Diagnostic) Validate() error {
	if d.Stage < StageReceive || d.Stage > StageRefresh || d.Code < CodeInvalid || d.Code > CodeInternal {
		return ErrResult
	}
	if d.Limit == "" {
		return validateNoLimit(d)
	}
	return validateLimit(d)
}

func validateNoLimit(d Diagnostic) error {
	if d.Observed != 0 || d.Maximum != 0 {
		return ErrResult
	}
	return nil
}

func validateLimit(d Diagnostic) error {
	if len(d.Limit) > maxLimitNameLength || d.Maximum <= 0 || d.Observed < 0 || !validLimitName(d.Limit) {
		return ErrResult
	}
	return nil
}

func validLimitName(value string) bool {
	for index := range len(value) {
		character := value[index]
		if character != '_' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return len(value) > 0
}

type Error struct {
	Diagnostic Diagnostic
	Cause      error
}

func (e *Error) Error() string {
	if e == nil {
		return "update: rejected"
	}
	diagnostic := e.Diagnostic
	if diagnostic.Limit != "" {
		return fmt.Sprintf("update: %s %s %s observed=%d maximum=%d",
			diagnostic.Stage, diagnostic.Code, diagnostic.Limit,
			diagnostic.Observed, diagnostic.Maximum)
	}
	return fmt.Sprintf("update: %s %s", diagnostic.Stage, diagnostic.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) Is(target error) bool { return target == ErrRejected }

func (s Stage) String() string {
	names := [...]string{"", "receive", "authenticate", "decode", "parse", "style", "layout", "raster", "refresh"}
	if s == 0 || int(s) >= len(names) {
		return "unknown"
	}
	return names[s]
}

func (c Code) String() string {
	names := [...]string{"", "invalid", "unsupported", "limit", "conflict", "cancelled", "stalled", "unavailable", "internal"}
	if c == 0 || int(c) >= len(names) {
		return "unknown"
	}
	return names[c]
}

type Status uint8

const (
	StatusAccepted Status = 1 + iota
	StatusUnchanged
	StatusRefreshed
	StatusRejected
)

type Result struct {
	ID          ID
	Status      Status
	ContentHash Digest
	Diagnostic  Diagnostic
}

func (r Result) Validate() error {
	if zeroID(r.ID) || r.Status < StatusAccepted || r.Status > StatusRejected {
		return ErrResult
	}
	if r.Status == StatusRejected {
		return r.Diagnostic.Validate()
	}
	if r.ContentHash == (Digest{}) || r.Diagnostic != (Diagnostic{}) {
		return ErrResult
	}
	return nil
}
