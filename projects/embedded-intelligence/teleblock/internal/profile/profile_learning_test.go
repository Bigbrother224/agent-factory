package profile

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"teleblock/internal/observe"
	"teleblock/internal/rules"
)

var t0 = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func tv(n byte) net.IP { return net.IPv4(192, 168, 1, n) }

func learningRouter(t *testing.T, c *fakeClock, period time.Duration) (*Router, *observe.Store) {
	t.Helper()
	store, err := observe.New(filepath.Join(t.TempDir(), "observe.json"),
		observe.WithNow(c.Now), observe.WithPeriod(period))
	if err != nil {
		t.Fatalf("observe.New: %v", err)
	}
	r := New(rules.DefaultSets())
	r.EnableLearning(store)
	return r, store
}

// An assigned device starts in observation: every domain gets Observe
// while the verdicts proposed by the brand rules pile up in the log.
func TestLearningDeviceObservesWhileLearning(t *testing.T) {
	c := &fakeClock{now: t0}
	r, store := learningRouter(t, c, 24*time.Hour)
	ip := tv(10)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if got := r.State(ip); got != DeviceLearning {
		t.Errorf("state after assignment: got %v, want DeviceLearning", got)
	}
	if got := r.Verdict(ip, "samsungacr.com"); got != rules.Observe {
		t.Errorf("telemetry domain during learning: got %q, want %q", got, rules.Observe)
	}
	if got := r.Verdict(ip, "netflix.com"); got != rules.Observe {
		t.Errorf("streaming domain during learning: got %q, want %q", got, rules.Observe)
	}
	entries := store.Entries(ip)
	if len(entries) != 2 {
		t.Fatalf("got %d recorded domains, want 2", len(entries))
	}
	if entries[0].Domain != "netflix.com" || entries[0].Proposed != observe.Allow ||
		entries[1].Domain != "samsungacr.com" || entries[1].Proposed != observe.Block {
		t.Errorf("proposed verdicts must be recorded: got %+v", entries)
	}
}

// At the end of the period the learned profile is enforced verbatim, and
// unseen domains fall back to the brand rule set.
func TestLearnedProfileEnforcedAfterPeriod(t *testing.T) {
	c := &fakeClock{now: t0}
	r, _ := learningRouter(t, c, 24*time.Hour)
	ip := tv(11)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	for _, domain := range []string{"samsungacr.com", "netflix.com", "samsungcloud.com"} {
		r.Verdict(ip, domain)
	}
	c.advance(25 * time.Hour)
	if got := r.State(ip); got != DeviceReady {
		t.Errorf("state after the period: got %v, want DeviceReady", got)
	}
	for _, tc := range []struct {
		domain string
		want   rules.Verdict
	}{
		{"samsungacr.com", rules.Block},       // learned
		{"netflix.com", rules.Allow},          // learned
		{"samsungcloud.com", rules.Observe},   // learned
		{"smetrics.samsung.com", rules.Block}, // unseen: brand rules apply
		{"apps.samsung.com", rules.Allow},     // unseen: brand rules apply
		{"doubleclick.net", rules.Allow},      // unseen: samsung set says nothing
	} {
		if got := r.Verdict(ip, tc.domain); got != tc.want {
			t.Errorf("Verdict(%s): got %q, want %q", tc.domain, got, tc.want)
		}
	}
}

// The period boundary is inclusive: exactly one period after the start,
// the device switches from observation to the learned profile.
func TestPeriodBoundaryIsInclusive(t *testing.T) {
	c := &fakeClock{now: t0}
	r, _ := learningRouter(t, c, 24*time.Hour)
	ip := tv(12)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if got := r.Verdict(ip, "samsungacr.com"); got != rules.Observe {
		t.Fatalf("before the period: got %q, want %q", got, rules.Observe)
	}
	c.advance(24*time.Hour - time.Nanosecond)
	if got := r.State(ip); got != DeviceLearning {
		t.Errorf("one nanosecond before the deadline: got %v, want DeviceLearning", got)
	}
	c.advance(time.Nanosecond)
	if got := r.State(ip); got != DeviceReady {
		t.Errorf("exactly at the deadline: got %v, want DeviceReady", got)
	}
	if got := r.Verdict(ip, "samsungacr.com"); got != rules.Block {
		t.Errorf("at the deadline: got %q, want %q", got, rules.Block)
	}
}

