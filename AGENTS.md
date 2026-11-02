# AGENTS.md — agent-factory

## 1. Directory Purpose
Usine à projets : 3 groupes d'agents (A-Scouts, B-Exécutants flotte cloud, C-Vérificateurs) communiquant uniquement via issues/PRs/labels de ce dépôt.

## 2. Ownership & Sensitivity
- Owner: Zineb
- Sensitivity: `shared`
- Last Reviewed: 2026-11-02

## 3. File Tree Summary
- `ideas/` — un md par idée (4 critères obligatoires, sources datées) + `TEMPLATE.md`
- `projects/<nom>/` — un sous-dossier par projet exécuté (node/go/python auto-détecté par verify.sh)
- `verify.sh` — l'arbitre froid : lint + tests + build, refus si un seul échec
- `.github/workflows/verify.yml` — CI qui refuse toute PR rouge
- `ETAT.md` — état du projet (protocole)

## 4. Key Conventions
- Labels (protocole du bus) : `idea/draft`, `idea/approved`, `idea/rejected`, `ship/pr`, `verify/ok`, `verify/failed`.
- Toute PR doit passer `verify.sh` ; l'arbitre est froid, aucune exception manuelle.
- Un projet dans `projects/` doit être auto-vérifiable : node (`package.json` + scripts `lint`/`test`), go (`go.mod`), ou python (`pyproject.toml`).
- Test local : `./verify.sh` (tout) ou `./verify.sh <nom-du-projet>`.
- Ne jamais merger une PR rouge ; ne jamais ouvrir une issue sans label.

## 5. Agent Instructions
- Les agents de B (flotte) ne livrent QUE des PR, jamais de push direct sur main.
- Un agent reviewer (groupe C, lecture seule) commente les PR, ne les merge jamais.
- Interdiction de toucher à `.github/` sans validation humaine explicite (c'est le verrou de l'usine).

## 6. Related Docs
- `README.md` — architecture 3 groupes + les 2 portes humaines.
- `/Users/mac/PROJETS.md` — index des projets actifs.