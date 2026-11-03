package observe

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func newStore(t *testing.T, clock *fakeClock, opts ...Option) *Store {
	t.Helper()
	opts = append([]Option{WithNow(clock.Now)}, opts...)
	s, err := New(filepath.Join(t.TempDir(), "observe.json"), opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func tv(n byte) net.IP { return net.IPv4(192, 168, 1, n) }

func TestLearnLogsSeenDomains(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock)
	ip := tv(10)
	for _, d := range []struct {
		domain  string
		verdict Verdict
	}{
		{"samsungads.com", Block},
		{"netflix.com", Allow},
		{"samsungcloud.com", Observe},
	} {
		if err := s.Learn(ip, "samsung", d.domain, d.verdict); err != nil {
			t.Fatalf("Learn(%s): %v", d.domain, err)
		}
	}
	entries := s.Entries(ip)
	want := []Entry{
		{Domain: "netflix.com", Proposed: Allow, FirstSeen: t0, LastSeen: t0, Hits: 1},
		{Domain: "samsungads.com", Proposed: Block, FirstSeen: t0, LastSeen: t0, Hits: 1},
		{Domain: "samsungcloud.com", Proposed: Observe, FirstSeen: t0, LastSeen: t0, Hits: 1},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, w := range want {
		got := entries[i]
		if got.Domain != w.Domain {
			t.Errorf("entry %d: domain %q, want %q", i, got.Domain, w.Domain)
		}
		if got.Proposed != w.Proposed {
			t.Errorf("entry %s: verdict %q, want %q", got.Domain, got.Proposed, w.Proposed)
		}
		if got.Hits != w.Hits {
			t.Errorf("entry %s: %d hits, want %d", got.Domain, got.Hits, w.Hits)
		}
		if !got.FirstSeen.Equal(w.FirstSeen) || !got.LastSeen.Equal(w.LastSeen) {
			t.Errorf("entry %s: seen at %v/%v, want %v/%v", got.Domain, got.FirstSeen, got.LastSeen, w.FirstSeen, w.LastSeen)
		}
	}
}

func TestLearnBumpsCounters(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock)
	ip := tv(11)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	clock.advance(5 * time.Second)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	entries := s.Entries(ip)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Hits != 2 {
		t.Errorf("got %d hits, want 2", e.Hits)
	}
	if !e.FirstSeen.Equal(t0) {
		t.Errorf("first seen %v, want %v", e.FirstSeen, t0)
	}
	if !e.LastSeen.Equal(t0.Add(5 * time.Second)) {
		t.Errorf("last seen %v, want %v", e.LastSeen, t0.Add(5*time.Second))
	}
}

func TestLearnRejectsInvalidInput(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock)
	ip := tv(12)
	cases := []struct {
		name     string
		ip       net.IP
		brand    string
		domain   string
		proposed Verdict
	}{
		{"nil IP", nil, "samsung", "samsungads.com", Block},
		{"empty brand", ip, "", "samsungads.com", Block},
		{"empty domain", ip, "samsung", "", Block},
		{"unknown verdict", ip, "samsung", "samsungads.com", Verdict("deny")},
	}
	for _, tc := range cases {
		if err := s.Learn(tc.ip, tc.brand, tc.domain, tc.proposed); err == nil {
			t.Errorf("%s: Learn must fail", tc.name)
		}
	}
	if entries := s.Entries(ip); len(entries) != 0 {
		t.Errorf("rejected input must not be logged, got %d entries", len(entries))
	}
}

func TestLearningPeriodGatesReady(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock) // default period: roughly one week
	ip, stranger := tv(13), tv(99)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if s.Ready(stranger) {
		t.Error("unknown device must not be ready")
	}
	if s.Ready(ip) {
		t.Error("device must not be ready at the start of the period")
	}
	clock.advance(DefaultPeriod - time.Second)
	if s.Ready(ip) {
		t.Error("device must not be ready one second before the period ends")
	}
	clock.advance(2 * time.Second)
	if !s.Ready(ip) {
		t.Error("device must be ready once the period has elapsed")
	}
}

func TestDecideObservesWhileLearning(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithBaseline(func(brand, domain string) Verdict { return Block }))
	ip, stranger := tv(14), tv(98)
	if got := s.Decide(stranger, "samsungads.com"); got != Observe {
		t.Errorf("unknown device: got %q, want %q", got, Observe)
	}
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	for _, domain := range []string{"samsungads.com", "netflix.com"} {
		if got := s.Decide(ip, domain); got != Observe {
			t.Errorf("during learning %s: got %q, want %q", domain, got, Observe)
		}
	}
}

