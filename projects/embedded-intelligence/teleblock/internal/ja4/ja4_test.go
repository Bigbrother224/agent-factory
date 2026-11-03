package ja4

import (
	"bytes"
	"strings"
	"testing"
)

// ext encodes a type-length-value extension entry.
func ext(typ uint16, data []byte) []byte {
	e := []byte{byte(typ >> 8), byte(typ), byte(len(data) >> 8), byte(len(data))}
	return append(e, data...)
}

// u16vector encodes a two-byte-length list of u16 values.
func u16vector(values []uint16) []byte {
	var v bytes.Buffer
	v.Write([]byte{byte(len(values) * 2 >> 8), byte(len(values) * 2)})
	for _, x := range values {
		v.Write([]byte{byte(x >> 8), byte(x)})
	}
	return v.Bytes()
}

// alpnvector encodes an application_layer_protocol_negotiation body.
func alpnvector(names ...string) []byte {
	var body bytes.Buffer
	for _, n := range names {
		body.Write([]byte{byte(len(n))})
		body.WriteString(n)
	}
	v := []byte{byte(body.Len() >> 8), byte(body.Len())}
	return append(v, body.Bytes()...)
}

// helloRecord assembles a full TLS record carrying a ClientHello.
func helloRecord(legacyVersion uint16, ciphers, extensions []uint16, bodies map[uint16][]byte) []byte {
	var exts bytes.Buffer
	for _, e := range extensions {
		exts.Write(ext(e, bodies[e]))
	}

	var hello bytes.Buffer
	hello.Write([]byte{byte(legacyVersion >> 8), byte(legacyVersion)})
	hello.Write(make([]byte, 32)) // random
	hello.Write([]byte{0})        // session_id: empty
	hello.Write(byteVec(ciphers))
	hello.Write([]byte{1, 0}) // compression_methods: null
	hello.Write([]byte{byte(exts.Len() >> 8), byte(exts.Len())})
	hello.Write(exts.Bytes())

	handshake := append([]byte{1, byte(hello.Len() >> 16), byte(hello.Len() >> 8), byte(hello.Len())}, hello.Bytes()...)
	return append([]byte{contentTypeHandshake, 0x03, 0x01,
		byte(len(handshake) >> 8), byte(len(handshake))}, handshake...)
}

func byteVec(values []uint16) []byte {
	v := []byte{byte(len(values) * 2 >> 8), byte(len(values) * 2)}
	for _, x := range values {
		v = append(v, byte(x>>8), byte(x))
	}
	return v
}

// hello with a host_name server_name extension body.
const testHost = "example.com"

func sniBody() []byte {
	entry := append([]byte{0, 0, byte(len(testHost))}, testHost...)
	list := append([]byte{byte(len(entry) >> 8), byte(len(entry))}, entry...)
	return list
}

// The reference example of the FoxIO JA4 spec: 15 cipher suites, 16
// extensions, ALPN h2, TLS 1.3 through supported_versions.
func specExampleRecord() []byte {
	ciphers := []uint16{0x1301, 0x1302, 0x1303, 0xc02b, 0xc02f, 0xc02c, 0xc030,
		0xcca9, 0xcca8, 0xc013, 0xc014, 0x009c, 0x009d, 0x002f, 0x0035}
	extensions := []uint16{0x001b, 0x0000, 0x0033, 0x0010, 0x4469, 0x0017,
		0x002d, 0x000d, 0x0005, 0x0023, 0x0012, 0x002b, 0xff01, 0x000b, 0x000a, 0x0015}
	bodies := map[uint16][]byte{
		extensionServerName:    sniBody(),
		extensionALPN:          alpnvector("h2"),
		extensionSigAlgs:       u16vector([]uint16{0x0403, 0x0804, 0x0401, 0x0503, 0x0805, 0x0501, 0x0806, 0x0601}),
		extensionSupportedVers: u16vector([]uint16{0x0304}),
		0x001b:                 {0x01, 0x00},
		0x0033:                 {0x00, 0x17, 0x00, 0x41, 0x04},
		0x4469:                 {0x04, 0x00, 0x02, 0x04, 0x01},
		0x0017:                 {},
		0x002d:                 {0x00, 0x02, 0x00, 0x01},
		0x0005:                 {},
		0x0023:                 {},
		0x0012:                 {},
		0xff01:                 {0x01, 0x00},
		0x000b:                 {0x02, 0x00, 0x1d},
		0x000a:                 {0x00, 0x0a, 0x00, 0x08},
		0x0015:                 {0x00},
		0x0033 - 0x0032:        nil,
	}
	return helloRecord(0x0303, ciphers, extensions, bodies)
}

