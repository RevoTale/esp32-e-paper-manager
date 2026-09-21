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

func TestScenePriorityBelongsToRevision(t *testing.T) {
	_, s := newScreenAPI(t)
	s.queue, _ = renderbatch.New(renderbatch.Policy{Debounce: time.Minute})
	urgent := refreshpolicy.Options{Priority: refreshpolicy.Urgent, Mode: refreshpolicy.Full}
	if _, err := s.submitOptionsAt(0, []byte("urgent"), func() time.Duration { return 0 }, urgent); err != nil {
		t.Fatal(err)
	}
	d, err := s.RenderNext(context.Background(), 0, true)
	if err != nil || d.Revision != 1 || d.Options != urgent {
		t.Fatal(d, err)
	}
	if _, err := s.Submit(1, []byte("normal"), 0); err != nil {
		t.Fatal(err)
	}
	if d.Options != urgent {
		t.Fatal("in-flight metadata changed")
	}
	if err := s.Resolve(1, true); err != nil {
		t.Fatal(err)
	}
	if d, err := s.RenderNext(context.Background(), time.Second, true); err != nil || d.Revision != 0 {
		t.Fatal(d, err)
	}
}

func TestInvalidRefreshHeadersNeverMutateScene(t *testing.T) {
	for _, tc := range []struct {
		name, priority, mode string
		code                 int
	}{
		{"unknown", "now", "", 400}, {"unsupported", "", "partial", 501},
		{"legacy", "urgent", "auto", 501},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, s := newScreenAPI(t)
			r := httptest.NewRequest("PUT", "/v2/screen", strings.NewReader("changed"))
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
			r.Header.Set("If-Match", a.tag(0))
			r.Header.Set("Content-Type", "text/html")
			if tc.priority != "" {
				r.Header.Set("X-Update-Priority", tc.priority)
			}
			if tc.mode != "" {
				r.Header.Set("X-Refresh-Mode", tc.mode)
			}
			w := httptest.NewRecorder()
			a.ServeHTTP(w, r)
			if w.Code != tc.code || s.Status().Current != 0 {
				t.Fatal(w.Code, w.Body, s.Status())
			}
		})
	}
}
