package screendelivery

import (
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func TestFullAfterPartialUsesLatestCompletion(t *testing.T) {
	// 2026-10-01 list soak: last partial ACK ~22:38:39Z, next frame submitted
	// at22:38:58Z, full started22:39:09Z. See docs/partial-candidate-20261001.md.
	c := Cadence{Policy: refreshpolicy.Policy{Normal: 30 * time.Second, Urgent: 30 * time.Second}}
	if err := c.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	last := time.Date(2026, 9, 30, 22, 38, 39, 0, time.UTC)
	full := c.Complete(last)
	submitted := last.Add(19 * time.Second)
	got := c.Deadline(full, refreshpolicy.Options{Mode: refreshpolicy.Full})
	if got.Sub(submitted) != 11*time.Second {
		t.Fatalf("full wait=%s, want11s after most recent partial", got.Sub(submitted))
	}
	if !submitted.After(c.Deadline(full, refreshpolicy.Options{Mode: refreshpolicy.Partial})) {
		t.Fatal("partial should already be allowed")
	}
}
