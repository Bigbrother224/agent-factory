#!/usr/bin/env bash
# chantier.sh — LA BOUCLE RÉSIDENTE de l'usine : une équipe d'agents qui fait
# avancer le chantier actif en arrière-plan, tranche après tranche, jusqu'à
# « shippé ». Lancé par launchd (~/Library/LaunchAgents/com.usine.chantier.plist).
#
# Un cycle :
#   1. Trouve le chantier actif (issue idea/approved sans label shipped)
#   2. ROADMAP.md absente ? → le PLANIFICATEUR (pi-team, flash) la rédige
#   3. Prochaine tranche « - [ ] » → pi-team CREW (plan/codeur/vérifieur, flash+nemotron)
#      l'implémente dans projects/<axe>/<projet>/ et coche la tranche
#   4. Arbitre local verify.sh → PR « chantier/<projet> » sur GitHub
#   5. GitHub : CI + revue kimi → workflow auto-merge (vert + GO uniquement)
#   6. ROADMAP finie → label ship/pr sur l'issue → PORTE 2 humaine
set -euo pipefail
export PATH="/Users/mac/.npm-global/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

REPO="/Users/mac/Projects/factory"
R="Bigbrother224/agent-factory"

LOG="/tmp/usine-chantier.log"

# Auth pi : launchd n'hérite pas de .zshrc (pattern du fix scout-daily 2026-11-03).
export PI_PROVIDER="${PI_PROVIDER:-ollama}"
if [ -z "${OLLAMA_API_KEY:-}" ]; then
  KEY=$(security find-generic-password -s pi-ollama -a "$USER" -w 2>/dev/null || true)
  [ -n "$KEY" ] && export OLLAMA_API_KEY="$KEY"
fi
[ -n "${GITHUB_TOKEN:-}" ] || export GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"

log() { echo "[$(date '+%F %T')] $*" | tee -a "$LOG"; }
log "=== cycle chantier ==="

# Une seule instance : verrou mkdir atomique (macOS n'a pas flock)
LOCKDIR="/tmp/usine-chantier-cycle"
if ! mkdir "$LOCKDIR" 2>/dev/null; then
  log "un cycle tourne déjà — sortie."
  exit 0
fi
trap 'rmdir "$LOCKDIR" 2>/dev/null' EXIT

cd "$REPO"
git fetch -q origin

