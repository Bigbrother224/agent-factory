package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

// put16 appends v as a big-endian uint16.
func put16(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}

// buildQuery assembles a DNS query for name.
func buildQuery(id uint16, name string, qtype uint16) []byte {
	var q []byte
	q = put16(q, id)
	q = put16(q, 0x0100) // RD
	q = put16(q, 1)       // QDCOUNT
	q = put16(q, 0)       // ANCOUNT
	q = put16(q, 0)       // NSCOUNT
	q = put16(q, 0)       // ARCOUNT
	for _, label := range strings.Split(name, ".") {
		q = append(q, byte(len(label)))
		q = append(q, label...)
	}
	q = append(q, 0) // root label
	q = put16(q, qtype)
	q = put16(q, classIN)
	return q
}

// staticLookup returns a LookupFunc always answering with ips.
func staticLookup(ips ...net.IP) LookupFunc {
	return func(context.Context, string) ([]net.IP, error) {
		return ips, nil
	}
}

// failingLookup returns a LookupFunc always failing with err.
func failingLookup(err error) LookupFunc {
	return func(context.Context, string) ([]net.IP, error) {
		return nil, err
	}
}

// newTestServer returns a Server with the given devices assigned.
func newTestServer(t *testing.T, lookup LookupFunc, devices map[string]string) *Server {
	t.Helper()
	router := profile.New(rules.DefaultSets())
	for ip, brand := range devices {
		if err := router.Assign(net.ParseIP(ip), brand); err != nil {
			t.Fatalf("Assign(%s, %s): %v", ip, brand, err)
		}
	}
	return New(router, lookup)
}

// rcodeOf returns the response code of a reply.
func rcodeOf(resp []byte) int { return int(resp[3] & 0x0F) }

// answerIP returns the RDATA of the first answer record.
func answerIP(resp []byte) net.IP {
	off := 12
	for resp[off] != 0 { // walk the question name
		off += 1 + int(resp[off])
	}
	off += 1 + 4 // root label, QTYPE, QCLASS
	off += 2 + 8 // answer name pointer, TYPE, CLASS, TTL
	rdlen := int(resp[off])<<8 | int(resp[off+1])
	off += 2
	return net.IP(resp[off : off+rdlen])
}

func TestParseQuery(t *testing.T) {
	q, err := parseQuery(buildQuery(0xBEEF, "www.SamsungACR.com", typeA))
	if err != nil {
		t.Fatalf("parseQuery: %v", err)
	}
	if q.name != "www.samsungacr.com" {
		t.Errorf("name = %q, want www.samsungacr.com", q.name)
	}
	if q.qtype != typeA || q.qclass != classIN {
		t.Errorf("type/class = %d/%d, want %d/%d", q.qtype, q.qclass, typeA, classIN)
	}
}

func TestRespondBlocked(t *testing.T) {
	s := newTestServer(t, staticLookup(net.ParseIP("93.184.216.34")),
		map[string]string{"192.168.1.10": rules.BrandSamsung})
	resp := s.respond(buildQuery(42, "us.samsungacr.com", typeA), net.ParseIP("192.168.1.10"))

	if rcodeOf(resp) != rcodeNoError {
		t.Fatalf("rcode = %d, want %d", rcodeOf(resp), rcodeNoError)
	}
	if got := answerIP(resp).String(); got != "0.0.0.0" {
		t.Errorf("answer = %s, want 0.0.0.0 (sinkhole)", got)
	}
	if id := binary.BigEndian.Uint16(resp[0:2]); id != 42 {
		t.Errorf("id = %d, want 42", id)
	}
	if resp[2]&0x80 == 0 {
		t.Error("QR bit not set")
	}
}

