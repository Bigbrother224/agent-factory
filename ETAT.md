# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-03

## Vision (verrouillée le 2026-11-02 au matin)
- Usine non-stop : les agents trouvent des idées, les construisent, les vérifient ; l'humain n'a que 2 portes : approuver l'idée (`idea/approved`), merger la PR.
- 3 groupes : **A-Scouts** (idées, veille sourcée), **B-Exécutants** (flotte cloud, 1 VM/idée — gagnant de l'audition étape 3), **C-Vérificateurs** (CI `verify.sh` + agent reviewer lecture seule).
- Le dépôt GitHub est l'unique bus de communication : issues/PRs/labels. Pas de chat entre agents.
- Base existante seulement, pas d'orchestrateur maison : GitHub (tracker+CI) + une flotte managée (Devin/Codex/Jules…). OpenClaw exclu.

## Plan (5 étapes)
1. **[VALIDÉE] Dépôt + arbitre froid** : repo privé `Bigbrother224/agent-factory`, `ideas/`, `verify.sh`, CI qui refuse les PR rouges, 6 labels. Preuve : PR #1 défaillante → run 41451930498 = failure.
2. **[VALIDÉE] Groupe A branché** : `scripts/veille.sh` (digest live HN+GitHub sourcé) + scout pi-team lecture-seule → `ideas/AAAA-MM-JJ-slug.md` → issue `idea/draft`. Cron 08:30 UTC (`scripts/scout-daily.sh`, idempotent, off-peak −50%).
3. **[VALIDÉE] Audition de B** : contrainte « zéro payant, plan Ollama seul » (décision humaine 2026-11-02). Aider ✅ (PR #3, CI verte run 41459426998, 2 relances) · OpenHands ❌ (0 livré en 4417 s / 60 itérations max) · SWE-agent installé non évalué. Devin/Codex/Jules exclus (payants). **Verdict : flotte B = Aider** (config verrouillée dans `audition/RESULTATS.md`).
4. **[BRANCHÉE — validée sur l'issue de test #4] Brancher B et C** : workflow `fleet.yml` — `idea/approved` → session Aider auto dans le runner GitHub (brief = `fleet/CONTRAT-B.md` + spec issue, modèle `glm-5.3:cloud` via secret `OLLAMA_API_KEY`) → arbitre entre 3 tentatives max → branche `fleet/<slug>` → PR → attente CI → `verify/ok` sur la PR + `ship/pr` sur l'issue → porte 2 humaine. Preuve : issue #4 → run 41514787497 → PR #5, CI verte (run 41516914883), livrable écrit 100% sans intervention.
   Réglages posés : secret `OLLAMA_API_KEY` dans le repo · `can_approve_pull_request_reviews=true` (Actions peut créer des PRs) · push `--force` sur branches fleet (idempotence).
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
- **Porte 2 passée sur #2** : PR #3 (TeleBlock MVP, Aider) mergée → `projects/teleblock/` sur main.
- Issue #2 enrichie (différenciation Pi-hole explicite + installation 3 chemins) et **approuvée** (porte 1). Brief d'audition identique : `audition/BRIEF-B.md`.
- Décision humaine : **zéro payant** → candidats libres pilotés par l'API Ollama (glm-5.3:cloud). Devin/Codex/Jules exclus.
- Audition réelle : **Aider VAINQUEUR** — PR #3 livrée, CI GitHub verte (run 41459426998), 2 relances opérateur ; hallucination de vérif détectée en session 1 (l'arbitre seul dit la vérité). **OpenHands : échec** (0 fichier en 4417 s, 60/60 itérations) ; worktree nettoyé. SWE-agent installé, non évalué (budget temps).
- Verdict go/no-go : **flotte B = Aider**, config verrouillée dans `audition/RESULTATS.md`.

## Fait — tour 5 (étape 4, ce tour)
- Workflow `.github/workflows/fleet.yml` + `fleet/CONTRAT-B.md` + `fleet/slugify.py` : cycle complet branché.
- Test end-to-end réel sur l'issue #4 (MotDuJour) : session Aider 100% autonome dans le runner (brief → code → verify.sh VERT → branche poussée). Deux réglages d'infra posés en route : permission Actions de créer des PRs ; `--force` sur les pushes fleet (idempotence des reruns).
- PR #5 ouverte, CI verte (run 41516914883), labels `verify/ok` + `ship/pr` posés — **en attente de ta porte 2**.
- Resté à faire dans ce circuit : la revue lecture-seule du groupe C (l'humain relit la PR au merge) — pour l'instant C = CI seul.

## Fait — tour 6 (2026-11-03, 2e agent — rôle C-Vérificateur, PR `fix/c-verificateur`)
- Audit parallèle lecture-seule du pipeline (zéro collision avec l'agent live, travail en worktree) — 6 failles identifiées, dont :
  1. `verify/failed` cul-de-sac : CI rouge sur la PR → label posé, RIEN ne relançait B.
  2. Règle 3 du contrat (min 8 tests) déclarative, non arbitrée par la CI.
  3. `!fix` sur un projet à PR fermée : commentaire posté sur la PR fermée au lieu de la rouvrir.
  4. Retrait/remise de `idea/approved` = double session Aider possible (double PR).
- Correctifs livrés dans la PR (porte 2 humaine) :
  - `fleet.yml` : **boucle de relance CI** — CI rouge malgré arbitre local vert → log d'échec extrait → session Aider correctrice → push → ré-arbitrage, max 3 cycles ; `verify/failed` ne tombe qu'après épuisement réel. + garde anti double-run (skip si `ship/pr` déjà posé) + hygiène (`idea/draft` retiré à l'approbation).
  - `verify.sh` : **comptage des tests** (node/go/python) — refus explicite si < 8 tests (contrat B règle 3 désormais arbitré).
  - `retour-humain.yml` : rouvre la PR fermée si besoin ; retire le `verify/failed` stale quand le retour passe au vert.
  - `example-hello` : 2 → 8 tests réels (l'arbitre reste sans exception).
- Tests locaux : vert sur les 2 projets de main (13 + 27 tests), négatif prouvé (projet à 2 tests → REFUSÉ, exit 1).

## Fait — tour 7 (2026-11-03, 2e agent — couche qualité, PR `fix/qualite` empilée sur `fix/c-verificateur`)
- **🚨 URGENT (action humaine)** : abonnement Ollama en impayé — l'API répond `403 payment past due` → flotte B et revue kimi échoueront sur tous les prochains cycles. À régler : https://ollama.com/settings/billing. (Le !fix motdujour du 08:31 a pu échouer pour ça.)
- **Panne découverte** : le cron scout du 2026-11-03 08:30 a ÉCHOUÉ (auth pi absente du cron : `PI_PROVIDER`/`OLLAMA_API_KEY` sont dans .zshrc). Fixé dans scout-daily.sh (résolution trousseau + PI_PROVIDER explicite). De plus, toute panne du pipeline poste maintenant une issue « ⚠️ » sur le bus — plus de panne silencieuse.
- **Qualité des idées (groupe A, 2 agents)** : le scout propose désormais 3 candidats DISTINCTS ; un **Sélectionneur** (pi-team raisonneur, anti-hallucination) vérifie les sources, note sur 5 axes (ancrage, douleur, différenciation, faisabilité std-lib, valeur d'usage) et rédige le md COMPLET du gagnant. Aucune idée publiée sans tri. TEMPLATE enrichi : pourquoi maintenant, choix du langage JUSTIFIÉ, périmètre/hors-périmètre, critères de succès mesurables.
- **Qualité du code (groupes B/C)** : nouveau `fleet/BAREME.md` (efficacité, intelligence, idiomatisme, choix de langage, design, tests — anti-patterns = refus), joint au brief de B (fleet.yml) et à l'invite du réviseur ; CONTRAT-B règle 5 durcie (vert ≠ bon).
- **Revue renforcée** (`fleet/revue.py`) : le barème est intégré à l'invite, notes 0-5 par dimension, verdict GO/NO-GO inchangé (parsable), nuance pour les PRs d'infra (juge l'orchestration, pas le périmètre).
- **Nouvel agent — Correcteur (C3)** : workflow `correcteur.yml` (patrouille */20 min) + `fleet/correcteur-queue.py` : dernière revue NO-GO → session Aider qui corrige les BLOQUANT/SÉRIEUX sur la branche de la PR, arbitre vert, push → la revue repart (synchronize). Max 2 cycles, puis `revue/bloquee` → porte 2 humaine. Une revue en PANNE ne consomme jamais un cycle. File testée en lecture sur le repo réel (zéro faux positif).
- Non testable localement tant que l'impayé Ollama n'est pas réglé (revue.py : 403 vérifié en direct).

## Prochaine étape
Porte 2 : trancher la PR #5 (`gh pr merge 5 -R Bigbrother224/agent-factory` ou rejet). Ensuite, l'usine tourne seule : le cron scout (08:30 UTC) poste une idée/jour, tu approuves (`idea/approved`), la flotte construit, la CI arbitre, tu merges.

## Décisions en attente
- Protection de branche (bloquée : GitHub Pro requis sur repo privé). Options : passer `agent-factory` en public (protection gratuite) ou prendre Pro.
- Choix de la flotte B (étape 3 : audition Devin vs Codex vs Jules).
- Plafond budget par cycle idée→PR.