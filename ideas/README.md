# Idées — groupe A (Scouts)

Une idée = un md ici + une issue GitHub labelée `idea/draft` **et** `axe/<axe>`.
Jamais d'idée sans sources datées, jamais d'idée hors axes.

## Le portfolio d'abord

L'usine existe pour construire un **portfolio GitHub** articulé sur 3 axes :

| Axe | Ce qu'il démontre |
|---|---|
| `ai-automation-and-systems` | ingénierie système réelle : agents, orchestration, pipelines, sandboxing, CI — pas des scripts autour d'une API LLM |
| `embedded-intelligence` | l'IA dans le matériel : embarqué, edge ML, capteurs, contraintes réelles (mémoire, latence, énergie) |
| `experimental-computing` | informatique expérimentale poussée jusqu'à la démonstration : paradigmes, calcul non conventionnel, art algorithmique |

**Barre portfolio** (une idée qui ne la passe pas n'est pas postée) :
1. **Axe clair** : le projet illustre évidemment l'un des 3 axes.
2. **Tendance sourcée** — liens datés (news, GitHub trending, HN, rapports).
3. **Besoin réel du marché** — preuve, pas intuition.
4. **Problème résolu** en une phrase.
5. **Différenciateur concret** vs solutions existantes (nommées).
6. **Shippable en MVP** : démontrable en < 3 commandes, démo lisible dans un README.

## Cycle de vie

`idea/draft` + `axe/*` → porte 1 humaine → `idea/approved` (la flotte construit
dans `projects/<axe>/<slug>/`) ou `idea/rejected` + raison (le scout réitère).

## Convention de nommage

`AAAA-MM-JJ-slug-court.md` — ex. `2026-11-03-tv-privacy-gateway.md`.

## Template

Voir `TEMPLATE.md`.