func TestRespondAllowed(t *testing.T) {
	s := newTestServer(t, staticLookup(net.ParseIP("93.184.216.34")),
		map[string]string{"192.168.1.10": rules.BrandSamsung})
	resp := s.respond(buildQuery(7, "www.netflix.com", typeA), net.ParseIP("192.168.1.10"))

	if rcodeOf(resp) != rcodeNoError {
		t.Fatalf("rcode = %d, want %d", rcodeOf(resp), rcodeNoError)
	}
	if got := answerIP(resp).String(); got != "93.184.216.34" {
		t.Errorf("answer = %s, want 93.184.216.34", got)
	}
}

func TestRespondRoutesByDevice(t *testing.T) {
	// The same domain is sinkholed for a Samsung TV but resolved for an LG.
	s := newTestServer(t, staticLookup(net.ParseIP("93.184.216.34")), map[string]string{
		"192.168.1.10": rules.BrandSamsung,
		"192.168.1.11": rules.BrandLG,
	})
	fromSamsung := s.respond(buildQuery(1, "samsungacr.com", typeA), net.ParseIP("192.168.1.10"))
	if got := answerIP(fromSamsung).String(); got != "0.0.0.0" {
		t.Errorf("samsung: answer = %s, want 0.0.0.0", got)
	}
	fromLG := s.respond(buildQuery(1, "samsungacr.com", typeA), net.ParseIP("192.168.1.11"))
	if got := answerIP(fromLG).String(); got != "93.184.216.34" {
		t.Errorf("lg: answer = %s, want 93.184.216.34", got)
	}
}

func TestRespondServFailOnUpstreamError(t *testing.T) {
	s := newTestServer(t, failingLookup(errors.New("upstream down")), nil)
	resp := s.respond(buildQuery(7, "www.netflix.com", typeA), net.ParseIP("192.168.1.10"))
	if rcodeOf(resp) != rcodeServFail {
		t.Errorf("rcode = %d, want %d (SERVFAIL)", rcodeOf(resp), rcodeServFail)
	}
}

func TestRespondNXDomainOnNotFound(t *testing.T) {
	notFound := &net.DNSError{Err: "no such host", Name: "nope.example", IsNotFound: true}
	s := newTestServer(t, failingLookup(notFound), nil)
	resp := s.respond(buildQuery(7, "nope.example", typeA), net.ParseIP("192.168.1.10"))
	if rcodeOf(resp) != rcodeNXDomain {
		t.Errorf("rcode = %d, want %d (NXDOMAIN)", rcodeOf(resp), rcodeNXDomain)
	}
}

func TestRespondNotImplementedForNonA(t *testing.T) {
	s := newTestServer(t, staticLookup(), nil)
	resp := s.respond(buildQuery(7, "www.netflix.com", 28), net.ParseIP("192.168.1.10")) // AAAA
	if rcodeOf(resp) != rcodeNotImp {
		t.Errorf("rcode = %d, want %d (NOTIMP)", rcodeOf(resp), rcodeNotImp)
	}
}

func TestRespondFormErrOnGarbage(t *testing.T) {
	s := newTestServer(t, staticLookup(), nil)
	if resp := s.respond([]byte{0x2a}, net.ParseIP("192.168.1.10")); resp != nil {
		t.Error("want a packet without ID dropped")
	}
	resp := s.respond([]byte{0x00, 0x2a, 0xff}, net.ParseIP("192.168.1.10"))
	if rcodeOf(resp) != rcodeFormErr {
		t.Errorf("rcode = %d, want %d (FORMERR)", rcodeOf(resp), rcodeFormErr)
	}
	if id := binary.BigEndian.Uint16(resp[0:2]); id != 0x2a {
		t.Errorf("id = %#x, want 0x2a", id)
	}
}

func TestRespondFormErrOnMultipleQuestions(t *testing.T) {
	s := newTestServer(t, staticLookup(), nil)
	q := buildQuery(7, "a.example", typeA)
	binary.BigEndian.PutUint16(q[4:6], 2) // lie about the question count
	resp := s.respond(q, net.ParseIP("192.168.1.10"))
	if rcodeOf(resp) != rcodeFormErr {
		t.Errorf("rcode = %d, want %d (FORMERR)", rcodeOf(resp), rcodeFormErr)
	}
}