func TestSpecExampleFingerprint(t *testing.T) {
	got, err := Fingerprint(specExampleRecord())
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	// Vector from the FoxIO JA4 technical details (JA4.md): cipher
	// hash 8daaf6152771, extension+sigalgs hash e5627efa2ab1.
	want := "t13d1516h2_8daaf6152771_e5627efa2ab1"
	if got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
}

func TestTLS12WithoutALPN(t *testing.T) {
	// TLS 1.2 negotiated through the legacy version field, one GREASE
	// cipher and one GREASE extension that must be ignored, ALPN
	// http/1.1, no signature algorithms.
	ciphers := []uint16{0xc02f, 0xc030, 0x0a0a, 0x002f}
	extensions := []uint16{0x0000, 0x000b, 0x0010, 0x0a0a}
	bodies := map[uint16][]byte{
		extensionServerName: sniBody(),
		extensionALPN:       alpnvector("http/1.1"),
		0x000b:              {0x02, 0x00, 0x1d},
	}
	got, err := Fingerprint(helloRecord(0x0303, ciphers, extensions, bodies))
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	want := "t12d0303h1_7adbb56324b3_" + shortHash("000b")
	if got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
}

func TestTLS13NoSNIAndNoALPN(t *testing.T) {
	// No server_name (→ i), no ALPN (→ 00), version taken from
	// supported_versions, GREASE versions ignored there too.
	ciphers := []uint16{0x1301}
	extensions := []uint16{0x000d, 0x002b}
	bodies := map[uint16][]byte{
		extensionSigAlgs:       u16vector([]uint16{0x1301, 0x1302}),
		extensionSupportedVers: u16vector([]uint16{0x0a0a, 0x0304, 0x0303}),
	}
	got, err := Fingerprint(helloRecord(0x0303, ciphers, extensions, bodies))
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	want := "t13i010200_0f2cb44170f4_" + shortHash("000d,002b_1301,1302")
	if got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
}

func TestParseAttributes(t *testing.T) {
	h, err := Parse(specExampleRecord())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if h.Version != "13" {
		t.Errorf("Version = %q, want %q", h.Version, "13")
	}
	if !h.HasSNI {
		t.Error("HasSNI = false, want true")
	}
	if len(h.CipherSuites) != 15 {
		t.Errorf("CipherSuites count = %d, want 15", len(h.CipherSuites))
	}
	if len(h.Extensions) != 16 {
		t.Errorf("Extensions count = %d, want 16", len(h.Extensions))
	}
	wantSig := []uint16{0x0403, 0x0804, 0x0401, 0x0503, 0x0805, 0x0501, 0x0806, 0x0601}
	for i, v := range wantSig {
		if h.SigAlgs[i] != v {
			t.Errorf("SigAlgs[%d] = %04x, want %04x", i, h.SigAlgs[i], v)
		}
	}
	if len(h.ALPN) != 1 || h.ALPN[0] != "h2" {
		t.Errorf("ALPN = %v, want [h2]", h.ALPN)
	}
}

