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
import re
import subprocess
import sys


def parse_repo(url):
    """owner/repo depuis une remote https ou ssh, sans corruption d'URL."""
    url = (url or "").strip().removesuffix(".git")
    m = re.match(r"(?:https?://[^/]+/|ssh://git@[^/]+/|git@[^/]+[:/])(.+?/.+)$", url)
    return m.group(1) if m else url


REPO = os.environ.get("GITHUB_REPOSITORY") or parse_repo(
    subprocess.run(["git", "config", "--get", "remote.origin.url"], capture_output=True, text=True).stdout
)


def gh(*args):
    """Commandes gh de haut niveau (pr, issue…) — --repo est supporté."""
    r = subprocess.run(["gh", *args, "--repo", REPO], capture_output=True, text=True, check=False)
    return r.stdout if r.returncode == 0 else ""


def gh_api(endpoint):
    """Appel REST API — endpoints ABSOLUS requis (gh api ne connaît pas --repo).
    Une erreur (404, 5xx, auth) est loggée sur stderr et rend {} : jamais d'échec silencieux total."""
    r = subprocess.run(
        ["gh", "api", f"/repos/{REPO}/{endpoint}"], capture_output=True, text=True, check=False
    )
    if r.returncode != 0:
        print(f"[correcteur-queue] API {endpoint} → échec {r.returncode} : {r.stderr.strip()[:120]}", file=sys.stderr)
        return {}
    try:
        return json.loads(r.stdout)
    except (json.JSONDecodeError, TypeError):
        return {}


def project_of(title):
    """'[axe] slug — …' → (axe, slug). Les PRs de la flotte sont titrées ainsi ;
    tout autre format (PR d'infra) → ("", "") → jamais corrigées automatiquement."""
    m = re.match(r"^\[([a-z0-9-]+)\]\s+([\w.-]+)", (title or "").strip().lower())
    return (m.group(1), m.group(2)) if m else ("", "")


def main():
    prs = json.loads(gh("pr", "list", "--state", "open", "--json", "number,headRefName,title,headRefOid") or "[]")
    for pr in prs:
        num, branch, title, oid = pr["number"], pr["headRefName"], pr["title"], pr["headRefOid"]
        axe, slug = project_of(title)

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

        # Cycles réellement consommés : corrections tentées (cycle) ou échouées (échec).
        # Le BLOCAGE final ne consomme pas de cycle (il n'y en a plus à consommer).
        cycles = sum(
            1
            for c in comments
            if (c.get("body") or "").startswith(("## Correcteur — cycle", "## Correcteur — échec"))
        )

        # Push après la dernière revue ? → la revue suivante est en cours, on attend.
        commit = gh_api(f"commits/{oid}")
        head_date = (commit.get("commit") or {}).get("committer", {}).get("date") or ""
        if head_date and last_review.get("created_at", "") and head_date > last_review["created_at"]:
            continue

        if cycles >= 2:
            # Épuisement : escalade à l'humain, une seule fois.
            if not any((c.get("body") or "").startswith("## Correcteur — BLOCAGE") for c in comments):
                print(f"bloque\t{num}\t{branch}\t{axe}\t{slug}\t{cycles}")
            continue

        # Seul un vrai projet de la flotte est corrigeable automatiquement (le brief
        # cible projects/<axe>/<slug>/). Une PR d'infra reste humaine.
        if not axe or not slug:
            continue
        if not gh_api(f"contents/projects/{axe}/{slug}?ref={branch}"):
            continue

        with open(f"/tmp/correcteur-{num}-revue.md", "w", encoding="utf-8") as f:
            f.write(body)
        print(f"correct\t{num}\t{branch}\t{axe}\t{slug}\t{cycles + 1}")


if __name__ == "__main__":
    sys.exit(main())
