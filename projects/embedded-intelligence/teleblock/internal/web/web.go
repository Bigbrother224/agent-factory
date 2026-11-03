// Package web serves TéléBloc's local dashboard at http://teleblock.local.
// One embedded page, three actions: see the machines on the LAN (ARP cache +
// reverse DNS), assign a brand to a TV in one click, and state in plain
// language how many tracking attempts were blocked today, per device.
// Zero account, zero telemetry: no external font, script or request, and the
// counters only ever live in this process. Verdict sources (DNS/TCP decision
// path) feed Counter.RecordBlocked; that wiring is a later tranche because
// those packages are owned elsewhere.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

//go:embed templates/index.html
var templatesFS embed.FS

var indexTmpl = template.Must(template.ParseFS(templatesFS, "templates/index.html"))

// brandDisplay states a brand the way the household knows it.
var brandDisplay = map[string]string{
	"samsung": "Samsung",
	"lg":      "LG",
	"sony":    "Sony",
	"tcl":     "TCL",
	"generic": "Generic",
}

const (
	incompleteMAC = "00:00:00:00:00:00"
	broadcastMAC  = "ff:ff:ff:ff:ff:ff"
	plainZero     = "Aucune tentative de tracking bloquée aujourd'hui"
)

// Host is a machine seen on the LAN.
type Host struct {
	IP  string
	MAC string
}

// brandOption is one assignable brand, drawn as one clickable chip.
type brandOption struct {
	Value    string
	Label    string
	Selected bool
}

// device is one row of the panel: a TV or any machine seen on the LAN.
type device struct {
	IP           string
	Brand        string
	MAC          string
	Hostname     string
	Name         string
	Addr         string
	Lamp         string
	IsTV         bool
	Blocked      int
	BlockedText  string
	CountPhrase  string
	Options      []brandOption
}

type pageData struct {
	Devices []device
	TVsText string
}

