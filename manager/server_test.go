package manager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RevoTale/esp32-e-paper-manager/securetransport"
	"github.com/RevoTale/esp32-e-paper-manager/update"
)

func TestServerAuthSubmitIdempotencyAndStatus(t *testing.T) {
	now := time.Unix(1_788_340_800, 0)
	record := DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := NewStaticRegistry([]DeviceRecord{record})
	server, err := NewServer(NewStore(), registry, []byte(strings.Repeat("t", 32)), func() time.Time { return now }, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/v1/devices/%x/updates", record.ID)
	unauthorized := request(server, http.MethodGet, path, "", "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized=%d", unauthorized.Code)
	}
	key := "01000000000000000000000000000000"
	created := request(server, http.MethodPost, path, "Bearer "+strings.Repeat("t", 32), key, "<p>safe</p>")
	if created.Code != http.StatusAccepted || strings.Contains(created.Body.String(), "device_key") {
		t.Fatalf("created=%d %s", created.Code, created.Body.String())
	}
	repeated := request(server, http.MethodPost, path, "Bearer "+strings.Repeat("t", 32), key, "<p>safe</p>")
	if repeated.Code != http.StatusOK {
		t.Fatalf("repeated=%d", repeated.Code)
	}
	conflict := request(server, http.MethodPost, path, "Bearer "+strings.Repeat("t", 32), key, "<p>changed</p>")
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict=%d", conflict.Code)
	}
	status := request(server, http.MethodGet, path, "Bearer "+strings.Repeat("t", 32), "", "")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"pending":true`) {
		t.Fatalf("status=%d %s", status.Code, status.Body.String())
	}
}

func TestServerRejectsWrongContentAndUnknownDevice(t *testing.T) {
	record := DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := NewStaticRegistry([]DeviceRecord{record})
	server, _ := NewServer(NewStore(), registry, []byte(strings.Repeat("t", 32)), time.Now, time.Hour)
	auth := "Bearer " + strings.Repeat("t", 32)
	unknown := request(server, http.MethodGet, "/v1/devices/02000000000000000000000000000000/updates", auth, "", "")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown=%d", unknown.Code)
	}
	path := fmt.Sprintf("/v1/devices/%x/updates", record.ID)
	invalid := httptest.NewRequest(http.MethodPost, path, strings.NewReader("<p>x</p>"))
	invalid.Header.Set("Authorization", auth)
	invalid.Header.Set("Idempotency-Key", "01000000000000000000000000000000")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, invalid)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("content type=%d", response.Code)
	}
}

func TestServerRateLimitsSubmission(t *testing.T) {
	record := DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := NewStaticRegistry([]DeviceRecord{record})
	server, _ := NewServer(NewStore(), registry, []byte(strings.Repeat("t", 32)), time.Now, time.Hour)
	path := fmt.Sprintf("/v1/devices/%x/updates", record.ID)
	for index := 0; index < updatesPerMinute; index++ {
		key := fmt.Sprintf("%032x", index+1)
		response := request(server, http.MethodPost, path, "Bearer "+strings.Repeat("t", 32), key, "<p>x</p>")
		if response.Code != http.StatusAccepted {
			t.Fatalf("index=%d status=%d", index, response.Code)
		}
	}
	limited := request(server, http.MethodPost, path, "Bearer "+strings.Repeat("t", 32),
		"ff000000000000000000000000000000", "<p>x</p>")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("limited=%d", limited.Code)
	}
}

func TestServerRejectsMalformedRoutesMethodsAndBodies(t *testing.T) {
	record := DeviceRecord{ID: securetransport.DeviceID{1}, Key: securetransport.Key{2}, Timezone: "Europe/Kyiv"}
	registry, _ := NewStaticRegistry([]DeviceRecord{record})
	server, _ := NewServer(NewStore(), registry, []byte(strings.Repeat("t", 32)), time.Now, time.Hour)
	auth := "Bearer " + strings.Repeat("t", 32)
	path := fmt.Sprintf("/v1/devices/%x/updates", record.ID)
	cases := []struct {
		method, path, key, body, contentType string
		status                               int
	}{
		{http.MethodGet, "/bad", "", "", "", http.StatusNotFound},
		{http.MethodPut, path, "", "", "", http.StatusMethodNotAllowed},
		{http.MethodPost, path, "bad", "<p>x</p>", "text/html", http.StatusBadRequest},
		{http.MethodPost, path, "01000000000000000000000000000000", "", "text/html", http.StatusBadRequest},
		{http.MethodPost, path, "01000000000000000000000000000000", "<p>x</p>", "application/json", http.StatusBadRequest},
	}
	for _, item := range cases {
		request := httptest.NewRequest(item.method, item.path, strings.NewReader(item.body))
		request.Header.Set("Authorization", auth)
		request.Header.Set("Idempotency-Key", item.key)
		request.Header.Set("Content-Type", item.contentType)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != item.status {
			t.Fatalf("%s %s status=%d", item.method, item.path, response.Code)
		}
	}
}

func TestServerConfigurationAndTimezoneFailures(t *testing.T) {
	if _, err := NewServer(nil, nil, nil, nil, 0); err == nil {
		t.Fatal("invalid server accepted")
	}
	if _, err := managerRequest(update.ID{1}, []byte("<p>x</p>"), "Invalid/Zone", time.Now()); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	if _, err := parseUpdateID(strings.Repeat("0", 32)); err == nil {
		t.Fatal("zero update identity accepted")
	}
}

func request(handler http.Handler, method, path, authorization, key, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", authorization)
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "text/html; charset=utf-8")
		request.Header.Set("Idempotency-Key", key)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
