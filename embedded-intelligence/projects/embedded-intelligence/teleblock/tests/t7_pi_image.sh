#!/usr/bin/env bash
# T7 — Raspberry Pi flashable image.
#
# Guards the tranche end to end: static arm64 cross-build, boot overlay
# contents, systemd unit, first-boot provisioning (hostname, mDNS, DNS gateway)
# and the operator docs. Run it from anywhere:
#   bash tests/t7_pi_image.sh
set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 1

pass=0
fail=0
skip=0

ok()      { pass=$((pass + 1)); printf '  ok   %s\n' "$1"; }
ko()      { fail=$((fail + 1)); printf '  FAIL %s\n' "$1"; }
skipped() { skip=$((skip + 1)); printf '  skip %s\n' "$1"; }

assert_file() { if [ -f "$2" ]; then ok "$1"; else ko "$1 (missing: $2)"; fi; }
assert_exec() { if [ -x "$2" ]; then ok "$1"; else ko "$1 (not executable: $2)"; fi; }
assert_parses() { if bash -n "$2" 2>/dev/null; then ok "$1"; else ko "$1 (bash -n failed on $2)"; fi; }
assert_grep() {
	if grep -qE -- "$2" "$3" 2>/dev/null; then
		ok "$1"
	else
		ko "$1 (pattern not found in $3: $2)"
	fi
}

printf 'T7 — Raspberry Pi image\n'

# --- boot overlay scripts ---------------------------------------------------
assert_file   "deploy/pi/firstrun.sh exists"           deploy/pi/firstrun.sh
assert_exec   "deploy/pi/firstrun.sh is executable"    deploy/pi/firstrun.sh
assert_parses "deploy/pi/firstrun.sh parses"           deploy/pi/firstrun.sh
assert_file   "deploy/pi/build-image.sh exists"        deploy/pi/build-image.sh
assert_exec   "deploy/pi/build-image.sh is executable" deploy/pi/build-image.sh
assert_parses "deploy/pi/build-image.sh parses"        deploy/pi/build-image.sh

# --- systemd unit -----------------------------------------------------------
unit=deploy/pi/teleblock.service
assert_file "systemd unit is shipped"            "$unit"
assert_grep "unit starts the installed binary"   '^ExecStart=/usr/local/bin/teleblock' "$unit"
assert_grep "unit restarts on failure"           '^Restart=on-failure' "$unit"
assert_grep "unit is enabled at boot"            '^WantedBy=multi-user.target' "$unit"
assert_grep "unit may bind the privileged port"  'CAP_NET_BIND_SERVICE' "$unit"

# --- runtime configuration --------------------------------------------------
assert_grep "service listens on port 53" '0\.0\.0\.0:53' deploy/pi/teleblock.env

# --- first boot: teleblock.local + DNS gateway ------------------------------
assert_grep "first boot sets the hostname"             'HOSTNAME_TARGET=teleblock' deploy/pi/firstrun.sh
assert_grep "first boot enables mDNS (avahi)"          'avahi-daemon' deploy/pi/firstrun.sh
assert_grep "first boot frees port 53 from resolved"   'DNSStubListener=no' deploy/pi/firstrun.sh
assert_grep "first boot installs the systemd unit"     'teleblock\.service' deploy/pi/firstrun.sh
assert_grep "first boot is one-shot (hooks removed)"   'removing the first-boot hooks' deploy/pi/firstrun.sh

# --- operator docs ----------------------------------------------------------
readme=deploy/pi/README-Pi.md
assert_file "README-Pi.md is shipped"             "$readme"
assert_grep "docs mention Raspberry Pi Imager"    'Raspberry Pi Imager' "$readme"
assert_grep "docs mention teleblock.local"        'teleblock\.local' "$readme"
assert_grep "docs explain the DNS gateway role"   'DNS gateway' "$readme"

