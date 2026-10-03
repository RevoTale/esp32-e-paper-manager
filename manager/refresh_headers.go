package manager

import (
	"net/http"

	"github.com/RevoTale/esp32-e-paper-manager/refreshpolicy"
)

// An absent option defaults; an explicitly empty or repeated option is invalid.
// Do not use Header.Get alone: it hides duplicate authoring instructions.
func refreshHeaders(h http.Header) (refreshpolicy.Options, error) {
	priority, mode := h.Values("X-Update-Priority"), h.Values("X-Refresh-Mode")
	if !singleOption(priority) || !singleOption(mode) {
		return refreshpolicy.Options{}, refreshpolicy.ErrPolicy
	}
	return refreshpolicy.Parse(h.Get("X-Update-Priority"), h.Get("X-Refresh-Mode"))
}

func singleOption(values []string) bool {
	return len(values) == 0 || (len(values) == 1 && values[0] != "")
}