// A device that never entered learning is governed directly by its brand
// rules — unknown IPs get the generic profile, with no observation delay.
func TestUnknownDevicesAreEnforcedImmediately(t *testing.T) {
	c := &fakeClock{now: t0}
	r, store := learningRouter(t, c, 24*time.Hour)
	stranger := tv(13)
	if got := r.State(stranger); got != DeviceUnknown {
		t.Errorf("state: got %v, want DeviceUnknown", got)
	}
	for _, domain := range []string{"doubleclick.net", "graph.facebook.com"} {
		if got := r.Verdict(stranger, domain); got != rules.Block {
			t.Errorf("generic blocklist (%s): got %q, want %q", domain, got, rules.Block)
		}
	}
	if entries := store.Entries(stranger); len(entries) != 0 {
		t.Errorf("unassigned devices must not be learned, got %+v", entries)
	}
}

// Without a learning store the Router keeps its pre-learning contract:
// brand rules apply immediately, no device enters observation.
func TestWithoutLearningStoreBrandRulesApplyImmediately(t *testing.T) {
	r := New(rules.DefaultSets())
	ip := tv(14)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if got := r.State(ip); got != DeviceUnknown {
		t.Errorf("state without learning: got %v, want DeviceUnknown", got)
	}
	if got := r.Verdict(ip, "samsungads.com"); got != rules.Block {
		t.Errorf("enforcement without learning: got %q, want %q", got, rules.Block)
	}
	if got := r.Verdict(ip, "netflix.com"); got != rules.Allow {
		t.Errorf("enforcement without learning: got %q, want %q", got, rules.Allow)
	}
}

// The learning state is local and durable: a restarted teleblock resumes
// devices mid-period and enforces learned profiles due meanwhile.
func TestLearningStateSurvivesRestart(t *testing.T) {
	c := &fakeClock{now: t0}
	path := filepath.Join(t.TempDir(), "observe.json")
	store, err := observe.New(path, observe.WithNow(c.Now), observe.WithPeriod(24*time.Hour))
	if err != nil {
		t.Fatalf("observe.New: %v", err)
	}
	r := New(rules.DefaultSets())
	r.EnableLearning(store)
	ip := tv(15)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	for _, domain := range []string{"samsungacr.com", "netflix.com"} {
		r.Verdict(ip, domain)
	}

	// Restart at the same instant: the device is mid-period.
	reloaded, err := observe.New(path, observe.WithNow(c.Now), observe.WithPeriod(24*time.Hour))
	if err != nil {
		t.Fatalf("observe.New after restart: %v", err)
	}
	r2 := New(rules.DefaultSets())
	r2.EnableLearning(reloaded)
	if got := r2.State(ip); got != DeviceLearning {
		t.Errorf("state after restart: got %v, want DeviceLearning", got)
	}
	if got := r2.Verdict(ip, "samsungacr.com"); got != rules.Observe {
		t.Errorf("verdict after restart: got %q, want %q", got, rules.Observe)
	}

	// Restart after the period: the learned profile is enforced.
	c.advance(48 * time.Hour)
	reborn, err := observe.New(path, observe.WithNow(c.Now), observe.WithPeriod(24*time.Hour))
	if err != nil {
		t.Fatalf("observe.New after the period: %v", err)
	}
	r3 := New(rules.DefaultSets())
	r3.EnableLearning(reborn)
	if got := r3.Verdict(ip, "samsungacr.com"); got != rules.Block {
		t.Errorf("learned verdict after restart: got %q, want %q", got, rules.Block)
	}
	if got := r3.Verdict(ip, "netflix.com"); got != rules.Allow {
		t.Errorf("learned verdict after restart: got %q, want %q", got, rules.Allow)
	}
}

// Reassigning a brand keeps the domain log: the verdicts recorded before
// the change remain, and unseen domains fall back to the new brand rules.
func TestReassignmentKeepsTheLogAndSwitchesBaseline(t *testing.T) {
	c := &fakeClock{now: t0}
	r, store := learningRouter(t, c, 24*time.Hour)
	ip := tv(16)
	if err := r.Assign(ip, "samsung"); err != nil {
		t.Fatalf("Assign samsung: %v", err)
	}
	if got := r.Verdict(ip, "samsungacr.com"); got != rules.Observe {
		t.Fatalf("during learning: got %q, want %q", got, rules.Observe)
	}
	if err := r.Assign(ip, "lg"); err != nil {
		t.Fatalf("Assign lg: %v", err)
	}
	c.advance(48 * time.Hour)
	if got := r.Verdict(ip, "samsungacr.com"); got != rules.Block {
		t.Errorf("pre-assignment verdict must survive: got %q, want %q", got, rules.Block)
	}
	if got := r.Verdict(ip, "adinfo.lge.com"); got != rules.Block {
		t.Errorf("unseen domain must follow the new brand: got %q, want %q", got, rules.Block)
	}
	for _, e := range store.Entries(ip) {
		if e.Domain == "adinfo.lge.com" {
			t.Errorf("baseline domains must not join the learned log: %+v", e)
		}
	}
}
