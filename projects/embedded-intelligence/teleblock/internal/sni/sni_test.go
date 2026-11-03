package sni

import (
	"testing"

	"teleblock/internal/rules"
)

// buildClientHello assembles a minimal ClientHello advertising
// serverName (empty for none), with a dummy extension first to prove
// the parser walks the extension list.
func buildClientHello(serverName string) []byte {
	var extensions []byte
	extensions = append(extensions, extension(0x000b, []byte{0x00, 0x02, 0x00, 0x1d})...) // supported_groups
	if serverName != "" {
		host := []byte(serverName)
		// RFC 6066 §3: the extension body is a two-byte-length
		// ServerNameList of (type, length, name) entries.
		entry := append([]byte{serverNameTypeHost, byte(len(host) >> 8), byte(len(host))}, host...)
		nameList := append([]byte{byte(len(entry) >> 8), byte(len(entry))}, entry...)
		extensions = append(extensions, extension(extensionServerName, nameList)...)
	}

	var hello []byte
	hello = append(hello, 0x03, 0x03)            // legacy_version: TLS 1.2
	hello = append(hello, make([]byte, 32)...)    // random
	hello = append(hello, 0)                      // session_id: empty
	hello = append(hello, 0x00, 0x02, 0x13, 0x01) // cipher_suites: TLS_AES_128_GCM_SHA256
	hello = append(hello, 0x01, 0x00)             // compression_methods: null
	hello = append(hello, byte(len(extensions)>>8), byte(len(extensions)))
	hello = append(hello, extensions...)

	handshake := append([]byte{handshakeTypeClientHello,
		byte(len(hello) >> 16), byte(len(hello) >> 8), byte(len(hello))}, hello...)
	return append([]byte{contentTypeHandshake, 0x03, 0x01,
		byte(len(handshake) >> 8), byte(len(handshake))}, handshake...)
}

// extension encodes a type-length-value extension entry.
func extension(typ uint16, data []byte) []byte {
	ext := []byte{byte(typ >> 8), byte(typ), byte(len(data) >> 8), byte(len(data))}
	return append(ext, data...)
}

func TestParse(t *testing.T) {
	got, err := Parse(buildClientHello("www.netflix.com"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got != "www.netflix.com" {
		t.Errorf("Parse = %q, want www.netflix.com", got)
	}
}

func TestParseWithoutServerName(t *testing.T) {
	if _, err := Parse(buildClientHello("")); err == nil {
		t.Error("Parse without SNI: want error, got nil")
	}
}

func TestParseRejectsNonHandshakeRecord(t *testing.T) {
	record := buildClientHello("example.com")
	record[0] = 23 // application_data
	if _, err := Parse(record); err == nil {
		t.Error("Parse of a non-handshake record: want error, got nil")
	}
}

func TestParseRejectsNonClientHello(t *testing.T) {
	record := buildClientHello("example.com")
	record[5] = 2 // ServerHello
	if _, err := Parse(record); err == nil {
		t.Error("Parse of a non-ClientHello handshake: want error, got nil")
	}
}

func TestParseRejectsTruncatedRecord(t *testing.T) {
	record := buildClientHello("example.com")
	if _, err := Parse(record[:len(record)-10]); err == nil {
		t.Error("Parse of a truncated record: want error, got nil")
	}
}

func TestClassify(t *testing.T) {
	samsung := rules.DefaultSets()[rules.BrandSamsung]
	tests := []struct {
		name string
		want rules.Verdict
	}{
		{"us.samsungacr.com", rules.Block},
		{"www.netflix.com", rules.Allow},
		{"samsungcloud.com", rules.Observe},
	}
	for _, tt := range tests {
		verdict, name, err := Classify(buildClientHello(tt.name), samsung)
		if err != nil {
			t.Fatalf("Classify(%s): %v", tt.name, err)
		}
		if name != tt.name {
			t.Errorf("Classify(%s) parsed %q", tt.name, name)
		}
		if verdict != tt.want {
			t.Errorf("Classify(%s) = %q, want %q", tt.name, verdict, tt.want)
		}
	}
}
