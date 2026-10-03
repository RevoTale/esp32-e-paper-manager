package manager

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func patchRequest(a *ScreenAPI, tag, source, kind string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPatch, "/v2/screen", strings.NewReader(source))
	r.Header.Set("Authorization", "Bearer "+strings.Repeat("t", 32))
	r.Header.Set("Content-Type", kind)
	r.Header.Set("If-Match", tag)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}

func TestScreenPatchAtomic(t *testing.T) {
	a, s := newScreenAPI(t)
	markup := `<p id="a">old</p><p id="b">keep</p>`
	if w := apiRequest(a, "PUT", "/v2/screen", a.tag(0), markup); w.Code != 202 {
		t.Fatal(w)
	}
	patch := `{"edits":[{"id":"a","text":"new"},{"id":"b","attributes":{"style":"color:red"}}]}`
	if w := patchRequest(a, a.tag(1), patch, "application/json"); w.Code != 202 {
		t.Fatal(w)
	}
	current := apiRequest(a, "GET", "/v2/screen", "", "")
	if s.Status().Current != 2 || !strings.Contains(current.Body.String(), `<p id="a">new</p>`) || !strings.Contains(current.Body.String(), `style="color:red"`) {
		t.Fatal(current, s.Status())
	}
	for _, patch := range []string{`{"edits":[{"id":"a","remove":true},{"id":"absent","text":"bad"}]}`, `{"edits":[{"id":"a","html":"<div>moved</div>"}]}`} {
		if w := patchRequest(a, a.tag(2), patch, "application/json"); w.Code != 400 {
			t.Fatal(w)
		}
		if after := apiRequest(a, "GET", "/v2/screen", "", ""); after.Body.String() != current.Body.String() || s.Status().Current != 2 {
			t.Fatal("partial mutation", after)
		}
	}
}

func TestScreenPatchAdmission(t *testing.T) {
	for _, test := range []struct {
		tag, body, kind string
		status          int
	}{
		{"", `{}`, "application/json", 428},
		{`"stale"`, `{}`, "application/json", 412},
		{"current", `{}`, "text/html", 415},
		{"current", strings.Repeat(" ", 32769), "application/json", 413},
		{"current", `{"edits":[]}`, "application/json", 400},
	} {
		a, _ := newScreenAPI(t)
		if test.tag == "current" {
			test.tag = a.tag(0)
		}
		if w := patchRequest(a, test.tag, test.body, test.kind); w.Code != test.status {
			t.Fatal(test, w)
		}
	}
}
