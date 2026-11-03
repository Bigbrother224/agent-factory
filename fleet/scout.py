#!/usr/bin/env python3
"""Scout du groupe A, autonome (VM GitHub ou local).
1. Digest réel du jour (HN Algolia + GitHub search API)
2. Idée sourcée via nemotron-3-super (plan Ollama) — alignée sur un axe portfolio
3. ideas/<date>-<slug>.md committé + issue idea/draft + axe/* créée
Idempotent : une idée par jour."""
import json
import os
import re
import subprocess
import sys
import unicodedata
import urllib.request

REPO = os.environ.get("REPO", "")
KEY = os.environ["OLLAMA_KEY"]
DAY = subprocess.run(["date", "+%F"], capture_output=True, text=True).stdout.strip()

# 0) Idempotence
existing = subprocess.run(
    ["gh", "issue", "list", "-R", REPO, "--label", "idea/draft", "--state", "open", "--limit", "50",
     "--json", "title"], capture_output=True, text=True).stdout
if any(DAY in t["title"] for t in json.loads(existing)):
    print("une idée draft contient déjà la date du jour — rien à faire")
    sys.exit(0)

# 1) Digest réel
digest = ["# Digest veille du " + DAY, ""]

req = urllib.request.Request(
    "https://hn.algolia.com/api/v1/search?tags=front_page&hitsPerPage=25")
with urllib.request.urlopen(req, timeout=60) as r:
    hits = json.load(r)["hits"]
digest.append("## Hacker News — front page (API Algolia, live)\n")
for h in hits:
    url = h.get("url") or f"https://news.ycombinator.com/item?id={h.get('objectID')}"
    digest.append(f"- [{h.get('title') or ''}]({url}) — {h.get('points')} pts, {h.get('num_comments')} coms")
digest.append("")

trend = subprocess.run(
    ["gh", "api", "search/repositories?q=created:>" + DAY[:-2] + "28+stars:>30&sort=stars&order=desc&per_page=15",
     "--jq", ".items[] | \"- [\\(.full_name)](\\(.html_url)) — ⭐\\(.stargazers_count) — \\(.description // 'sans description')\""],
    capture_output=True, text=True).stdout
digest.append("## GitHub — repos créés cette semaine, les plus étoilés\n")
digest.append(trend)
DIGEST = "\n".join(digest)
open("/tmp/digest.md", "w").write(DIGEST)

# 2) Idée sourcée (nemotron)
PROMPT = f"""Tu es le groupe A (Scouts) de l'usine. Lis /tmp/digest.md (tendances réelles du jour, sourcées) ci-dessous.
Le but de l'usine est un PORTFOLIO GitHub articulé sur 3 axes : (1) ai-automation-and-systems — ingénierie système réelle autour des agents, orchestration, pipelines, CI ; (2) embedded-intelligence — IA dans le matériel, embarqué, edge ML ; (3) experimental-computing — informatique expérimentale poussée jusqu'à la démonstration.
Propose LA meilleure idée de projet : créative, cutting-edge, basée sur une tendance RÉELLE du digest, qui résout un vrai problème avec un vrai besoin marché, shippable en MVP démontrable en moins de 3 commandes, illustrant ÉVIDEMMENT un des 3 axes. PAS une réinvention, PAS un projet de test.
Réponds UNIQUEMENT avec le draft markdown complet : première ligne = titre « # ... », puis métadonnées (Date : {DAY}, Scout, Axe : <un des 3 noms anglais>, Label : idea/draft), puis les sections du template : 1) Axe & légitimité portfolio 2) Tendance sourcée 3) Besoin réel du marché 4) Problème résolu 5) Ce qui manque aux solutions existantes 6) Premier contrat technique. Sources datées prises du digest uniquement. Une seule idée. ≤60 lignes.

{DIGEST}"""

req = urllib.request.Request(
    "https://ollama.com/v1/chat/completions",
    data=json.dumps({
        "model": "nemotron-3-super:cloud",
        "messages": [{"role": "user", "content": PROMPT}],
        "max_tokens": 8192,
    }).encode(),
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {KEY}"},
)
with urllib.request.urlopen(req, timeout=600) as r:
    out = json.load(r)["choices"][0]["message"]["content"]
if not out.strip():
    print("réponse vide du scout — abandon")
    sys.exit(1)

# 3) Validations
assert out.startswith("#"), "le draft doit commencer par un titre '#'"
AXE = ""
m = re.search(r"(ai-automation-and-systems|embedded-intelligence|experimental-computing)", out)
if m:
    AXE = m.group(1)
if not AXE:
    print("pas d'axe valide dans le draft — abandon")
    sys.exit(1)
title = out.splitlines()[0].lstrip("# ").strip()

# 4) Fichier ideas/ + commit
t = unicodedata.normalize("NFKD", title).encode("ascii", "ignore").decode()
slug = "-".join(re.sub(r"[^a-z0-9]+", "-", t.lower()).strip("-").split("-")[:3])[:24] or "idee"
MD = f"ideas/{DAY}-{slug}.md"
open(MD, "w").write(out)
subprocess.run(["git", "add", MD], check=True)
subprocess.run(["git", "commit", "-qm", f"scout : idée du {DAY} — {slug} (axe {AXE})"], check=True)
subprocess.run(["git", "push", "-q"], check=True)

# 5) Issue = porte 1
subprocess.run(
    ["gh", "issue", "create", "-R", REPO,
     "--title", f"{title} (idea/draft)",
     "--label", "idea/draft", "--label", f"axe/{AXE}",
     "--body-file", MD], check=True)
print(f"idée publiée : {title} — axe {AXE}")