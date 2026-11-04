# TeleBlock

Stop the tracking your smart TV does behind your back — without breaking Netflix.

TeleBlock runs on your network as a DNS resolver with a short memory and strong
opinions: each TV it sees starts in an observation week, every domain it contacts
gets recorded, and when the week is over TeleBlock enforces a per-brand profile —
ad and telemetry domains blocked, streaming allowed.

Then there is the traffic that never touches your resolver. TeleBlock inspects the
TLS ClientHello on plain port 443 (iptables redirect, local TCP proxy), classifies
each handshake by JA4 fingerprint, and resets the flows its profile blocks. It
never reads, decrypts or stores video content — a blocked connection is a reset,
that is all.

No account, no cloud, no phone-home of its own.

## Why not Pi-hole or NextDNS?

Both are excellent DNS blocklists, and both share two assumptions TeleBlock drops:
one rule set for every device on the LAN, and the client asking your resolver
before it connects. Smart TVs break both.

| | Pi-hole / NextDNS | TeleBlock |
|---|---|---|
| Policy | one blocklist per network | a profile per device, per brand |
| Enforcement | DNS answers only | DNS plus the SNI seen on the wire |
| Unknown traffic | allow or block, nothing between | JA4 fingerprint, parked in Observe |
| First days | everything changes at once | a silent observation week, then auto-enforce |
| Feedback | query logs | plain words: which TV, what was blocked |

The SNI check is the difference that matters. When a TV pulls ads from servers it
never resolved through you, a DNS blocklist cannot even see the connection.
TeleBlock reads the server name straight out of the ClientHello and resets just
that flow. Netflix next door keeps streaming.

## The dashboard

Point a browser at `http://teleblock.local`:

```
Devices on this network

192.168.1.42   living-room TV   Samsung   learning — 5 days left
192.168.1.51   bedroom TV       LG        enforced

bedroom TV: 96 tracking attempts blocked today
living-room TV: 214 tracking attempts blocked today
```

TeleBlock spots the TVs itself (ARP and DNS), you assign a brand in one click,
and the counter talks like a person, not a firewall.

## Three ways to install

1. **From source, today.** Clone the repository, then from the `teleblock` folder:

   ```
   go build -o teleblock .
   sudo ./teleblock
   ```

   Root is needed for the resolver's port 53 and the iptables redirect. The
   binary starts the DNS resolver, the SNI proxy and the dashboard.

2. **Docker Compose.** For homes that already run Pi-hole: multi-arch image,
   state on a volume, and a one-line importer that takes over from an existing
   install — `deploy/install.sh` in this repo, on the host that runs Docker.

3. **A flashable Raspberry Pi image.** Preconfigured card via Raspberry Pi
   Imager; the Pi boots straight into gateway duty and the dashboard answers on
   first boot. `deploy/pi/imagegen` assembles the image payload and its systemd
   unit, and `tests/e2e.sh` cross-compiles the ARM64 binary and checks both on
   every run.

## What the tests pin down

Every change must keep `./verify.sh teleblock` green: vet, build, the unit test
suite, and `tests/e2e.sh` — a real binary on real sockets, plus the ARM64 image
payload.

- DNS — response encoding, flags and RCODE carried over from the query
- Rules — exact and subdomain matching, verdict resolution, per-brand default sets
- JA4 — ClientHello parsing, including short and truncated handshake buffers
- TCP — SNI extraction, verdict routing through the proxy, targeted resets
- Learning — observation window, persisted state, automatic enforcement
- Dashboard — handlers, per-TV counters, brand assignment

## Validating the shipped definition

What no unit test can certify is your living room: a real Samsung and a real LG,
Netflix and YouTube playing end to end, telemetry counted as blocked at the
dashboard. That evidence lives in [`docs/validation.md`](docs/validation.md), a
dated report with one row per check and the proof behind it.

`go run ./deploy/validation -report docs/validation.md` reads that report and
fails on a missing check, an unknown status, or a row without evidence.
`go run ./deploy/validation -gate -report docs/validation.md` additionally
demands every row be `PASS` and the report signed. That is what keeps the report
honest: it stays unverified until the hardware session happens, and nobody can
call the project shipped by asserting it.

## License

MIT.
