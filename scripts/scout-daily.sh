#!/usr/bin/env bash
# scout-daily.sh — pipeline quotidien du groupe A (Scouts + Sélectionneur), tourné par cron.
# 1. Digest veille du jour (sources réelles HN + GitHub)
# 2. Scout (lecture-seule) : propose 3 candidats DISTINCTS sourcés, chacun ancré dans un axe
# 3. Sélectionneur (raisonneur, lecture-seule, anti-hallucination) : vérifie les sources,
#    note chaque candidat sur 5 axes, rédige le md COMPLET du gagnant selon TEMPLATE.md
# 4. ideas/<date>-<slug>.md  5. Issue GitHub labelée idea/draft + axe/<axe> → porte 1 humaine.
# Toute panne est postée sur le bus (issue « ⚠️ ») : une panne silencieuse est interdite.
set -euo pipefail
export PATH="/Users/mac/.npm-global/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

REPO="/Users/mac/Projects/factory"
DAY=$(date +%F)
DIGEST="$REPO/.veille/$DAY.md"
LOG_FILE="/dev/null"

# Auth pi : le cron n'hérite pas de .zshrc — provider + clé Ollama explicitement résolus
# depuis le trousseau (le cron du 2026-11-03 08:30 a échoué en silence pour cette raison).
export PI_PROVIDER="${PI_PROVIDER:-ollama}"
if [ -z "${OLLAMA_API_KEY:-}" ]; then
  KEY=$(security find-generic-password -s pi-ollama -a "$USER" -w 2>/dev/null \
     || security find-generic-password -s plandex-ollama -a "$USER" -w 2>/dev/null || true)
  [ -n "$KEY" ] && export OLLAMA_API_KEY="$KEY"
fi

log() { echo "[$(date '+%F %T')] $*"; }

panne() { # panne <message> : log + issue sur le bus (visible par l'humain), puis exit 1.
  log "ÉCHEC : $1"
  local body alerte
  body=$(printf "Le pipeline idées du %s a échoué : %s\n\nLog (5 dernières lignes) :\n%s\n%s\n%s\nÀ investiguer — le groupe A est muet aujourd'hui." \
    "$DAY" "$1" '```' "$(tail -5 "$LOG_FILE" 2>/dev/null || echo indisponible)" '```')
  # gh résout le repo depuis la remote du checkout — pas de nom codé en dur.
  alerte=$(cd "$REPO" && gh issue create --title "⚠️ Pipeline scout en échec ($DAY)" --body "$body" 2>/dev/null || true)
  [ -n "$alerte" ] && log "Alerte postée sur le bus : $alerte" || log "⚠️ ALERTE NON POSTÉE (gh indisponible ?) — panne visible uniquement ici."
  exit 1
}

# 0) Idempotence : une idée par jour, pas plus
if ls "$REPO/ideas/$DAY"-*.md >/dev/null 2>&1; then
  log "OK : idée du $DAY déjà présente — rien à faire."
  exit 0
fi

# 1) Digest sourcé du jour (panne → alerte sur le bus, jamais de sortie silencieuse)
RUN=$(mktemp -d /tmp/scout-daily-XXXXXX)
"$REPO/scripts/veille.sh" > "$RUN/veille.log" 2>&1 || { LOG_FILE="$RUN/veille.log"; panne "digest veille.sh en échec"; }
log "Digest : $DIGEST"

# 2) Scout (lecture-seule) — 3 candidats DISTINCTS sourcés, chacun ancré dans un axe
~/bin/pi-team scout "Tu es le groupe A (Scouts) de l'usine. Lis $DIGEST (tendances réelles du jour, sourcées), ideas/README.md + ideas/TEMPLATE.md et la skill factory-ideation si tu la vois. Le but de l'usine est un PORTFOLIO GitHub articulé sur 3 axes : (1) ai-automation-and-systems — ingénierie système réelle autour des agents, orchestration, pipelines, CI ; (2) embedded-intelligence — IA dans le matériel, embarqué, edge ML ; (3) experimental-computing — informatique expérimentale poussée jusqu'à la démonstration. Propose TROIS projets DISTINCTS basés sur des tendances RÉELLES du digest — créatifs, cutting-edge, chacun résolvant un vrai problème avec un vrai besoin, shippable en MVP démontrable en moins de 3 commandes. PAS des jouets ni des réinventions. Pour chaque candidat : axe, tendance + source datée (du digest UNIQUEMENT), problème résolu, cible, différenciateur concret vs solutions existantes NOMMÉES, périmètre MVP réaliste en std lib (zéro dépendance), langage recommandé + pourquoi. Format strict : '## Candidat 1 — <titre>' puis les points. ≤90 lignes au total." "$REPO" > "$RUN/pi-team.log" 2>&1 || true

