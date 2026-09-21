package manager

import (
	"net/http"
	"testing"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

func TestRefreshHeaders(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header http.Header
		want   refreshpolicy.Options
		valid  bool
	}{
		{"default", http.Header{}, refreshpolicy.Options{}, true},
		{"urgent full", http.Header{"X-Update-Priority": {"urgent"}, "X-Refresh-Mode": {"full"}}, refreshpolicy.Options{Priority: refreshpolicy.Urgent, Mode: refreshpolicy.Full}, true},
		{"normal partial", http.Header{"X-Update-Priority": {"normal"}, "X-Refresh-Mode": {"partial"}}, refreshpolicy.Options{Mode: refreshpolicy.Partial}, true},
		{"empty", http.Header{"X-Update-Priority": {""}}, refreshpolicy.Options{}, false},
		{"duplicate", http.Header{"X-Update-Priority": {"urgent", "normal"}}, refreshpolicy.Options{}, false},
		{"list", http.Header{"X-Update-Priority": {"normal, urgent"}}, refreshpolicy.Options{}, false},
		{"unknown", http.Header{"X-Refresh-Mode": {"fast"}}, refreshpolicy.Options{}, false},
		{"empty mode", http.Header{"X-Refresh-Mode": {""}}, refreshpolicy.Options{}, false},
		{"duplicate mode", http.Header{"X-Refresh-Mode": {"auto", "auto"}}, refreshpolicy.Options{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := refreshHeaders(tc.header)
			if (err == nil) != tc.valid || got != tc.want {
				t.Fatalf("options=%+v error=%v", got, err)
			}
		})
	}
}
