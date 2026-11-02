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
3. **[BLOQUÉE — accès] Audition de B** : brief identique prêt (`audition/BRIEF-B.md`). Codex : compte ChatGPT refuse tous les modèles (dernière session réelle : juin 2026 ; CLI à jour 0.166.0 ; diagnostic dans `audition/RESULTATS.md`). Devin : pas de clé Cognition. Jules : pas d'accès. Reprise documentée.
4. Brancher B et C : `idea/approved` → session flotte (API) ; PR → `verify.sh` + revue C ; `verify/failed` → relance B (max 3) ; `verify/ok` → `ship/pr` → porte 2.
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

## Fait — tour 3 (2026-11-02, ce tour)
- Issue #2 **enrichie** après ton retour : §4 tableau de différenciation explicite vs Pi-hole/NextDNS (blocage par connexion/domaine, apprentissage, zéro-config, local-only), §5 installation en 3 chemins (image Pi flashable, migration Pi-hole en 1 ligne, binaire Go statique) — body de l'issue mis à jour.
- **Porte 1 exécutée** : `idea/approved` posé sur l'issue #2.
- `audition/BRIEF-B.md` : brief d'audition identique pour les 3 candidats (MVP Go std-lib : résolveur DNS + profils par marque + parseur ClientHello/SNI + 8 tests min., `./verify.sh teleblock` non négociable).
- **Audition tentée en réel — 3 blocages d'accès** : Codex (compte ChatGPT refuse tous les modèles — abonnement ? à re-authentifier), Devin (pas de clé API Cognition), Jules (pas installé). CLI Codex mise à jour 0.146.0 → 0.166.0 au passage.
- Résultats + diagnostic + procédure de reprise exacte : `audition/RESULTATS.md`.

## Prochaine étape
Toi : débloquer UN accès (1. `codex logout && codex login` + vérifier abonnement ChatGPT/Codex · 2. clé api.devin.ai · 3. accès Jules). Ensuite : relancer l'audition selon `audition/RESULTATS.md` (commandes prêtes) et remplir le tableau factuel.

## Décisions en attente
- Protection de branche (bloquée : GitHub Pro requis sur repo privé). Options : passer `agent-factory` en public (protection gratuite) ou prendre Pro.
- Choix de la flotte B (étape 3 : audition Devin vs Codex vs Jules).
- Plafond budget par cycle idée→PR.