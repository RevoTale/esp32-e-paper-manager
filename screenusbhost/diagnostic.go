package screenusbhost

import "errors"

// Proxy exit codes are a local worker ABI, not EPS2 or physical-delivery evidence.
// Never transport stderr: it may contain private paths or packet contents.
type proxyFailure struct {
	code  int
	cause error
}

func (e proxyFailure) Error() string { return proxyStage(e.code) }
func (e proxyFailure) Unwrap() error { return e.cause }

func proxyStage(code int) string {
	switch code {
	case 20:
		return "screen USB: serial-open failed"
	case 21:
		return "screen USB: serial-setup failed"
	case 22:
		return "screen USB: serial-write failed"
	case 23:
		return "screen USB: serial-read failed"
	case 24:
		return "screen USB: reply-validation failed"
	default:
		return "screen USB: worker exited without a known diagnostic"
	}
}

// ProxyExitCode emits only a fixed, non-secret category from the supervised CLI.
// Unknown failures retain exit1; older workers remain compatible with the parent.
func ProxyExitCode(err error) int {
	if err == nil {
		return 0
	}
	var failure proxyFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	return 1
}
