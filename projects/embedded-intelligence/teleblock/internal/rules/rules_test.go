package rules

import "testing"

func TestVerdictBlock(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	for _, name := range []string{
		"samsungacr.com",
		"us.samsungacr.com", // subdomains are covered
		"smetrics.samsung.com",
	} {
		if got := set.Verdict(name); got != Block {
			t.Errorf("Verdict(%q) = %q, want block", name, got)
		}
	}
}

func TestVerdictAllow(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	for _, name := range []string{"netflix.com", "www.netflix.com", "unknown.example"} {
		if got := set.Verdict(name); got != Allow {
			t.Errorf("Verdict(%q) = %q, want allow", name, got)
		}
	}
}

func TestVerdictObserve(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	if got := set.Verdict("samsungcloud.com"); got != Observe {
		t.Errorf("Verdict(samsungcloud.com) = %q, want observe", got)
	}
}

func TestVerdictCaseInsensitive(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	if got := set.Verdict("US.SamsungACR.COM"); got != Block {
		t.Errorf("Verdict(US.SamsungACR.COM) = %q, want block", got)
	}
}

func TestVerdictTrailingDot(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	if got := set.Verdict("samsungacr.com."); got != Block {
		t.Errorf("Verdict(samsungacr.com.) = %q, want block", got)
	}
}

func TestMatchStaysOnDotBoundaries(t *testing.T) {
	set := DefaultSets()[BrandSamsung]
	if _, ok := set.Match("notsamsungacr.com"); ok {
		t.Error("Match(notsamsungacr.com) matched, want no match")
	}
}

func TestDefaultSets(t *testing.T) {
	sets := DefaultSets()
	for _, brand := range []string{BrandSamsung, BrandLG, BrandSony, BrandTCL, BrandGeneric} {
		set, ok := sets[brand]
		if !ok {
			t.Fatalf("no rule set for brand %q", brand)
		}
		if len(set) < 6 {
			t.Errorf("brand %q: %d rules, want at least 6", brand, len(set))
		}
		for _, r := range set {
			if r.Domain == "" || r.Comment == "" {
				t.Errorf("brand %q: rule %+v lacks domain or comment", brand, r)
			}
			switch r.Verdict {
			case Allow, Block, Observe:
			default:
				t.Errorf("brand %q: rule %q has invalid verdict %q", brand, r.Domain, r.Verdict)
			}
		}
	}
}
