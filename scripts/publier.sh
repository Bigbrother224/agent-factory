#!/usr/bin/env bash
# publier.sh — la promotion portfolio : sort un projet de l'usine (repo privé)
# vers son PROPRE repo GitHub PUBLIC, autonome (CI incluse).
#
# Usage : publier.sh <axe>/<projet> [--dry]
#   <axe>/<projet>  ex. embedded-intelligence/teleblock
#   --dry           tout prépare mais ne crée/pousse RIEN
#
# Conditions : arbitre verify.sh vert + README.md portfolio présent (contrat B règle 6).
set -euo pipefail
export PATH="/Users/mac/.npm-global/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

REPO="/Users/mac/Projects/factory"
R="Bigbrother224/agent-factory"
DRY=0; [ "${2:-}" = "--dry" ] || [ "${1:-}" = "--dry" ] && DRY=1
[ "${1:-}" = "--dry" ] && shift
TARGET="${1:-}"
[ -n "$TARGET" ] || { echo "usage: publier.sh <axe>/<projet> [--dry]"; exit 1; }

cd "$REPO"
SRC="projects/$TARGET"
[ -d "$SRC" ] || { echo "ÉCHEC : projects/$TARGET n'existe pas."; exit 1; }
PROJ="${TARGET##*/}"
GH_REPO="Bigbrother224/$PROJ"

echo "=== 1/4 Arbitre ==="
./verify.sh "$TARGET" > /dev/null && echo "verify.sh $TARGET : VERT" || { echo "ÉCHEC : arbitre rouge — pas de publication."; exit 1; }

echo "=== 2/4 Contrat portfolio ==="
[ -f "$SRC/README.md" ] || { echo "ÉCHEC : README.md manquant (contrat B règle 6) — un repo public sans README n'est pas un portfolio."; exit 1; }
DESCRIPTION=$(python3 -c "
import re
lines=[l for l in open('$SRC/README.md') if l.strip()]
title=lines[0].lstrip('# ').strip()
# première phrase non titre non label
for l in lines[1:]:
    s=l.strip().strip('*_')
    if s and not s.startswith(('#','-','[','!','<')) and len(s)>20:
        desc=s; break
else: desc=title
print((title+' — '+desc)[:350])")
echo "description : $DESCRIPTION"
if gh repo view "$GH_REPO" >/dev/null 2>&1; then
  echo "ÉCHEC : $GH_REPO existe déjà."; exit 1
fi

echo "=== 3/4 Copie autonome + CI ==="
STAGE=$(mktemp -d "/tmp/publier-$PROJ-XXXXXX")
cp -R "$SRC/." "$STAGE/"
mkdir -p "$STAGE/.github/workflows"
cat > "$STAGE/.github/workflows/verify.yml" <<'EOF'
name: verify
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 22 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - name: Tests
        run: |
          if [ -f package.json ]; then npm test; fi
          if [ -f go.mod ]; then go vet ./... && go build ./... && go test ./...; fi
          if [ -f pyproject.toml ]; then pip install -q ruff pytest && ruff check . && pytest -q; fi
EOF
git -C "$STAGE" init -q -b main
git -C "$STAGE" add -A
git -C "$STAGE" -c user.name="$(git config user.name)" -c user.email="$(git config user.email)" commit -qm "$PROJ — publié depuis l'usine agent-factory (portfolio)"

echo "=== 4/4 Publication ==="
if [ "$DRY" = "1" ]; then
  echo "[DRY] tout est prêt dans $STAGE — repo public qui SERAIT créé : $GH_REPO"
  echo "[DRY] rien créé, rien poussé."
  exit 0
fi
gh repo create "$GH_REPO" --public --description "$DESCRIPTION" --source "$STAGE" --push
gh repo edit "$GH_REPO" --add-topic portfolio >/dev/null 2>&1 || true
echo ""
echo "✅ Publié : https://github.com/$GH_REPO"
echo "   (issue un commentaire de traçabilité sur l'issue d'origine si tu veux)"
rm -rf "$STAGE"