# 1) Chantier actif : issue idea/approved + axe/*, pas encore shipped
ACTIVE=$(gh issue list -R "$R" --label idea/approved --state open --limit 10 \
  --json number,title,labels | python3 -c "
import json, sys
for i in json.load(sys.stdin):
    labs = [l['name'] for l in i['labels']]
    if any(l.startswith('axe/') for l in labs) and 'shipped' not in labs:
        axe = next(l[4:] for l in labs if l.startswith('axe/'))
        print(f\"{i['number']}|{i['title']}|{axe}\")
        break
")
[ -n "$ACTIVE" ] || { log "aucun chantier actif (pas d'issue approuvée non shippée) — l'usine dort."; exit 0; }
NUM="${ACTIVE%%|*}"; REST="${ACTIVE#*|}"; TITLE="${REST%%|*}"; AXE="${REST#*|}"
SLUG=$(python3 "$REPO/fleet/slugify.py" "$TITLE")
log "chantier actif : #$NUM « $TITLE » (axe $AXE)"

# 2) Dossier projet : slug exact, sinon premier mot (ex. slug « teleblock-pare-feu » → « teleblock »)
PROJDIR=""
if [ -d "projects/$AXE/$SLUG" ]; then PROJDIR="$SLUG"
else
  FIRST="${SLUG%%-*}"
  for d in projects/$AXE/*/; do
    d="${d%/}"; d="${d##*/}"
    [ "$d" = "$FIRST" ] && PROJDIR="$d" && break
  done
fi
[ -n "$PROJDIR" ] || { log "ÉCHEC : aucun dossier projects/$AXE/$SLUG ni $FIRST/ — d'abord merger le MVP."; exit 1; }
PROJ="projects/$AXE/$PROJDIR"
log "projet : $PROJ"

# 3) Worktree du chantier (branch chantier/<projet>), resynchronisé sur main
WT="/tmp/usine-chantier-$PROJDIR"
BR="chantier/$PROJDIR"
if [ ! -d "$WT" ]; then
  git worktree add "$WT" -b "$BR" origin/main 2>/dev/null || git worktree add "$WT" -b "$BR" origin/main
else
  git -C "$WT" checkout -q "$BR" 2>/dev/null || git -C "$WT" checkout -q -B "$BR" origin/main
fi
git -C "$WT" fetch -q origin
git -C "$WT" rebase -q origin/main || { log "rebase impossible — worktree resynchronisé à neuf."; git -C "$WT" reset -q --hard origin/main; }

# 4) ROADMAP absente → le PLANIFICATEUR la rédige (lecture-seule → son rapport devient la roadmap)
if [ ! -f "$WT/$PROJ/ROADMAP.md" ]; then
  log "ROADMAP absente — planificateur (flash)…"
  gh issue view "$NUM" -R "$R" --json body -q .body > /tmp/usine-chantier-spec.md
  ~/bin/pi-team planificateur "Lis .spec-issue.md (spec de l'issue #$NUM, projet $PROJDIR) et le code existant dans $PROJ/. Rédige dans ton RAPPORT FINAL un document markdown qui sera copié tel quel en $PROJ/ROADMAP.md. Il découpe le chemin du MVP actuel vers la définition de « shippé » de l'issue en 5-8 tranches AUTONOMES (chacune faisable en une session). Format exact par tranche : « - [ ] T<n> : <objectif précis> — fichiers: <attendus> — vérif: <commande>. » La dernière tranche = « T<n> : définition de shippé atteinte » selon l'issue. Le document commence par un titre « # Roadmap — <projet> » et un tableau des tranches. Rien d'autre dans le rapport." "$WT" > /tmp/pi-team-plan.out 2>&1 || true
  PLAN=$(grep -o '/tmp/pi-team-[^ ]*planificateur\.md' /tmp/pi-team-plan.out | head -1)
  if [ -n "$PLAN" ] && [ -s "$PLAN" ]; then
    cp "$PLAN" "$WT/$PROJ/ROADMAP.md"
    git -C "$WT" add "$PROJ/ROADMAP.md"
    git -C "$WT" commit -qm "chantier : roadmap issue #$NUM (planificateur)"
    log "ROADMAP créée ($(grep -c '\- \[ \]' "$WT/$PROJ/ROADMAP.md" || true) tranches)."
  else
    log "ÉCHEC planificateur — pas de roadmap produite."; exit 1
  fi
fi

# 5) Prochaine tranche à faire
TRANCHE=$(grep -m1 '\- \[ \]' "$WT/$PROJ/ROADMAP.md" || true)
if [ -z "$TRANCHE" ]; then
  log "ROADMAP terminée — chantier $PROJDIR prêt à être shippé."
  gh issue comment "$NUM" -R "$R" --body "🏁 **Roadmap terminée** — le chantier $PROJDIR a atteint la définition de shippé de l'issue. Label \`ship/pr\` posé : à ta porte 2." >/dev/null 2>&1 || true
  gh issue edit "$NUM" -R "$R" --add-label ship/pr >/dev/null 2>&1 || true
  exit 0
fi
log "tranche : ${TRANCHE#*] }"

# 6) L'ÉQUIPE : pi-team crew (planificateur → codeur → verifieur ; modèles flash/nemotron)
~/bin/pi-team crew "Dans $PROJ/ : implémente la tranche «${TRANCHE#*] }». Lis d'abord $PROJ/ROADMAP.md pour le contexte global. Contraintes : ne modifie rien en dehors de $PROJ/, ./verify.sh $AXE/$PROJDIR doit finir VERT (exécute-le et rapporte sa sortie réelle), puis coche la tranche [x] dans $PROJ/ROADMAP.md." "$WT" >> /tmp/pi-team-plan.out 2>&1 || true
CREW=$(grep -o '/tmp/pi-crew-[^ ]*' /tmp/pi-team-plan.out | tail -1)
[ -n "$CREW" ] && log "rapports d'équipe : $CREW"

# 7) Arbitre local
if (cd "$WT" && ./verify.sh "$AXE/$PROJDIR"); then
  log "arbitre local : VERT"
else
  log "arbitre local : ROUGE — la tranche sera retentée au prochain cycle."
  exit 1
fi

# 8) Commit + push + PR (créée une fois, mise à jour ensuite)
git -C "$WT" add -A
git -C "$WT" commit -qm "chantier $PROJDIR : ${TRANCHE#*] }" --allow-empty || true
git -C "$WT" push -q -u origin "$BR" --force-with-lease
PR=$(gh pr list -R "$R" --head "$BR" --state open --json url -q '.[0].url' 2>/dev/null || true)
if [ -z "$PR" ]; then
  gh pr create -R "$R" --base main --head "$BR" \
    --title "[chantier] $PROJDIR — tranches consécutives (issue #$NUM)" \
    --body "Boucle chantier résidente : l'équipe d'agents avance tranche par tranche vers la définition de shippé de l'issue #$NUM. Auto-merge uniquement si CI verte + revue kimi GO." >/dev/null
  log "PR chantier créée."
else
  gh pr comment "$PR" -R "$R" --body "tranche avancée : ${TRANCHE#*] }" >/dev/null 2>&1 || true
  log "PR chantier mise à jour : $PR"
fi
log "=== cycle terminé — GitHub prend le relais (CI + revue kimi + auto-merge) ==="