#!/bin/sh
# TeleBlock — point d'entrée du conteneur : installe la règle iptables
# qui nourrit l'inspecteur SNI (T1), démarre le binaire, retire la règle
# à l'arrêt. Requiert cap NET_ADMIN et network_mode: host (compose) :
# c'est dans l'espace réseau de l'hôte que passent les flux 443 des TV —
# un bridge ne les laisserait jamais entrer et l'inspection serait
# inopérante.
set -eu

TLS_PORT="${TELEBLOCK_TLS_PORT:-8449}"

# Règle idempotente : -C teste son existence, -A l'ajoute si absente.
iptables -t nat -C PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT" 2>/dev/null ||
	iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT"

cleanup() {
	iptables -t nat -D PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT" 2>/dev/null || true
	kill "$child" 2>/dev/null || true
}
trap cleanup TERM INT

/usr/local/bin/teleblock "$@" &
child=$!
wait "$child"
