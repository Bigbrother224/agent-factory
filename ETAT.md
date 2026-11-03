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

## Prochaine étape
Valider le premier cycle complet sous la nouvelle structure : le cron scout de demain (08:30 UTC) doit produire une idée alignée axe (sinon échec explicite) → porte 1 → flotte → porte 2. En parallèle : brancher le dashboard (3e agent pi) sur les labels (source de vérité : idea/*, axe/*, ship/pr, verify/*).

## Règles humaines verrouillées (2026-11-03)
- **glm-5.3:cloud (flagship) INTERDIT partout** — uniquement `glm-5.3-flash:cloud` (abonnement). Verrouillé dans verify.sh : toute PR l'introduisant est refusée par la CI.
- Le dashboard est l'outil personnel de l'humain — pas d'axes d'amélioration imposés par l'usine.

## Décisions en attente
- Dashboard : architecture (lit GitHub en direct via gh, ou s'appuie sur un fichier d'état ?) — à coordonner avec le 3e agent pi.
- Public/privé : le repo doit-il devenir public pour être un vrai portfolio GitHub ? (branch protection gratuite au passage).
- Garder ou tuer le daemon rabbit-agent (99% CPU depuis le 13 oct.).
