# Brief d'audition — flotte B (identique pour tous les candidats)

Issue source : #2 (TeleBlock — pare-feu de confidentialité pour smart TVs), label `idea/approved`.

## Périmètre (MVP, une seule session)

Implémente `projects/teleblock/` — **Go pur, std lib uniquement** (aucune dépendance externe — le sandbox de l'audition est sans réseau, et le produit final est un binaire statique) :

1. **Résolveur DNS** : serveur UDP minimal qui répond aux requêtes A, avec **profils par périphérique** : map IP source → marque de TV (samsung, lg, sony, tcl, generic), chaque profil ayant ses règles.
2. **Classifieur SNI** : parse du ClientHello TLS (parsing réel des enregistrements TLS, pas une regex sur des octets magiques) → extraction du SNI → verdict `allow` / `block` / `observe` selon les règles du profil de la marque (ex. `samsungacr.com` → block, `netflix.com` → allow).
3. **Règles par marque** : fichiers de règles par défaut en dur, raisonnables et documentées (au moins 6 règles par marque, basées sur les domaines de télémétrie connus des marques).
4. **Tests unitaires réels** : `go test ./...` doit couvrir le parsing ClientHello, le verdict par règle, et le routage par profil. Minimum 8 tests.

## Non-négociables

- `./verify.sh teleblock` doit passer (go vet + go build + go test, exécuté à la fin, résultat rapporté tel quel).
- Ne touche à RIEN en dehors de `projects/teleblock/` (pas de modification de `verify.sh`, `README.md`, etc.).
- Pas d'architecture au-delà du MVP : pas de dashboard, pas d'image Docker, pas de mode apprentissage.
- Code idiomatique, commentaires sobres, tout en anglais dans le code.

## Métriques relevées par l'arbitre de l'audition
PR livrée ? · CI verte du 1er coup ? · interventions humaines ? · durée ? · coût ?