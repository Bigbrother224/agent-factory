package profile

import (
	"net"
	"testing"

	"teleblock/internal/rules"
)

func mustAssign(t *testing.T, r *Router, ip, brand string) {
	t.Helper()
	if err := r.Assign(net.ParseIP(ip), brand); err != nil {
		t.Fatalf("Assign(%s, %s): %v", ip, brand, err)
	}
}

func TestAssignAndBrand(t *testing.T) {
	r := New(rules.DefaultSets())
	mustAssign(t, r, "192.168.1.42", rules.BrandSamsung)
	if got := r.Brand(net.ParseIP("192.168.1.42")); got != rules.BrandSamsung {
		t.Errorf("Brand = %q, want %q", got, rules.BrandSamsung)
	}
}

func TestUnknownDeviceGetsGenericProfile(t *testing.T) {
	r := New(rules.DefaultSets())
	mustAssign(t, r, "192.168.1.42", rules.BrandLG)
	if got := r.Brand(net.ParseIP("192.168.1.99")); got != rules.BrandGeneric {
		t.Errorf("Brand(unknown) = %q, want %q", got, rules.BrandGeneric)
	}
}

func TestAssignRejectsUnknownBrand(t *testing.T) {
	r := New(rules.DefaultSets())
	if err := r.Assign(net.ParseIP("192.168.1.42"), "philips"); err == nil {
		t.Error("Assign(unknown brand): want error, got nil")
	}
}

func TestAssignRejectsNilIP(t *testing.T) {
	r := New(rules.DefaultSets())
	if err := r.Assign(nil, rules.BrandSamsung); err == nil {
		t.Error("Assign(nil IP): want error, got nil")
	}
}

func TestVerdictRoutesByProfile(t *testing.T) {
	r := New(rules.DefaultSets())
	mustAssign(t, r, "192.168.1.10", rules.BrandSamsung)
	mustAssign(t, r, "192.168.1.11", rules.BrandLG)

	tests := []struct {
		ip   string
		name string
		want rules.Verdict
	}{
		// samsungacr.com is telemetry for Samsung TVs only.
		{"192.168.1.10", "samsungacr.com", rules.Block},
		{"192.168.1.11", "samsungacr.com", rules.Allow},
		// Unknown devices fall back to the generic profile.
		{"192.168.1.12", "doubleclick.net", rules.Block},
		{"192.168.1.10", "doubleclick.net", rules.Allow},
	}
	for _, tt := range tests {
		got := r.Verdict(net.ParseIP(tt.ip), tt.name)
		if got != tt.want {
			t.Errorf("Verdict(%s, %q) = %q, want %q", tt.ip, tt.name, got, tt.want)
		}
	}
}
