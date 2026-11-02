# agent-factory

Usine à projets logiciels pilotée par 3 groupes d'agents. Le dépôt est le bus de
communication : les issues/PRs et leurs labels sont le protocole. Aucun chat
entre agents.

## Les 3 groupes

| Groupe | Rôle | Qui | Produit |
|---|---|---|---|
| **A — Scouts** | trouver des idées (veille, tendances, besoin marché) | pi-team scout (flash local) | issue `idea/draft` + md dans `ideas/` |
| **B — Exécutants** | construire jusqu'à la PR | flotte cloud (Devin / Codex / Jules — gagnant de l'audition étape 3) | PR verte |
| **C — Vérificateurs** | arbitre froid + revue lecture-seule | CI `verify.sh` + agent reviewer | `verify/ok` / `verify/failed` + commentaires PR |

## Les 2 portes humaines (les seules)

1. **Porte 1** — approuver/rejeter une issue : `idea/approved` / `idea/rejected` (raison obligatoire si rejet).
2. **Porte 2** — merger la PR (on ne voit jamais une PR rouge : le CI les relance vers B, max 3 fois).

## Protocole des labels

- `idea/draft` → idée candidate postée par un scout, en attente de la porte 1.
- `idea/approved` → déclenche automatiquement B (API de la flotte).
- `idea/rejected` → retour au scout avec la raison, qui réitère.
- `ship/pr` → PR vérifiée par C, en attente de la porte 2.
- `verify/ok` / `verify/failed` → verdict de C ; `verify/failed` relance B (max 3).

## Structure

```
ideas/          # un md par idée : sources, besoin, problème, différenciateur
projects/       # un sous-dossier par projet exécuté (convention par langage)
verify.sh       # l'arbitre froid : lint + tests + build, exécuté par la CI
.github/workflows/verify.yml   # refuse toute PR rouge
```

## Convention projet

Chaque dossier de `projects/` doit être auto-vérifiable par `verify.sh` :
- **node** : `package.json` avec scripts `lint`, `test`, `build` (optionnels sauf `test`).
- **go** : `go.mod` → `go vet` + `go build` + `go test`.
- **python** : `pyproject.toml` → `ruff check` + `pytest`.
- Tout autre type : rejeté par l'arbitre (échec explicite).