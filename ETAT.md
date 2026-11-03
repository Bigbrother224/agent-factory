# État — agent-factory (usine à projets, 3 groupes d'agents)

Dernière mise à jour : 2026-11-03 (13:54)

## Fait — tour 13 (2026-11-03, 13:54 — scène E2E teleblock déboguée et poussée + glm-5.3 résiduel éradiqué)
- **La scène E2E (règle 7) est VERTE et POUSSÉE** sur `chantier/teleblock` (clos le point du tour 12 « à déboguer ») : 3 bugs trouvés — (1) deadlock du test (étape 9 + cleanup consommaient 2× le canal `done` → motif canal fermé) ; (2) `-observe-state` activait par défaut le learning 168h qui suspend tout blocage → drapeau passé vide ; (3) **bug produit réel** : `ja4.Parse` rejetait tout ClientHello TLS 1.3 réel — `supported_versions` porte sa longueur sur UN octet (RFC 8446 §4.2.1), le parseur en attendait deux — corrigé + tests remis au vrai format filaire.
- **Retours kimi PR #11 actionnables traités** : fichier vide parasite `embedded-intelligence/teleblock` à la racine supprimé (artefact du batch résident) ; tableau ROADMAP renuméroté (T4 dashboard, T5 README, T6 Docker, T7 Pi, T8 validation — conforme à l'historique réel) ; ETAT.md accidentellement embarqué dans un commit de branche (git add -A) → retiré du diff par amend. Restent au prochain cycle : IPv6 SO_ORIGINAL_DST (SÉRIEUX) et validation RedirectPort (MINEUR).
- **Head bed93db** : arbitre vert, scène passée 3× en local (~1,1 s) ; CI + revue kimi relancées dessus (auto-merge si vert + GO).
- **glm-5.3 résiduel (signalement humain)** : audit complet du dépôt — workflows tous en flash ✓ ; README racine corrigé (« Aider (glm-5.3) » → glm-5.3-flash, poussé ce tour) ; seules mentions restantes légitimes : `verify.sh` (la garde elle-même) et `audition/RESULTATS.md` (archive historique, note explicitement l'interdiction depuis le 2026-11-03).

## Fait — tour 12 (2026-11-03, dashboard : la revue NO-GO enfin lisible)
Problème remonté par l'humain : impossible d'ouvrir le NO-GO de la PR #11, infos incomplètes. Cause racine × 2 :
- **La revue n'apparaissait même pas sur la carte TeleBlock** : le client cherchait la PR par slug exact (`teleblock-pare-feu`, 3 premiers mots du titre) alors que la branche est `chantier/teleblock` → CI, verdict et roadmap invisibles sur la carte. Correspondance tolérante ajoutée (même logique que la recherche de roadmap côté serveur).
- **Aucun moyen de lire la revue** : seul un extrait tronqué de 100 car. était affiché. Maintenant : **modale revue complète** (verdict + compte de gravités, 7 problèmes avec pastilles BLOQUANT/SÉRIEUX/MINEUR, 6 notes qualité /5 avec barres, bilan, lien direct vers le commentaire GitHub) — ouverte par la puce « revue NO-GO », la ligne résumé de la carte (« Revue kimi 2 h : NO-GO · 1 bloquant, 4 sérieux… ») ou le bouton Ouvrir du panneau revues. Escape/clic hors modale pour fermer.
- Bonus : stat « digest de veille » qui affichait toujours « aucun » (s.lastDigest jamais rempli) → date réelle du dernier `ideas/YYYY-MM-DD-*.md` (2026-11-02).
- Vérifié : 14/14 tests de rendu en réel (DOM stub + état live du serveur) ; skill design respecté (focus-visible, pas de tiret cadratin visible, échelle z documentée toast=30 / modale=20).

## Fait — tour 11 (2026-11-03, reprise après fermeture des 3 fenêtres agents)
- **Aucun impact** : l'usine tourne à 100% sur VM GitHub depuis le tour 9 — les fenêtres Mac fermées n'ont rien interrompu.
- **Dashboard relancé** (`./dashboard/start.sh`, http://localhost:7777 → 200) — c'était la seule dépendance locale tombée.
- **État du bus au 13:29** : PR #11 (chantier/teleblock) vivante, MERGEABLE ; dernière revue kimi = **NO-GO** (retours injectés au prochain cycle résident, qui tourne en batch toutes les 6 h) ; 2 runs verify/revue-c `action_required` (12:59) seront auto-déverrouillés par le cron auto-merge (≤20 min). PRs atelier #12 mergée, #10 fermée (contenus coulés sur main au tour 10).

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

## Fait — tour 9 (2026-11-03, 2e agent — l'usine équipe ses agents, PR `fix/skills` atelier)
- Skills externes curés depuis GitHub : `frontend-design` d'**anthropics/skills** (Apache 2.0, étalon anti-vibecoding) + TDD/systematic-debugging/verification-before-completion d'**obra/superpowers** (MIT). Canoniques dans `.agents/skills/`, condensés flash-friendly dans `fleet/skills/` pour les briefs Aider.
- Injection dans le chantier résident : session code en `--message-file` avec discipline non négociable + étalon design si face visible ; planificateur prévoit une tranche design pour les projets UI.
- Dashboard : refonte de l'agent résident gardée ; bugfix porté (PRs `chantier/` invisibles sur les fiches).
- **Règle de gouvernance (humain, ce tour)** : GitHub = la PRODUCTION uniquement (idées des 3 axes, projets, notifications). Le développement de l'usine elle-même = **atelier** (label `atelier`), géré hors production : PRs atelier auto-mergées (CI verte + GO), dashboard les sépare.

## Fait — tour 10 (2026-11-03, 2e agent — SÉPARATION ATELIER / PRODUCTION, dépôt direct sur main)
**Règle de gouvernance posée par l'humain : GitHub = la PRODUCTION uniquement** (idées des 3 axes, projets, leurs notifications). Le développement de l'usine elle-même = **atelier**, géré hors bus :
- Label **`atelier`** créé : toute PR/issue d'outillage le porte ; **auto-merge** étendu (CI verte + revue GO = squash, comme les tranches chantier) — l'outillage ne séjourne plus jamais sur le bus. Dépôts directs sur main pour l'atelier : validés localement + CI après-coup.
- **Dashboard** : les éléments atelier sont exclus des vues production (chaînes, ligne de process) ; une ligne discrète en pied de page les référence.
- Coulée ce tour (PRs #10/#12 retirées du bus, contenus atterrés directement) : `fleet/BAREME.md` joint au brief de la flotte + craft node/go/python injecté selon le langage de la spec ; garde anti double-run + relances CI auto (max 3, garde-fou périmètre) + hygiène de labels dans fleet.yml ; réouverture de PR fermée au !fix ; règle 3 (min 8 tests) arbitrée par verify.sh ; revue kimi : barème intégré, notes par dimension, relance auto 25k sur réponse vide, nuance PRs d'infra ; **correcteur C3** (revue NO-GO d'une PR projet → session Aider correctrice, max 2 cycles → revue/bloquee → porte 2).

## Fait — tour 12 (2026-11-03, 14:40 — cadence non-stop + E2E, verrou flagship reconfirmé)
- **RÈGLE RECONFIRMÉE** : glm-5.3:cloud (flagship) INTERDIT dans l'usine — flash partout. (Le quota brûlé aujourd'hui venait de la session pi de l'humain elle-même, lancée en flagship : relancer en `pi -m glm-5.3-flash:cloud`.)
- **Repo public** + chantier **horaire** (minutes illimitées) + batch 4 tranches/run, budget 2400 s — ~6× la cadence d'hier.
- **E2E partout** : verify.sh exécute `tests/e2e.sh` si présent ; le planificateur doit planifier une tranche E2E ; kimi refuse GO final sans scène E2E réelle (barème règle 7).
- **Scène E2E teleblock écrite** (worktree /tmp/factory-teleblock, branche chantier/teleblock) : binaire réel démarré, DNS sinkhole/passthrough, RST sur SNI bloqué, inspection sans blocage, dashboard. **NON poussée** : le run local a été interrompu (à déboguer : probablement la route du dashboard T4 ou l'attente de disponibilité) — à finir avant de pousser sur la branche.
- Récupéré au passage : le travail dashboard non commité des 3 fenêtres fermées (modale revue kimi) — mergé sur main.

## Prochaine étape
Toi : rien de manuel — le cycle est automatisé. La revue kimi sur le head bed93db arrive dans ~5 min : GO + CI verte → auto-merge de la PR #11 ; NO-GO → le résident (chantier horaire) reçoit les retours et corrige au cycle suivant (IPv6 + RedirectPort sont déjà identifiés). Tu peux suivre sur le dashboard (http://localhost:7777, carte TeleBlock). Toujours en attente : trancher public/privé du repo (portfolio visible des recruteurs).

## Règles humaines verrouillées (2026-11-03)
- **glm-5.3:cloud (flagship) INTERDIT partout** — uniquement `glm-5.3-flash:cloud` (abonnement). Verrouillé dans verify.sh : toute PR l'introduisant est refusée par la CI.
- Le dashboard est l'outil personnel de l'humain — pas d'axes d'amélioration imposés par l'usine.

## Diagnostic « autonomie totale » (2026-11-03, 13:45 — réponse à la frustration humaine)
L'objectif « développer non-stop jusqu'au résultat parfait sans harnessing » est freiné par 3 murs mesurés :
1. **Cadence** : 7 runs résident aujourd'hui, 3 réussis, cadence 6h (quota privé 2000 min/mois) → 3-4 itérations efficaces/jour.
2. **Plafond codeur** : glm-5.3-flash nécessite 2-3 cycles NO-GO pour converger (kimi plus exigeant que le codeur n'est bon) ; règle flagship-interdit verrouille ce plafond.
3. **Mur physique** : « shippé » teleblock = validation sur Samsung/LG réelles (T7-T8) — aucune VM ne peut le faire ; le dernier kilomètre est hardware humain.
Décisions proposées : (a) repo public → minutes illimitées, 10-20× ; (b) flagship autorisé pour le codeur résident sur tranches dures uniquement ; (c) redéfinir ship = image Pi flashable + guide de validation (l'humain installe, il ne harness pas).

## Décisions en attente
- ~~Dashboard : architecture~~ **Tranché (tour 8)** : lit GitHub en direct via `gh` (zéro fichier d'état parallèle), le composeur passe par les commentaires d'issue (`!fix` = instructions exécutables).
- Public/privé : le repo doit-il devenir public pour être un vrai portfolio GitHub ? (branch protection gratuite au passage).
- Garder ou tuer le daemon rabbit-agent (99% CPU depuis le 13 oct.).
