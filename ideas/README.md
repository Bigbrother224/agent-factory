# Idées — groupe A (Scouts)

Une idée = un fichier md ici + une issue GitHub labelée `idea/draft` (le md est
attaché ou collé dans l'issue). Jamais d'idée sans sources datées.

## Critères obligatoires (les 5)

1. **Tendance sourcée — pourquoi maintenant** — liens datés (news, GitHub trending, HN, rapports) qui prouvent que le sujet est chaud *maintenant*.
2. **Besoin réel du marché** — qui a ce problème, preuve (forum, issue GitHub récurrente, étude), pas une intuition.
3. **Problème résolu** — la formulation précise du problème que le projet résout, en une phrase.
4. **Différenciation vs solutions existantes** — 2-3 concurrents nommés + la faille que notre projet comble (différenciateur concret, pas "meilleure UX").
5. **Choix technique justifié** — langage argumenté en une phrase, périmètre MVP + hors-périmètre explicite, découpage cœur/interface, critères de succès mesurables.

Une idée ne validant pas les 5 → le draft n'est pas posté. Le pipeline est en 2 agents :
le **Scout** propose 3 candidats sourcés, le **Sélectionneur** (anti-hallucination) vérifie
les sources, note chaque candidat sur 5 axes et ne publie QUE le gagnant complet.
Un projet jouet ne gagne jamais par défaut — si les 3 candidats sont faibles, rien n'est publié.

## Cycle de vie

`idea/draft` (porte 1 humaine) → `idea/approved` (la flotte B construit) ou
`idea/rejected` + raison (le scout réitère en tenant compte de la raison).

## Convention de nommage

`AAAA-MM-DD-slug-court.md` — ex. `2026-11-02-observabilite-esp32.md`.

## Template

Voir `TEMPLATE.md`.