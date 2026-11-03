package tcp

import (
	"fmt"
	"strconv"
	"strings"
)

// RedirectIPPort is the clear-text port TeleBlock inspects.
const RedirectIPPort = 443

// RedirectRule returns the iptables rule that feeds the inspection
// listener: transparent HTTPS traffic crossing the router (port 443)
// is REDIRECTed to the local listen port. Applied on the TV-facing
// interface; the rule is documented here and applied by the
// deployment layer, never executed by the package.
func RedirectRule(listenPort int) string {
	return fmt.Sprintf("iptables -t nat -A PREROUTING -p tcp --dport %d -j REDIRECT --to-ports %d",
		RedirectIPPort, listenPort)
}

// RedirectPort validates that rule is a TeleBlock REDIRECT rule for
// port 443 and returns the local listen port it targets.
func RedirectPort(rule string) (int, error) {
	fields := strings.Fields(rule)
	var want []string
	for _, f := range fields {
		switch f {
		case "-t", "-A", "-p", "--dport", "-j", "--to-ports":
			want = append(want, f)
		}
	}
	// Exactly the six flags of our rule, in order.
	got := fmt.Sprint(want)
	if got != "[-t -A -p --dport -j --to-ports]" {
		return 0, fmt.Errorf("tcp: not a TeleBlock REDIRECT rule: %q", rule)
	}
	i := index(fields, "--dport")
	if i < 0 || fields[i+1] != strconv.Itoa(RedirectIPPort) {
		return 0, fmt.Errorf("tcp: rule must redirect --dport %d: %q", RedirectIPPort, rule)
	}
	i = index(fields, "--to-ports")
	port, err := strconv.Atoi(fields[i+1])
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("tcp: invalid --to-ports in %q", rule)
	}
	return port, nil
}

// index returns the position of s in fields, -1 when absent.
func index(fields []string, s string) int {
	for i, f := range fields {
		if f == s {
			return i
		}
	}
	return -1
}