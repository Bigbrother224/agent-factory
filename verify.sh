#!/usr/bin/env bash
# verify.sh — l'arbitre froid de l'usine.
# Usage : ./verify.sh [projet1 projet2...]   (sans argument : tous les projets)
# Échoue si UN SEUL point de contrôle échoue. Pas d'exception, pas de warning toléré.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
PROJECTS_DIR="$REPO_ROOT/projects"
FAILURES=0
CHECKED=0

say()  { printf '\n\033[1m[verify]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[verify] ÉCHEC:\033[0m %s\n' "$*"; FAILURES=$((FAILURES+1)); }

if [ ! -d "$PROJECTS_DIR" ] || [ -z "$(ls -A "$PROJECTS_DIR" 2>/dev/null)" ]; then
  say "Aucun projet dans projects/ — rien à vérifier (OK)."
  exit 0
fi

if [ $# -gt 0 ]; then
  TARGETS="$*"
else
  TARGETS=$(ls "$PROJECTS_DIR")
fi

for name in $TARGETS; do
  dir="$PROJECTS_DIR/$name"
  if [ ! -d "$dir" ]; then
    fail "projects/$name : n'existe pas"
    continue
  fi
  CHECKED=$((CHECKED+1))
  say "=== $name ==="

  if [ -f "$dir/package.json" ]; then
    cd "$dir"
    if [ -f package-lock.json ]; then
      npm ci --no-audit --no-fund --silent || { fail "$name : npm ci"; continue; }
    fi
    has() { node -e "const s=require('./package.json').scripts||{};process.exit(s['$1']?0:1)" 2>/dev/null; }
    if has lint; then npm run --silent lint || { fail "$name : lint (node)"; continue; }; else
      fail "$name (node) : script 'lint' manquant"; continue; fi
    if has test; then npm run --silent test || { fail "$name : tests (node)"; continue; }; else
      fail "$name (node) : script 'test' manquant"; continue; fi
    if has build; then npm run --silent build || { fail "$name : build (node)"; continue; }; fi

  elif [ -f "$dir/go.mod" ]; then
    cd "$dir"
    go vet ./...        || { fail "$name : go vet"; continue; }
    go build ./...      || { fail "$name : go build"; continue; }
    go test ./...       || { fail "$name : go test"; continue; }

  elif [ -f "$dir/pyproject.toml" ]; then
    cd "$dir"
    python3 -m pip install --quiet ruff pytest >/dev/null 2>&1 || true
    [ -x "$(command -v ruff)" ] || pip install --quiet ruff >/dev/null 2>&1 || { fail "$name : ruff indisponible"; continue; }
    ruff check .        || { fail "$name : ruff"; continue; }
    pytest -q           || { fail "$name : pytest"; continue; }

  else
    fail "$name : type de projet inconnu (ni package.json, ni go.mod, ni pyproject.toml)"
  fi
done

printf '\n'
if [ "$FAILURES" -gt 0 ]; then
  printf '\033[31m[verify] %d échec(s) sur %d projet(s). PR REFUSÉE.\033[0m\n' "$FAILURES" "$CHECKED"
  exit 1
fi
printf '\033[32m[verify] %d projet(s) vérifié(s) — tout est vert. PR ACCEPTÉE côté CI.\033[0m\n' "$CHECKED"