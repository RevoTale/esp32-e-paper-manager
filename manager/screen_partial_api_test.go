package manager

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func TestPartialAPIRequiresConfiguredTransport(t *testing.T) {
	s, _ := partialFixture(t)
	a, err := NewScreenAPI(s, []byte(strings.Repeat("t", 32)), [16]byte{1}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewScreenPumpWithPolicy(s, &dispatchSender{}, refreshpolicy.Policy{Normal: time.Minute, Urgent: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if w := partialAPIRequest(a); w.Code != 501 {
		t.Fatal(w)
	}
	if err := p.ConfigurePartial(refreshpolicy.Policy{}); err == nil {
		t.Fatal("invalid cadence accepted")
	}
	if w := partialAPIRequest(a); w.Code != 501 {
		t.Fatal(w)
	}
	if err := p.ConfigurePartial(refreshpolicy.Policy{Normal: time.Second, Urgent: time.Second}); err != nil {
		t.Fatal(err)
	}
	if w := partialAPIRequest(a); w.Code != 202 {
		t.Fatal(w)
	}
	if s.options.Mode != refreshpolicy.Partial || s.Status().Current != 1 {
		t.Fatal(s.Status(), s.options)
	}
}

func partialAPIRequest(a *ScreenAPI) *httptest.ResponseRecorder {
	r := httptest.NewRequest("PUT", "/v2/screen", strings.NewReader("<p>partial</p>"))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	r.Header.Set("Content-Type", "text/html")
	r.Header.Set("If-Match", a.tag(0))
	r.Header.Set("X-Refresh-Mode", "partial")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
