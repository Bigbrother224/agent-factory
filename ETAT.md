# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-03

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

## Fait — tour 8 (2026-11-03, dashboard v2 — portfolio vivant + point de communication)
- **Dashboard v2** (`dashboard/`) adapté à la structure portfolio : 3 colonnes = les 3 axes, une carte vivante par projet (timeline du cycle Scout→Porte 1→Flotte B→CI+revue→Porte 2, badges idea/axe/verify/ship, CI, verdict kimi, stats, lien portfolio).
- **Point de communication humain↔agents intégré** : chaque carte affiche le fil des derniers commentaires (toi + l'usine) et un composeur — **mode `🔧 !fix`** publie un commentaire `!fix …` → workflow retour-humain → la flotte exécute ; **mode `💬 Note`** publie un commentaire simple (contexte lisible par toi et toute relecture). GitHub reste l'unique bus : zéro fichier d'état parallèle (décision « en attente » tranchée).
- Actions contextuelles : Porte 1, Porte 2 (merger — grisée si CI ≠ verte), rejeter avec raison (commentaire + `idea/rejected`), relancer la flotte (retirer/reposé `idea/approved`), poser l'axe (bloc avertissement pour idées sans axe — la flotte refuserait), fermer, relancer un run.
- Strip pipeline (portes 1/2, chantier, à réparer, shippés, date du dernier digest de veille) + tables runs agents et revues kimi + portfolio main par axe (flag README manquant — teleblock signalé, règle 6 pas encore satisfaite).
- Vérifié en réel : 3 issues / 7 PRs / 30 runs / digest / fil des 4 messages de #2 / kimi NO-GO PR #10 / garde-fous (note vide, axe invalide, action inconnue refusés).

## Fait — tour 8 (2026-11-03, l'âme de l'usine : l'équipe résidente)
- **Boucle chantier** (`scripts/chantier.sh` + launchd com.usine.chantier, 45 min) : planificateur → ROADMAP (8 tranches) → crew pi-team (flash/nemotron) → arbitre → PR `chantier/<projet>` → CI + revue kimi → auto-merge GitHub (vert+GO uniquement, PRs chantier/* seulement).
- **Prouvé en réel, 2 cycles** : T1 (inspection TCP/SNI + reset ciblé, internal/tcp) puis T2 (empreinte JA4, internal/ja4) — arbiter vert, PR #11 vivante, kimi joue tech lead (a refusé 2 NO-GO légitimes : violation de périmètre crew corrigée, intégration JA4 à finir, IPv6 à traiter — ses retours sont injectés au crew du cycle suivant).
- Réviseur durci : 32k tokens, diff plafonné 40 Ko (saturation détectée et corrigée).
- Fin de roadmap = label `ship/pr` sur l'issue → porte 2 humaine. Les portes humaines restent : valider les idées, merger le ship.

## Fait — tour 9 (2026-11-03, refonte design du dashboard via skill GitHub)
- **Skill adopté** : `h3nryprod01/design-taste` (synthèse des 3 skills de référence : Emil Kowalski design-engineering + impeccable + taste-skill) — cloné dans /tmp/design-taste, mode **Operate** (dashboard), dials variance 4 / motion 3 / densité 6.
- **dashboard.html refondu selon le skill** : famille unique Outfit + JetBrains Mono pour les données (plus Inter par défaut) · palette off-black + accent orange brûlé unique verrouillé, neutres teintés vers l'accent, sémantique ok/bad/warn/info standardisée · zéro carte imbriquée (colonnes d'axe plates, un seul niveau d'élévation) · bordure latérale du fil supprimée (AI-slop) · timeline du cycle reconstruite · composeur avec label visible au-dessus du champ (plus de placeholder-étiquette) · états d'interaction complets (:focus-visible, :active scale(.97), disabled, hover en media-query, squelettes de chargement) · motion 180 ms ease-out custom + prefers-reduced-motion · icônes Lucide inline (plus d'emojis) · copy auditée sans tiret cadratin · z-index documenté (toast=30).
- **Pre-flight du skill exécuté** : `scripts/preflight.mjs` → 0 violation dure après correctifs (outline:focus-visible ajouté, em dash supprimé) ; 3 warnings assumés (CDN fonts avec fallback hors ligne, micro-étiquettes sous la limite).
- Fonctionnalités v2 intactes : 3 axes, fil de communication + composeur !fix/note, portes 1/2, relance flotte, rejet motivé, pose d'axe, portfolio main avec flag README.

## Fait — tour 9 (2026-11-03, usine déportée : zéro dépendance au Mac)
- **Toute l'usine tourne sur VM GitHub gratuites** (sandbox neuve par run) : `chantier.yml` (résident, toutes les 2 h — roadmap→tranche→arbitre→PR, avec feedback kimi injecté et garde-fous : session stérile = échec, arbitre rouge = rien poussé), `scout.yml` (quotidien 08:30, `fleet/scout.py` autonome : digest réel + nemotron + issue), plus les existants fleet/revue/auto-merge.
- Mac déchargé : launchd chantier déchargé, cron scout retiré. `scripts/chantier.sh` et `scout-daily.sh` gardés comme fallback local.
- Preuves du tour : cycle résident sur VM (T3 partiellement appliqué, échec proprement absorbé — l'arbitre a refusé, rien de cassé), 3 bugs de workflow corrigés en itérant (sentinelle null vs '0', AXE non exporté, garde-fou mesuré après la session).
- **Quota (décision à trancher)** : repo privé = 2000 min Actions/mois ≈ 2-4 cycles/jour → cron réglé sur 2 h. Repo **public** = minutes standard illimitées + branch protection gratuite + portfolio visible des recruteurs.
- **Option serveur dédié (si besoin persistant) : Oracle Cloud Always Free** — 4 cœurs ARM Ampere + 24 Go RAM, gratuit à vie, carte demandée à l'inscription sans débit. Pour un daemon 24/7 ou un self-hosted runner.

## Prochaine étape
Toi : recharger http://localhost:7777 (design refondu), puis trancher la PR #10 (couche qualité, kimi NO-GO : lire la revue sur la carte ou `gh pr view 10 -R Bigbrother224/agent-factory`), et demain valider le premier cycle scout complet sous les 3 axes.

## Règles humaines verrouillées (2026-11-03)
- **glm-5.3:cloud (flagship) INTERDIT partout** — uniquement `glm-5.3-flash:cloud` (abonnement). Verrouillé dans verify.sh : toute PR l'introduisant est refusée par la CI.
- Le dashboard est l'outil personnel de l'humain — pas d'axes d'amélioration imposés par l'usine.

## Décisions en attente
- ~~Dashboard : architecture~~ **Tranché (tour 8)** : lit GitHub en direct via `gh` (zéro fichier d'état parallèle), le composeur passe par les commentaires d'issue (`!fix` = instructions exécutables).
- Public/privé : le repo doit-il devenir public pour être un vrai portfolio GitHub ? (branch protection gratuite au passage).
- Garder ou tuer le daemon rabbit-agent (99% CPU depuis le 13 oct.).