func TestCountsAreTwoDigits(t *testing.T) {
	// More than 9 ciphers: counts keep two decimal digits.
	var ciphers []uint16
	for i := 0; i < 11; i++ {
		ciphers = append(ciphers, uint16(0x1300+i))
	}
	h, err := Parse(helloRecord(0x0303, ciphers, nil, nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := h.protocolBlock(); len(got) != 10 || got != "t12i110000" {
		t.Errorf("protocolBlock = %q, want t12i110000", got)
	}
	want := "1300,1301,1302,1303,1304,1305,1306,1307,1308,1309,130a"
	if got := strings.Join(sortedHex(h.CipherSuites), ","); got != want {
		t.Errorf("sorted ciphers = %q, want %q", got, want)
	}
	if got := h.extensionHash(); got != "000000000000" {
		t.Errorf("extensionHash = %q, want 000000000000", got)
	}
}

func TestCountCappedAt99(t *testing.T) {
	var ciphers []uint16
	for i := 0; i < 105; i++ {
		ciphers = append(ciphers, uint16(0x1000+i))
	}
	h, err := Parse(helloRecord(0x0303, ciphers, nil, nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := h.protocolBlock(); !strings.HasPrefix(got, "t12i99") {
		t.Errorf("protocolBlock = %q, want t12i99 prefix (cap 99)", got)
	}
}

func TestUnknownVersionIs00(t *testing.T) {
	h, err := Parse(helloRecord(0x0750, []uint16{0x1301}, nil, nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if h.Version != "00" {
		t.Errorf("Version = %q, want 00", h.Version)
	}
	if !strings.HasPrefix(h.String(), "t00") {
		t.Errorf("fingerprint = %q, want t00 prefix", h.String())
	}
}

func TestEmptyFingerprintFields(t *testing.T) {
	// Only a SNI extension: the extension hash list is empty after
	// removing server_name → c = 000000000000. The SNI still counts
	// as one extension in the protocol block.
	rec := helloRecord(0x0303, nil, []uint16{0x0000}, map[uint16][]byte{0x0000: sniBody()})
	got, err := Fingerprint(rec)
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	want := "t12d000100_000000000000_000000000000"
	if got != want {
		t.Errorf("Fingerprint = %q, want %q", got, want)
	}
}

func TestALPNFields(t *testing.T) {
	tests := []struct {
		alpn []string
		want string
	}{
		{nil, "00"},
		{[]string{""}, "00"},
		{[]string{"h2", "http/1.1"}, "h2"},   // first value only
		{[]string{"http/1.1", "h2"}, "h1"},   // first value only
		{[]string{"a"}, "aa"},                // single character: both ends
		{[]string{"spdy/3.1"}, "s1"},         // first and last alnum chars
		{[]string{"\xab\xcd"}, "ad"},         // hex fallback (spec example)
		{[]string{" \x61"}, "21"},            // hex fallback (spec example)
		{[]string{"\x30\x31\xab\xcd"}, "3d"}, // hex fallback (spec example)
	}
	for _, tc := range tests {
		h := &Hello{Version: "12", ALPN: tc.alpn}
		if got := h.alpnField(); got != tc.want {
			t.Errorf("alpnField(%q) = %q, want %q", tc.alpn, got, tc.want)
		}
	}
}

func TestNotHandshakeAndNotClientHello(t *testing.T) {
	if _, err := Fingerprint([]byte("GET / HTTP/1.1\r\n")); err == nil {
		t.Error("plain HTTP accepted, want error")
	}
	notHello := append([]byte{contentTypeHandshake, 0x03, 0x01, 0x00, 0x04}, []byte{2, 0, 0, 0}...)
	if _, err := Fingerprint(notHello); err == nil {
		t.Error("ServerHello accepted, want error")
	}
}

func TestTruncatedRecords(t *testing.T) {
	full := specExampleRecord()
	for _, n := range []int{1, 5, 9, 40, len(full) - 1} {
		if _, err := Fingerprint(full[:n]); err == nil {
			t.Errorf("Fingerprint(full[:%d]) accepted a truncated record", n)
		}
	}
}
