// Package ja4 computes JA4 fingerprints of TLS ClientHello records
// (FoxIO-LLC, github.com/FoxIO-LLC/ja4). The fingerprint is a compact
// identifier of the client TLS stack — version, cipher suites,
// extensions — that TeleBlock uses to class unknown flows observed by
// the proxy: flows without a known SNI verdict are keyed by their
// fingerprint and routed to the Observe verdict instead of Block.
// Nothing here decrypts traffic: the ClientHello is read in the clear,
// exactly like the SNI parser.
package ja4

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// TLS wire constants (RFC 5246, RFC 8446).
const (
	contentTypeHandshake     = 22 // 0x16
	handshakeTypeClientHello = 1  // 0x01
	extensionServerName      = 0x0000
	extensionALPN            = 0x0010
	extensionSigAlgs         = 0x000d
	extensionSupportedVers   = 0x002b
)

var (
	errTruncated      = errors.New("ja4: truncated record")
	errNotHandshake   = errors.New("ja4: not a handshake record")
	errNotClientHello = errors.New("ja4: not a ClientHello")
)

// Hello holds the ClientHello attributes hashed into a JA4
// fingerprint, kept as a struct so callers can expose raw data (logs,
// classification tables) alongside the fingerprint string.
type Hello struct {
	Version      string   // 2 characters: "13", "12", "10"… "00" unknown
	HasSNI       bool     // server_name extension present (→ domain)
	CipherSuites []uint16 // GREASE excluded, in wire order
	Extensions   []uint16 // GREASE excluded, in wire order
	SigAlgs      []uint16 // signature_algorithms (0x000d), wire order
	ALPN         []string // application_layer_protocol_negotiation (0x0010)
}

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

func (c *cursor) take(n int) ([]byte, error) {
	if n < 0 || c.i+n > len(c.b) {
		return nil, errTruncated
	}
	v := c.b[c.i : c.i+n]
	c.i += n
	return v, nil
}

// skipVector8 skips a one-byte-length vector (session id, compression
// methods).
func (c *cursor) skipVector8() error {
	n, err := c.u8()
	if err != nil {
		return err
	}
	_, err = c.take(int(n))
	return err
}

// Parse decodes a TLS record and returns the Hello attributes of its
// ClientHello. The input is a full record (header + body), the same
// shape sni.Parse accepts. GREASE values are ignored everywhere, as
// the FOxIO spec requires.
func Parse(record []byte) (*Hello, error) {
	// Record: content type, version, length.
	c := &cursor{b: record}
	typ, err := c.u8()
	if err != nil {
		return nil, err
	}
	if typ != contentTypeHandshake {
		return nil, errNotHandshake
	}
	if _, err := c.take(2); err != nil { // record version
		return nil, err
	}
	recLen, err := c.u16()
	if err != nil {
		return nil, err
	}
	body, err := c.take(recLen)
	if err != nil {
		return nil, err
	}

	// Handshake: type, length.
	h := &cursor{b: body}
	msgType, err := h.u8()
	if err != nil {
		return nil, err
	}
	if msgType != handshakeTypeClientHello {
		return nil, errNotClientHello
	}
	helloLen, err := h.u24()
	if err != nil {
		return nil, err
	}
	hello, err := h.take(helloLen)
	if err != nil {
		return nil, err
	}

	// ClientHello: legacy version, random, session id, cipher
	// suites, compression methods, extensions.
	parsed := &Hello{}
	ch := cursor{b: hello}
	legacy, err := ch.u16()
	if err != nil {
		return nil, err
	}
	parsed.Version = versionField(uint16(legacy))
	if _, err := ch.take(32); err != nil { // random
		return nil, err
	}
	if err := ch.skipVector8(); err != nil { // session id
		return nil, err
	}

	cipherLen, err := ch.u16()
	if err != nil {
		return nil, err
	}
	for i := 0; i < cipherLen; i += 2 {
		suite, err := ch.u16()
		if err != nil {
			return nil, err
		}
		if isGrease(uint16(suite)) {
			continue
		}
		parsed.CipherSuites = append(parsed.CipherSuites, uint16(suite))
	}

	if err := ch.skipVector8(); err != nil { // compression methods
		return nil, err
	}
	extLen, err := ch.u16()
	if err != nil {
		return nil, err
	}
	exts, err := ch.take(extLen)
	if err != nil {
		return nil, err
	}

	extensions := &cursor{b: exts}
	supported := []uint16{}
	for extensions.i < len(extensions.b) {
		extType, err := extensions.u16()
		if err != nil {
			return nil, err
		}
		bodyLen, err := extensions.u16()
		if err != nil {
			return nil, err
		}
		extBody, err := extensions.take(bodyLen)
		if err != nil {
			return nil, err
		}
		if isGrease(uint16(extType)) {
			continue
		}
		parsed.Extensions = append(parsed.Extensions, uint16(extType))
		switch extType {
		case extensionServerName:
			parsed.HasSNI = true
		case extensionALPN:
			parsed.ALPN, err = parseALPN(extBody)
			if err != nil {
				return nil, err
			}
		case extensionSigAlgs:
			parsed.SigAlgs, err = parseU16List(extBody)
			if err != nil {
				return nil, err
			}
		case extensionSupportedVers:
			// RFC 8446 §4.2.1 : la liste des versions supportées du
			// ClientHello est préfixée par une longueur sur UN octet,
			// contrairement à signature_algorithms (RFC 5246, deux octets).
			supported, err = parseU16List8(extBody)
			if err != nil {
				return nil, err
			}
		}
	}

	// The negotiated version: highest supported_versions value when
	// the extension exists (GREASE ignored), else the legacy version.
	nonGrease := make([]uint16, 0, len(supported))
	for _, v := range supported {
		if !isGrease(v) {
			nonGrease = append(nonGrease, v)
		}
	}
	if len(nonGrease) > 0 {
		max := nonGrease[0]
		for _, v := range nonGrease {
			if v > max {
				max = v
			}
		}
		parsed.Version = versionField(max)
	}
	return parsed, nil
}

