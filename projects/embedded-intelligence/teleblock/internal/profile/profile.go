// Package profile routes devices to the rule set of their TV brand.
package profile

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"teleblock/internal/observe"
	"teleblock/internal/rules"
)

// DeviceState is where a device stands in the learning mode lifecycle.
type DeviceState int

const (
	// DeviceUnknown: the device is not tracked by the learning store —
	// never assigned, or learning disabled. Its brand profile is
	// enforced immediately.
	DeviceUnknown DeviceState = iota
	// DeviceLearning: observation period running; verdicts are recorded,
	// not enforced.
	DeviceLearning
	// DeviceReady: the period has elapsed; the learned profile is
	// enforced, with the brand rules as fallback for unseen domains.
	DeviceReady
)

// Router maps client IPs to brand profiles; unknown IPs get the generic
// profile. With a learning store attached, every assigned device starts
// in observation, and its learned profile is enforced automatically once
// its learning period has elapsed. It is safe for concurrent use.
type Router struct {
	mu      sync.RWMutex
	devices map[string]string    // client IP -> brand
	sets    map[string]rules.Set // brand -> rules
	learn   *observe.Store       // nil: learning mode disabled
}

// New returns a Router serving the given brand rule sets.
func New(sets map[string]rules.Set) *Router {
	return &Router{
		devices: make(map[string]string),
		sets:    sets,
	}
}

// EnableLearning attaches the per-device learning store: assigned devices
// start (or resume, their state being reloaded from the store) in
// observation, and their learned profile is enforced once their learning
// period has elapsed. Domains unseen during the period fall back to the
// brand rule set. Passing nil restores the immediate enforcement of the
// brand rules.
func (r *Router) EnableLearning(s *observe.Store) {
	if s != nil {
		// The baseline resolves rule sets at call time; the sets map is
		// immutable after New, so reading it takes no lock.
		s.SetBaseline(func(brand, domain string) observe.Verdict {
			return observe.Verdict(r.set(brand).Verdict(domain))
		})
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.learn = s
}

// Assign binds a client IP to a brand profile. With learning enabled the
// device also starts (or resumes) its observation period; a failure to
// persist the learning state fails the assignment.
func (r *Router) Assign(ip net.IP, brand string) error {
	if ip == nil {
		return errors.New("invalid device IP")
	}
	if _, ok := r.sets[brand]; !ok {
		return fmt.Errorf("unknown brand %q", brand)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Persister AVANT de muter la map : si Register échoue, l'assignation
	// échoue entière (contrat « l'échec de persistance fait échouer
	// l'assignation ») — jamais un appareil assigné en mémoire sans état
	// persisté (revue : atomicité).
	if r.learn != nil {
		if err := r.learn.Register(ip, brand); err != nil {
			return fmt.Errorf("device %s: %w", ip, err)
		}
	}
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

// State returns where the device stands in the learning mode lifecycle.
func (r *Router) State(ip net.IP) DeviceState {
	r.mu.RLock()
	learn := r.learn
	r.mu.RUnlock()
	if learn == nil || ip == nil || !learn.Known(ip) {
		return DeviceUnknown
	}
	if learn.Ready(ip) {
		return DeviceReady
	}
	return DeviceLearning
}

// Verdict returns the verdict for domain as seen from ip. A device in
// observation gets Observe — nothing is blocked while it learns — while
// the verdicts proposed by its brand rule set are recorded. Once its
// learning period has elapsed the learned profile is enforced, with the
// brand rules as fallback for unseen domains. Devices that never entered
// learning — unknown to the store, or with learning disabled — are
// governed directly by their brand rules.
func (r *Router) Verdict(ip net.IP, domain string) rules.Verdict {
	r.mu.RLock()
	learn := r.learn
	r.mu.RUnlock()
	if learn == nil {
		return r.Set(ip).Verdict(domain)
	}
	switch r.State(ip) {
	case DeviceUnknown:
		return r.Set(ip).Verdict(domain)
	case DeviceLearning:
		r.record(learn, ip, domain)
		return rules.Observe
	default: // DeviceReady
		r.record(learn, ip, domain)
		return rules.Verdict(learn.Decide(ip, domain))
	}
}

// record feeds the observed domain to the learning store, with the verdict
// its brand rule set proposes. A persistence failure does not change the
// verdict: it is logged and left to the next query to fix.
func (r *Router) record(learn *observe.Store, ip net.IP, domain string) {
	brand := r.Brand(ip)
	if err := learn.Learn(ip, brand, domain, observe.Verdict(r.set(brand).Verdict(domain))); err != nil {
		log.Printf("profile: device %s: record %s: %v", ip, domain, err)
	}
}

// set returns the rule set of a brand, falling back to the generic one.
func (r *Router) set(brand string) rules.Set {
	if s, ok := r.sets[brand]; ok {
		return s
	}
	return r.sets[rules.BrandGeneric]
}

// brand is the lock-free lookup behind the exported methods.
func (r *Router) brand(ip net.IP) string {
	if brand, ok := r.devices[ip.String()]; ok {
		return brand
	}
	return rules.BrandGeneric
}
