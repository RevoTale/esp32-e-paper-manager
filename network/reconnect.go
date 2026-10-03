package network

import "time"

// RetryUntilConnected retries a link operation until it succeeds. The caller
// owns cancellation by making attempt return success when the runtime stops.
func RetryUntilConnected(attempt func() error, sleep func(time.Duration)) error {
	if attempt == nil || sleep == nil {
		return ErrPolling
	}
	for failures := uint(0); ; failures++ {
		if err := attempt(); err == nil {
			return nil
		}
		sleep(RetryDelay(failures))
	}
}

// EnsureConnected restores a lost link and then refreshes its network
// configuration. An already healthy link is left untouched.
func EnsureConnected(
	isUp func() bool,
	join func() error,
	configure func() error,
	sleep func(time.Duration),
) error {
	if isUp == nil || configure == nil {
		return ErrPolling
	}
	if isUp() {
		return nil
	}
	if err := RetryUntilConnected(join, sleep); err != nil {
		return err
	}
	return configure()
}
