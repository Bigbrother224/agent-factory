#!/usr/bin/env python3
"""Revue du groupe C : relecture lecture-seule d'un diff par kimi-k2.7-code (plan Ollama).
Usage : revue.py <fichier-diff>  → imprime la revue markdown sur stdout."""
import json
import os
import sys
import urllib.request

DIFF = open(sys.argv[1], encoding="utf-8").read()
if len(DIFF) > 40000:
    DIFF = DIFF[:40000] + "\n[… diff tronqué — taille plafonnée pour tenir dans le budget de tokens du réviseur …]"

PROMPT = f"""Tu es le groupe C (Vérificateurs) de l'usine agent-factory : agent de revue LECTURE SEULE.
Relis uniquement le diff ci-dessous. Réponds en français, format markdown strict :

## Revue du groupe C
### Verdict
**GO** (mergeable tel quel) ou **NO-GO** (bloquant) — un seul mot en gras.

### Problèmes (du plus grave au moins grave)
Pour chacun : `[BLOQUANT|SÉRIEUX|MINEUR]` + fichier/ligne + explication 1-2 phrases + ce qu'il faudrait faire.
Si un problème que tu signales n'est pas PROUVÉ par le diff, marque-le explicitement comme hypothèse.

### Qualité générale
2 phrases max : lisibilité, structure, respect apparent du contrat (tests réels, périmètre projects/ uniquement).

Ne félicite pas. Ne réécris pas le code. Si le code est bon, dis-le en une ligne et liste zéro problème.

---

```diff
{DIFF}
```"""

req = urllib.request.Request(
    "https://ollama.com/v1/chat/completions",
    data=json.dumps(
        {
            "model": "kimi-k2.7-code:cloud",
            "messages": [{"role": "user", "content": PROMPT}],
            "max_tokens": 32768,
        }
    ).encode(),
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {os.environ['OLLAMA_KEY']}",
    },
)
try:
    with urllib.request.urlopen(req, timeout=300) as r:
        out = json.load(r)["choices"][0]["message"]["content"]
except Exception as e:  # noqa: BLE001
    out = f"## Revue du groupe C\n**PANNE du réviseur** (erreur API) : {e}\nVerdict par défaut : **NO-GO** à trancher par l'humain."
if not out or not out.strip():
    out = "## Revue du groupe C\n**PANNE du réviseur** : réponse du modèle vide (budget de tokens épuisé par le raisonnement ?). Verdict par défaut : **NO-GO** à trancher par l'humain."
print(out.strip())