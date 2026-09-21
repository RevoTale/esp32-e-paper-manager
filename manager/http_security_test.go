package manager

import (
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScreenRejectsAmbiguousSecurityHeaders(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		code                int
	}{
		{"Authorization", "Bearer " + strings.Repeat("t", 32), "Bearer other", 401},
		{"Authorization", "Bearer " + strings.Repeat("t", 32), "Bearer " + strings.Repeat("t", 32), 401},
		{"Content-Type", "text/html", "application/json", 415},
		{"Content-Type", "text/html", "text/html", 415},
		{"Content-Encoding", "", "gzip", 415},
	} {
		a, s := newScreenAPI(t)
		r := httptest.NewRequest("PUT", "/v2/screen", strings.NewReader("bounded"))
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
		r.Header.Set("Content-Type", "text/html")
		r.Header.Set("If-Match", a.tag(0))
		r.Header.Set(tc.name, tc.first)
		r.Header.Add(tc.name, tc.second)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != tc.code || s.Status().Current != 0 {
			t.Fatalf("%s: status=%d current=%d", tc.name, w.Code, s.Status().Current)
		}
	}
}

type heldBody struct {
	io.Reader
	started, release chan struct{}
	once             sync.Once
}

func (b *heldBody) Read(dst []byte) (int, error) {
	b.once.Do(func() { close(b.started) })
	<-b.release
	return b.Reader.Read(dst)
}

func TestScreenBoundsConcurrentMutationBodies(t *testing.T) {
	a, _ := newScreenAPI(t)
	release, done := make(chan struct{}), make(chan struct{}, 2)
	workers := 0
	t.Cleanup(func() {
		close(release)
		for range workers {
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("mutation worker did not stop")
			}
		}
	})
	for range 2 {
		body := &heldBody{Reader: strings.NewReader("body"), started: make(chan struct{}), release: release}
		r := httptest.NewRequest("PUT", "/v2/screen", body)
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
		r.Header.Set("Content-Type", "text/html")
		r.Header.Set("If-Match", a.tag(0))
		go func() { a.ServeHTTP(httptest.NewRecorder(), r); done <- struct{}{} }()
		workers++
		select {
		case <-body.started:
		case <-time.After(time.Second):
			t.Fatal("body did not start")
		}
	}
	if response := apiRequest(a, "PUT", "/v2/screen", a.tag(0), "overflow"); response.Code != 429 {
		t.Fatalf("third concurrent mutation admitted: %d", response.Code)
	}
}
