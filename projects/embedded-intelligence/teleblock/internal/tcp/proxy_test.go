package tcp

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

// tls wire constants for the test ClientHello builder.
const (
	testContentTypeHandshake = 22
	testExtensionServerName  = 0
)

// testClientHello assembles a minimal ClientHello record advertising
// serverName (empty for none), mirroring the builder used by the sni
// package tests.
func testClientHello(serverName string) []byte {
	var extensions []byte
	extensions = append(extensions, ext(0x000b, []byte{0x00, 0x02, 0x00, 0x1d})...) // supported_groups
	if serverName != "" {
		host := []byte(serverName)
		entry := append([]byte{0, byte(len(host) >> 8), byte(len(host))}, host...)
		nameList := append([]byte{byte(len(entry) >> 8), byte(len(entry))}, entry...)
		extensions = append(extensions, ext(testExtensionServerName, nameList)...)
	}

	var hello []byte
	hello = append(hello, 0x03, 0x03)             // legacy_version: TLS 1.2
	hello = append(hello, make([]byte, 32)...)    // random
	hello = append(hello, 0)                      // session_id: empty
	hello = append(hello, 0x00, 0x02, 0x13, 0x01) // cipher_suites
	hello = append(hello, 0x01, 0x00)             // compression_methods: null
	hello = append(hello, byte(len(extensions)>>8), byte(len(extensions)))
	hello = append(hello, extensions...)

	handshake := append([]byte{1, byte(len(hello) >> 16), byte(len(hello) >> 8), byte(len(hello))}, hello...)
	return append([]byte{testContentTypeHandshake, 0x03, 0x01,
		byte(len(handshake) >> 8), byte(len(handshake))}, handshake...)
}

// ext encodes a type-length-value extension entry.
func ext(typ uint16, data []byte) []byte {
	e := []byte{byte(typ >> 8), byte(typ), byte(len(data) >> 8), byte(len(data))}
	return append(e, data...)
}

// startInspector starts an inspection Listener on a loopback port
// routing relays to targetAddr.
func startInspector(t *testing.T, router *profile.Router, targetAddr string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen inspector: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	ins := New(router, func(net.Conn) (string, error) { return targetAddr, nil })
	go ins.Serve(ln)
	return ln.Addr().String()
}

// startEcho starts an upstream server that echoes every payload back.
func startEcho(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	return ln.Addr().String()
}

// samsungRouter returns a router with 127.0.0.1 on the Samsung
// profile.
func samsungRouter(t *testing.T) *profile.Router {
	t.Helper()
	router := profile.New(rules.DefaultSets())
	if err := router.Assign(net.ParseIP("127.0.0.1"), rules.BrandSamsung); err != nil {
		t.Fatalf("assign: %v", err)
	}
	return router
}

func TestBlockedSNIIsReset(t *testing.T) {
	ins := startInspector(t, samsungRouter(t), startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write(testClientHello("us.samsungacr.com")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(testClientHello("us.samsungacr.com")))
	n, err := conn.Read(buf)
	if err == nil || n > 0 {
		t.Errorf("blocked flow must be reset without data, got %d bytes, err %v", n, err)
	}
}

func TestAllowedSNIIsRelayed(t *testing.T) {
	ins := startInspector(t, samsungRouter(t), startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	payload := testClientHello("www.netflix.com")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	for i := range payload {
		if got[i] != payload[i] {
			t.Fatalf("relayed payload modified at byte %d: got %02x want %02x", i, got[i], payload[i])
		}
	}
}

func TestObservedSNIIsRelayed(t *testing.T) {
	ins := startInspector(t, samsungRouter(t), startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	payload := testClientHello("samsungcloud.com") // Observe verdict
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
}

func TestOversizedRecordIsRelayedUninspected(t *testing.T) {
	// A handshake record whose declared length exceeds the RFC 5246
	// maximum (16 KiB) is invalid: the inspector must not allocate
	// on it and must relay the flow byte for byte, uninspected —
	// even when the SNI inside would be blocked.
	ins := startInspector(t, samsungRouter(t), startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	payload := append([]byte{testContentTypeHandshake, 0x03, 0x01, 0xFF, 0xFF},
		bytes.Repeat([]byte("A"), 64)...)
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("oversized record relay modified: got %d bytes", len(got))
	}
}

func TestNonTLSStreamIsRelayed(t *testing.T) {
	ins := startInspector(t, samsungRouter(t), startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	payload := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("non-TLS payload modified: got %q want %q", got, payload)
	}
}

func TestUnknownDevicesUseGenericSet(t *testing.T) {
	router := profile.New(rules.DefaultSets()) // no device assigned
	ins := startInspector(t, router, startEcho(t))
	conn, err := net.Dial("tcp", ins)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Generic set does not block samsungacr.com, so the flow relays.
	payload := testClientHello("us.samsungacr.com")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
}

func TestConcurrentFlows(t *testing.T) {
	ins := startInspector(t, samsungRouter(t), startEcho(t))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "www.netflix.com"
			wantEcho := true
			if i%2 == 0 {
				name = "samsungads.com"
				wantEcho = false
			}
			conn, err := net.Dial("tcp", ins)
			if err != nil {
				t.Errorf("dial: %v", err)
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			payload := testClientHello(name)
			if _, err := conn.Write(payload); err != nil {
				t.Errorf("write: %v", err)
				return
			}
			buf := make([]byte, len(payload))
			n, err := conn.Read(buf)
			if wantEcho {
				if _, rerr := io.ReadFull(conn, buf[n:]); rerr != nil {
					t.Errorf("flow %d: allowed SNI not relayed fully: %v", i, rerr)
					return
				}
				// First chunk + remainder equals a full echo.
				if len(buf) != len(payload) {
					t.Errorf("flow %d: echo truncated", i)
				}
				return
			}
			if err == nil && n > 0 {
				t.Errorf("flow %d: blocked SNI %s produced data", i, name)
			}
		}(i)
	}
	wg.Wait()
}

func TestNewWithNilTargetUsesOriginalDst(t *testing.T) {
	ins := New(profile.New(rules.DefaultSets()), nil)
	if ins.target == nil {
		t.Fatal("New: nil target must select OriginalDst")
	}
}
