# Barème qualité — l'étalon de l'usine

Ce barème s'applique à tout code livré par la flotte B et à toute correction (retour humain, correcteur).
Il est joint au brief du groupe B et à l'invite du réviseur : les deux bouts de la chaîne parlent du même langage.
Vert ≠ bon. `verify.sh` vert est un plancher, pas une garantie. Ce barème est le plafond attendu.

## 1. Efficacité
- Complexité adaptée au volume réel : pas de O(n²) là où O(n) est trivial, pas de re-scan là où un index/dict suffit.
- Structures de données choisies pour l'usage : map quand on cherche, set quand on dédoublonne, slice/itérateur quand on traverse.
- Pas d'allocations inutiles en boucle chaude ; le travail fait est proportionnel au problème.

## 2. Intelligence du code
- La bonne abstraction au bon endroit : ni copier-coller déguisé, ni usine à gaz pour un seul usage.
- Cas limites traités par design (entrées vides, bornes, encodage, erreurs), pas par accident.
- Zéro réinvention de la std lib : si le langage le fait nativement, on l'utilise.
- Le code révèle l'intention : une lecture suffit à comprendre *pourquoi*, pas seulement *comment*.

## 3. Idiomatisme
- Le code doit ressembler à son langage : `errors.Is` en Go, exceptions précises en Python, ESM propre en Node — pas de « Python écrit en C » ni de C écrit en JavaScript.
- Nommage idiomatique (`camelCase`/`snake_case` selon le langage), fichiers organisés selon les conventions du langage (`internal/`, `test/`, `pyproject`).
- Erreurs : le mécanisme du langage, pas de codes de retour improvisés ni de silences suspects.

## 4. Choix du langage
- Justifié par le problème, pas par l'habitude : CLI robuste à distribuer → Go ; outil texte/data → Node ou Python selon l'écosystème cible ; perf ou typage fort → Go.
- Std lib d'abord (contrainte de l'usine) : le choix doit rester naturel en std lib seule ; si la spec exige un langage inadapté, B le signale dans son rapport au lieu de subir.

## 5. Design
- Découpage net : cœur métier séparé de l'interface (CLI, tests, futur HTTP).
- Surface publique minimale : on n'exporte que ce qui sert ; API stable et nommée avec soin.
- Un module = une responsabilité ; les dépendances pointent dans un seul sens.

## 6. Tests
- Comportement d'abord, cas limites et chemins d'erreur inclus — pas que le happy path.
- Un test qui ne peut pas échouer n'est pas un test.
- Chaque règle métier annoncée dans la spec a au moins un test qui la protège.

## Anti-patterns = refus immédiat
- Réinventer la std lib, code mort, commentaires narratifs qui paraphrasent le code.
- Over-engineering : abstraction, config ou paramètre jamais utilisé.
- Sous-engineering : tout dans un fichier unique passé 150 lignes de logique, fonctions de 50 lignes à imbrication profonde.
- Tests taillés sur mesure pour l'implémentation (assertions qui reflètent le code, pas le besoin).