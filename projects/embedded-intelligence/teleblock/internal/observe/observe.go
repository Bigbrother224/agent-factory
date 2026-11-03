// Package observe implements per-device learning mode: each TV on the
// network starts in observation for a learning period (~ one week). During
// that period the domains the device contacts and the verdict proposed for
// each of them are recorded; when the period elapses, the recorded verdicts
// become the profile applied automatically for that device. The learning
// state is persisted locally as JSON and never leaves the machine.
package observe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Verdict is what a learned profile proposes for a domain.
type Verdict string

const (
	// Allow keeps traffic for a domain.
	Allow Verdict = "allow"
	// Block resets traffic for a domain.
	Block Verdict = "block"
	// Observe watches traffic for a domain without acting on it.
	Observe Verdict = "observe"
)

// DefaultPeriod is the default learning period: roughly one week.
const DefaultPeriod = 7 * 24 * time.Hour

// Entry is one domain recorded while a device was in observation.
type Entry struct {
	Domain    string    `json:"domain"`
	Proposed  Verdict   `json:"proposed"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Hits      int       `json:"hits"`
}

// Device is the learning state of one TV, keyed by client IP.
type Device struct {
	IP        string           `json:"ip"`
	Brand     string           `json:"brand"`
	StartedAt time.Time        `json:"started_at"`
	Entries   map[string]Entry `json:"domains"`
}

// Rule is one line of the profile applied automatically at the end of the
// learning period of a device.
type Rule struct {
	Domain  string
	Verdict Verdict
}

// Set is the learned profile of a device, ordered by domain name.
type Set []Rule

// Baseline proposes the verdict for a domain when the learned profile says
// nothing about it. An empty verdict falls back to Allow.
type Baseline func(brand, domain string) Verdict

// Store holds the learning state of every device and persists it as one
// JSON file on the local filesystem.
type Store struct {
	mu       sync.Mutex
	path     string
	period   time.Duration
	now      func() time.Time
	baseline Baseline
	devices  map[string]*Device
}

// Option configures a Store at construction time.
type Option func(*Store)

// WithPeriod overrides the learning period. A period of zero or less skips
// learning altogether: devices are considered ready immediately and only
// the baseline applies to them.
func WithPeriod(period time.Duration) Option {
	return func(s *Store) { s.period = period }
}

// WithNow overrides the clock used to measure the learning period, for
// tests and dry-run checks.
func WithNow(now func() time.Time) Option {
	return func(s *Store) { s.now = now }
}

// WithBaseline sets the default profile consulted once the learning period
// is over, for the domains the device never got to learn.
func WithBaseline(baseline Baseline) Option {
	return func(s *Store) { s.baseline = baseline }
}

// New opens the learning state persisted at path, or starts an empty one
// when no file is there yet. A file that cannot be parsed is an error: the
// learning state is local, irreplaceable data.
func New(path string, opts ...Option) (*Store, error) {
	if path == "" {
		return nil, errors.New("observe: empty state path")
	}
	s := &Store{
		path:    path,
		period:  DefaultPeriod,
		now:     time.Now,
		devices: map[string]*Device{},
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.now == nil {
		s.now = time.Now
	}
	buf, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("observe: read %s: %w", path, err)
	}
	var devices map[string]*Device
	if err := json.Unmarshal(buf, &devices); err != nil {
		return nil, fmt.Errorf("observe: parse %s: %w", path, err)
	}
	if devices == nil {
		devices = map[string]*Device{}
	}
	for _, d := range devices {
		// A hand-edited state must not turn a later Learn into a
		// nil-map panic.
		if d.Entries == nil {
			d.Entries = map[string]Entry{}
		}
	}
	s.devices = devices
	return s, nil
}

// SetBaseline installs or replaces the baseline profile consulted after
// the learning period, for the domains the device never learned.
func (s *Store) SetBaseline(baseline Baseline) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.baseline = baseline
}

// Learn records the domain seen on the device together with the verdict
// proposed for it. While the device is in observation every new domain
// joins the log; once the period has elapsed the profile is frozen — seen
// domains only bump their in-memory counters and unseen domains are
// ignored. Only structural changes (a new device, a brand change, a new
// domain) trigger a disk write: counter updates ride along with the next
// one, so a chatty device cannot wear out the local storage.
func (s *Store) Learn(ip net.IP, brand, domain string, proposed Verdict) error {
	if ip == nil {
		return errors.New("observe: nil device IP")
	}
	if brand == "" {
		return fmt.Errorf("observe: device %s: empty brand", ip)
	}
	if domain == "" {
		return fmt.Errorf("observe: device %s: empty domain", ip)
	}
	switch proposed {
	case Allow, Block, Observe:
	default:
		return fmt.Errorf("observe: device %s: invalid proposed verdict %q", ip, proposed)
	}
	s.mu.Lock()
	key := ip.String()
	now := s.now()
	d, ok := s.devices[key]
	dirty := false
	if !ok {
		d = &Device{IP: key, Brand: brand, StartedAt: now, Entries: map[string]Entry{}}
		s.devices[key] = d
		dirty = true
	} else if d.Brand != brand {
		d.Brand = brand
		dirty = true
	}
	if e, seen := d.Entries[domain]; seen {
		e.Hits++
		e.LastSeen = now
		d.Entries[domain] = e
	} else if !d.ready(s.period, now) {
		d.Entries[domain] = Entry{
			Domain:    domain,
			Proposed:  proposed,
			FirstSeen: now,
			LastSeen:  now,
			Hits:      1,
		}
		dirty = true
	}
	if !dirty {
		s.mu.Unlock()
		return nil
	}
	// Sérialisation sous le verrou (rapide), écriture disque HORS verrou : garder
	// s.mu pendant l'I/O sérialise tous les apprenants (revue).
	state, encErr := json.MarshalIndent(s.devices, "", "  ")
	s.mu.Unlock()
	if encErr != nil {
		return fmt.Errorf("observe: encode state: %w", encErr)
	}
	return s.writeState(state)
}

// Register declares a device and starts its learning period: a newly seen
// device begins its period now; a known one keeps its earlier start and
// only gets its brand refreshed. The state is persisted.
func (s *Store) Register(ip net.IP, brand string) error {
	if ip == nil {
		return errors.New("observe: nil device IP")
	}
	if brand == "" {
		return fmt.Errorf("observe: device %s: empty brand", ip)
	}
	s.mu.Lock()
	key := ip.String()
	d, ok := s.devices[key]
	if !ok {
		d = &Device{IP: key, Brand: brand, StartedAt: s.now(), Entries: map[string]Entry{}}
		s.devices[key] = d
	} else if d.Brand == brand {
		s.mu.Unlock()
		return nil
	} else {
		d.Brand = brand
	}
	// Sérialisation sous le verrou (rapide), écriture disque HORS verrou : garder
	// s.mu pendant l'I/O sérialise tous les apprenants (revue).
	state, encErr := json.MarshalIndent(s.devices, "", "  ")
	s.mu.Unlock()
	if encErr != nil {
		return fmt.Errorf("observe: encode state: %w", encErr)
	}
	return s.writeState(state)
}

// Known reports whether the device has ever been declared to the store.
func (s *Store) Known(ip net.IP) bool {
	if ip == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.devices[ip.String()]
	return ok
}

// ready reports whether the learning period of the device is over.
func (d *Device) ready(period time.Duration, now time.Time) bool {
	return period <= 0 || now.Sub(d.StartedAt) >= period
}

// Ready reports whether the learning period of the device has elapsed, i.e.
// whether its learned profile can be applied. Unknown devices are never
// ready.
func (s *Store) Ready(ip net.IP) bool {
	if ip == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[ip.String()]
	if !ok {
		return false
	}
	return d.ready(s.period, s.now())
}

// Entries returns the log of the domains recorded for the device, ordered
// by domain name. Unknown devices have no entries.
func (s *Store) Entries(ip net.IP) []Entry {
	if ip == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[ip.String()]
	if !ok {
		return nil
	}
	entries := make([]Entry, 0, len(d.Entries))
	for _, e := range d.Entries {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Domain < entries[j].Domain
	})
	return entries
}

// Set returns the profile learned by the device, applied automatically once
// its learning period has elapsed. While the device is still in observation
// it reports false. The set can be empty when the period elapsed before the
// device was ever seen.
func (s *Store) Set(ip net.IP) (Set, bool) {
	if ip == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[ip.String()]
	if !ok || !d.ready(s.period, s.now()) {
		return nil, false
	}
	set := make(Set, 0, len(d.Entries))
	for _, e := range d.Entries {
		set = append(set, Rule{Domain: e.Domain, Verdict: e.Proposed})
	}
	sort.Slice(set, func(i, j int) bool {
		return set[i].Domain < set[j].Domain
	})
	return set, true
}

// Decide returns the verdict enforced now on the device for the domain.
// Unknown devices and devices still in observation get Observe; after the
// learning period the learned verdict applies, falling back to the baseline
// for domains the profile does not know, and to Allow when no baseline is
// set. Decide never mutates the learning state.
func (s *Store) Decide(ip net.IP, domain string) Verdict {
	// Guard nil comme les autres méthodes publiques : un IP nil est un appel
	// buggé, pas un crash du résolveur (revue : panic possible).
	if ip == nil {
		return Observe
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[ip.String()]
	if !ok || !d.ready(s.period, s.now()) {
		return Observe
	}
	if e, seen := d.Entries[domain]; seen {
		return e.Proposed
	}
	if s.baseline != nil {
		if v := s.baseline(d.Brand, domain); v != "" {
			return v
		}
	}
	return Allow
}

// writeState écrit un instantané déjà sérialisé hors s.mu, de façon atomique :
// il est écrit à côté de sa cible puis renommé en place, pour qu'un crash ne
// laisse jamais un état à moitié écrit derrière lui. Toujours appelé HORS
// verrou : la sérialisation se fait sous le verrou de l'appelant, l'I/O ici.
func (s *Store) writeState(buf []byte) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("observe: create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(s.path)+".*")
	if err != nil {
		return fmt.Errorf("observe: create temp state: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name) // no-op after a successful rename
	if _, err := tmp.Write(buf); err != nil {
		tmp.Close()
		return fmt.Errorf("observe: write state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("observe: close state: %w", err)
	}
	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("observe: publish state: %w", err)
	}
	return nil
}
