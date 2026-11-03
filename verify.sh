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
  # Convention : projects/<axe>/<projet> (2 niveaux). Les projets posés à plat
  # (legacy) ne sont pas découverts : la CI n'accepte que la structure par axes.
  TARGETS=$(cd "$PROJECTS_DIR" && ls -d */*/ 2>/dev/null | sed 's:/$::' | tr '\n' ' ')
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
    # Contrat B, règle 3 : minimum 8 tests réels (comptés dans les fichiers de test).
    # find -exec (portable macOS/Linux) : pas de xargs, pas de souci d'entrée vide.
    TESTS=$(find "$dir" \( -name '*.test.js' -o -path '*/test/*' \) -type f -name '*.js' -exec grep -hE '^[[:space:]]*(test|it)\(' {} + 2>/dev/null | wc -l | tr -d ' ')
    [ "${TESTS:-0}" -ge 8 ] || { fail "$name (node) : contrat B règle 3 — $TESTS tests trouvés, minimum 8"; continue; }

  elif [ -f "$dir/go.mod" ]; then
    cd "$dir"
    go vet ./...        || { fail "$name : go vet"; continue; }
    go build ./...      || { fail "$name : go build"; continue; }
    go test ./...       || { fail "$name : go test"; continue; }
    # Contrat B, règle 3 : minimum 8 tests réels.
    TESTS=$(grep -rE '^func Test' "$dir" --include='*_test.go' 2>/dev/null | wc -l | tr -d ' ')
    [ "${TESTS:-0}" -ge 8 ] || { fail "$name (go) : contrat B règle 3 — $TESTS tests trouvés, minimum 8"; continue; }

  elif [ -f "$dir/pyproject.toml" ]; then
    cd "$dir"
    python3 -m pip install --quiet ruff pytest >/dev/null 2>&1 || true
    [ -x "$(command -v ruff)" ] || pip install --quiet ruff >/dev/null 2>&1 || { fail "$name : ruff indisponible"; continue; }
    ruff check .        || { fail "$name : ruff"; continue; }
    pytest -q           || { fail "$name : pytest"; continue; }
    # Contrat B, règle 3 : minimum 8 tests réels.
    TESTS=$(grep -rE '^[[:space:]]*def test_' "$dir" --include='*.py' --exclude-dir='.venv' --exclude-dir='__pycache__' 2>/dev/null | wc -l | tr -d ' ')
    [ "${TESTS:-0}" -ge 8 ] || { fail "$name (python) : contrat B règle 3 — $TESTS tests trouvés, minimum 8"; continue; }

  else
    fail "$name : type de projet inconnu (ni package.json, ni go.mod, ni pyproject.toml)"
  fi
  # Scène E2E (règle humaine 2026-11-03) : si le projet définit tests/e2e.sh,
  # le scénario réel bout-en-bout doit passer — le produit démarre pour de
  # vrai et son comportement observable est vérifié, pas seulement ses unités.
  if [ -f "$dir/tests/e2e.sh" ]; then
    say "$name : scène E2E (tests/e2e.sh)"
    ( cd "$dir" && bash tests/e2e.sh ) || { fail "$name : E2E — tests/e2e.sh a échoué"; continue; }
  fi
done

printf '\n'

# Garde budgétaire (ordre humain 2026-11-03, reconfirmé et ÉTENDU le même
# jour : GLM est INTERDIT — tout modèle glm-*, quelque soit le tier — l'abonnement
# a failli y passer. Remplacement : deepseek-v4.1-flash (et deepseek-v4-pro:0813
# pour les tâches difficiles). La qualité passe par la boucle (scène E2E +
# relances + barème kimi), jamais par le modèle.
if grep -rqi 'glm-' "$REPO_ROOT/.github/workflows" 2>/dev/null; then
  fail "budget : modèle GLM détecté dans .github/workflows — INTERDIT (abonnement presque épuisé), deepseek-v4.1-flash:cloud uniquement"
fi

if [ "$FAILURES" -gt 0 ]; then
  printf '\033[31m[verify] %d échec(s) sur %d projet(s). PR REFUSÉE.\033[0m\n' "$FAILURES" "$CHECKED"
  exit 1
fi
printf '\033[32m[verify] %d projet(s) vérifié(s) — tout est vert. PR ACCEPTÉE côté CI.\033[0m\n' "$CHECKED"