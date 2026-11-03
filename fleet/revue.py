#!/usr/bin/env python3
"""Revue du groupe C : relecture lecture-seule d'un diff par kimi-k2.7-code (plan Ollama).
Usage : revue.py <fichier-diff>  → imprime la revue markdown sur stdout.
Le barème qualité (fleet/BAREME.md) est joint à l'invite : la revue juge la qualité
(efficacité, intelligence, idiomatisme, langage, design, tests), pas seulement la correction.

Diff trop grand pour une seule passe (plafond 40 Ko ≈ budget de tokens du réviseur) :
découpage PAR FICHIERS en passes ≤ plafond, chaque passe relue séparément — plus aucune
partie du diff n'est invisible (bug réel corrigé : la coupe naïve par la queue a fait
juger « absence de tests/e2e.sh » 3× une scène qui passait pourtant la CI).

Le commentaire final porte UN seul verdict (fusion : le plus sévère gagne). L'auto-merge
greppe « **GO** » n'importe où dans le commentaire : quand une passe dit NO-GO, le motif
est sanitarisé dans le corps fusionné pour interdire tout faux merge."""
import json
import os
import re
import sys
import urllib.request

CAP = 40000  # plafond d'une passe, en caractères (~10k tokens d'entrée)
EMPTY_RETRY = 25000  # relance d'une passe vide : tranche plus courte

BAREME = open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "BAREME.md"), encoding="utf-8").read()

PROMPT_HEAD = f"""Tu es le groupe C (Vérificateurs) de l'usine agent-factory : agent de revue LECTURE SEULE, exigeant, sans complaisance.
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
Exigence E2E (règle 7 du barème) : un projet dont la roadmap est terminée (proche du ship) SANS `tests/e2e.sh` réel démarrant le produit et vérifiant son comportement observable = NO-GO. Avant ce stade, signale son absence en SÉRIEUX — mais uniquement si le fichier est réellement absent du diff présent.
Cette passe ne couvre qu'une partie des fichiers du diff : ne conclus JAMAIS à l'absence d'un fichier hors de cette passe, et ne compte pas cette absence comme un problème.

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
"""


def split_file_parts(diff: str):
    """[(nom, contenu)] — une entrée par fichier du diff, noms dé-quotés
    (git cite entre guillemets les chemins contenant des espaces)."""
    parts = [p for p in re.split(r"(?m)^(?=diff --git )", diff) if p]
    out = []
    for p in parts:
        m = re.match(r'diff --git ?"?a/(.+?)"? ?"?b/(.+?)"?(?:\n|$)', p)
        name = m.group(2).strip('"') if m else "?"
        out.append((name, p))
    return out


def pack_passes(parts):
    """Regroupe les fichiers entiers en passes ≤ CAP ; un fichier seul plus
    gros que le plafond devient des passes-tranches étiquetées."""
    passes, cur, cur_len = [], [], 0
    for name, p in parts:
        if len(p) > CAP:
            if cur:
                passes.append(cur)
                cur, cur_len = [], 0
            for i in range(0, len(p), CAP - 200):
                tranche = p[i : i + CAP - 200]
                n = i // (CAP - 200) + 1
                if i + CAP - 200 < len(p):
                    tranche += f"\n[… extrait seulement — tranche {n} de {name} …]\n"
                passes.append([(f"{name} (tranche {n})", tranche)])
        elif cur_len + len(p) > CAP and cur:
            passes.append(cur)
            cur, cur_len = [(name, p)], len(p)
        else:
            cur.append((name, p))
            cur_len += len(p)
    if cur:
        passes.append(cur)
    return passes


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