func TestDecideAppliesLearnedProfileAfterPeriod(t *testing.T) {
	clock := &fakeClock{now: t0}
	var learnedBrand string
	s := newStore(t, clock, WithPeriod(24 * time.Hour), WithBaseline(func(brand, domain string) Verdict {
		learnedBrand = brand
		return Block
	}))
	ip := tv(16)
	learned := []struct {
		domain  string
		verdict Verdict
	}{
		{"samsungacr.com", Block},
		{"netflix.com", Allow},
		{"samsungcloud.com", Observe},
	}
	for _, d := range learned {
		if err := s.Learn(ip, "samsung", d.domain, d.verdict); err != nil {
			t.Fatalf("Learn(%s): %v", d.domain, err)
		}
	}
	clock.advance(48 * time.Hour)
	for _, d := range learned {
		if got := s.Decide(ip, d.domain); got != d.verdict {
			t.Errorf("Decide(%s): got %q, want %q", d.domain, got, d.verdict)
		}
	}
	if got := s.Decide(ip, "telemetry.example.com"); got != Block {
		t.Errorf("unknown domain: got %q, want the baseline %q", got, Block)
	}
	if learnedBrand != "samsung" {
		t.Errorf("baseline consulted for brand %q, want %q", learnedBrand, "samsung")
	}
}

func TestDecideFallsBackToAllowWithoutBaseline(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithPeriod(24 * time.Hour))
	ip := tv(17)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	clock.advance(48 * time.Hour)
	if got := s.Decide(ip, "netflix.com"); got != Allow {
		t.Errorf("unknown domain without baseline: got %q, want %q", got, Allow)
	}
}

func TestSetAppliesProfileAtEndOfPeriod(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithPeriod(24 * time.Hour))
	ip := tv(15)
	if _, ok := s.Set(ip); ok {
		t.Error("profile must not be applied before the device is seen")
	}
	for _, d := range []struct {
		domain  string
		verdict Verdict
	}{
		{"samsungcloud.com", Observe},
		{"samsungads.com", Block},
		{"netflix.com", Allow},
	} {
		if err := s.Learn(ip, "samsung", d.domain, d.verdict); err != nil {
			t.Fatalf("Learn(%s): %v", d.domain, err)
		}
	}
	if _, ok := s.Set(ip); ok {
		t.Error("profile must not be applied while the device is still learning")
	}
	clock.advance(48 * time.Hour)
	set, ok := s.Set(ip)
	if !ok || set == nil {
		t.Fatal("profile must be applied once the period has elapsed")
	}
	want := []Rule{
		{Domain: "netflix.com", Verdict: Allow},
		{Domain: "samsungads.com", Verdict: Block},
		{Domain: "samsungcloud.com", Verdict: Observe},
	}
	if len(set) != len(want) {
		t.Fatalf("got %d rules, want %d", len(set), len(want))
	}
	for i, w := range want {
		if set[i] != w {
			t.Errorf("rule %d: got %+v, want %+v", i, set[i], w)
		}
	}
}

func TestProfileIsFrozenAfterPeriod(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithPeriod(24 * time.Hour))
	ip := tv(19)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if err := s.Learn(ip, "samsung", "samsungcloud.com", Observe); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	clock.advance(48 * time.Hour)
	if err := s.Learn(ip, "samsung", "samsungads.com", Observe); err != nil {
		t.Fatalf("Learn after the period: %v", err)
	}
	entries := s.Entries(ip)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Domain != "samsungads.com" || entries[0].Proposed != Block || entries[0].Hits != 2 {
		t.Errorf("seen domain must keep its proposed verdict and bump its counter, got %+v", entries[0])
	}
	if err := s.Learn(ip, "samsung", "apps.samsung.com", Block); err != nil {
		t.Fatalf("Learn of unseen domain: %v", err)
	}
	if entries := s.Entries(ip); len(entries) != 2 {
		t.Errorf("unseen domain after the period must be ignored, got %d entries", len(entries))
	}
}

func TestZeroPeriodSkipsLearning(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithPeriod(0), WithBaseline(func(brand, domain string) Verdict { return Block }))
	ip := tv(18)
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if !s.Ready(ip) {
		t.Error("device must be ready immediately with a zero period")
	}
	if set, ok := s.Set(ip); !ok || len(set) != 0 {
		t.Errorf("nothing must be learned with a zero period, got %v (ok=%v)", set, ok)
	}
	if got := s.Decide(ip, "samsungads.com"); got != Block {
		t.Errorf("baseline must be enforced, got %q, want %q", got, Block)
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	clock := &fakeClock{now: t0}
	path := filepath.Join(t.TempDir(), "observe.json")
	ip := tv(20)
	first, err := New(path, WithNow(clock.Now), WithPeriod(24 * time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := first.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	reopened, err := New(path, WithNow(clock.Now), WithPeriod(24 * time.Hour))
	if err != nil {
		t.Fatalf("New after restart: %v", err)
	}
	clock.advance(48 * time.Hour)
	if !reopened.Ready(ip) {
		t.Fatal("the learning period must survive a restart")
	}
	entries := reopened.Entries(ip)
	if len(entries) != 1 {
		t.Fatalf("got %d entries after restart, want 1", len(entries))
	}
	e := entries[0]
	if e.Domain != "samsungads.com" || e.Proposed != Block || e.Hits != 1 || !e.FirstSeen.Equal(t0) {
		t.Errorf("recorded domain must survive a restart, got %+v", e)
	}
	if got := reopened.Decide(ip, "netflix.com"); got != Allow {
		t.Errorf("after restart: got %q, want %q", got, Allow)
	}
}

func TestNewRejectsCorruptStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observe.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Error("New must fail on a corrupt state file")
	}
}

