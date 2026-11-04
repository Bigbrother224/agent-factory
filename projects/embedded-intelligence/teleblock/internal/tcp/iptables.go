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
	// Forme STRICTE, positionnelle : exactement 12 tokens, la valeur de
	// certains changé (ports). Aucun token supplémentaire toléré entre
	// les drapeaux (revue : `-s 192.168.1.0/24` ou --comment d'une règle
	// étrangère passait pour une règle TeleBlock).
	const (
		wantTable, wantChain, wantProto, wantTarget = "nat", "PREROUTING", "tcp", "REDIRECT"
	)
	fields := strings.Fields(rule)
	// Préfixe optionnel « iptables » (sortie réelle de RedirectRule) : un
	// seul token, strictement ce mot — tout autre préfixe est rejeté.
	if len(fields) > 0 && fields[0] == "iptables" {
		fields = fields[1:]
	}
	if len(fields) != 12 ||
		fields[0] != "-t" || fields[1] != wantTable ||
		fields[2] != "-A" || fields[3] != wantChain ||
		fields[4] != "-p" || fields[5] != wantProto ||
		fields[6] != "--dport" || fields[7] != strconv.Itoa(RedirectIPPort) ||
		fields[8] != "-j" || fields[9] != wantTarget ||
		fields[10] != "--to-ports" {
		return 0, fmt.Errorf("tcp: not a TeleBlock REDIRECT rule: %q", rule)
	}
	port, err := strconv.Atoi(fields[11])
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("tcp: invalid --to-ports in %q", rule)
	}
	return port, nil
}

// Strict positional shape: index() removed — the exact 12-token check
// above makes positional scanning by flags unnecessary.
