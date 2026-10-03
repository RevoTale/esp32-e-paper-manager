// Package screendelivery defines the manager's physical delivery boundary.
// Only its caller owns scene revisions, refresh timestamps and retry decisions.
package screendelivery

import (
	"context"
	"errors"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

// Sender confirms a physical full refresh, not merely receipt of bytes.
type Sender interface {
	Send(context.Context, display.Frame) error
}

// PolicySender is configured once before its owner starts. Implementations
// require peer capability negotiation; they must never silently downgrade.
type PolicySender interface {
	Sender
	ConfigureRefresh(refreshpolicy.Policy) error
	SendWithOptions(context.Context, display.Frame, refreshpolicy.Options) error
}

// RegionSender consumes immutable prepared crops under the same confirmed
// baseline as its last successful Send. It never silently performs a full update.
type RegionSender interface {
	Sender
	SendRegion(context.Context, RegionPlan, refreshpolicy.Options) error
}

// RegionPolicySender negotiates both full and partial operator policies before
// delivery starts. ConfigureRefresh must precede ConfigurePartial.
type RegionPolicySender interface {
	PolicySender
	RegionSender
	ConfigurePartial(refreshpolicy.Policy) error
}

// Outcome resolves only a previously ambiguous Send in the same caller.
type Outcome uint8

const (
	NoPending Outcome = iota
	PendingConfirmed
	PendingUnconfirmed
)

// Readiness carries a conservative physical refresh deadline.
type Readiness struct {
	Pending                Outcome
	NotBefore              time.Time
	UrgentNotBefore        time.Time // Zero means no distinct urgent permission.
	PartialNotBefore       time.Time // Zero means no distinct partial permission.
	PartialUrgentNotBefore time.Time
}

// RecoveringSender is serialized by one pump. Changed is a coalesced wake-up,
// not ownership transfer; WaitReady may reconcile but must never resend pixels.
type RecoveringSender interface {
	Sender
	Changed() <-chan struct{}
	WaitReady(context.Context) (Readiness, error)
	ResetSession()
}

var (
	ErrTransportLost   = errors.New("screen transport lost; delivery requires reconciliation")
	ErrTransportResync = errors.New("screen transport requires a new full baseline")
)
