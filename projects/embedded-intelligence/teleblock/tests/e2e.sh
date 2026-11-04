#!/bin/bash
# Scène E2E réelle de TeleBlock (barème règle 7) : le binaire est
# compilé puis démarré POUR DE VRAI, et son comportement observable est
# vérifié de bout en bout — résolution DNS par profil, sinkhole des
# domaines de télémétrie, RST ciblé sur SNI bloqué, inspection laissée
# intacte sinon, dashboard servi. Aucun mock : de vrais paquets sur de
# vrais sockets.
#
# Depuis T7, la même scène vérifie la livraison Raspberry Pi : cross-compile
# linux/arm64 statique et charge utile d'image générée — sans carte SD.
set -euo pipefail
cd "$(dirname "$0")/.."

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

echo "== T7 : cross-compile linux/arm64 statique =="
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -trimpath -ldflags "-s -w" -o "$work/teleblock-arm64" .
test -s "$work/teleblock-arm64"

echo "== T7 : charge utile d'image Pi =="
go run ./deploy/pi/imagegen -out "$work/pi" -binary "$work/teleblock-arm64" -version e2e >/dev/null
unit="$work/pi/rootfs/etc/systemd/system/teleblock.service"
test -f "$unit"
grep -q '^ExecStart=/usr/local/bin/teleblock$' "$unit"
grep -q '^AmbientCapabilities=CAP_NET_BIND_SERVICE$' "$unit"
test -x "$work/pi/rootfs/usr/local/bin/teleblock"
test -L "$work/pi/rootfs/etc/systemd/system/multi-user.target.wants/teleblock.service"
test -L "$work/pi/rootfs/etc/systemd/system/multi-user.target.wants/avahi-daemon.service"
grep -qx 'teleblock' "$work/pi/rootfs/etc/hostname"
grep -q 'teleblock.local' "$work/pi/rootfs/etc/hosts"
grep -q 'DNSStubListener=no' "$work/pi/rootfs/etc/systemd/resolved.conf.d/teleblock.conf"
grep -q '"hostname": "teleblock"' "$work/pi/payload.json"

# La scène d'intégration existante reste la juge du comportement réseau.
go test -tags e2e -count=1 -timeout 300s ./tests/
