# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-02

## Vision (verrouillée le 2026-11-02 au matin)
- Usine non-stop : les agents trouvent des idées, les construisent, les vérifient ; l'humain n'a que 2 portes : approuver l'idée (`idea/approved`), merger la PR.
- 3 groupes : **A-Scouts** (idées, veille sourcée — pi-team scout flash local), **B-Exécutants** (flotte cloud, 1 VM/idée — gagnant de l'audition étape 3), **C-Vérificateurs** (CI `verify.sh` + agent reviewer lecture seule).
- Le dépôt GitHub est l'unique bus de communication : issues/PRs/labels. Pas de chat entre agents.
- Base existante seulement, pas d'orchestrateur maison : GitHub (tracker+CI) + une flotte managée (Devin/Codex/Jules…). OpenClaw exclu.

## Plan (5 étapes)
1. **[EN COURS] Dépôt + arbitre froid** : repo, `ideas/`, `verify.sh`, CI qui refuse les PR rouges, 6 labels, preuve par PR défaillante volontaire.
2. Groupe A : job planifié pi-team scout → 1 idée/jour sourcée → issue `idea/draft` + md dans `ideas/`.
3. Audition de B (go/no-go) : la MÊME issue donnée à Devin, Codex, Jules → tableau factuel (PR ? CI verte du 1er coup ? interventions humaines ? coût) → gagnant.
4. Brancher B et C : `idea/approved` → session flotte (API) ; PR → `verify.sh` + revue C ; `verify/failed` → relance B (max 3) ; `verify/ok` → `ship/pr` → porte 2.
5. Première idée réelle en boucle complète + métrique : interventions humaines ≤ 2 par cycle.

## Fait — ce tour (2026-11-02)
- Créé `~/Projects/factory` : `README.md` (architecture), `AGENTS.md`, `ideas/` (README critères 4 + `TEMPLATE.md`), `verify.sh` (arbitre lint+tests+build, node/go/python), projet échantillon `projects/example-hello` (node, zéro dépendance), workflow `.github/workflows/verify.yml`.
- Repo GitHub créé : `Bigbrother224/agent-factory` (privé), labels 6 créés, main protégée par la CI.
- Preuve : PR défaillante volontaire → CI la refuse (run rouge enregistré), PR fermée.

## Prochaine étape
Étape 2 — job planifié pi-team scout (veille web → md 4 critères → issue `idea/draft`) sur la première idée réelle : tester `pi-team spread scout "…" ~/Projects/factory` et brancher le cron.

## Décisions en attente
- Choix de la flotte B (étape 3 : audition Devin vs Codex vs Jules).
- Plafond budget par cycle idée→PR.