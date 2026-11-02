#!/usr/bin/env bash
# veille.sh — collecte de vraies tendances dev du jour (sources datées, pas d'hallucination).
# Produit : .veille/YYYY-MM-DD.md  (digest lu ensuite par le scout pi-team, lecture-seule)
set -euo pipefail
REPO="$(cd "$(dirname "$0")/.." && pwd)"
DAY=$(date +%F)
OUT="$REPO/.veille/$DAY.md"
mkdir -p "$REPO/.veille"

{
  echo "# Digest veille du $DAY"
  echo
  echo "## Hacker News — front page (API Algolia, live)"
  echo
  curl -sf "https://hn.algolia.com/api/v1/search?tags=front_page&hitsPerPage=25" \
    | python3 -c '
import json, sys
d = json.load(sys.stdin)
for h in d.get("hits", []):
    t = h.get("title") or ""
    u = h.get("url") or ("https://news.ycombinator.com/item?id=" + str(h.get("objectID")))
    print("- [%s](%s) — %s pts, %s coms" % (t, u, h.get("points"), h.get("num_comments")))
'
  echo
  echo "## GitHub — repos créés les 7 derniers jours, les plus étoilés (API live)"
  echo
  SINCE=$(date -v-7d +%F 2>/dev/null || date -d "7 days ago" +%F)
  gh api "search/repositories?q=created:>$SINCE+stars:>30&sort=stars&order=desc&per_page=15" \
    --jq '.items[] | "- [\(.full_name)](\(.html_url)) — ⭐\(.stargazers_count) — \(.description // "sans description")"'
  echo
  echo "## GitHub — repos à forte croissance cette semaine (top actifs)"
  echo
  gh api "search/repositories?q=pushed:>$SINCE+stars:>200&sort=stars&order=desc&per_page=10" \
    --jq '.items[] | "- [\(.full_name)](\(.html_url)) — ⭐\(.stargazers_count) — \(.language // "?") — \(.description // "sans description")"'
} > "$OUT"

echo "Digest écrit : $OUT ($(wc -l < "$OUT" | tr -d ' ') lignes)"