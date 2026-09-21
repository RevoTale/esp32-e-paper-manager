package manager

import (
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
	"github.com/RevoTale/esp32-e-paper-manager/renderbatch"
)

// ScreenAPI serves one explicitly bound screen. It never constructs devices
// from request paths. Mount only behind HTTPS and for trusted content authors
// until the renderer capability/resource-isolation gates pass.
type ScreenAPI struct {
	access    apiAccess
	screen    *Screen
	epoch     string
	now       Clock
	mutations chan struct{}
}

// NewScreenAPI requires a fresh cryptographically random epoch per process.
// It prevents stale If-Match values from becoming valid after a restart.
func NewScreenAPI(screen *Screen, token []byte, epoch [16]byte, now Clock) (*ScreenAPI, error) {
	if screen == nil || len(token) < 32 || epoch == ([16]byte{}) || now == nil {
		return nil, ErrConfiguration
	}
	return &ScreenAPI{screen: screen, access: apiAccess{token: TokenDigest(token)}, epoch: hex.EncodeToString(epoch[:]), now: now, mutations: make(chan struct{}, 2)}, nil
}

func (a *ScreenAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if !a.access.authorized(r) {
		writeError(w, 401, "unauthorized")
		return
	}
	if headerBytes(r.Header) > maxHeaderBytes {
		writeError(w, 431, "headers_too_large")
		return
	}
	switch r.URL.Path {
	case "/v2/screen":
		a.scene(w, r)
	case "/v2/screen/status":
		if r.Method != http.MethodGet {
			screenMethod(w, "GET")
			return
		}
		writeJSON(w, 200, a.screen.Status())
	default:
		writeError(w, 404, "not_found")
	}
}

func (a *ScreenAPI) scene(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.screen.mu.Lock()
		markup, revision := a.screen.markup, a.screen.state.Current
		a.screen.mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("ETag", a.tag(revision))
		_, _ = io.WriteString(w, markup)
	case http.MethodPut, http.MethodPatch:
		a.mutate(w, r)
	default:
		screenMethod(w, "GET, PUT, PATCH")
	}
}

func (a *ScreenAPI) mutate(w http.ResponseWriter, r *http.Request) {
	options, err := refreshHeaders(r.Header)
	if err != nil {
		writeError(w, 400, "invalid_refresh_options")
		return
	}
	if options.Mode == refreshpolicy.Partial || (options.Priority == refreshpolicy.Urgent && !a.screen.refreshEnabled) {
		writeError(w, 501, "refresh_unsupported")
		return
	}
	select {
	case a.mutations <- struct{}{}:
		defer func() { <-a.mutations }()
	default:
		writeError(w, 429, "busy")
		return
	}
	if r.Method == http.MethodPut {
		a.replace(w, r, options)
	} else {
		a.patch(w, r, options)
	}
}

func (a *ScreenAPI) replace(w http.ResponseWriter, r *http.Request, options refreshpolicy.Options) {
	base, ok := a.mutationBase(w, r)
	if !ok {
		return
	}
	markup, status, err := screenMarkup(w, r)
	if err != nil {
		writeError(w, status, "invalid_document")
		return
	}
	revision, err := a.screen.submitOptionsAt(base, markup, a.screen.elapsed, options)
	writeMutation(w, revision, err)
}

func (a *ScreenAPI) mutationBase(w http.ResponseWriter, r *http.Request) (renderbatch.Revision, bool) {
	if !a.access.allow(a.now()) {
		writeError(w, 429, "rate_limited")
		return 0, false
	}
	if r.Header.Get("If-Match") == "" {
		writeError(w, 428, "base_required")
		return 0, false
	}
	base := a.screen.Status().Current
	// Require exactly one strong tag; wildcard, weak tags and lists cannot
	// authorize a stale full replacement. RFC 9110 section 13.1.1.
	if len(r.Header.Values("If-Match")) != 1 || r.Header.Get("If-Match") != a.tag(base) {
		writeError(w, 412, "stale_base")
		return 0, false
	}
	return base, true
}

func writeMutation(w http.ResponseWriter, revision renderbatch.Revision, err error) {
	if errors.Is(err, errPatchUnsupported) {
		writeError(w, 501, "patch_unsupported")
		return
	}
	if errors.Is(err, ErrConflict) {
		writeError(w, 412, "stale_base")
		return
	}
	if err != nil {
		writeError(w, 400, "invalid_document")
		return
	}
	writeJSON(w, 202, map[string]any{"revision": revision})
}

func screenMarkup(w http.ResponseWriter, r *http.Request) ([]byte, int, error) {
	return screenBody(w, r, "text/html")
}

func screenBody(w http.ResponseWriter, r *http.Request, expected string) ([]byte, int, error) {
	if len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Content-Encoding")) != 0 {
		return nil, 415, ErrConfiguration
	}
	kind, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != expected || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		return nil, 415, ErrConfiguration
	}
	if r.ContentLength > 32768 {
		return nil, 413, ErrConfiguration
	}
	// Bounds unknown/chunked bodies too: https://pkg.go.dev/net/http#MaxBytesReader
	body := http.MaxBytesReader(w, r.Body, 32768)
	markup, err := io.ReadAll(body)
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		return nil, 413, err
	}
	return markup, 400, err
}

func (a *ScreenAPI) tag(revision renderbatch.Revision) string {
	return `"` + a.epoch + "-" + strconv.FormatUint(uint64(revision), 10) + `"`
}

func (s *Screen) elapsed() time.Duration {
	if s.elapsedClock != nil {
		return s.elapsedClock()
	}
	return time.Since(s.started)
}

func screenMethod(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, 405, "method_not_allowed")
}
