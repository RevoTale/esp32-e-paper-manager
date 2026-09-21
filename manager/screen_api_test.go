package manager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/display"
	"github.com/RevoTale/esp32-e-paper-manager/engine"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

type apiRenderer struct{}

func (apiRenderer) ValidateDocument(ctx context.Context, source []byte) error {
	return (&engine.Renderer{}).ValidateDocument(ctx, source)
}

func (apiRenderer) Render(context.Context, display.Size, []byte) (display.Frame, error) {
	return display.NewFrame(display.Size{Width: 8, Height: 1}, 1, []byte{0})
}

func newScreenAPI(t *testing.T) (*ScreenAPI, *Screen) {
	t.Helper()
	s, err := NewScreen(apiRenderer{}, display.Size{Width: 8, Height: 1}, renderbatch.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewScreenAPI(s, []byte(strings.Repeat("t", 32)), [16]byte{1}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return a, s
}

func apiRequest(a http.Handler, method, path, etag, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	r.Header.Set("Content-Type", "text/html; charset=utf-8")
	if etag != "" {
		r.Header.Set("If-Match", etag)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func TestScreenAPISceneAndStatus(t *testing.T) {
	a, s := newScreenAPI(t)
	initial := apiRequest(a, "GET", "/v2/screen", "", "")
	if initial.Code != 200 || initial.Body.Len() != 0 || initial.Header().Get("ETag") == "" {
		t.Fatal(initial)
	}
	etag := initial.Header().Get("ETag")
	w := apiRequest(a, "PUT", "/v2/screen", etag, "<p>Привіт</p>")
	if w.Code != 202 {
		t.Fatal(w)
	}
	current := apiRequest(a, "GET", "/v2/screen", "", "")
	if current.Body.String() != "<p>Привіт</p>" || current.Header().Get("ETag") == etag {
		t.Fatal(current)
	}
	checkScreenPreconditions(t, a, s, etag, current.Header().Get("ETag"))
}

func checkScreenPreconditions(t *testing.T, a *ScreenAPI, s *Screen, etag, currentTag string) {
	t.Helper()
	if w := apiRequest(a, "PUT", "/v2/screen", etag, "stale"); w.Code != 412 {
		t.Fatal(w)
	}
	if w := apiRequest(a, "PUT", "/v2/screen", "", "missing"); w.Code != 428 {
		t.Fatal(w)
	}
	status := apiRequest(a, "GET", "/v2/screen/status", "", "")
	if status.Code != 200 || status.Header().Get("ETag") != "" || !strings.Contains(status.Body.String(), `"current":1`) {
		t.Fatal(status)
	}
	if s.Status().Current != 1 {
		t.Fatal(s.Status())
	}
	other, _ := NewScreenAPI(s, []byte(strings.Repeat("t", 32)), [16]byte{2}, time.Now)
	if w := apiRequest(other, "PUT", "/v2/screen", currentTag, "restart"); w.Code != 412 {
		t.Fatal(w)
	}
}

func TestScreenAPIConcurrentWriters(t *testing.T) {
	a, s := newScreenAPI(t)
	etag := apiRequest(a, "GET", "/v2/screen", "", "").Header().Get("ETag")
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); codes <- apiRequest(a, "PUT", "/v2/screen", etag, "new").Code }()
	}
	wg.Wait()
	if x, y := <-codes, <-codes; x+y != 202+412 {
		t.Fatal(x, y)
	}
	if s.Status().Current != 1 {
		t.Fatal(s.Status())
	}
}

func TestScreenAPIAuthBeforeBodyOrLookup(t *testing.T) {
	a, s := newScreenAPI(t)
	for _, path := range []string{"/v2/screen", "/unknown", "/v2/screen/status"} {
		r := httptest.NewRequest("PUT", path, nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 401 || strings.Contains(w.Body.String(), "panic") {
			t.Fatal(w)
		}
	}
	if s.Status().Current != 0 {
		t.Fatal(s.Status())
	}
}
