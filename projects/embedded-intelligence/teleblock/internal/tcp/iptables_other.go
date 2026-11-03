//go:build !linux

// OriginalDst on non-Linux systems: SO_ORIGINAL_DST is a Linux
// socket option, so redirected flows cannot be relayed there. This
// keeps the package buildable and testable everywhere (tests inject
// their own TargetFunc).
package tcp

import (
	"errors"
	"net"
)

// OriginalDst always fails on non-Linux systems.
func OriginalDst(conn net.Conn) (string, error) {
	return "", errors.New("tcp: SO_ORIGINAL_DST requires Linux (iptables REDIRECT)")
}
