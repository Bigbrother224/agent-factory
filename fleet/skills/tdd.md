# TDD — la loi de fer du codeur de l'usine

> Source : skill `test-driven-development` de obra/superpowers (MIT), condensé pour la flotte.

## LA LOI

```
AUCUN CODE DE PRODUCTION SANS TEST QUI ÉCHOUE D'ABORD
```

Tu as écrit du code avant le test ? Supprime-le. Recommence. Le garder « en référence »,
c'est rationaliser.

## Le cycle (rouge → vert → refactor)

1. **Écris UN test** qui exprime le comportement attendu de la tranche.
2. **Exécute-le et regarde-le échouer.** Si tu n'as pas vu le test échouer, tu ne sais pas
   s'il teste la bonne chose. Un test qui passe immédiatement ne prouve rien.
3. Écris le MINIMUM de code pour le faire passer.
4. Exécute l'arbitre (`./verify.sh <axe>/<projet>`) — vert ?
5. Refactore si besoin, l'arbitre re-vert entre chaque mouvement.

## Ce que les tests doivent être

- **Comportement, pas implémentation** : teste ce que le code FAIT pour l'utilisateur,
  jamais les détails internes (une assertion qui reflète le code, pas le besoin, n'est pas un test).
- **Un test qui ne peut pas échouer n'est pas un test.**
- Chaque règle métier de la spec a AU MOINS un test qui la protège.
- Cas limites et chemins d'erreur au même titre que le chemin nominal —
  entrée vide, bornes, encodage, erreur attendue.

## Quand l'arbitre est rouge

Ne JAMAIS modifier un test pour le faire passer. Le test décrit le besoin ;
si code et test divergent, c'est le code qui a tort (ou le test a mal capturé le besoin —
mais alors change-le en le JUSTIFIANT explicitement, pas pour « débloquer »).

## Exceptions

Aucune pour l'usine : la spec des projets est du vrai code livré au portfolio.
Config pure et fichiers de données : les couvrir par un test de chargement reste souvent possible.