type deviceJSON struct {
	IP          string `json:"ip"`
	MAC         string `json:"mac,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	Brand       string `json:"brand,omitempty"`
	Blocked     int    `json:"blocked_today"`
	BlockedText string `json:"blocked_today_text"`
}

// Server is the dashboard; it is an http.Handler to attach to a listener.
type Server struct {
	router   *profile.Router
	counter  *Counter
	hostname func(ip string) string
	arp      func() ([]Host, error)

	mu    sync.Mutex
	known map[string]string // ip -> brand, mirror of assignments since startup
}

// New builds the dashboard. known pre-fills the TVs assigned at launch
// (the -device flags); LAN discovery adds the rest.
func New(router *profile.Router, known map[string]string) *Server {
	k := make(map[string]string, len(known))
	for ip, brand := range known {
		k[ip] = brand
	}
	return &Server{
		router:   router,
		counter:  NewCounter(),
		hostname: systemHostname,
		arp:      readSystemArp,
		known:    k,
	}
}

// ServeHTTP routes the three dashboard endpoints; anything else is 404.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.serveDashboard(w, r)
	case "/assign":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.serveAssign(w, r)
	case "/api/devices":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.serveJSON(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveDashboard(w http.ResponseWriter, _ *http.Request) {
	devs := s.devices()
	data := pageData{Devices: devs}
	tvs := 0
	for _, d := range devs {
		if d.IsTV {
			tvs++
		}
	}
	switch {
	case tvs == 0:
		data.TVsText = "No TV under protection yet"
	case tvs == 1:
		data.TVsText = "1 TV under protection"
	default:
		data.TVsText = fmt.Sprintf("%d TVs under protection", tvs)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, data); err != nil {
		log.Printf("web: dashboard render: %v", err)
	}
}

// serveAssign applies a one-click brand assignment: POST ip + brand.
func (s *Server) serveAssign(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "unreadable form", http.StatusBadRequest)
		return
	}
	ip := r.PostForm.Get("ip")
	brand := r.PostForm.Get("brand")
	if net.ParseIP(ip) == nil {
		http.Error(w, fmt.Sprintf("invalid IP address %q", ip), http.StatusBadRequest)
		return
	}
	if !isBrand(brand) {
		http.Error(w, fmt.Sprintf("unknown brand %q", brand), http.StatusBadRequest)
		return
	}
	if err := s.router.Assign(net.ParseIP(ip), brand); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.known[ip] = brand
	s.mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) serveJSON(w http.ResponseWriter, _ *http.Request) {
	out := make([]deviceJSON, 0)
	for _, d := range s.devices() {
		out = append(out, deviceJSON{
			IP:          d.IP,
			MAC:         d.MAC,
			Hostname:    d.Hostname,
			Brand:       d.Brand,
			Blocked:     d.Blocked,
			BlockedText: PlainBlocked(d.Blocked),
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("web: devices json: %v", err)
	}
}

// devices merges LAN discovery (ARP cache, reverse DNS) with the TVs
// already known, and states each counter in plain language.
func (s *Server) devices() []device {
	s.mu.Lock()
	known := make(map[string]string, len(s.known))
	for ip, brand := range s.known {
		known[ip] = brand
	}
	s.mu.Unlock()

	byIP := map[string]Host{}
	if hosts, err := s.arp(); err != nil {
		log.Printf("web: LAN discovery unavailable: %v", err)
	} else {
		for _, h := range hosts {
			byIP[h.IP] = h
		}
	}
	for ip := range known {
		if _, ok := byIP[ip]; !ok {
			byIP[ip] = Host{IP: ip}
		}
	}

	devs := make([]device, 0, len(byIP))
	for ip, h := range byIP {
		brand := known[ip]
		d := device{IP: ip, Brand: brand, MAC: h.MAC, IsTV: brand != ""}
		d.Hostname = s.hostname(ip)
		switch {
		case d.Hostname != "":
			d.Name = d.Hostname
		case d.IsTV:
			d.Name = "TV " + displayName(brand)
		default:
			d.Name = "unknown device"
		}
		d.Addr = ip
		if h.MAC != "" {
			d.Addr = ip + "  ·  " + h.MAC
		}
		d.Blocked = s.counter.BlockedToday(ip)
		switch {
		case d.Blocked > 0:
			d.Lamp = "lamp hot"
			d.CountPhrase = countPhrase(d.Blocked)
		case d.IsTV:
			d.Lamp = "lamp on"
			d.BlockedText = plainZero
		default:
			d.Lamp = "lamp"
			d.BlockedText = plainZero
		}
		d.Options = brandOptions(brand)
		devs = append(devs, d)
	}
	sort.Slice(devs, func(i, j int) bool { return lessIP(devs[i].IP, devs[j].IP) })
	return devs
}

// Counter counts blocked tracking attempts per device, in memory, one
// bucket per day. It stays local and forgets previous days immediately.
type Counter struct {
	mu   sync.Mutex
	now  func() time.Time
	days map[string]int
}

// NewCounter starts an empty counter.
func NewCounter() *Counter {
	return &Counter{now: time.Now, days: make(map[string]int)}
}

// RecordBlocked adds one blocked tracking attempt to a device, today.
func (c *Counter) RecordBlocked(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.days[c.keyLocked(ip)]++
}

// BlockedToday states how many attempts were blocked today for one device.
func (c *Counter) BlockedToday(ip string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.days[c.keyLocked(ip)]
}

// keyLocked builds the day|ip key and prunes previous days (mu held).
func (c *Counter) keyLocked(ip string) string {
	today := c.now().Format("2006-01-02")
	prefix := today + "|"
	for k := range c.days {
		if !strings.HasPrefix(k, prefix) {
			delete(c.days, k)
		}
	}
	return prefix + ip
}

// PlainBlocked states in plain language how many tracking attempts were
// blocked today for one device: the exact wording shown on the panel.
func PlainBlocked(n int) string {
	switch {
	case n <= 0:
		return plainZero
	case n == 1:
		return "1 tentative de tracking bloquée aujourd'hui"
	default:
		return fmt.Sprintf("%d tentatives de tracking bloquées aujourd'hui", n)
	}
}

// countPhrase states the rest of the sentence for a non-zero count.
func countPhrase(n int) string {
	if n == 1 {
		return "tentative de tracking bloquée aujourd'hui"
	}
	return "tentatives de tracking bloquées aujourd'hui"
}

// readSystemArp reads the kernel ARP cache (Linux /proc/net/arp).
func readSystemArp() ([]Host, error) {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	return parseArp(data), nil
}

// parseArp keeps the resolved entries of an ARP cache: header, incomplete
// (00:00:00:00:00:00 or broadcast) and malformed lines are dropped.
func parseArp(data []byte) []Host {
	hosts := []Host{}
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		ip, mac := fields[0], fields[3]
		if net.ParseIP(ip) == nil {
			continue
		}
		if mac == incompleteMAC || strings.EqualFold(mac, broadcastMAC) {
			continue
		}
		hosts = append(hosts, Host{IP: ip, MAC: mac})
	}
	return hosts
}

// systemHostname asks the resolver for the reverse-DNS name of a machine on
// the LAN; a miss or a stall (>500 ms) simply yields an empty name.
func systemHostname(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	names, err := net.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// brands lists every brand that has a rule set.
func brands() []string {
	sets := rules.DefaultSets()
	names := make([]string, 0, len(sets))
	for name := range sets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func isBrand(brand string) bool {
	_, ok := rules.DefaultSets()[brand]
	return brand != "" && ok
}

func displayName(brand string) string {
	if label, ok := brandDisplay[brand]; ok {
		return label
	}
	if brand == "" {
		return ""
	}
	return strings.ToUpper(brand[:1]) + brand[1:]
}

func brandOptions(current string) []brandOption {
	opts := make([]brandOption, 0)
	for _, b := range brands() {
		opts = append(opts, brandOption{Value: b, Label: displayName(b), Selected: b == current})
	}
	return opts
}

// lessIP orders IPs numerically (10.0.x before 192.168.x).
func lessIP(a, b string) bool {
	ia, ib := net.ParseIP(a).To16(), net.ParseIP(b).To16()
	if ia == nil || ib == nil {
		return a < b
	}
	return bytes.Compare(ia, ib) < 0
}
