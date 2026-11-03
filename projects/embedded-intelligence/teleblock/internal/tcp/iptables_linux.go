//go:build linux

// OriginalDst recovers the pre-redirection destination of a connection
// redirected by the nat PREROUTING REDIRECT rule (iptables.go): the
// kernel keeps the original address in the socket and hands it back
// through getsockopt(SO_ORIGINAL_DST). Linux only — TeleBlock targets
// Linux routers.
package tcp

import (
	"encoding/binary"
	"errors"
	"net"
	"syscall"
	"unsafe"
)

// soOriginalDst is SO_ORIGINAL_DST (linux/ip.h); SOL_IP is the
// socket-option level of the IP layer.
const (
	soOriginalDst = 80
	sockaddrInLen = 16
)

// OriginalDst returns the original destination address of a
// REDIRECTed connection, as "ip:port".
func OriginalDst(conn net.Conn) (string, error) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return "", errors.New("tcp: not a TCP connection")
	}
	raw, err := tc.SyscallConn()
	if err != nil {
		return "", err
	}
	var addr [sockaddrInLen]byte
	var serr error
	raw.Control(func(fd uintptr) {
		n := uint32(sockaddrInLen)
		_, _, errno := syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd,
			uintptr(syscall.SOL_IP), soOriginalDst,
			uintptr(unsafe.Pointer(&addr[0])), uintptr(unsafe.Pointer(&n)), 0)
		if errno != 0 {
			serr = errno
		}
	})
	if serr != nil {
		return "", serr
	}
	if family := binary.NativeEndian.Uint16(addr[0:2]); family != syscall.AF_INET {
		return "", errors.New("tcp: SO_ORIGINAL_DST: not an IPv4 socket")
	}
	dst := &net.TCPAddr{
		IP:   net.IPv4(addr[4], addr[5], addr[6], addr[7]),
		Port: int(binary.BigEndian.Uint16(addr[2:4])),
	}
	return dst.String(), nil
}
