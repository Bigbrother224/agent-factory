// TeleBlock is a privacy firewall for smart TVs: a DNS server that
// answers according to the rule set of the TV brand assigned to the
// querying device. Telemetry domains are sinkholed, everything else is
// resolved upstream.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"teleblock/internal/dns"
	"teleblock/internal/profile"
	"teleblock/internal/rules"
)

// device is a TV declared on the command line.
type device struct {
	ip    string
	brand string
}

// deviceFlags collects repeated -device IP=brand flags.
type deviceFlags []device

func (d *deviceFlags) String() string { return "" }

func (d *deviceFlags) Set(v string) error {
	ip, brand, ok := strings.Cut(v, "=")
	if !ok || net.ParseIP(ip) == nil {
		return fmt.Errorf("want IP=brand, got %q", v)
	}
	*d = append(*d, device{ip: ip, brand: brand})
	return nil
}

func main() {
	listen := flag.String("listen", ":53", "UDP address to serve DNS on")
	upstream := flag.String("upstream", "1.1.1.1:53", "upstream DNS server for allowed names")
	var devices deviceFlags
	flag.Var(&devices, "device", "TV to protect, as IP=brand (samsung, lg, sony, tcl, generic); repeatable")
	flag.Parse()

	router := profile.New(rules.DefaultSets())
	for _, d := range devices {
		if err := router.Assign(net.ParseIP(d.ip), d.brand); err != nil {
			log.Fatalf("device %s: %v", d.ip, err)
		}
		log.Printf("device %s: %s profile", d.ip, d.brand)
	}

	server := dns.New(router, dns.Upstream(*upstream))
	log.Printf("teleblock: DNS firewall on %s, upstream %s", *listen, *upstream)
	if err := server.ListenAndServe(*listen); err != nil {
		log.Fatal(err)
	}
}
