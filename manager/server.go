package manager

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/document"
	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

const (
	maxHeaderBytes   = 4096
	updatesPerMinute = 30
)

type Clock func() time.Time

type Server struct {
	*apiAccess
	store    *Store
	registry Registry
	now      Clock
	ttl      time.Duration
}

func NewServer(store *Store, registry Registry, token []byte, now Clock, ttl time.Duration) (*Server, error) {
	if store == nil || registry == nil || len(token) < 32 || now == nil || ttl <= 0 {
		return nil, ErrConfiguration
	}
	return &Server{store: store, registry: registry, apiAccess: &apiAccess{token: TokenDigest(token)}, now: now, ttl: ttl}, nil
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	if !s.authorized(request) {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseDevicePath(request.URL.Path)
	if !ok {
		writeError(writer, http.StatusNotFound, "not_found")
		return
	}
	record, found := s.registry.Lookup(id)
	if !found {
		writeError(writer, http.StatusNotFound, "not_found")
		return
	}
	switch request.Method {
	case http.MethodPost:
		s.submit(writer, request, record)
	case http.MethodGet:
		s.status(writer, record)
	default:
		writer.Header().Set("Allow", "GET, POST")
		writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (s *Server) submit(writer http.ResponseWriter, request *http.Request, record DeviceRecord) {
	if !s.allow(s.now()) {
		writeError(writer, http.StatusTooManyRequests, "rate_limited")
		return
	}
	now := s.now()
	updateRequest, code, err := decodeSubmission(request, record.Timezone, now)
	if err != nil {
		writeError(writer, http.StatusBadRequest, code)
		return
	}
	generation, created, err := s.store.Submit(record.ID, updateRequest, now, s.ttl)
	if errors.Is(err, ErrConflict) {
		writeError(writer, http.StatusConflict, "idempotency_conflict")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal")
		return
	}
	status := http.StatusAccepted
	if !created {
		status = http.StatusOK
	}
	writeJSON(writer, status, map[string]any{"generation": generation, "created": created})
}

func decodeSubmission(request *http.Request, timezone string, now time.Time) (update.Request, string, error) {
	if request.ContentLength > document.MaxEncodedBytes || headerBytes(request.Header) > maxHeaderBytes ||
		!strings.HasPrefix(request.Header.Get("Content-Type"), "text/html") {
		return update.Request{}, "invalid_request", ErrConfiguration
	}
	id, err := parseUpdateID(request.Header.Get("Idempotency-Key"))
	if err != nil {
		return update.Request{}, "invalid_idempotency_key", err
	}
	markup, err := io.ReadAll(io.LimitReader(request.Body, document.MaxEncodedBytes+1))
	if err != nil || len(markup) == 0 || len(markup) > document.MaxEncodedBytes {
		return update.Request{}, "invalid_document", ErrConfiguration
	}
	decoded, err := managerRequest(id, markup, timezone, now)
	return decoded, "invalid_document", err
}

func (s *Server) status(writer http.ResponseWriter, record DeviceRecord) {
	status := s.store.Status(record.ID, s.now())
	writeJSON(writer, http.StatusOK, map[string]any{"pending": status.Pending,
		"connected": status.Connected, "last_seen": status.LastSeen,
		"generation": status.Generation, "last_result_status": status.LastResult.Status})
}

func managerRequest(id update.ID, markup []byte, timezone string, now time.Time) (update.Request, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return update.Request{}, err
	}
	source, err := document.NewSource(document.Version1, document.ProfileDashboard, markup)
	if err != nil {
		return update.Request{}, err
	}
	return update.NewRequest(id, now.Unix(), now.In(location).Format("2006-01-02 15:04"), timezone, source)
}

func parseDevicePath(path string) (securetransport.DeviceID, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "devices" || parts[3] != "updates" {
		return securetransport.DeviceID{}, false
	}
	var id securetransport.DeviceID
	decoded, err := hex.DecodeString(parts[2])
	if err != nil || len(decoded) != len(id) {
		return id, false
	}
	copy(id[:], decoded)
	return id, true
}

func parseUpdateID(value string) (update.ID, error) {
	var id update.ID
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(id) {
		return id, ErrConfiguration
	}
	copy(id[:], decoded)
	if id == (update.ID{}) {
		return id, ErrConfiguration
	}
	return id, nil
}

func headerBytes(header http.Header) int {
	total := 0
	for name, values := range header {
		total += len(name)
		for _, value := range values {
			total += len(value)
		}
	}
	return total
}

func writeError(writer http.ResponseWriter, status int, code string) {
	writeJSON(writer, status, map[string]string{"error": code})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
