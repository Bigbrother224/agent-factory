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
	"sync"

	"teleblock/internal/ja4"
	"teleblock/internal/profile"
	"teleblock/internal/rules"
	"teleblock/internal/sni"
)

// TLS wire constants (RFC 5246).
const (
	tlsRecordHeaderLen   = 5  // content type, version, length
	contentTypeHandshake = 22 // 0x16
)

// tlsMaxRecordLen is the maximum TLSPlaintext record length the RFC
// 5246 allows (16 KiB); anything larger is not a record we inspect.
const tlsMaxRecordLen = 16384

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
// its SNI, then relay or reset. Single-close owner: each conn is closed
// EXACTLY once — explicit close on early return, reset(), or relay() ;
// no defer, to avoid any double close (review) with the exit path of the
// two relay goroutines.
func (l *Listener) handle(conn net.Conn) {
	src := remoteIP(conn)

	header := make([]byte, tlsRecordHeaderLen)
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close() // no data: quiet close
		return
	}

	// Only a handshake record carries a ClientHello we can read in
	// the clear; anything else is relayed untouched — no decryption.
	// The record length is capped at the RFC maximum: a larger value
	// is invalid, and allocating on it would let a client pin memory,
	// so the flow is relayed uninspected without allocation.
	var sniName, fp string
	prefix := header
	if header[0] == contentTypeHandshake {
		length := int(header[3])<<8 | int(header[4])
		if length > tlsMaxRecordLen {
			log.Printf("tcp: record length %d exceeds %d, relaying uninspected", length, tlsMaxRecordLen)
		} else {
			body := make([]byte, length)
			if _, err := io.ReadFull(conn, body); err != nil {
				conn.Close()
				return
			}
			prefix = append(header, body...)
			if name, err := sni.Parse(prefix); err == nil {
				sniName = name
			}
			display := sniName
			if display == "" {
				display = "-"
			}
			// JA4 fingerprint of the same ClientHello: classification
			// key for unknown flows, exposed to the Observe verdict.
			if hello, err := ja4.Parse(prefix); err == nil {
				fp = hello.String()
			}
			if fp != "" {
				log.Printf("tcp: ja4 %s sni=%s src=%s", fp, display, conn.RemoteAddr())
			}
		}
	}

	if sniName != "" {
		switch l.router.Verdict(src, sniName) {
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
// unmodified in both directions. The two directions are joined by a
// WaitGroup and EACH end is closed exactly once by its owner — dst by
// relay when the reverse direction is finished, conn by the goroutine —
// the end of one direction releases the other. No concurrent close, no
// resource leak (review).
func (l *Listener) relay(conn net.Conn, prefix []byte) {
	addr, err := l.target(conn)
	if err != nil {
		log.Printf("tcp: original destination: %v", err)
		conn.Close()
		return
	}
	dst, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("tcp: relay to %s: %v", addr, err)
		conn.Close()
		return
	}
	if _, err := dst.Write(prefix); err != nil {
		log.Printf("tcp: relay to %s: %v", addr, err)
		dst.Close()
		conn.Close()
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(dst, conn); err != nil {
			log.Printf("tcp: relay src→dst: %v", err)
		}
		conn.Close() // src side is finished: releases the reverse copy
	}()
	if _, err := io.Copy(conn, dst); err != nil {
		log.Printf("tcp: relay dst→src: %v", err)
	}
	dst.Close() // dst side is finished: releases the forward copy
	wg.Wait()
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
