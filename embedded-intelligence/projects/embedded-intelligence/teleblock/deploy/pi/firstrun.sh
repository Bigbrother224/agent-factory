#!/bin/bash
# teleblock — one-shot first-boot provisioning for Raspberry Pi OS.
#
# deploy/pi/build-image.sh drops this file on the card's boot partition; the
# hook Raspberry Pi Imager installs in /etc/rc.local (and the systemd.run= in
# cmdline.txt when present) runs it exactly once. The script then removes both
# hooks and itself from the card, so a reboot never replays it.
set -euo pipefail

LOG_FILE=/var/log/teleblock-firstrun.log
mkdir -p /var/log
exec > >(tee -a "$LOG_FILE") 2>&1

BOOT_DIR=/boot/firmware
if [ ! -f "$BOOT_DIR/firstrun.sh" ]; then BOOT_DIR=/boot; fi

HOSTNAME_TARGET=teleblock
INSTALL_BIN=/usr/local/bin/teleblock
CONFIG_DIR=/etc/teleblock
SERVICE=teleblock.service

log()  { printf '[teleblock-firstrun] %s\n' "$*"; }
warn() { printf '[teleblock-firstrun] warning: %s\n' "$*" >&2; }
die()  { printf '[teleblock-firstrun] error: %s\n' "$*" >&2; exit 1; }

log "provisioning starting (boot directory: $BOOT_DIR)"

# ---------------------------------------------------------------------------
# 1. The Pi answers as teleblock.local
# ---------------------------------------------------------------------------
log "hostname: $HOSTNAME_TARGET"
if command -v hostnamectl >/dev/null 2>&1; then
	hostnamectl set-hostname "$HOSTNAME_TARGET"
else
	printf '%s\n' "$HOSTNAME_TARGET" > /etc/hostname
fi

if grep -qE '^127\.0\.1\.1[[:space:]]' /etc/hosts; then
	sed -i -E "s/^127\.0\.1\.1.*/127.0.1.1\t$HOSTNAME_TARGET/" /etc/hosts
else
	printf '127.0.1.1\t%s\n' "$HOSTNAME_TARGET" >> /etc/hosts
fi

# mDNS is what makes "teleblock.local" resolve before any DNS record exists.
if ! dpkg -s avahi-daemon >/dev/null 2>&1; then
	log "installing avahi-daemon for mDNS"
	DEBIAN_FRONTEND=noninteractive apt-get update -qq || warn "apt-get update failed"
	DEBIAN_FRONTEND=noninteractive apt-get install -y -qq avahi-daemon ||
		warn "avahi-daemon not installed; teleblock.local will not resolve until it is"
fi
systemctl enable --now avahi-daemon || warn "avahi-daemon could not be started"

# ---------------------------------------------------------------------------
# 2. teleblock itself
# ---------------------------------------------------------------------------
[ -f "$BOOT_DIR/teleblock" ] || die "$BOOT_DIR/teleblock is missing from the boot partition"

install -d -m 0755 "$CONFIG_DIR" /var/lib/teleblock
install -m 0755 "$BOOT_DIR/teleblock" "$INSTALL_BIN"

if ! id -u teleblock >/dev/null 2>&1; then
	log "creating the teleblock service account"
	useradd --system --home-dir /var/lib/teleblock --shell /usr/sbin/nologin teleblock
fi
chown -R teleblock:teleblock /var/lib/teleblock

if [ -f "$BOOT_DIR/teleblock.env" ]; then
	install -m 0640 "$BOOT_DIR/teleblock.env" "$CONFIG_DIR/teleblock.env"
	chown root:teleblock "$CONFIG_DIR/teleblock.env"
fi
[ -f "$BOOT_DIR/teleblock.service" ] || die "$BOOT_DIR/teleblock.service is missing"
install -m 0644 "$BOOT_DIR/teleblock.service" "/etc/systemd/system/$SERVICE"

# systemd-resolved holds 127.0.0.53:53 by default; teleblock needs port 53.
install -d -m 0755 /etc/systemd/resolved.conf.d
cat > /etc/systemd/resolved.conf.d/teleblock.conf <<'EOF'
[Resolve]
DNSStubListener=no
DNS=127.0.0.1
FallbackDNS=1.1.1.1 9.9.9.9
EOF
systemctl restart systemd-resolved || warn "systemd-resolved could not be restarted"

systemctl daemon-reload
systemctl enable "$SERVICE" || warn "$SERVICE could not be enabled"
systemctl restart "$SERVICE" ||
	warn "$SERVICE did not start on the first attempt — check: journalctl -u $SERVICE"

# ---------------------------------------------------------------------------
# 3. Tell the operator where to point the LAN
# ---------------------------------------------------------------------------
LAN_ADDRESS="$(hostname -I 2>/dev/null | awk '{print $1}')"
log "this Pi filters DNS on ${LAN_ADDRESS:-<lan-address>}:53"
log "set your router's DHCP DNS server to that address to protect the whole LAN"

# ---------------------------------------------------------------------------
# 4. This script runs once
# ---------------------------------------------------------------------------
log "removing the first-boot hooks"
if [ -f "$BOOT_DIR/cmdline.txt" ] && grep -q 'firstrun.sh' "$BOOT_DIR/cmdline.txt"; then
	sed -i -E 's#[[:space:]]*systemd\.run=[^[:space:]]*firstrun\.sh[^[:space:]]*##g' "$BOOT_DIR/cmdline.txt"
fi
if [ -f /etc/rc.local ] && grep -q 'firstrun.sh' /etc/rc.local; then
	sed -i -E '/firstrun\.sh/d' /etc/rc.local
fi
rm -f "$BOOT_DIR/firstrun.sh" "$BOOT_DIR/teleblock" "$BOOT_DIR/teleblock.env" \
      "$BOOT_DIR/teleblock.service" "$BOOT_DIR/README-BOOTFS.txt"

log "done — $HOSTNAME_TARGET.local is live, the Pi is the DNS gateway"
