// Package rules defines the per-brand filtering rules of TeleBlock.
//
// Every TV brand has a rule set: an ordered list of rules mapping a
// domain suffix to a verdict. The first matching rule wins; names that
// match no rule are allowed — TeleBlock is a blocklist firewall, not a
// default-deny one.
package rules

import "strings"

// Verdict is the decision taken for a domain.
type Verdict string

const (
	// Allow: the domain is needed for the TV to work (streaming, updates).
	Allow Verdict = "allow"
	// Block: known telemetry or ad domain; sinkholed by the resolver.
	Block Verdict = "block"
	// Observe: mixed-use domain; allowed but logged for review.
	Observe Verdict = "observe"
)

// Brand profile names.
const (
	BrandSamsung = "samsung"
	BrandLG      = "lg"
	BrandSony    = "sony"
	BrandTCL     = "tcl"
	BrandGeneric = "generic"
)

// Rule applies a verdict to a domain and all of its subdomains.
type Rule struct {
	// Domain is matched on dot boundaries: "samsungacr.com" matches
	// "samsungacr.com" and "us.samsungacr.com", not "notsamsungacr.com".
	Domain string
	// Verdict is applied when Domain matches.
	Verdict Verdict
	// Comment documents why the rule exists.
	Comment string
}

// Set is an ordered rule list; the first match wins.
type Set []Rule

// DefaultSets returns the built-in rule sets keyed by brand. Domains
// are drawn from public smart-TV blocklists and vendor documentation;
// treat them as a documented starting point, to be tuned per
// deployment.
func DefaultSets() map[string]Set {
	return map[string]Set{
		BrandSamsung: {
			{Domain: "samsungacr.com", Verdict: Block, Comment: "ACR telemetry: fingerprints everything the TV plays."},
			{Domain: "samsungads.com", Verdict: Block, Comment: "Samsung Ads: ad delivery and tracking."},
			{Domain: "smetrics.samsung.com", Verdict: Block, Comment: "Adobe Analytics metrics endpoint."},
			{Domain: "samsungcloud.com", Verdict: Observe, Comment: "Cloud services: needed for remote features, carries usage data."},
			{Domain: "apps.samsung.com", Verdict: Allow, Comment: "App store."},
			{Domain: "netflix.com", Verdict: Allow, Comment: "Streaming."},
			{Domain: "youtube.com", Verdict: Allow, Comment: "Streaming."},
		},
		BrandLG: {
			{Domain: "lgtvcommon.com", Verdict: Block, Comment: "webOS common services: background viewing telemetry."},
			{Domain: "adinfo.lge.com", Verdict: Block, Comment: "Advertising information endpoint."},
			{Domain: "lgtvsdp.com", Verdict: Observe, Comment: "Smart TV platform services: app framework mixed with analytics."},
			{Domain: "snu.lge.com", Verdict: Allow, Comment: "Software updates."},
			{Domain: "lgappstv.com", Verdict: Allow, Comment: "App store content."},
			{Domain: "netflix.com", Verdict: Allow, Comment: "Streaming."},
			{Domain: "youtube.com", Verdict: Allow, Comment: "Streaming."},
		},
		BrandSony: {
			{Domain: "doubleclick.net", Verdict: Block, Comment: "Google ads (Bravia runs Google TV)."},
			{Domain: "googleadservices.com", Verdict: Block, Comment: "Google ad conversion tracking."},
			{Domain: "app-measurement.com", Verdict: Block, Comment: "Firebase measurement telemetry."},
			{Domain: "sonyentertainmentnetwork.com", Verdict: Observe, Comment: "SEN backend: auth and streaming mixed with usage data."},
			{Domain: "sony.net", Verdict: Observe, Comment: "Bravia services, including telemetry endpoints."},
			{Domain: "netflix.com", Verdict: Allow, Comment: "Streaming."},
			{Domain: "youtube.com", Verdict: Allow, Comment: "Streaming."},
		},
		BrandTCL: {
			{Domain: "tracking.roku.com", Verdict: Block, Comment: "Roku viewing tracking (TCL Roku TVs)."},
			{Domain: "logs.roku.com", Verdict: Block, Comment: "Roku log collection."},
			{Domain: "advertising.roku.com", Verdict: Block, Comment: "Roku ad platform."},
			{Domain: "roku.com", Verdict: Observe, Comment: "Roku backend: needed for the channel store, mixed with telemetry."},
			{Domain: "tcl.com", Verdict: Observe, Comment: "Vendor services and firmware information."},
			{Domain: "netflix.com", Verdict: Allow, Comment: "Streaming."},
			{Domain: "youtube.com", Verdict: Allow, Comment: "Streaming."},
		},
		BrandGeneric: {
			{Domain: "doubleclick.net", Verdict: Block, Comment: "Google ads: present on most smart TVs."},
			{Domain: "googleadservices.com", Verdict: Block, Comment: "Google ad tracking."},
			{Domain: "graph.facebook.com", Verdict: Block, Comment: "Facebook SDK telemetry."},
			{Domain: "scorecardresearch.com", Verdict: Block, Comment: "Analytics beacon used by many TV apps."},
			{Domain: "netflix.com", Verdict: Allow, Comment: "Streaming."},
			{Domain: "youtube.com", Verdict: Allow, Comment: "Streaming."},
		},
	}
}

// Match returns the first rule matching name, if any.
func (s Set) Match(name string) (Rule, bool) {
	name = normalize(name)
	for _, r := range s {
		if matchDomain(name, normalize(r.Domain)) {
			return r, true
		}
	}
	return Rule{}, false
}

// Verdict returns the verdict for name, defaulting to Allow.
func (s Set) Verdict(name string) Verdict {
	if r, ok := s.Match(name); ok {
		return r.Verdict
	}
	return Allow
}

// normalize lowercases a domain name and drops its trailing dot.
func normalize(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, "."))
}

// matchDomain reports whether name is domain or a subdomain of it.
func matchDomain(name, domain string) bool {
	return name == domain || strings.HasSuffix(name, "."+domain)
}
