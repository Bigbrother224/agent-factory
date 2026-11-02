// Package profile routes devices to the rule set of their TV brand.
package profile

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"teleblock/internal/rules"
)

// Router maps client IPs to brand profiles; unknown IPs get the generic
// profile. It is safe for concurrent use.
type Router struct {
	mu      sync.RWMutex
	devices map[string]string    // client IP -> brand
	sets    map[string]rules.Set // brand -> rules
}

// New returns a Router serving the given brand rule sets.
func New(sets map[string]rules.Set) *Router {
	return &Router{
		devices: make(map[string]string),
		sets:    sets,
	}
}

// Assign binds a client IP to a brand profile.
func (r *Router) Assign(ip net.IP, brand string) error {
	if ip == nil {
		return errors.New("invalid device IP")
	}
	if _, ok := r.sets[brand]; !ok {
		return fmt.Errorf("unknown brand %q", brand)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[ip.String()] = brand
	return nil
}

// Brand returns the brand profile assigned to ip, defaulting to the
// generic profile for unknown devices.
func (r *Router) Brand(ip net.IP) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.brand(ip)
}

// Set returns the rule set that applies to ip.
func (r *Router) Set(ip net.IP) rules.Set {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sets[r.brand(ip)]
}

// Verdict returns the verdict for domain as seen from ip.
func (r *Router) Verdict(ip net.IP, domain string) rules.Verdict {
	return r.Set(ip).Verdict(domain)
}

// brand is the lock-free lookup behind the exported methods.
func (r *Router) brand(ip net.IP) string {
	if brand, ok := r.devices[ip.String()]; ok {
		return brand
	}
	return rules.BrandGeneric
}
