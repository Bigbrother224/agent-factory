# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-02

## Vision (restructurée 2026-11-03)
- **Le portfolio est le but** : un GitHub articulé sur 3 axes — **ai-automation-and-systems**, **embedded-intelligence**, **experimental-computing**. L'usine (3 groupes d'agents : scouts / Aider / CI+revue kimi) ne construit QUE des projets alignés sur un axe, shippables en MVP démontrable, avec README anglais niveau portfolio.
- Structure : `projects/<axe>/<projet>/`. Zéro payant : plan Ollama seul.

## Plan
1. ✅ Arbitre CI (verify.sh, PR rouge bloquée — prouvé).
2. ✅ Groupe A branché : digest sourcé HN+GitHub → idée → issue `idea/draft` + `axe/*` (cron 08:30 UTC). **Contrainte depuis 2026-11-03 : idées alignées sur les 3 axes uniquement.**
3. ✅ Flotte B = Aider (audition gagnée) ; workflow `fleet.yml` exige le label `axe/*` (sinon refuse + retire idea/approved).
4. ✅ Cycle complet branché : `idea/approved` → Aider → verify.sh (max 3) → PR → CI + revue kimi → `ship/pr` → porte 2. `!fix` en commentaire → corrections.
5. **[EN COURS] Remplir le portfolio** : 1 idée/axe minimum, puis 1 projet porté à terme de bout en bout.

## Fait — tour 7 (2026-11-03, restructuration portfolio)
- **Nettoyage** : PRs #7/#8/#9 fermées + branches supprimées ; issue #4 (MotDuJour, test) fermée « not planned » ; projects/example-hello supprimé ; worktrees parallèles retirés (c-verificateur arrêté : ses PRs #8/#9 ferment — le flux brouillait l'humain).
- **Restructuration axes** : `projects/{ai-automation-and-systems,embedded-intelligence,experimental-computing}/` avec README d'axe ; teleblock déplacé dans embedded-intelligence ; issue #2 relabelée `axe/embedded-intelligence` (labels idea/draft + ship/pr retirés).
- **Plomberie** : verify.sh découvre les projets 2 niveaux ; fleet.yml exige `axe/*` (sinon commente, refuse, retire idea/approved) et construit dans `projects/<axe>/<slug>/` ; fix-target ressort slug|branche|axe ; retour-humain utilise l'axe ; labels GitHub `axe/*` créés.
- **Barre portfolio** : CONTRAT-B règle 6 (README anglais : why/demo/how/tests, lisible en 2 min) ; TEMPLATE 6 sections avec Axe obligatoire ; scout-daily n'accepte un rapport QUE s'il porte un axe valide et pose le label `axe/*` sur l'issue.
- README racine = landing portfolio (axes en tête, projet teleblock listé).

## Fait — tour 8 (2026-11-03, 2e agent — réconciliation qualité sur la structure 3-axes, PR `fix/qualite-2`)
- **Réconciliation complète** de la couche qualité avec la restructuration portfolio du live agent (aucun écrasement — ses axes, son garde anti faux-succès !fix, sa garde budgétaire CI et son dashboard sont intacts et repris tels quels).
- **Barème qualité** (`fleet/BAREME.md`) joint au brief de la flotte B (fleet.yml) + CONTRAT-B règle 5 durcie : *vert ≠ bon*. Règle 3 (min 8 tests) **arbitrée** par verify.sh (node/go/python, comptage portable).
- **Revue renforcée** (`fleet/revue.py`) : barème intégré à l'invite kimi, notes 0-5 par dimension, nuance pour les PRs d'infra. Prouvée en production : 3 rounds sur ma PR précédente ont attrapé 3 vrais bugs.
- **Correcteur C3** (`correcteur.yml` + `correcteur-queue.py`, patrouille */20) adapté au format `[$AXE] $SLUG` : revue NO-GO → session Aider flash qui corrige les BLOQUANT/SÉRIEUX dans `projects/<axe>/<slug>/`, garde-fou périmètre (awk littéral), max 2 cycles → `revue/bloquee` → porte 2.
- **Skills** (demande humaine « équipe les agents ») : 2 skills pi pour le groupe A (`.agents/skills/factory-ideation`, `factory-selection` — craft sourcé + notation 0-5 + redlines) découvertes automatiquement par les pi headless du cron ; 3 skills langage pour la flotte B (`fleet/skills/{node,go,python}.md` — idiomes, structure, pièges) **injectés dans le brief Aider selon le langage déclaré par la spec**.
- **Pipeline idées 2 agents** : scout → 3 candidats DISTINCTS sourcés → Sélectionneur (anti-hallucination) note sur 5 axes, rédige le md du gagnant selon TEMPLATE enrichi (langage justifié, hors-périmètre, critères mesurables) ; **aucune idée publiée sans tri**, panne → issue ⚠️ sur le bus.
- fleet.yml : garde anti double-run (ship/pr déjà posé = skip), hygiène labels (idea/draft retiré, verify/failed nettoyé au vert), relances CI auto max 3. retour-humain : réouverture de PR fermée au !fix.
- Tout est **flash-only** (règle verrouillée) ; CI verte sur la branche.

## Prochaine étape
Valider le premier cycle complet sous la nouvelle structure : le cron scout de demain (08:30 UTC) doit produire une idée alignée axe (sinon échec explicite) → porte 1 → flotte → porte 2. En parallèle : brancher le dashboard (3e agent pi) sur les labels (source de vérité : idea/*, axe/*, ship/pr, verify/*).

## Règles humaines verrouillées (2026-11-03)
- **glm-5.3:cloud (flagship) INTERDIT partout** — uniquement `glm-5.3-flash:cloud` (abonnement). Verrouillé dans verify.sh : toute PR l'introduisant est refusée par la CI.
- Le dashboard est l'outil personnel de l'humain — pas d'axes d'amélioration imposés par l'usine.

## Décisions en attente
- Dashboard : architecture (lit GitHub en direct via gh, ou s'appuie sur un fichier d'état ?) — à coordonner avec le 3e agent pi.
- Public/privé : le repo doit-il devenir public pour être un vrai portfolio GitHub ? (branch protection gratuite au passage).
- Garder ou tuer le daemon rabbit-agent (99% CPU depuis le 13 oct.).
