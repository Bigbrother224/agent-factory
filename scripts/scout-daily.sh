#!/usr/bin/env bash
# scout-daily.sh — pipeline quotidien du groupe A (Scouts), tourné par cron.
# 1. Digest veille du jour (sources réelles HN + GitHub)  2. Scout pi-team (lecture-seule)
# 3. ideas/<date>-<slug>.md  4. Issue GitHub label idea/draft  → porte 1 humaine.
set -euo pipefail
export PATH="/Users/mac/.npm-global/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

REPO="/Users/mac/Projects/factory"
DAY=$(date +%F)
DIGEST="$REPO/.veille/$DAY.md"

log() { echo "[$(date '+%F %T')] $*"; }

# 0) Idempotence : une idée par jour, pas plus
if ls "$REPO/ideas/$DAY"-*.md >/dev/null 2>&1; then
  log "OK : idée du $DAY déjà présente — rien à faire."
  exit 0
fi

# 1) Digest sourcé du jour
"$REPO/scripts/veille.sh" >/dev/null
log "Digest : $DIGEST"

# 2) Scout (lecture-seule) — propose UNE idée sourcée depuis le digest
RUN=$(mktemp -d /tmp/scout-daily-XXXXXX)
~/bin/pi-team scout "Tu es le groupe A (Scouts) de l'usine. Lis $DIGEST (tendances réelles du jour, sourcées) et ideas/README.md + ideas/TEMPLATE.md (4 critères obligatoires). Propose LA meilleure idée de projet : créative, cutting-edge, basée sur une tendance RÉELLE du digest, qui résout un vrai problème avec un vrai besoin marché — PAS une réinvention. Dans ton rapport : le draft COMPLET du md selon TEMPLATE.md, première ligne = titre '# ...', avec les métadonnées (Date : $DAY, Scout, Label : idea/draft), les 4 critères avec sources datées prises du digest uniquement, et le premier contrat technique. Une seule idée, pas de liste. ≤60 lignes." "$REPO" > "$RUN/pi-team.log" 2>&1 || true

REPORT=$(grep -o '/tmp/pi-team-[^ ]*scout\.md' "$RUN/pi-team.log" | head -1)
if [ -z "$REPORT" ] || [ ! -s "$REPORT" ]; then
  log "ÉCHEC : aucun rapport scout produit. Log : $(tail -5 "$RUN/pi-team.log" | tr '\n' ' ')"
  exit 1
fi

# 3) Le rapport EST le md : slug dérivé du titre
TITLE=$(head -1 "$REPORT" | sed 's/^# *//; s/ *$//')
[ -n "$TITLE" ] || { log "ÉCHEC : rapport sans titre."; exit 1; }
grep -q "idea/draft" "$REPORT" || { log "ÉCHEC : rapport sans label idea/draft."; exit 1; }
SLUG=$(python3 -c "
import unicodedata, re, sys
t = unicodedata.normalize('NFKD', sys.argv[1]).encode('ascii','ignore').decode()
s = re.sub(r'[^a-z0-9]+', '-', t.lower()).strip('-')
print((s[:40].rstrip('-')) or 'idee')
" "$TITLE")
MD="$REPO/ideas/$DAY-$SLUG.md"
cp "$REPORT" "$MD"
log "Idée écrite : ideas/$DAY-$SLUG.md"

# 4) Issue GitHub = porte 1 humaine
gh issue create -R Bigbrother224/agent-factory \
  --title "$TITLE (idea/draft)" \
  --label "idea/draft" \
  --body-file "$MD" > "$RUN/issue.url"
log "Issue : $(cat "$RUN/issue.url")"
rm -rf "$RUN"