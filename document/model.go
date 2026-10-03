// Package document defines the bounded input contract shared by every update
// transport and the on-device renderer.
package document

import (
	"errors"
	"unicode/utf8"
)

const (
	MaxEncodedBytes = 32 * 1024
	MaxNodes        = 256
	MaxDepth        = 16
	MaxAttributes   = 8
	MaxStyles       = 512
	MaxTextBytes    = 24 * 1024
	MaxWork         = 4 * MaxEncodedBytes
)

var (
	ErrDocument = errors.New("document: invalid size")
	ErrEncoding = errors.New("document: invalid UTF-8")
	ErrLimits   = errors.New("document: invalid limits")
	ErrProfile  = errors.New("document: unsupported profile")
	ErrVersion  = errors.New("document: unsupported version")
)

type Version uint8

const Version1 Version = 1

type Profile uint8

const ProfileDashboard Profile = 1

// Limits bounds parser, style, layout, and asset work. V2-04 selects the
// measured values; transports always enforce MaxEncodedBytes first.
type Limits struct {
	MaxEncodedBytes      int
	MaxNodes             int
	MaxDepth             int
	MaxAttributesPerNode int
	MaxStyleDeclarations int
	MaxTextBytes         int
	MaxAssetBytes        int
	MaxWork              int
}

func (l Limits) Validate() error {
	if l.MaxEncodedBytes <= 0 || l.MaxEncodedBytes > MaxEncodedBytes ||
		!l.validStructure() || !l.validContent() {
		return ErrLimits
	}
	return nil
}

func (l Limits) validStructure() bool {
	return l.MaxNodes > 0 && l.MaxNodes <= MaxNodes &&
		l.MaxDepth > 0 && l.MaxDepth <= MaxDepth &&
		l.MaxAttributesPerNode > 0 && l.MaxAttributesPerNode <= MaxAttributes &&
		l.MaxStyleDeclarations > 0 && l.MaxStyleDeclarations <= MaxStyles
}

func (l Limits) validContent() bool {
	return l.MaxTextBytes > 0 && l.MaxTextBytes <= l.MaxEncodedBytes &&
		l.MaxAssetBytes >= 0 && l.MaxAssetBytes <= l.MaxEncodedBytes &&
		l.MaxWork > 0 && l.MaxWork <= MaxWork
}

func DashboardLimits() Limits {
	return Limits{
		MaxEncodedBytes: MaxEncodedBytes, MaxNodes: MaxNodes, MaxDepth: MaxDepth,
		MaxAttributesPerNode: MaxAttributes, MaxStyleDeclarations: MaxStyles,
		MaxTextBytes: MaxTextBytes, MaxAssetBytes: 0, MaxWork: MaxWork,
	}
}

// Source borrows one caller-owned UTF-8 HTML byte slice. The caller keeps it
// stable until parsing and rendering finish.
type Source struct {
	version Version
	profile Profile
	html    []byte
}

func NewSource(version Version, profile Profile, html []byte) (Source, error) {
	source := Source{version: version, profile: profile, html: html}
	if err := source.Validate(); err != nil {
		return Source{}, err
	}
	return source, nil
}

func (s Source) Validate() error {
	if s.version != Version1 {
		return ErrVersion
	}
	if s.profile != ProfileDashboard {
		return ErrProfile
	}
	if len(s.html) == 0 || len(s.html) > MaxEncodedBytes {
		return ErrDocument
	}
	if !utf8.Valid(s.html) {
		return ErrEncoding
	}
	return nil
}

func (s Source) Version() Version { return s.version }
func (s Source) Profile() Profile { return s.profile }
func (s Source) HTML() []byte     { return s.html }