CANDIDATS=$(grep -o '/tmp/pi-team-[^ ]*scout\.md' "$RUN/pi-team.log" | head -1 || true)
if [ -z "$CANDIDATS" ] || [ ! -s "$CANDIDATS" ]; then
  LOG_FILE="$RUN/pi-team.log"; panne "aucun rapport scout produit (auth ?)"
fi
log "Candidats : $CANDIDATS"

# 3) Sélectionneur (raisonneur, lecture-seule, anti-hallucination) — trie et rédige
~/bin/pi-team raisonneur "Tu es le Sélectionneur du groupe A de l'usine. Lis $CANDIDATS (3 candidats du scout), $DIGEST (sources réelles du jour) et la skill factory-selection si tu la vois.

Étape 1 — anti-hallucination : vérifie que CHAQUE source citée par un candidat est RÉELLEMENT dans le digest ; note toute source inventée (disqualifiante).
Étape 2 — note chaque candidat 0-5 sur 5 axes : (a) ancrage réel dans les tendances du jour, (b) douleur/besoin réel, (c) différenciation vs existants nommés, (d) faisabilité MVP en std lib en UN cycle Aider, (e) valeur d'usage (est-ce que quelqu'un s'en sert vraiment ?).
Étape 3 — choisis LE meilleur total. Un projet jouet ne doit JAMAIS gagner par défaut ; si les 3 sont faibles, dis-le et refuse.
Étape 4 — rédige le md COMPLET du gagnant selon ideas/TEMPLATE.md, TOUTES les sections remplies, intégrant tes critiques : différenciation affinée vs concurrents nommés, périmètre MVP précis avec hors-périmètre explicite, choix du langage JUSTIFIÉ en 1 phrase, critères de succès mesurables. Première ligne = '# <titre>'. Métadonnées : Date : $DAY, Scout : pi-team scout, Sélectionneur : pi-team raisonneur, Axe : <un des 3 noms anglais>, Label : idea/draft + axe/<axe>. Termine par une section '## Pourquoi ce choix' : 2 phrases + les scores des 3 candidats. ≤80 lignes." "$REPO" > "$RUN/pi-team2.log" 2>&1 || true

REPORT=$(grep -o '/tmp/pi-team-[^ ]*raisonneur\.md' "$RUN/pi-team2.log" | head -1 || true)
if [ -z "$REPORT" ] || [ ! -s "$REPORT" ]; then
  LOG_FILE="$RUN/pi-team2.log"; panne "aucun rapport du sélectionneur (le tri qualité n'a pas eu lieu — aucune idée publiée sans tri)"
fi
log "Sélection : $REPORT"

# 4) Le rapport du sélectionneur EST le md : titre + axe, slug dérivé du titre
TITLE=$(head -1 "$REPORT" | sed 's/^# *//; s/ *$//')
[ -n "$TITLE" ] || { LOG_FILE="$REPORT"; panne "rapport sans titre"; }
grep -q "idea/draft" "$REPORT" || { LOG_FILE="$REPORT"; panne "rapport sans label idea/draft"; }
AXE=$(grep -ioE "(ai-automation-and-systems|embedded-intelligence|experimental-computing)" "$REPORT" | head -1)
[ -n "$AXE" ] || { LOG_FILE="$REPORT"; panne "rapport sans axe valide (ai-automation-and-systems|embedded-intelligence|experimental-computing)"; }
SLUG=$(python3 -c "
import unicodedata, re, sys
t = unicodedata.normalize('NFKD', sys.argv[1]).encode('ascii','ignore').decode()
s = re.sub(r'[^a-z0-9]+', '-', t.lower()).strip('-')
print((s[:40].rstrip('-')) or 'idee')
" "$TITLE") || { LOG_FILE="$REPORT"; panne "génération du slug impossible"; }
MD="$REPO/ideas/$DAY-$SLUG.md"
cp "$REPORT" "$MD" || { LOG_FILE="$REPORT"; panne "écriture du md impossible ($MD)"; }
log "Idée écrite : ideas/$DAY-$SLUG.md"

# 5) Issue GitHub = porte 1 humaine (gh résout le repo depuis la remote du checkout)
# Labels défensifs : créer au besoin (échec silencieux interdit).
(cd "$REPO" && gh label create "idea/draft" --color FBCA04 >/dev/null 2>&1 || true
 gh label create "axe/$AXE" --color 0E8A16 >/dev/null 2>&1 || true)
ISSUE=$(cd "$REPO" && gh issue create \
  --title "$TITLE (idea/draft)" \
  --label "idea/draft" --label "axe/$AXE" \
  --body-file "$MD") || { LOG_FILE="$MD"; panne "création de l'issue impossible (gh auth ?)"; }
log "Issue : $ISSUE — axe: $AXE"
rm -rf "$RUN"