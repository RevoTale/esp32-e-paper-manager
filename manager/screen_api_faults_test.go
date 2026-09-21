package manager

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestScreenAPIRejectsBadRequests(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, header, value, body string
		code                                    int
	}{
		{"path", "/v2/other", "GET", "", "", "", 404},
		{"method", "/v2/screen", "POST", "", "", "", 405},
		{"status method", "/v2/screen/status", "PUT", "", "", "", 405},
		{"weak", "/v2/screen", "PUT", "If-Match", `W/"1"`, "", 412},
		{"wildcard", "/v2/screen", "PUT", "If-Match", "*", "", 412},
		{"list", "/v2/screen", "PUT", "If-Match", `"a", "b"`, "", 412},
		{"json", "/v2/screen", "PUT", "Content-Type", "application/json", "{}", 415},
		{"type prefix", "/v2/screen", "PUT", "Content-Type", "text/html-evil", "", 415},
		{"charset", "/v2/screen", "PUT", "Content-Type", "text/html; charset=utf-16", "", 415},
		{"gzip", "/v2/screen", "PUT", "Content-Encoding", "gzip", "", 415},
		{"large", "/v2/screen", "PUT", "", "", strings.Repeat("x", 32769), 413},
		{"utf8", "/v2/screen", "PUT", "", "", "\xff", 400},
		{"headers", "/v2/screen", "PUT", "X-Extra", strings.Repeat("x", 4096), "", 431},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, s := newScreenAPI(t)
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
			r.Header.Set("Content-Type", "text/html")
			r.Header.Set("If-Match", a.tag(0))
			if tc.header != "" {
				r.Header.Set(tc.header, tc.value)
			}
			w := httptest.NewRecorder()
			a.ServeHTTP(w, r)
			if w.Code != tc.code || s.Status().Current != 0 {
				t.Fatal(w, s.Status())
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal(w.Header())
			}
		})
	}
}

type screenReadFailure struct{}

func (screenReadFailure) Read([]byte) (int, error) { return 0, errors.New("private internal error") }

func TestScreenAPIUnknownLengthAndReaderFailure(t *testing.T) {
	for _, body := range []io.Reader{strings.NewReader(strings.Repeat("x", 32769)), screenReadFailure{}} {
		a, s := newScreenAPI(t)
		r := httptest.NewRequest("PUT", "/v2/screen", body)
		r.ContentLength = -1
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
		r.Header.Set("Content-Type", "text/html")
		r.Header.Set("If-Match", a.tag(0))
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code < 400 || strings.Contains(w.Body.String(), "private") || s.Status().Current != 0 {
			t.Fatal(w)
		}
	}
}

func TestScreenAPIConfigurationAndBudget(t *testing.T) {
	a, s := newScreenAPI(t)
	for _, tc := range []struct {
		s     *Screen
		token []byte
		epoch [16]byte
		now   Clock
	}{
		{nil, []byte(strings.Repeat("t", 32)), [16]byte{1}, time.Now},
		{s, nil, [16]byte{1}, time.Now},
		{s, []byte(strings.Repeat("t", 32)), [16]byte{}, time.Now},
		{s, []byte(strings.Repeat("t", 32)), [16]byte{1}, nil},
	} {
		if _, err := NewScreenAPI(tc.s, tc.token, tc.epoch, tc.now); !errors.Is(err, ErrConfiguration) {
			t.Fatal(err)
		}
	}
	for range updatesPerMinute {
		_ = apiRequest(a, "PUT", "/v2/screen", "*", "")
	}
	if w := apiRequest(a, "PUT", "/v2/screen", "*", ""); w.Code != 429 {
		t.Fatal(w)
	}
}
