#!/bin/sh
# TeleBlock — point d'entrée du conteneur : installe la règle iptables
# qui nourrit l'inspecteur SNI (T1), démarre le binaire, retire la règle
# à l'arrêt. Requiert cap NET_ADMIN et network_mode: host (compose) :
# c'est dans l'espace réseau de l'hôte que passent les flux 443 des TV —
# un bridge ne les laisserait jamais entrer et l'inspection serait
# inopérante.
set -eu

TLS_PORT="${TELEBLOCK_TLS_PORT:-8449}"

# Règle idempotente. -C rend 1 quand la règle est absente : c'est le seul
# cas où l'on tente l'ajout. Tout autre code est une vraie erreur (cap
# NET_ADMIN manquante, iptables absent) : on s'arrête en l'écrivant au
# lieu de laisser un -A échouer en silence (revue : masquer -C coupait
# tout diagnostic). Le rc passe par || rc=$? pour ne pas déclencher
# set -e avant le case.
rc=0
iptables -t nat -C PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT" 2>/dev/null || rc=$?
case $rc in
0)
	;; # règle déjà présente
1)
	iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT" ||
		{ echo "[teleblock] ERREUR : impossible d'installer la règle REDIRECT 443->$TLS_PORT (cap NET_ADMIN manquante ?)" >&2; exit 1; }
	;;
*)
	echo "[teleblock] ERREUR : iptables -C a échoué (rc=$rc : cap NET_ADMIN manquante, iptables absent ?)" >&2
	exit 1
	;;
esac

# La règle vit dans l'espace réseau de l'HÔTE (network_mode: host) :
# elle doit disparaître quoi qu'il arrive — arrêt normal, erreur fatale
# (set -e sort via le trap EXIT), docker stop (TERM) — sinon le HTTPS
# du LAN reste coupé après la mort du conteneur (revue : fuite
# bloquante). cleanup est idempotente : un double passage est inoffensif.
child=""
# Cleanup idempotente : un double passage est inoffensif. child est
# initialisé AVANT le trap : un signal reçu entre le trap et l'assignation
# (démarrage lent du binaire) ne tue pas le shell sous set -u — la règle
# iptables doit toujours partir avec le conteneur (revue : fuite bloquante).
cleanup() {
	iptables -t nat -D PREROUTING -p tcp --dport 443 -j REDIRECT --to-ports "$TLS_PORT" 2>/dev/null || true
	if [ -n "$child" ]; then
		kill "$child" 2>/dev/null || true
	fi
}
trap cleanup EXIT INT TERM

/usr/local/bin/teleblock "$@" &
child=$!
wait "$child"
