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
	"net/http"
	"strconv"
	"strings"

	"teleblock/internal/dns"
	"teleblock/internal/observe"
	"teleblock/internal/profile"
	"teleblock/internal/rules"
	"teleblock/internal/tcp"
	"teleblock/internal/web"
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
	tlsListen := flag.String("tls-listen", ":8449", "local port where redirected HTTPS (443) traffic arrives for passive SNI inspection; must match the iptables REDIRECT 443->8449 installed by deploy/entrypoint.sh (TELEBLOCK_TLS_PORT); empty disables")
	observeState := flag.String("observe-state", "observe.json", "local JSON file holding the per-device learning state; empty disables learning mode")
	webListen := flag.String("web-listen", ":8080", "local dashboard (TV panel) on this address; empty disables")
	flag.Parse()

	var learn *observe.Store
	if *observeState != "" {
		store, err := observe.New(*observeState)
		if err != nil {
			log.Fatalf("observe state %s: %v", *observeState, err)
		}
		learn = store
	}
	router := profile.New(rules.DefaultSets())
	if learn != nil {
		router.EnableLearning(learn)
		log.Printf("teleblock: per-device learning state %s, period %v", *observeState, observe.DefaultPeriod)
	}
	known := make(map[string]string)
	for _, d := range devices {
		if err := router.Assign(net.ParseIP(d.ip), d.brand); err != nil {
			log.Fatalf("device %s: %v", d.ip, err)
		}
		known[d.ip] = d.brand
		log.Printf("device %s: %s profile", d.ip, d.brand)
	}

	if *tlsListen != "" {
		listenPort := tcp.RedirectIPPort
		if _, portStr, err := net.SplitHostPort(*tlsListen); err == nil {
			if p, err := strconv.Atoi(portStr); err == nil {
				listenPort = p
			}
		}
		ln, err := net.Listen("tcp", *tlsListen)
		if err != nil {
			log.Fatalf("tls-listen %s: %v", *tlsListen, err)
		}
		inspector := tcp.New(router, nil)
		log.Printf("teleblock: TLS SNI inspection on %s (%s)", ln.Addr(), tcp.RedirectRule(listenPort))
		go func() {
			if err := inspector.Serve(ln); err != nil {
				log.Printf("teleblock: tcp inspection stopped: %v", err)
			}
		}()
	}

	if *webListen != "" {
		dash := web.New(router, known)
		log.Printf("teleblock: dashboard on http://localhost%s", *webListen)
		go func() {
			if err := http.ListenAndServe(*webListen, dash); err != nil {
				log.Printf("teleblock: dashboard stopped: %v", err)
			}
		}()
	}

	server := dns.New(router, dns.Upstream(*upstream))
	log.Printf("teleblock: DNS firewall on %s, upstream %s", *listen, *upstream)
	if err := server.ListenAndServe(*listen); err != nil {
		log.Fatal(err)
	}
}
