#!/usr/bin/env python3
"""File d'attente du correcteur (groupe C3) — patrouille des PRs ouvertes.

Pour chaque PR ouverte, décide :
  - correct : dernière revue NO-GO (pas de PANNE), < 2 cycles correctifs déjà faits,
              aucun push après la dernière revue (sinon une nouvelle revue arrive),
              et le slug du titre correspond à un vrai projet projects/<slug>.
  - bloque  : 2 cycles correctifs épuisés ET la dernière revue reste NO-GO
              → un seul commentaire de blocage + label revue/bloquee (porte humaine).
  - (rien)  : PR saine, revue GO, revue en panne, revue en cours, déjà bloquée, ou hors périmètre.

Sortie stdout : une ligne TSV par action  `action <TAB> numéro <TAB> branche <TAB> slug <TAB> cycle`.
Le corps de la dernière revue est sauvé dans /tmp/correcteur-<num>-revue.md (action correct)."""
import json
import os
import subprocess
import sys

REPO = os.environ.get("GITHUB_REPOSITORY") or subprocess.run(
    ["git", "config", "--get", "remote.origin.url"], capture_output=True, text=True
).stdout.strip().replace(":", "/").removesuffix(".git").removeprefix("https://").removeprefix("git@")


def gh(*args):
    r = subprocess.run(["gh", *args, "--repo", REPO], capture_output=True, text=True, check=False)
    return r.stdout if r.returncode == 0 else ""


def gh_api(endpoint):
    out = gh("api", endpoint)
    try:
        return json.loads(out)
    except (json.JSONDecodeError, TypeError):
        return {}


def slug_of(title):
    title = title or ""
    if not title.startswith("["):
        return ""
    return title[1 : title.find("]")] if "]" in title else ""


def main():
    prs = json.loads(gh("pr", "list", "--state", "open", "--json", "number,headRefName,title,headRefOid") or "[]")
    for pr in prs:
        num, branch, title, oid = pr["number"], pr["headRefName"], pr["title"], pr["headRefOid"]
        slug = slug_of(title)

        detail = gh_api(f"issues/{num}")
        comments = gh_api(f"issues/{num}/comments") or []
        if not isinstance(comments, list):
            comments = []
        labels = {l["name"] for l in (detail.get("labels") or [])}
        if "revue/bloquee" in labels:
            continue  # déjà escaladée à l'humain

        reviews = [c for c in comments if (c.get("body") or "").startswith("## Revue du groupe C")]
        if not reviews:
            continue  # pas encore de revue
        last_review = reviews[-1]
        body = last_review.get("body") or ""
        if "PANNE" in body:
            continue  # panne du réviseur : ne pas brûler un cycle correctif sur une revue vide
        if "**NO-GO**" not in body:
            continue  # GO : rien à faire

        cycles = sum(1 for c in comments if (c.get("body") or "").startswith("## Correcteur"))

        # Push après la dernière revue ? → la revue suivante est en cours, on attend.
        commit = gh_api(f"commits/{oid}")
        head_date = (commit.get("commit") or {}).get("committer", {}).get("date") or ""
        if head_date and last_review.get("created_at", "") and head_date > last_review["created_at"]:
            continue

        if cycles >= 2:
            # Épuisement : escalade à l'humain, une seule fois.
            if not any((c.get("body") or "").startswith("## Correcteur — BLOCAGE") for c in comments):
                print(f"bloque\t{num}\t{branch}\t{slug}\t{cycles}")
            continue

        # Seul un vrai projet est corrigeable automatiquement (le brief de la session
        # cible projects/<slug>/). Une PR d'infra (titre sans [slug] connu) reste humaine.
        if not slug:
            continue
        if not gh_api(f"contents/projects/{slug}?ref={branch}"):
            continue

        with open(f"/tmp/correcteur-{num}-revue.md", "w", encoding="utf-8") as f:
            f.write(body)
        print(f"correct\t{num}\t{branch}\t{slug}\t{cycles + 1}")


if __name__ == "__main__":
    sys.exit(main())