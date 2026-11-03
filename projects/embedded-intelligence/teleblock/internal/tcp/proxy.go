// Package tcp implements passive TLS inspection on port 443: the
// listener reads the first record of every redirected connection in
// the clear — the ClientHello header carrying the SNI — classifies the
// advertised server name against the rule set of the device brand,
// then either relays the flow untouched to its original destination or
// resets it with a targeted TCP RST. Nothing is ever decrypted: the
// ClientHello is the only TLS structure TeleBlock reads.
//
// On a Linux router the listener is fed by the iptables REDIRECT rule
// returned by RedirectRule; the real (pre-redirection) destination is
// recovered from the socket with SO_ORIGINAL_DST.
package tcp

import (
	"io"
	"log"
	"net"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
	"teleblock/internal/sni"
)

// TLS wire constants (RFC 5246).
const (
	tlsRecordHeaderLen   = 5  // content type, version, length
	contentTypeHandshake = 22 // 0x16
)

// TargetFunc returns the real destination address of a redirected
// connection. The default implementation, OriginalDst, recovers it
// from the socket; tests inject a plain loopback address.
type TargetFunc func(conn net.Conn) (string, error)

// Listener inspects TLS ClientHellos on redirected port 443 traffic
// and applies the brand rule set of the source device. It is safe for
// concurrent use.
type Listener struct {
	router *profile.Router
	target TargetFunc
}

// New returns a Listener applying router to every inspected flow. A
// nil target selects OriginalDst (SO_ORIGINAL_DST, Linux).
func New(router *profile.Router, target TargetFunc) *Listener {
	if target == nil {
		target = OriginalDst
	}
	return &Listener{router: router, target: target}
}

// ListenAndServe listens on the TCP address addr and inspects every
// accepted connection. It blocks until the listener fails.
func (l *Listener) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return l.Serve(ln)
}

// Serve inspects every connection accepted from ln. It blocks until
// the listener fails.
func (l *Listener) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go l.handle(conn)
	}
}

// handle inspects one connection: read the first TLS record, classify
// its SNI, then relay or reset.
func (l *Listener) handle(conn net.Conn) {
	defer conn.Close()

	src := remoteIP(conn)

	header := make([]byte, tlsRecordHeaderLen)
	if _, err := io.ReadFull(conn, header); err != nil {
		return // no data: let the connection close quietly
	}

	// Only a handshake record carries a ClientHello we can read in
	// the clear; anything else is relayed untouched — no decryption.
	var sniName string
	prefix := header
	if header[0] == contentTypeHandshake {
		body := make([]byte, int(header[3])<<8|int(header[4]))
		if _, err := io.ReadFull(conn, body); err != nil {
			return
		}
		prefix = append(header, body...)
		if name, err := sni.Parse(prefix); err == nil {
			sniName = name
		}
	}

	if sniName != "" {
		switch l.router.Set(src).Verdict(sniName) {
		case rules.Block:
			log.Printf("tcp: blocked SNI %s from %s", sniName, conn.RemoteAddr())
			reset(conn)
			return
		case rules.Observe:
			log.Printf("tcp: observed SNI %s from %s", sniName, conn.RemoteAddr())
		}
	}

	l.relay(conn, prefix)
}

// relay pipes the connection, with the already-read prefix, to its
// real destination. The inspection is strictly passive: bytes flow
// unmodified in both directions.
func (l *Listener) relay(conn net.Conn, prefix []byte) {
	addr, err := l.target(conn)
	if err != nil {
		log.Printf("tcp: original destination: %v", err)
		return
	}
	dst, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("tcp: relay to %s: %v", addr, err)
		return
	}
	defer dst.Close()

	if _, err := dst.Write(prefix); err != nil {
		log.Printf("tcp: relay to %s: %v", addr, err)
		return
	}
	go func() {
		io.Copy(dst, conn)
		dst.Close()
	}()
	io.Copy(conn, dst)
}

// reset aborts the source connection with a targeted TCP RST: closing
// with SO_LINGER 0 makes the kernel emit an RST segment for this
// connection only. No other flow is touched.
func reset(conn net.Conn) {
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetLinger(0)
	}
	conn.Close()
}

// remoteIP extracts the source IP of a connection, nil when the
// address is not TCP (unknown devices fall back to the generic
// profile).
func remoteIP(conn net.Conn) net.IP {
	if a, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		return a.IP
	}
	return nil
}