#!/usr/bin/env python3
"""Revue du groupe C : relecture lecture-seule d'un diff par kimi-k2.7-code (plan Ollama).
Usage : revue.py <fichier-diff>  → imprime la revue markdown sur stdout.
Le barème qualité (fleet/BAREME.md) est joint à l'invite : la revue juge la qualité
(efficacité, intelligence, idiomatisme, langage, design, tests), pas seulement la correction.
Réponse vide → UNE relance avec diff tronqué à 25k (cause la plus fréquente : budget de raisonnement)."""
import json
import os
import sys
import urllib.request

DIFF = open(sys.argv[1], encoding="utf-8").read()
CAP = 40000
if len(DIFF) > CAP:
    # Budget partagé entre fichiers : chaque fichier du diff obtient une
    # part du plafond et le réviseur voit un EXTRAIT DE TOUT (bug réel
    # corrigé : la coupe naïve par la queue rendait tests/, internal/ et
    # l'entrypoint invisibles — kimi a 3× jugé « absence de tests/e2e.sh »
    # une scène qui passait la CI).
    import re

    parts = [p for p in re.split(r"(?m)^(?=diff --git )", DIFF) if p]
    per = max(400, CAP // max(1, len(parts)))
    chunks, cut_files = [], []
    for p in parts:
        m = re.match(r"diff --git a/(\S+) b/(\S+)", p)
        name = m.group(2) if m else "?"
        if len(p) > per:
            chunks.append(p[:per] + "\n[… extrait seulement — fichier tronqué …]\n")
            cut_files.append(name)
        else:
            chunks.append(p)
    joined = "".join(chunks)
    if len(joined) > CAP:
        joined = joined[:CAP] + "\n[… plafond global atteint …]\n"
    joined += (
        "\n[… budget de tokens partagé entre les fichiers : seuls des extraits sont visibles. "
        + f"{len(cut_files)} fichier(s) tronqué(s) : " + ", ".join(cut_files) + " …]\n"
        + "Les fichiers tronqués sont « non évalués » : ne conclus JAMAIS à leur absence, "
        + "et ne compte pas leur non-visibilité comme un problème.\n"
    )
    DIFF = joined

BAREME = open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "BAREME.md"), encoding="utf-8").read()

PROMPT = f"""Tu es le groupe C (Vérificateurs) de l'usine agent-factory : agent de revue LECTURE SEULE, exigeant, sans complaisance.
Tu appliques strictement ce barème qualité — vert ≠ bon :

---
{BAREME}
---

Relis uniquement le diff ci-dessous. Réponds en français, format markdown STRICT (les titres ci-dessous sont obligatoires, dans cet ordre) :

## Revue du groupe C
### Verdict
**GO** (mergeable tel quel) ou **NO-GO** (bloquant) — un seul mot en gras, rien d'autre sur la ligne.

### Problèmes (du plus grave au moins grave)
Pour chacun : `[BLOQUANT|SÉRIEUX|MINEUR]` + fichier/ligne + explication 1-2 phrases + ce qu'il faudrait faire.
BLOQUANT ou SÉRIEUX : tout écart au barème qui dégrade durablement le projet — algorithme naïf inadapté, mauvaise structure de données, code non idiomatique, design de module incohérent, tests limités au happy path, modification hors de `projects/`.
MINEUR : cosmétique ou hypothétique. Si un problème n'est pas PROUVÉ par le diff, marque-le explicitement comme hypothèse.
Exception : un diff qui modifie l'usine elle-même (`.github/`, `fleet/`, `verify.sh` — maintenance de l'orchestration) est légitime. Ne le juge pas au périmètre `projects/`, mais à la robustesse de l'orchestration : races, échecs silencieux, boucles infinies, secrets mal gérés.
Exigence E2E (règle 7 du barème) : un projet dont la roadmap est terminée (proche du ship) SANS `tests/e2e.sh` réel démarrant le produit et vérifiant son comportement observable = NO-GO. Avant ce stade, signale son absence en SÉRIEUX — mais uniquement s'il n'apparaît ni dans le diff ni dans le manifeste des fichiers tronqués.

### Notes qualité (0-5, sans complaisance)
Une ligne par dimension, format `Efficacité : N/5 — une phrase de justification.` :
- Efficacité (complexité, structures de données)
- Intelligence du code (abstractions, cas limites, non-réinvention)
- Idiomatisme du langage
- Choix du langage au regard du problème
- Design (découpage, surface publique, lisibilité)
- Tests (comportement, cas limites, erreurs)

### Bilan
2 phrases max : le point fort et le point faible principal. Ne félicite pas. Ne réécris pas le code. Si le code est bon, dis-le en une ligne et liste zéro problème.

---

```diff
{DIFF}
```"""


def call(prompt: str) -> str:
    req = urllib.request.Request(
        "https://ollama.com/v1/chat/completions",
        data=json.dumps(
            {
                "model": "kimi-k2.7-code:cloud",
                "messages": [{"role": "user", "content": prompt}],
                "max_tokens": 32768,
            }
        ).encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {os.environ['OLLAMA_KEY']}",
        },
    )
    with urllib.request.urlopen(req, timeout=300) as r:
        return (json.load(r)["choices"][0]["message"]["content"] or "").strip()


out = ""
try:
    out = call(PROMPT)
except Exception as e:  # noqa: BLE001
    out = f"## Revue du groupe C\n**PANNE du réviseur** (erreur API) : {e}\nVerdict par défaut : **NO-GO** à trancher par l'humain."
# Réponse vide = budget mangé par le raisonnement : UNE relance, diff tronqué à 25k.
if not out and len(DIFF) > 25000:
    try:
        out = call(
            PROMPT.replace(
                DIFF, DIFF[:25000] + "\n[… diff tronqué à 25k — relance après réponse vide …]\n"
            )
        )
    except Exception:  # noqa: BLE001
        pass
if not out:
    out = "## Revue du groupe C\n**PANNE du réviseur** : réponse du modèle vide (2 tentatives, diff complet puis tronqué). Verdict par défaut : **NO-GO** à trancher par l'humain."
print(out)