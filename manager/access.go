package manager

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"
)

// apiAccess is the existing single-process token and submission budget policy,
// shared by the legacy and opt-in screen APIs without changing its semantics.
type apiAccess struct {
	token       [sha256.Size]byte
	mu          sync.Mutex
	windowStart time.Time
	used        int
}

func (s *apiAccess) authorized(request *http.Request) bool {
	if len(request.Header.Values("Authorization")) != 1 {
		return false
	}
	value := request.Header.Get("Authorization")
	if !strings.HasPrefix(value, "Bearer ") {
		return false
	}
	digest := sha256.Sum256([]byte(strings.TrimPrefix(value, "Bearer ")))
	return subtle.ConstantTimeCompare(digest[:], s.token[:]) == 1
}

func (s *apiAccess) allow(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.windowStart.IsZero() || now.Sub(s.windowStart) >= time.Minute {
		s.windowStart, s.used = now, 0
	}
	if s.used >= updatesPerMinute {
		return false
	}
	s.used++
	return true
}