// Fingerprint computes the JA4 fingerprint of a TLS ClientHello
// record, e.g. "t13d1516h2_8daaf6152771_e5627efa2ab1".
func Fingerprint(record []byte) (string, error) {
	h, err := Parse(record)
	if err != nil {
		return "", err
	}
	return h.String(), nil
}

// String returns the JA4 fingerprint of the Hello: protocol block, a
// truncated sha256 of the sorted cipher suites, then a truncated
// sha256 of the sorted extensions followed by the signature
// algorithms in wire order.
func (h *Hello) String() string {
	return fmt.Sprintf("%s_%s_%s", h.protocolBlock(), h.cipherHash(), h.extensionHash())
}

// Raw returns the JA4_r raw fingerprint: unhashed sorted cipher and
// extension lists.
func (h *Hello) Raw() string {
	return fmt.Sprintf("%s_%s_%s", h.protocolBlock(),
		strings.Join(sortedHex(h.CipherSuites), ","),
		strings.Join(sortedHex(h.Extensions), ","))
}

// protocolBlock is the first JA4 field: protocol, TLS version,
// domain/IP, cipher count, extension count, ALPN characters.
func (h *Hello) protocolBlock() string {
	ciphers, extensions := len(h.CipherSuites), len(h.Extensions)
	if ciphers > 99 {
		ciphers = 99
	}
	if extensions > 99 {
		extensions = 99
	}
	kind := "i" // no server_name: destination is an IP address
	if h.HasSNI {
		kind = "d"
	}
	return fmt.Sprintf("t%s%s%02d%02d%s", h.Version, kind, ciphers, extensions, h.alpnField())
}

// alpnField is the first and last character of the first ALPN value,
// or "00" when absent. When the first or last byte is not an ASCII
// alphanumeric character, the first and last characters of the hex
// representation are used instead (FoxIO spec examples).
func (h *Hello) alpnField() string {
	if len(h.ALPN) == 0 || h.ALPN[0] == "" {
		return "00"
	}
	v := h.ALPN[0]
	first, last := v[0], v[len(v)-1]
	if isASCIIAlnum(first) && isASCIIAlnum(last) {
		return string([]byte{first, last})
	}
	hs := hex.EncodeToString([]byte(v))
	return hs[:1] + hs[len(hs)-1:]
}

// cipherHash is the truncated sha256 of the cipher suites sorted in
// hex order, or 000000000000 when none.
func (h *Hello) cipherHash() string {
	if len(h.CipherSuites) == 0 {
		return "000000000000"
	}
	return shortHash(strings.Join(sortedHex(h.CipherSuites), ","))
}

