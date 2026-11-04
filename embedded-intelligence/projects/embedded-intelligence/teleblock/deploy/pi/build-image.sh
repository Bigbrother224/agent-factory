#!/usr/bin/env bash
# Assemble the boot-partition overlay that turns a stock Raspberry Pi OS card
# into a teleblock DNS gateway.
#
#   make build-pi && make pi-bundle
#
# Then either point Raspberry Pi Imager at dist/pi-bootfs/firstrun.sh, or copy
# the overlay straight onto a mounted boot partition:
#   ./deploy/pi/build-image.sh --bootfs /media/$USER/bootfs
set -euo pipefail

PI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$PI_DIR/../.." && pwd)"
OUT_DIR="${OUT_DIR:-$PROJECT_ROOT/dist/pi-bootfs}"
BINARY="${BINARY:-$PROJECT_ROOT/dist/teleblock-linux-arm64}"
BOOTFS="${BOOTFS:-}"

usage() {
	cat <<'EOF'
usage: deploy/pi/build-image.sh [options]

  --out DIR      staging directory for the boot overlay (default: dist/pi-bootfs)
  --binary PATH  arm64 binary to ship (default: dist/teleblock-linux-arm64)
  --bootfs DIR   copy the overlay onto a mounted boot partition
  -h, --help     show this message

Build the binary first:  make build-pi
EOF
}

die() { printf 'build-image: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
	case "$1" in
		--out)     OUT_DIR="${2:?--out needs a directory}"; shift 2 ;;
		--binary)  BINARY="${2:?--binary needs a path}"; shift 2 ;;
		--bootfs)  BOOTFS="${2:?--bootfs needs a directory}"; shift 2 ;;
		-h|--help) usage; exit 0 ;;
		*)         usage >&2; die "unknown argument: $1" ;;
	esac
done

[ -f "$BINARY" ] || die "$BINARY is missing — run 'make build-pi' first"

# The Pi runs a different libc than the build host: ship nothing but a static
# AArch64 executable. Catch that here instead of on a customer's desk.
if command -v file >/dev/null 2>&1; then
	description="$(file -b "$BINARY")"
	case "$description" in
		*AArch64*|*aarch64*) ;;
		*) die "$BINARY is not an AArch64 binary ($description)" ;;
	esac
	case "$description" in
		*dynamically\ linked*|*interpreter*) die "$BINARY is dynamically linked; rebuild with CGO_ENABLED=0" ;;
	esac
fi

if [ -d "$OUT_DIR" ]; then rm -rf "$OUT_DIR"; fi
mkdir -p "$OUT_DIR"

install -m 0755 "$BINARY"                   "$OUT_DIR/teleblock"
install -m 0755 "$PI_DIR/firstrun.sh"       "$OUT_DIR/firstrun.sh"
install -m 0644 "$PI_DIR/teleblock.service" "$OUT_DIR/teleblock.service"
install -m 0644 "$PI_DIR/teleblock.env"     "$OUT_DIR/teleblock.env"
install -m 0644 "$PI_DIR/README-Pi.md"      "$OUT_DIR/README-Pi.md"

cat > "$OUT_DIR/README-BOOTFS.txt" <<'EOF'
teleblock — boot overlay
========================

Copy every file in this directory to the root of the card's boot partition
(the FAT32 one that appears as "bootfs").

Raspberry Pi OS runs firstrun.sh once, on the very first boot. It installs
teleblock, sets the hostname to "teleblock", enables mDNS and hands port 53 to
the service. The Pi answers as teleblock.local a few seconds after that first
boot — no SSH session needed.

The command line lives in teleblock.env; edit it on the card before flashing to
change ports, the upstream resolver or the blocklists.
EOF

if [ -n "$BOOTFS" ]; then
	[ -d "$BOOTFS" ] || die "$BOOTFS is not a directory"
	[ -f "$BOOTFS/cmdline.txt" ] ||
		die "$BOOTFS does not look like a boot partition (no cmdline.txt)"
	cp -a "$OUT_DIR"/. "$BOOTFS"/
	printf 'build-image: overlay copied to %s\n' "$BOOTFS"
fi

printf 'build-image: overlay ready in %s\n' "$OUT_DIR"
printf 'build-image: next — Raspberry Pi Imager, "Use custom", pick %s/firstrun.sh\n' "$OUT_DIR"
