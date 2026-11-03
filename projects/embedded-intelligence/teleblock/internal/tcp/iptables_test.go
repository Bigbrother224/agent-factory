package tcp

import "testing"

func TestRedirectRule(t *testing.T) {
	got := RedirectRule(8843)
	want := "iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 8843"
	if got != want {
		t.Errorf("RedirectRule(8843) = %q, want %q", got, want)
	}
}

func TestRedirectPortRoundTrip(t *testing.T) {
	port, err := RedirectPort(RedirectRule(8843))
	if err != nil {
		t.Fatalf("RedirectPort: %v", err)
	}
	if port != 8843 {
		t.Errorf("RedirectPort = %d, want 8843", port)
	}
}

func TestRedirectPortRejectsForeignRules(t *testing.T) {
	tests := []string{
		"",
		"echo hello",
		"iptables -t filter -A INPUT -p tcp --dport 443 -j DROP",
		"iptables -t nat -A PREROUTING -p udp --dport 443 -j MARK --to-ports 8843",
		"iptables -t mangle -A OUTPUT -p tcp --dport 443 -j MARK --to-ports 8843",
		"iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-ports 8843",
		"iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 0",
		"iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 70000",
		"iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports foo",
		// Foreign tokens inside an otherwise well-shaped rule : rejected
		// by the strict positional shape (review).
		"iptables -t nat -A PREROUTING -s 192.168.1.0/24 -p tcp --dport 443 -j REDIRECT --to-ports 8843",
		"iptables -t nat -A PREROUTING -p tcp -m tcp --dport 443 -j REDIRECT --to-ports 8843",
		"iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports 8843 --comment teleblock",
	}
	for _, r := range tests {
		if port, err := RedirectPort(r); err == nil {
			t.Errorf("RedirectPort(%q) = %d, want error", r, port)
		}
	}
}