# --- the ./verify.sh entry point --------------------------------------------
# `./verify.sh` is resolved against the cwd, and the arbiter sits at the
# repository root: without the shim, running it from here dies with
# "/bin/sh: 1: ./verify.sh: not found".
assert_file   "verify.sh shim exists"        verify.sh
assert_exec   "verify.sh shim is executable" verify.sh
assert_parses "verify.sh shim parses"        verify.sh

if out="$(VERIFY_SH=/nonexistent/verify.sh ./verify.sh --print-arbiter 2>&1)"; then
	ko "verify.sh rejects a missing forced arbiter"
else
	case "$out" in
		*"/nonexistent/verify.sh"*) ok "verify.sh rejects a missing forced arbiter" ;;
		*) ko "verify.sh rejects a missing forced arbiter (unhelpful message: $out)" ;;
	esac
fi

if resolved="$(./verify.sh --print-arbiter 2>/dev/null)"; then
	if [ "$resolved" = "$PWD/verify.sh" ]; then
		ko "verify.sh never resolves to itself"
	elif [ -f "$resolved" ]; then
		ok "verify.sh resolves the arbiter above the project directory"
	else
		ko "verify.sh resolves the arbiter above the project directory (got: $resolved)"
	fi
else
	skipped "no arbiter above the project directory — resolution not exercised"
fi

# --- Makefile ---------------------------------------------------------------
if command -v make >/dev/null 2>&1; then
	if make -n build-pi 2>/dev/null | grep -q 'GOOS=linux GOARCH=arm64'; then
		ok "make build-pi targets linux/arm64"
	else
		ko "make build-pi targets linux/arm64"
	fi
	if make -n build-pi 2>/dev/null | grep -q 'CGO_ENABLED=0'; then
		ok "make build-pi disables cgo (static binary)"
	else
		ko "make build-pi disables cgo (static binary)"
	fi
	if make -n verify 2>/dev/null | grep -qE '"[^"]*/verify\.sh" teleblock'; then
		ok "make verify locates the arbiter above the project directory"
	else
		ko "make verify locates the arbiter above the project directory"
	fi
else
	skipped "make is not installed"
fi

# --- the real cross-build ---------------------------------------------------
if command -v go >/dev/null 2>&1; then
	if CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./... >/dev/null 2>&1; then
		ok "GOOS=linux GOARCH=arm64 go build ./... succeeds"
	else
		ko "GOOS=linux GOARCH=arm64 go build ./... succeeds"
	fi
else
	skipped "go toolchain is not installed"
fi

# --- overlay assembly -------------------------------------------------------
if command -v go >/dev/null 2>&1 && command -v make >/dev/null 2>&1; then
	staging="$(mktemp -d)"
	missing_out="$(mktemp -d)"

	if make build-pi >/dev/null 2>&1 && [ -f dist/teleblock-linux-arm64 ]; then
		ok "make build-pi produces dist/teleblock-linux-arm64"
	else
		ko "make build-pi produces dist/teleblock-linux-arm64"
	fi

	if ./deploy/pi/build-image.sh --out "$staging" >/dev/null 2>&1; then
		ok "build-image.sh stages the boot overlay"
	else
		ko "build-image.sh stages the boot overlay"
	fi

	if [ -f "$staging/firstrun.sh" ] && [ -f "$staging/teleblock" ] &&
	   [ -f "$staging/teleblock.service" ] && [ -f "$staging/teleblock.env" ]; then
		ok "overlay carries firstrun.sh, binary, unit and env file"
	else
		ko "overlay carries firstrun.sh, binary, unit and env file"
	fi

	if BINARY=/nonexistent ./deploy/pi/build-image.sh --out "$missing_out" >/dev/null 2>&1; then
		ko "build-image.sh refuses to stage a missing binary"
	else
		ok "build-image.sh refuses to stage a missing binary"
	fi

	rm -rf "$staging" "$missing_out"
else
	skipped "go/make missing — overlay assembly not exercised"
fi

printf '\nT7: %d passed, %d failed, %d skipped\n' "$pass" "$fail" "$skip"
[ "$fail" -eq 0 ]
