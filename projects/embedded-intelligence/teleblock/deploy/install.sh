#!/bin/sh
# TeleBlock — installateur « une ligne », compatible avec un Pi-hole existant.
#
# Sans lecture de stdin (pipable) :
#   curl -fsSL <URL_DE_CE_FICHIER> | TELEBLOCK_REPO=<url-git-du-dépôt> sh
#
# Depuis un clone du dépôt (chemin principal) :
#   sh deploy/install.sh
#
# Effets :
#   1. vérifie docker et le plugin compose ;
#   2. prend les sources dans le dépôt courant, sinon clone
#      $TELEBLOCK_REPO dans $TELEBLOCK_DIR ;
#   3. détecte un Pi-hole déjà installé (hôte ou conteneur) et importe sa
#      gravity (domaines bloqués, un par ligne) dans l'état TeleBlock ;
#   4. écrit deploy/.env : port DNS 53 par défaut, repli sur 5353 si le 53
#      est déjà occupé par pi-hole-FTL ;
#   5. démarre la pile : docker compose -f deploy/docker-compose.yml up -d.
#
# Variables d'environnement :
#   TELEBLOCK_DIR       destination du clone (défaut /opt/teleblock)
#   TELEBLOCK_REPO      url git du dépôt (obligatoire sans checkout local)
#   TELEBLOCK_STATE     chemin d'état sur l'hôte (défaut : deploy/data du
#                      dépôt ; l'installateur l'exporte pour le compose)
#   TELEBLOCK_DNS_PORT  force le port DNS hôte (défait le repli 5353)

set -eu

TELEBLOCK_DIR="${TELEBLOCK_DIR:-/opt/teleblock}"
DNS_PORT="${TELEBLOCK_DNS_PORT:-}"

log() { printf '[teleblock] %s\n' "$*" >&2; }
die() { log "ERREUR : $*"; exit 1; }

# 1. Prérequis : on meurt avant de toucher au système.
command -v docker >/dev/null 2>&1 || die "docker est requis sur cet hôte"
docker compose version >/dev/null 2>&1 || die "le plugin « docker compose » est requis"

# 2. Sources.
if [ -f deploy/docker-compose.yml ]; then
	log "sources du dépôt courant : $PWD"
elif [ -f "$TELEBLOCK_DIR/deploy/docker-compose.yml" ]; then
	cd "$TELEBLOCK_DIR"
	log "sources existantes : $TELEBLOCK_DIR"
else
	[ -n "${TELEBLOCK_REPO:-}" ] \
		|| die "TELEBLOCK_REPO manquant : lancez sh deploy/install.sh depuis un clone, ou exportez TELEBLOCK_REPO=<url git>"
	mkdir -p "$TELEBLOCK_DIR" \
		|| die "impossible de créer $TELEBLOCK_DIR (relancer en root/sudo ?)"
	git clone --depth 1 "$TELEBLOCK_REPO" "$TELEBLOCK_DIR" \
		|| die "échec du clone $TELEBLOCK_REPO"
	cd "$TELEBLOCK_DIR"
	log "sources clonées : $TELEBLOCK_DIR"
fi

# 3. Pi-hole existant ? Import de sa gravity pour ne pas repartir de zéro.
#    Le chemin d'état honore TELEBLOCK_STATE (le compose le montera tel
#    quel) et devient absolu : la gravity importée atterrit exactement là
#    où le conteneur la lira, quel que soit le répertoire d'appel.
STATE_DIR="${TELEBLOCK_STATE:-$PWD/deploy/data}"
GRAVITY_OUT="$STATE_DIR/pihole-gravity.txt"
mkdir -p "$STATE_DIR"
PIHOLE_PRESENT=0

# Liste des conteneurs lue une seule fois (revue : deux appels à docker ps
# peuvent désigner des conteneurs différents si la liste bouge entre eux).
RUNNING_CONTAINERS=$(docker ps --format '{{.Names}}' 2>/dev/null || true)

if [ -f /etc/pihole/gravity.db ]; then
	PIHOLE_PRESENT=1
	if command -v sqlite3 >/dev/null 2>&1 &&
		sqlite3 /etc/pihole/gravity.db 'SELECT domain FROM gravity;' >"$GRAVITY_OUT" 2>/dev/null; then
		log "gravity Pi-hole importée : $(wc -l <"$GRAVITY_OUT" | tr -d ' ') domaines -> $GRAVITY_OUT"
	else
		log "sqlite3 absent ou gravity illisible : pas d'import, l'installation continue"
	fi
elif echo "$RUNNING_CONTAINERS" | grep -qi pihole; then
	PIHOLE_PRESENT=1
	PIHOLE_CONTAINER=$(echo "$RUNNING_CONTAINERS" | grep -i pihole | head -n 1)
	if docker exec "$PIHOLE_CONTAINER" sqlite3 /etc/pihole/gravity.db \
		'SELECT domain FROM gravity;' >"$GRAVITY_OUT" 2>/dev/null; then
		log "gravity Pi-hole conteneurisée importée : $(wc -l <"$GRAVITY_OUT" | tr -d ' ') domaines -> $GRAVITY_OUT"
	else
		log "export de la gravity via le conteneur impossible : pas d'import, l'installation continue"
	fi
fi

# 4. Port DNS hôte : 53 par défaut ; repli sur 5353 (puis 5354) si un
#    Pi-hole occupe déjà 53 — le repli est validé comme réellement libre
#    avant d'écrire deploy/.env (sinon le bind du compose échouerait).
port_free() {
	port_hex=$(printf '%04X' "$1")
	for f in /proc/net/tcp /proc/net/tcp6; do
		if [ -r "$f" ] && awk -v p=":$port_hex" '$2 ~ p"$" && $4 == "0A" { found = 1 }
			END { exit !found }' "$f" 2>/dev/null; then
			return 1
		fi
	done
	command -v nc >/dev/null 2>&1 || return 0
	! nc -z 127.0.0.1 "$1" >/dev/null 2>&1
}

if [ "$PIHOLE_PRESENT" -eq 1 ] && [ -z "$DNS_PORT" ]; then
	if port_free 5353; then
		DNS_PORT=5353
	elif port_free 5354; then
		DNS_PORT=5354
	else
		die "Pi-hole occupe :53 et aucun port de repli libre : forcez TELEBLOCK_DNS_PORT=<port>"
	fi
	log "Pi-hole détecté : TeleBlock écoutera sur :$DNS_PORT (le port 53 reste à pi-hole-FTL)"
fi
printf 'TELEBLOCK_DNS_PORT=%s\n' "${DNS_PORT:-53}" >deploy/.env
log "port DNS hôte : ${DNS_PORT:-53} (deploy/.env)"

# 5. Démarrage de la pile (état persistant dans $STATE_DIR, exporté pour
#    le compose : la gravity importée est visible dans le conteneur).
export TELEBLOCK_STATE="$STATE_DIR"
docker compose -f deploy/docker-compose.yml up -d --build
log "TeleBlock actif : DNS sur :${DNS_PORT:-53}/udp+tcp, état persistant dans $STATE_DIR"
