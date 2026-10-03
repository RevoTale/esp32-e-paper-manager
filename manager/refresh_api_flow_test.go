package manager

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

func TestUrgentPatchCarriesMetadataWithAtomicScene(t *testing.T) {
	a, s := newScreenAPI(t)
	s.queue, _ = renderbatch.New(renderbatch.Policy{Debounce: time.Minute})
	if _, err := NewScreenPumpWithPolicy(s, &cadenceSender{}, refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	if w := apiRequest(a, "PUT", "/v2/screen", a.tag(0), `<p id="a">old</p>`); w.Code != 202 {
		t.Fatal(w)
	}
	r := httptest.NewRequest("PATCH", "/v2/screen", strings.NewReader(`{"edits":[{"id":"a","text":"important"}]}`))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	r.Header.Set("If-Match", a.tag(1))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Update-Priority", "urgent")
	r.Header.Set("X-Refresh-Mode", "full")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 202 {
		t.Fatal(w)
	}
	d, err := s.RenderNext(context.Background(), s.elapsed(), true)
	if err != nil || d.Revision != 2 || d.Options.Priority != refreshpolicy.Urgent || d.Options.Mode != refreshpolicy.Full {
		t.Fatal(d, err)
	}
	if !strings.Contains(s.markup, "important") {
		t.Fatal(s.markup)
	}
}

func TestMaintenanceDoesNotInheritUrgentPriority(t *testing.T) {
	_, s := newScreenAPI(t)
	s.options.Priority = refreshpolicy.Urgent
	s.cycles = &screenCycles{resync: true}
	if s.pendingOptions().Priority == refreshpolicy.Urgent {
		t.Fatal("recovery inherited urgency")
	}
}
