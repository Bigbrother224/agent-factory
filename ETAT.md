# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-02

## Vision (verrouillée le 2026-11-02 au matin)
- Usine non-stop : les agents trouvent des idées, les construisent, les vérifient ; l'humain n'a que 2 portes : approuver l'idée (`idea/approved`), merger la PR.
- 3 groupes : **A-Scouts** (idées, veille sourcée), **B-Exécutants** (flotte cloud, 1 VM/idée — gagnant de l'audition étape 3), **C-Vérificateurs** (CI `verify.sh` + agent reviewer lecture seule).
- Le dépôt GitHub est l'unique bus de communication : issues/PRs/labels. Pas de chat entre agents.
- Base existante seulement, pas d'orchestrateur maison : GitHub (tracker+CI) + une flotte managée (Devin/Codex/Jules…). OpenClaw exclu.

## Plan (5 étapes)
1. **[VALIDÉE] Dépôt + arbitre froid** : repo privé `Bigbrother224/agent-factory`, `ideas/`, `verify.sh`, CI qui refuse les PR rouges, 6 labels. Preuve : PR #1 défaillante → run 41451930498 = failure.
2. **[VALIDÉE] Groupe A branché** : `scripts/veille.sh` (digest live HN+GitHub sourcé) + scout pi-team lecture-seule → `ideas/AAAA-MM-JJ-slug.md` → issue `idea/draft`. Cron 08:30 UTC (`scripts/scout-daily.sh`, idempotent, off-peak −50%).
3. **[VALIDÉE] Audition de B** : contrainte « zéro payant, plan Ollama seul » (décision humaine 2026-11-02). Aider ✅ (PR #3, CI verte run 41459426998, 2 relances) · OpenHands ❌ (0 livré en 4417 s / 60 itérations max) · SWE-agent installé non évalué. Devin/Codex/Jules exclus (payants). **Verdict : flotte B = Aider** (config verrouillée dans `audition/RESULTATS.md`).
4. **[PROCHAINE] Brancher B et C** : workflow GitHub `idea/approved` → session Aider (commande dans `audition/RESULTATS.md`) ; PR → `verify.sh` + revue C ; `verify/failed` → relance (max 3) ; `verify/ok` → `ship/pr` → porte 2.
5. Première idée réelle en boucle complète + métrique : interventions humaines ≤ 2 par cycle.

## Fait — étape 1 (2026-11-02)
- `~/Projects/factory` : `README.md`, `AGENTS.md`, `ideas/` (critères 4 + `TEMPLATE.md`), `verify.sh` (lint+tests+build, node/go/python), `projects/example-hello` (node, zéro dépendance), workflow `verify.yml`.
- Arbiter vert sur main (run 41451851178) ; PR défaillante volontaire refusée (run 41451930498). Étape 1 validée.

## Fait — étape 2 (2026-11-02, ce tour)
- `scripts/veille.sh` : digest du jour réel et sourcé (HN front page API Algolia + GitHub search API) — le scout lit ce digest, aucune tendance hallucinée.
- Premier scout exécuté (pi-team scout, nemotron-3-super, sur le digest du jour) : idée **TeleBlock**, pare-feu de confidentialité smart TV, ancrée sur le fil HN « Which TV do you recommend for privacy? » (489 pts / 612 coms).
- Md : `ideas/2026-11-02-teleblock-tv-privacy-firewall.md` ; **issue #2 créée avec label `idea/draft`** → en attente de la porte 1 humaine.
- `scripts/scout-daily.sh` : pipeline complet journalier (digest → scout → md → issue), idempotent (1 idée/jour max), garde-fous (titre/label manquants = échec explicite). Testé : sort propre si l'idée du jour existe.
- Cron installé : `30 8 * * *` UTC, log dans `.veille/cron.log`.

## Fait — tours 3-4 (2026-11-02)
- Issue #2 enrichie (différenciation Pi-hole explicite + installation 3 chemins) et **approuvée** (porte 1). Brief d'audition identique : `audition/BRIEF-B.md`.
- Décision humaine : **zéro payant** → candidats libres pilotés par l'API Ollama (glm-5.3:cloud). Devin/Codex/Jules exclus.
- Audition réelle : **Aider VAINQUEUR** — PR #3 livrée, CI GitHub verte (run 41459426998), 2 relances opérateur ; hallucination de vérif détectée en session 1 (l'arbitre seul dit la vérité). **OpenHands : échec** (0 fichier en 4417 s, 60/60 itérations) ; worktree nettoyé. SWE-agent installé, non évalué (budget temps).
- Verdict go/no-go : **flotte B = Aider**, config verrouillée dans `audition/RESULTATS.md`.

## Prochaine étape
Étape 4 : brancher le cycle complet — GitHub Action `idea/approved` → session Aider → PR → CI + relances auto (max 3) → `ship/pr` ; en parallèle, ta porte 2 sur la PR #3 (merger ou pas).

## Décisions en attente
- Protection de branche (bloquée : GitHub Pro requis sur repo privé). Options : passer `agent-factory` en public (protection gratuite) ou prendre Pro.
- Choix de la flotte B (étape 3 : audition Devin vs Codex vs Jules).
- Plafond budget par cycle idée→PR.