def review_pass(files, all_files):
    """Relit une passe ; réponse vide → UNE relance sur une tranche plus
    courte (cause la plus fréquente : budget mangé par le raisonnement)."""
    names = ", ".join(n for n, _ in files)
    all_names = " ; ".join(n for n, _ in all_files)
    # Manifeste complet en fin de prompt : le réviseur doit JUGER uniquement
    # les fichiers de la passe, mais ne peut plus prétexter l'absence d'un
    # fichier qu'il voit nommé — cause récurrente et fausse de NO-GO
    # (« observe_test.go livré sans aucun test » alors qu'il est vert et
    # couvert dans une autre passe).
    footer = (
        "\n[mémento : cette passe ne contient QUE les fichiers de son en-tête ; "
        "le diff complet comporte exactement ces fichiers : " + all_names + ". "
        "Ne JAMAIS signer comme problème l'absence d'un fichier nommé ici, "
        "ni l'absence de tests d'un fichier dont le *_test.go est dans le diff ; "
        "un fichier hors passe est signalé au plus “non évalué dans cette passe”.]\n"
    )
    body = "".join(p for _, p in files)
    prompt = PROMPT_HEAD + footer + body + "\n```"
    try:
        out = call(prompt)
    except Exception as e:  # noqa: BLE001
        return "", f"**PANNE du réviseur** (erreur API) : {e}"
    if not out and len(body) > EMPTY_RETRY:
        try:
            out = call(
                PROMPT_HEAD
                + body[:EMPTY_RETRY]
                + "\n[… passe tronquée à 25k — relance après réponse vide …]\n```"
            )
        except Exception:  # noqa: BLE001
            pass
    return out, None


def extract_section(md: str, title: str) -> str:
    """Extrait le corps d'une section markdown (### Titre … jusqu'au
    prochain ### ou fin), sans le titre lui-même."""
    m = re.search(rf"(?ms)^{re.escape(title)}.*?\n(.*?)(?=^### |\Z)", md)
    return m.group(1).strip() if m else "(section absente)"


def verdict_of(md: str) -> str:
    m = re.search(r"\*\*(GO|NO-GO)\*\*", md)
    return m.group(1) if m else "PANNE"


def main() -> str:
    diff = open(sys.argv[1], encoding="utf-8").read()
    parts = split_file_parts(diff)
    passes = pack_passes(parts)
    if len(passes) == 1 and len(passes[0]) <= CAP:  # cas simple : une passe, sortie directe
        out, panne = review_pass(passes[0], parts)
        if panne:
            return f"## Revue du groupe C\n\n### Verdict\n**NO-GO**\n\n### Problèmes (du plus grave au moins grave)\n- [BLOQUANT] {panne} — Verdict par défaut à trancher par l'humain."
        if not out:
            return "## Revue du groupe C\n**PANNE du réviseur** : réponse du modèle vide (2 tentatives). Verdict par défaut : **NO-GO** à trancher par l'humain."
        return out

    # Multi-passes : chaque passe est relue, les revues sont fusionnées et
    # le verdict final est le plus sévère.
    reviewed = []
    for i, files in enumerate(passes, 1):
        out, panne = review_pass(files, parts)
        label = f"passe {i}/{len(passes)} — {', '.join(n for n, _ in files)}"
        reviewed.append((label, out, panne))

    verdicts = []
    for _, out, panne in reviewed:
        if panne or not out:
            verdicts.append("NO-GO")
        else:
            verdicts.append(verdict_of(out))
    overall = "NO-GO" if any(v != "GO" for v in verdicts) else "GO"

    lines = [
        "## Revue du groupe C",
        "",
        "### Verdict",
        f"**{overall}** — diff découpé en {len(reviewed)} passes par fichiers ; le verdict est le plus sévère des passes.",
        "",
        "### Problèmes (du plus grave au plus grave)",
    ]
    for label, out, panne in reviewed:
        if panne or not out:
            lines.append(f"\n**{label} :** PANNE du réviseur — défaut de preuve, la passe compte comme NO-GO.")
        else:
            lines.append(f"\n**{label} :**\n{extract_section(out, '### Problèmes')}")
    lines.append("\n### Notes qualité (0-5, sans complaisance)")
    for label, out, panne in reviewed:
        if panne or not out:
            continue
        lines.append(f"\n**{label} :**\n{extract_section(out, '### Notes qualité')}")
    lines.append("\n### Bilan")
    for label, out, panne in reviewed:
        if panne or not out:
            continue
        lines.append(f"\n**{label} :** {extract_section(out, '### Bilan')}")
    body = "\n".join(lines)
    if overall != "GO":
        # Garde-fou auto-merge : le motif **GO** ne doit apparaître nulle
        # part dans un commentaire de revue dont le verdict est NO-GO.
        body = body.replace("**GO**", "`GO`")
    return body


if __name__ == "__main__":
    print(main())