func TestLearnTracksBrandReassignment(t *testing.T) {
	clock := &fakeClock{now: t0}
	var learnedBrand string
	s := newStore(t, clock, WithPeriod(24 * time.Hour), WithBaseline(func(brand, domain string) Verdict {
		learnedBrand = brand
		return Allow
	}))
	ip := tv(21)
	if err := s.Learn(ip, "samsung", "samsungcloud.com", Observe); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if err := s.Learn(ip, "lg", "samsungcloud.com", Observe); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	clock.advance(48 * time.Hour)
	if got := s.Decide(ip, "netflix.com"); got != Allow {
		t.Errorf("baseline verdict: got %q, want %q", got, Allow)
	}
	if learnedBrand != "lg" {
		t.Errorf("baseline consulted with brand %q, want %q (brand was reassigned)", learnedBrand, "lg")
	}
}

func TestLearnHandlesConcurrentDevices(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock)
	const devices, hits = 8, 5
	var wg sync.WaitGroup
	for n := 0; n < devices; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := tv(byte(20 + n))
			domain := fmt.Sprintf("tv%d.example.com", n)
			for i := 0; i < hits; i++ {
				if err := s.Learn(ip, "samsung", domain, Block); err != nil {
					t.Errorf("Learn(%s): %v", domain, err)
					return
				}
			}
		}(n)
	}
	wg.Wait()
	for n := 0; n < devices; n++ {
		entries := s.Entries(tv(byte(20 + n)))
		if len(entries) != 1 || entries[0].Hits != hits {
			t.Errorf("tv%d: got %+v, want 1 entry with %d hits", n, entries, hits)
		}
	}
}

func TestRegisterStartsThePeriodAtAssignment(t *testing.T) {
	clock := &fakeClock{now: t0}
	path := filepath.Join(t.TempDir(), "observe.json")
	s, err := New(path, WithNow(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ip := tv(30)
	if err := s.Register(ip, "lg"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !s.Known(ip) {
		t.Error("registered device must be known")
	}
	if s.Ready(ip) {
		t.Error("device must not be ready right after assignment")
	}
	clock.advance(DefaultPeriod)
	if !s.Ready(ip) {
		t.Error("the period must start at assignment, not at the first query")
	}
}

func TestRegisterKeepsThePeriodOnReassignment(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock, WithPeriod(24 * time.Hour))
	var learnedBrand string
	s.SetBaseline(func(brand, domain string) Verdict {
		learnedBrand = brand
		return Block
	})
	ip := tv(31)
	if err := s.Register(ip, "samsung"); err != nil {
		t.Fatalf("Register samsung: %v", err)
	}
	if err := s.Learn(ip, "samsung", "samsungads.com", Block); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	clock.advance(2 * time.Hour)
	if err := s.Register(ip, "lg"); err != nil {
		t.Fatalf("Register lg: %v", err)
	}
	clock.advance(23 * time.Hour)
	if !s.Ready(ip) {
		t.Error("reassignment must not restart the learning period")
	}
	entries := s.Entries(ip)
	if len(entries) != 1 || entries[0].Domain != "samsungads.com" || entries[0].Proposed != Block {
		t.Errorf("the domain log must survive reassignment, got %+v", entries)
	}
	if got := s.Decide(ip, "doubleclick.net"); got != Block {
		t.Errorf("baseline after reassignment: got %q, want %q", got, Block)
	}
	if learnedBrand != "lg" {
		t.Errorf("baseline consulted with brand %q, want %q", learnedBrand, "lg")
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	clock := &fakeClock{now: t0}
	s := newStore(t, clock)
	if err := s.Register(nil, "lg"); err == nil {
		t.Error("Register must reject a nil IP")
	}
	if err := s.Register(tv(32), ""); err == nil {
		t.Error("Register must reject an empty brand")
	}
}

func TestRegisterSurvivesRestart(t *testing.T) {
	clock := &fakeClock{now: t0}
	path := filepath.Join(t.TempDir(), "observe.json")
	first, err := New(path, WithNow(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ip := tv(33)
	if err := first.Register(ip, "lg"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	reopened, err := New(path, WithNow(clock.Now))
	if err != nil {
		t.Fatalf("New after restart: %v", err)
	}
	if !reopened.Known(ip) {
		t.Fatal("the assignment must survive a restart")
	}
	clock.advance(DefaultPeriod)
	if !reopened.Ready(ip) {
		t.Error("the learning period must not restart across a restart")
	}
}
