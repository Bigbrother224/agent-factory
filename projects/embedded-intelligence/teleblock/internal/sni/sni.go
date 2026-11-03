// Package sni extracts the server name from TLS ClientHello records and
// classifies it against a rule set.
package sni

import (
	"errors"

	"teleblock/internal/rules"
)

// TLS wire constants (RFC 5246, RFC 8446, RFC 6066).
const (
	contentTypeHandshake     = 22 // 0x16
	handshakeTypeClientHello = 1  // 0x01
	extensionServerName      = 0  // 0x0000
	serverNameTypeHost       = 0  // host_name
)

var (
	errTruncated      = errors.New("sni: truncated record")
	errNotHandshake   = errors.New("sni: not a handshake record")
	errNotClientHello = errors.New("sni: not a ClientHello")
	errNoServerName   = errors.New("sni: no server_name extension")
)

// cursor walks TLS structures from left to right.
type cursor struct {
	b []byte
	i int
}

func (c *cursor) u8() (byte, error) {
	if c.i+1 > len(c.b) {
		return 0, errTruncated
	}
	v := c.b[c.i]
	c.i++
	return v, nil
}

func (c *cursor) u16() (int, error) {
	if c.i+2 > len(c.b) {
		return 0, errTruncated
	}
	v := int(c.b[c.i])<<8 | int(c.b[c.i+1])
	c.i += 2
	return v, nil
}

func (c *cursor) u24() (int, error) {
	if c.i+3 > len(c.b) {
		return 0, errTruncated
	}
	v := int(c.b[c.i])<<16 | int(c.b[c.i+1])<<8 | int(c.b[c.i+2])
	c.i += 3
	return v, nil
}

// take returns the next n bytes.
func (c *cursor) take(n int) ([]byte, error) {
	if n < 0 || c.i+n > len(c.b) {
		return nil, errTruncated
	}
	v := c.b[c.i : c.i+n]
	c.i += n
	return v, nil
}

// skipVector8 skips a vector prefixed by a one-byte length (session id,
// compression methods).
func (c *cursor) skipVector8() error {
	n, err := c.u8()
	if err != nil {
		return err
	}
	_, err = c.take(int(n))
	return err
}

// Parse extracts the server name from a TLS ClientHello record. It
// walks the actual wire structures — record, handshake, ClientHello,
// extensions — instead of pattern-matching raw bytes.
func Parse(record []byte) (string, error) {
	// TLS record: content type, version, length.
	c := &cursor{b: record}
	typ, err := c.u8()
	if err != nil {
		return "", err
	}
	if typ != contentTypeHandshake {
		return "", errNotHandshake
	}
	if _, err := c.take(2); err != nil { // record version
		return "", err
	}
	recLen, err := c.u16()
	if err != nil {
		return "", err
	}
	body, err := c.take(recLen)
	if err != nil {
		return "", err
	}

	// Handshake: message type, length.
	h := &cursor{b: body}
	msgType, err := h.u8()
	if err != nil {
		return "", err
	}
	if msgType != handshakeTypeClientHello {
		return "", errNotClientHello
	}
	helloLen, err := h.u24()
	if err != nil {
		return "", err
	}
	hello, err := h.take(helloLen)
	if err != nil {
		return "", err
	}

	// ClientHello: version, random, session id, cipher suites,
	// compression methods, extensions.
	ch := &cursor{b: hello}
	if _, err := ch.take(2 + 32); err != nil { // version + random
		return "", err
	}
	if err := ch.skipVector8(); err != nil { // session id
		return "", err
	}
	cipherLen, err := ch.u16()
	if err != nil {
		return "", err
	}
	if _, err := ch.take(cipherLen); err != nil {
		return "", err
	}
	if err := ch.skipVector8(); err != nil { // compression methods
		return "", err
	}
	extLen, err := ch.u16()
	if err != nil {
		return "", err
	}
	exts, err := ch.take(extLen)
	if err != nil {
		return "", err
	}

	// Extensions: walk the list until server_name.
	e := &cursor{b: exts}
	for e.i < len(e.b) {
		extType, err := e.u16()
		if err != nil {
			return "", err
		}
		bodyLen, err := e.u16()
		if err != nil {
			return "", err
		}
		extBody, err := e.take(bodyLen)
		if err != nil {
			return "", err
		}
		if extType == extensionServerName {
			return parseServerName(extBody)
		}
	}
	return "", errNoServerName
}

// parseServerName decodes a server_name extension body (RFC 6066 §3):
// a two-byte-length list of (type, length, name) entries, of which we
// want the host_name.
func parseServerName(ext []byte) (string, error) {
	c := &cursor{b: ext}
	listLen, err := c.u16()
	if err != nil {
		return "", err
	}
	list, err := c.take(listLen)
	if err != nil {
		return "", err
	}
	l := &cursor{b: list}
	for l.i < len(l.b) {
		nameType, err := l.u8()
		if err != nil {
			return "", err
		}
		nameLen, err := l.u16()
		if err != nil {
			return "", err
		}
		name, err := l.take(nameLen)
		if err != nil {
			return "", err
		}
		if nameType == serverNameTypeHost {
			return string(name), nil
		}
	}
	return "", errNoServerName
}

// Classify parses a ClientHello and returns the verdict that set
// applies to the advertised server name, along with the name itself.
func Classify(record []byte, set rules.Set) (rules.Verdict, string, error) {
	name, err := Parse(record)
	if err != nil {
		return "", "", err
	}
	return set.Verdict(name), name, nil
}
