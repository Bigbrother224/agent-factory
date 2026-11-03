# Debugging systématique — cause racine avant tout correctif

> Source : skill `systematic-debugging` de obra/superpowers (MIT), condensé pour la flotte.

## LA LOI

```
AUCUN CORRECTIF SANS INVESTIGATION DE CAUSE RACINE
```

Un correctif de symptôme est un échec. Si tu n'as pas fini la phase 1, tu ne proposes PAS de fix.

## Phase 1 — COMPRENDRE (obligatoire)

1. **Reproduire** : quelles entrées/actions exactes déclenchent le comportement ?
   Un bug non reproductible ne peut pas être vérifié corrigé.
2. **Lire le message d'erreur EN ENTIER** — stack trace comprise. Le nom du fichier
   et la ligne y sont presque toujours.
3. **Isoler** : le plus petit cas qui reproduit. Commente, coupe, bissect jusqu'au
   minimum qui échoue.
4. **Hypothèses écrites** : liste 2-3 causes possibles, avec pour chacune CE QUE TU
   OBSERVES qui l'étaye (pas ce que tu supposes).

## Phase 2 — INSTRUMENTER (avant de fixer)

- Ajoute un affichage/debug temporaire qui tranche ENTRE les hypothèses.
- Un instrument = une question précise, avec une réponse observable.
- La prédiction échoue ? Ton hypothèse était fausse : tant mieux, tu viens d'apprendre
  quelque chose de vrai. Passe à l'hypothèse suivante — n'insiste pas.

## Phase 3 — CORRIGER

- Fix la cause racine, pas le symptôme. Demande-toi : « ce fix empêcherait-il la
  même classe de bug ailleurs ? »
- Si le correctif te semble suspect ou trop large : régresse d'un cran, re-instrumente.
- Supprime le code de debug instrumenté.

## Phase 4 — VÉRIFIER

- Reproduis le cas minimal : il passe. Reproduis un cas voisin : rien n'a cassé.
- `./verify.sh <axe>/<projet>` vert, sortie réelle.
- Un test de régression verrouille le bug corrigé — sinon il reviendra.

## Anti-patterns interdits

- « J'essaie un truc » sans hypothèse (shotgun debugging).
- Réessayer le même fix une deuxième fois en espérant.
- Blâmer le compilateur, la std lib, le runner — c'est presque toujours ton code.
- Fixer en cascade 3 symptômes au lieu d'une cause.