// extensionHash is the truncated sha256 of the extensions sorted in
// hex order (no server_name, no ALPN — both are already in the
// protocol block), followed by the signature algorithms in wire
// order, or 000000000000 when the extension list is empty.
func (h *Hello) extensionHash() string {
	var list []string
	for _, e := range h.Extensions {
		if e == extensionServerName || e == extensionALPN {
			continue
		}
		list = append(list, hex4(e))
	}
	if len(list) == 0 {
		return "000000000000"
	}
	sort.Strings(list)
	if len(h.SigAlgs) > 0 {
		return shortHash(strings.Join(list, ",") + "_" + strings.Join(sortedHexKeepOrder(h.SigAlgs), ","))
	}
	return shortHash(strings.Join(list, ","))
}

// versionField maps a TLS version byte pair to its JA4 characters
// (FoxIOspec table); unknown versions map to "00".
func versionField(v uint16) string {
	switch v {
	case 0x0304:
		return "13"
	case 0x0303:
		return "12"
	case 0x0302:
		return "11"
	case 0x0301:
		return "10"
	case 0x0300:
		return "s3"
	case 0x0002:
		return "s2"
	case 0xfeff:
		return "d1"
	case 0xfefd:
		return "d2"
	case 0xfefc:
		return "d3"
	default:
		return "00"
	}
}

// isGrease reports whether v is a GREASE value (draft-davidben-tls-
// grease: 0x?A0A pattern, both bytes equal). GREASE must be ignored
// in cipher suites, extensions and supported_versions.
func isGrease(v uint16) bool {
	return v>>8 == v&0xFF && v&0x0F == 0x0A
}

// parseALPN decodes an application_layer_protocol_negotiation body: a
// two-byte list of one-byte-length protocol name strings.
func parseALPN(b []byte) ([]string, error) {
	c := cursor{b: b}
	listLen, err := c.u16()
	if err != nil {
		return nil, err
	}
	if listLen != len(b)-2 {
		return nil, errTruncated
	}
	var list []string
	for c.i < len(c.b) {
		n, err := c.u8()
		if err != nil {
			return nil, err
		}
		name, err := c.take(int(n))
		if err != nil {
			return nil, err
		}
		list = append(list, string(name))
	}
	return list, nil
}

// parseU16List decodes a two-byte-length list of u16 values
// (signature_algorithms, RFC 5246).
func parseU16List(b []byte) ([]uint16, error) {
	c := cursor{b: b}
	listLen, err := c.u16()
	if err != nil {
		return nil, err
	}
	if listLen != len(b)-2 {
		return nil, errTruncated
	}
	var list []uint16
	for c.i < len(c.b) {
		v, err := c.u16()
		if err != nil {
			return nil, err
		}
		list = append(list, uint16(v))
	}
	return list, nil
}

// parseU16List8 decodes a one-byte-length list of u16 values
// (supported_versions, RFC 8446 §4.2.1: the ClientHello carries
// SupportedVersions<24..2^8-2>, a one-byte length prefix).
func parseU16List8(b []byte) ([]uint16, error) {
	if len(b) < 1 {
		return nil, errTruncated
	}
	listLen := int(b[0])
	if listLen != len(b)-1 || listLen%2 != 0 {
		return nil, errTruncated
	}
	var list []uint16
	c := cursor{b: b[1:]}
	for c.i < len(c.b) {
		v, err := c.u16()
		if err != nil {
			return nil, err
		}
		list = append(list, uint16(v))
	}
	return list, nil
}

// sortedHex formats values as lower-case 4-character hex strings,
// sorted in hex order (identical to numeric order at fixed width).
func sortedHex(values []uint16) []string {
	vs := make([]string, len(values))
	for i, v := range values {
		vs[i] = hex4(v)
	}
	sort.Strings(vs)
	return vs
}

// sortedHexKeepOrder formats values without sorting (signature
// algorithms must keep their wire order in the extension hash).
func sortedHexKeepOrder(values []uint16) []string {
	vs := make([]string, len(values))
	for i, v := range values {
		vs[i] = hex4(v)
	}
	return vs
}

func hex4(v uint16) string {
	return fmt.Sprintf("%04x", v)
}

// shortHash is the first 12 characters of the lower-case sha256.
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

// isASCIIAlnum reports ASCII 0x30-0x39, 0x41-0x5A, 0x61-0x7A.
func isASCIIAlnum(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
