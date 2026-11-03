# Contrat B — règles non négociables pour toute session de la flotte

1. **Périmètre** : tu crées et modifies UNIQUEMENT des fichiers dans `projects/<projet>/`. Aucune modification de `verify.sh`, `.github/`, `README.md`, `ideas/` ou de tout autre fichier du dépôt.
2. **Auto-vérifiable** : le projet doit passer `./verify.sh <projet>` selon ses conventions :
   - **node** : `package.json` avec scripts `lint` (min. `node --check` par fichier) et `test` (`node --test`). Zéro dépendance externe (std lib uniquement).
   - **go** : `go.mod`, std lib uniquement, `go vet` + `go build` + `go test` verts.
   - **python** : `pyproject.toml`, code compatible `ruff check` + `pytest`.
3. **Tests réels** : minimum 8 tests unitaires qui testent le comportement, pas des trivialités. **Arbitré** : `verify.sh` compte les tests (node : `test(`/`it(` ; go : `func Test` ; python : `def test_`) et refuse tout projet < 8.
4. **Vérification obligatoire** : avant de terminer, exécute `./verify.sh <projet>` et rapporte sa sortie réelle. Ne déclare JAMAIS « vert » sans l'avoir exécuté. Si rouge : diagnostique, corrige les vraies causes (ne bidouille pas les tests), et relance.
5. **Qualité** : code idiomatique, commentaires sobres. Pas de fonctionnalité au-delà de la spécification de l'issue. Pas de comptes, de clés ni d'appels réseau non spécifiés.
6. **Rendu final** : un court rapport — ce qui a été construit, comment le vérifier, résultat de `./verify.sh` (sortie réelle), nombre de tests.