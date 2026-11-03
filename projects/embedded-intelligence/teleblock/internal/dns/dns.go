// Package dns implements the TeleBlock DNS server: a minimal UDP
// resolver that answers A queries according to the rule set of the
// querying device.
package dns

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

const (
	maxPacket = 512 // classic DNS over UDP limit
	answerTTL = 60  // seconds
	typeA     = 1
	classIN   = 1
)

// Response codes (RFC 1035 §4.1.1).
const (
	rcodeNoError  = 0
	rcodeFormErr  = 1
	rcodeServFail = 2
	rcodeNXDomain = 3
	rcodeNotImp   = 4
)

// LookupFunc resolves a domain to its A records. Production uses
// Upstream; tests inject fakes.
type LookupFunc func(ctx context.Context, domain string) ([]net.IP, error)

// Upstream returns a LookupFunc that forwards to the DNS server at
// addr and returns only its A records.
func Upstream(addr string) LookupFunc {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", addr)
		},
	}
	return func(ctx context.Context, domain string) ([]net.IP, error) {
		return resolver.LookupIP(ctx, "ip4", domain)
	}
}

// Server answers A queries with per-device verdicts: blocked names are
// sinkholed to 0.0.0.0, everything else is resolved upstream.
type Server struct {
	router  *profile.Router
	lookup  LookupFunc
	timeout time.Duration
}

// New returns a Server routing devices through router and resolving
// allowed names with lookup.
func New(router *profile.Router, lookup LookupFunc) *Server {
	return &Server{router: router, lookup: lookup, timeout: 5 * time.Second}
}

// ListenAndServe answers queries on addr until the socket fails.
func (s *Server) ListenAndServe(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("dns: listening on %s", conn.LocalAddr())

	for {
		buf := make([]byte, maxPacket)
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		pkt := buf[:n]
		go func() {
			if resp := s.respond(pkt, src.IP); resp != nil {
				if _, err := conn.WriteToUDP(resp, src); err != nil {
					log.Printf("dns: write to %s: %v", src, err)
				}
			}
		}()
	}
}

// respond builds the reply to a query coming from srcIP.
func (s *Server) respond(pkt []byte, srcIP net.IP) []byte {
	q, err := parseQuery(pkt)
	if err != nil {
		log.Printf("dns: malformed query from %s: %v", srcIP, err)
		return formErr(pkt)
	}
	if q.qtype != typeA || q.qclass != classIN {
		return buildResponse(q, rcodeNotImp, nil)
	}

	switch s.router.Verdict(srcIP, q.name) {
	case rules.Block:
		log.Printf("dns: block %s (%s profile)", q.name, s.router.Brand(srcIP))
		return buildResponse(q, rcodeNoError, []net.IP{net.IPv4zero})
	case rules.Observe:
		log.Printf("dns: observe %s (%s profile)", q.name, s.router.Brand(srcIP))
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	ips, err := s.lookup(ctx, q.name)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return buildResponse(q, rcodeNXDomain, nil)
		}
		return buildResponse(q, rcodeServFail, nil)
	}
	if len(ips) == 0 {
		return buildResponse(q, rcodeNXDomain, nil)
	}
	return buildResponse(q, rcodeNoError, ips)
}

// query is a parsed DNS query.
type query struct {
	id      uint16
	flags   uint16
	name    string // dotted, lowercased, without trailing dot
	qtype   uint16
	qclass  uint16
	section []byte // raw question section, echoed in the reply
}

// parseQuery parses the header and the single question of a DNS query.
func parseQuery(b []byte) (query, error) {
	if len(b) < 12 {
		return query{}, errors.New("dns: short header")
	}
	q := query{
		id:    binary.BigEndian.Uint16(b[0:2]),
		flags: binary.BigEndian.Uint16(b[2:4]),
	}
	if binary.BigEndian.Uint16(b[4:6]) != 1 {
		return q, errors.New("dns: expected exactly one question")
	}
	name, off, err := parseName(b, 12)
	if err != nil {
		return q, err
	}
	if len(b) < off+4 {
		return q, errors.New("dns: truncated question")
	}
	q.name = name
	q.qtype = binary.BigEndian.Uint16(b[off : off+2])
	q.qclass = binary.BigEndian.Uint16(b[off+2 : off+4])
	q.section = b[12 : off+4]
	return q, nil
}

// parseName decodes the QNAME labels starting at off and returns the
// name plus the offset just past it. Label lengths above 63 are
// invalid, which also rejects compression pointers — queries never
// contain them.
func parseName(b []byte, off int) (string, int, error) {
	var sb strings.Builder
	for {
		if off >= len(b) {
			return "", 0, errors.New("dns: truncated name")
		}
		n := int(b[off])
		if n == 0 {
			off++
			break
		}
		if n > 63 || off+1+n > len(b) {
			return "", 0, errors.New("dns: bad label")
		}
		if sb.Len() > 0 {
			sb.WriteByte('.')
		}
		sb.Write(b[off+1 : off+1+n])
		off += 1 + n
		if sb.Len() > 253 {
			return "", 0, errors.New("dns: name too long")
		}
	}
	return strings.ToLower(sb.String()), off, nil
}

// formErr builds a best-effort FORMERR reply for an unparseable query,
// or nil when there is not even an ID to echo.
func formErr(b []byte) []byte {
	if len(b) < 2 {
		return nil
	}
	resp := make([]byte, 12)
	binary.BigEndian.PutUint16(resp[0:2], binary.BigEndian.Uint16(b[0:2]))
	binary.BigEndian.PutUint16(resp[2:4], 0x8000|rcodeFormErr)
	return resp
}

// buildResponse assembles a reply that echoes the question and carries
// one A record per answer IP.
func buildResponse(q query, rcode int, ips []net.IP) []byte {
	// QR=1, RA=1, copy OPCODE and RD from the query, set RCODE.
	flags := q.flags&0x7900 | 0x8080 | uint16(rcode)

	resp := make([]byte, 12, 12+len(q.section)+len(ips)*16)
	binary.BigEndian.PutUint16(resp[0:2], q.id)
	binary.BigEndian.PutUint16(resp[2:4], flags)
	binary.BigEndian.PutUint16(resp[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(resp[6:8], uint16(len(ips)))
	resp = append(resp, q.section...)

	for _, ip := range ips {
		var ans [16]byte
		binary.BigEndian.PutUint16(ans[0:2], 0xC00C) // NAME: pointer to the question
		binary.BigEndian.PutUint16(ans[2:4], typeA)
		binary.BigEndian.PutUint16(ans[4:6], classIN)
		binary.BigEndian.PutUint32(ans[6:10], answerTTL)
		binary.BigEndian.PutUint16(ans[10:12], 4) // RDLENGTH
		copy(ans[12:16], ip.To4())
		resp = append(resp, ans[:]...)
	}
	return resp
}
