// Package update defines the versioned HTML update contract shared by USB,
// the home manager, and the encrypted device link.
package update

import (
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/RevoTale/esp32-e-paper-manager/document"
)

const (
	Version           = 1
	DisplayTimeLength = 16
	MaxTimezoneLength = 64
)

var (
	ErrID       = errors.New("update: invalid id")
	ErrTime     = errors.New("update: invalid time")
	ErrTimezone = errors.New("update: invalid timezone")
)

type ID [16]byte
type Digest [sha256.Size]byte

type Request struct {
	id          ID
	utc         int64
	displayTime [DisplayTimeLength]byte
	timezone    [MaxTimezoneLength]byte
	timezoneLen uint8
	document    document.Source
	contentHash Digest
}

func NewRequest(id ID, utc int64, displayTime, timezone string, source document.Source) (Request, error) {
	request := Request{id: id, utc: utc, document: source}
	if zeroID(id) {
		return Request{}, ErrID
	}
	if utc <= 0 || !validDisplayTime(displayTime) {
		return Request{}, ErrTime
	}
	if !validTimezone(timezone) {
		return Request{}, ErrTimezone
	}
	if err := source.Validate(); err != nil {
		return Request{}, err
	}
	copy(request.displayTime[:], displayTime)
	copy(request.timezone[:], timezone)
	request.timezoneLen = uint8(len(timezone))
	request.contentHash = sha256.Sum256(source.HTML())
	return request, nil
}

func (r Request) Validate() error {
	rebuilt, err := NewRequest(r.id, r.utc, r.DisplayTime(), r.Timezone(), r.document)
	if err != nil {
		return err
	}
	if rebuilt.contentHash != r.contentHash {
		return ErrRejected
	}
	return nil
}

func (r Request) SameIntent(other Request) bool {
	return r.id == other.id && r.utc == other.utc &&
		r.displayTime == other.displayTime && r.timezone == other.timezone &&
		r.timezoneLen == other.timezoneLen && r.contentHash == other.contentHash &&
		r.document.Version() == other.document.Version() &&
		r.document.Profile() == other.document.Profile()
}

func (r Request) ID() ID                                    { return r.id }
func (r Request) UTC() int64                                { return r.utc }
func (r Request) DisplayTime() string                       { return string(r.displayTime[:]) }
func (r Request) DisplayTimeBytes() [DisplayTimeLength]byte { return r.displayTime }
func (r Request) Timezone() string                          { return string(r.timezone[:r.timezoneLen]) }
func (r Request) Document() document.Source                 { return r.document }
func (r Request) ContentHash() Digest                       { return r.contentHash }

func zeroID(id ID) bool {
	var combined byte
	for _, value := range id {
		combined |= value
	}
	return combined == 0
}

func validDisplayTime(value string) bool {
	if !displayTimeShape(value) {
		return false
	}
	year := numberAt(value, 0, 4)
	month := numberAt(value, 5, 2)
	day := numberAt(value, 8, 2)
	hour := numberAt(value, 11, 2)
	minute := numberAt(value, 14, 2)
	return validDate(year, month, day) && hour < 24 && minute < 60
}

func displayTimeShape(value string) bool {
	return len(value) == DisplayTimeLength && displayTimeSeparators(value) &&
		digitsAt(value, 0, 4) && digitsAt(value, 5, 2) && digitsAt(value, 8, 2) &&
		digitsAt(value, 11, 2) && digitsAt(value, 14, 2)
}

func displayTimeSeparators(value string) bool {
	return value[4] == '-' && value[7] == '-' && value[10] == ' ' && value[13] == ':'
}

func validDate(year, month, day int) bool {
	if year < 1970 || month < 1 || month > 12 || day < 1 {
		return false
	}
	days := [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	maximum := days[month-1]
	if month == 2 && (year%400 == 0 || year%4 == 0 && year%100 != 0) {
		maximum = 29
	}
	return day <= maximum
}

func digitsAt(value string, start, length int) bool {
	for index := start; index < start+length; index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func numberAt(value string, start, length int) int {
	number := 0
	for index := start; index < start+length; index++ {
		number = number*10 + int(value[index]-'0')
	}
	return number
}

func validTimezone(value string) bool {
	if len(value) == 0 || len(value) > MaxTimezoneLength || value[0] == '/' ||
		value[len(value)-1] == '/' || strings.Contains(value, "..") || strings.Contains(value, "//") {
		return false
	}
	for index := range len(value) {
		if !timezoneByte(value[index]) {
			return false
		}
	}
	return true
}

func timezoneByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' || strings.ContainsRune("_+-./", rune(value))
}
