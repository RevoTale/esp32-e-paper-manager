// Package network defines fail-closed address and retry policy.
package network

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"time"
)

var (
	ErrEndpoint        = errors.New("network: invalid manager endpoint")
	ErrIPv6Unavailable = errors.New("network: IPv6 unavailable in target stack")
)

type Endpoint struct {
	Host string
	Port uint16
}

func ParseEndpoint(value string) (Endpoint, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "tcp" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Endpoint{}, ErrEndpoint
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil || host == "" {
		return Endpoint{}, ErrEndpoint
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return Endpoint{}, ErrEndpoint
	}
	return Endpoint{Host: host, Port: uint16(number)}, nil
}

func SelectIPv4(addresses []netip.Addr, port uint16) (netip.AddrPort, error) {
	for _, address := range addresses {
		if address.Is4() {
			return netip.AddrPortFrom(address, port), nil
		}
	}
	return netip.AddrPort{}, ErrIPv6Unavailable
}

func RetryDelay(failures uint) time.Duration {
	const maximum = 5 * time.Minute
	wait := 2 * time.Second << min(failures, 8)
	if wait > maximum {
		return maximum
	}
	return wait
}
