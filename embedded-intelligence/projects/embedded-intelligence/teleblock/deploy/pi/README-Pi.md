# teleblock on a Raspberry Pi

Flash one card, boot it, and the Pi is a DNS gateway: it answers on the LAN as
`teleblock.local` and filters every lookup that passes through it. No SSH, no
package install, no hand-written systemd unit.

## Flash it

```sh
make build-pi     # CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build
make pi-bundle    # assembles dist/pi-bootfs/
```

1. In Raspberry Pi Imager, pick **Raspberry Pi OS Lite (64-bit)** and your card.
2. Under **OS customisation → Edit settings**, set the user, the Wi-Fi and the
   locale. That step is what gives the Pi a network on first boot, and without a
   network it cannot fetch `avahi-daemon`.
3. Copy everything from `dist/pi-bootfs/` to the root of the card's boot
   partition — the FAT32 one.

Or, with the card already mounted:

```sh
./deploy/pi/build-image.sh --bootfs /media/$USER/bootfs
```

## First boot, in order

| Moment | What happens |
| --- | --- |
| 0 s | `firstrun.sh` runs once, from `/etc/rc.local`. |
| ~5 s | Hostname becomes `teleblock`; `/etc/hosts` follows. |
| ~10 s | `avahi-daemon` is enabled, so `teleblock.local` resolves over mDNS. |
| ~15 s | The static binary lands in `/usr/local/bin`, the unit in `/etc/systemd/system`. |
| ~20 s | `systemd-resolved` releases port 53 and `teleblock.service` takes it. |
| end | `firstrun.sh` deletes its own hooks from the card. |

From a machine on the same network:

```sh
ping -c1 teleblock.local
dig @teleblock.local example.com
```

## Make the LAN use it

The Pi is a resolver now, but nothing asks it yet. Two options:

- **Whole LAN** — in the router's DHCP settings, set the DNS server to the Pi's
  address; every client picks it up on the next lease renewal.
- **One machine** — point that machine's resolver at `teleblock.local`.

Leave the router's own upstream resolver alone: `teleblock` forwards whatever it
allows to `--upstream` from `/etc/teleblock/teleblock.env`.

## Configure without reflashing

`/etc/teleblock/teleblock.env` holds the command line the unit runs:

```sh
TELEBLOCK_ARGS=serve --listen 0.0.0.0:53 --upstream 1.1.1.1:53
```

Edit it on the boot partition *before* the first boot, or over SSH afterwards,
then `sudo systemctl restart teleblock.service`.

## Why the binary is static

`make build-pi` sets `CGO_ENABLED=0`, so no glibc from the build host leaks into
the artefact. A dynamically linked binary built on a desktop aborts on the Pi
with `version 'GLIBC_2.xx' not found`. `build-image.sh` refuses to stage one, so
the failure shows up on the desk rather than in the field:

```sh
file dist/teleblock-linux-arm64
# ELF 64-bit LSB executable, ARM aarch64, statically linked, stripped
```

## Troubleshooting

| Symptom | Check |
| --- | --- |
| `teleblock.local` does not resolve | `systemctl status avahi-daemon`, `hostname` |
| Service restarts in a loop | `journalctl -u teleblock.service -n 50` |
| Port 53 already taken | `ss -lunpt '( sport = :53 )'` — is the `resolved` drop-in in place? |
| First boot did nothing | `/var/log/teleblock-firstrun.log`, on the card and on the Pi |

## Verifying the artefacts

```sh
./verify.sh teleblock                    # arbiter — run it from the repository root
make verify                              # same arbiter, resolved from here
GOOS=linux GOARCH=arm64 go build ./...   # compiles everywhere, no artefact
bash tests/t7_pi_image.sh                # layout, unit, docs, overlay, cross-build
```

The arbiter lives at the repository root. On its own, `./verify.sh` is resolved
against the shell's working directory, so running it from here fails with the
bare `/bin/sh: 1: ./verify.sh: not found`. `verify.sh` in this directory is
therefore a shim: it walks up to the real arbiter and passes every argument
through. `--print-arbiter` shows which script would run, `VERIFY_SH=…` forces
one, and if nothing is found the shim says so and exits 3 rather than leaving
you with a missing-file message.

`make verify` searches the same way, starting *above* this directory so it can
never resolve to the shim.
