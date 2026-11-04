#!/bin/bash
# Construit la charge utile de l'image Raspberry Pi de TeleBlock (tranche T7).
#
# Aucune carte SD n'est touchée ici : le script cross-compile le binaire
# linux/arm64 statique depuis n'importe quelle machine (amd64 incluse), puis
# pose la configuration de l'image et son manifeste :
#   dist/pi-image/rootfs/            arborescence à poser sur la partition root
#   dist/pi-image/payload.json       manifeste (chemins, modes, SHA-256)
#   dist/teleblock-pi-rootfs.tar.gz  la même chose, prête à copier
set -euo pipefail
cd "$(dirname "$0")/../.."

out="${OUT:-dist/pi-image}"
binary="${BINARY:-dist/teleblock-linux-arm64}"
version="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

echo "== cross-compile linux/arm64 statique (CGO_ENABLED=0) =="
mkdir -p "$(dirname "$binary")"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -trimpath -ldflags "-s -w -X main.version=$version" \
    -o "$binary" .

echo "== charge utile : systemd, teleblock.local, port 53, manifeste =="
go run ./deploy/pi/imagegen -out "$out" -binary "$binary" -version "$version"

echo "== archive pour la carte SD =="
mkdir -p dist
tar -C "$out/rootfs" -czf dist/teleblock-pi-rootfs.tar.gz .

echo
echo "Charge utile : $out/$([ -d "$out/rootfs" ] && echo rootfs)"
echo "Archive      : dist/teleblock-pi-rootfs.tar.gz"
echo "Pose         : sudo tar -C <partition root de la SD> -xzf dist/teleblock-pi-rootfs.tar.gz"
