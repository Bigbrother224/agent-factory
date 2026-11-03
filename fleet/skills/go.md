# Craft Go — l'usine attend du Go qui respire la std lib

## Structure
- `main.go` = CLI (parsing flags, fmt, os.Exit) ; le cœur dans `internal/<domaine>/`.
- `internal/` pour tout ce qui n'est pas réutilisable publiquement — découpage par domaine, un paquet = une responsabilité.
- `*_test.go` à côté des fichiers, `go.mod` sans une seule dépendance externe.
- Vérification : `go vet ./...` + `go build ./...` + `go test ./...` verts.

## Idiomes attendus
- **Erreurs** : `if err != nil { return fmt.Errorf("charger le profil %s : %w", name, err) }` — wrap avec contexte, messages qui commencent par un verbe minuscule.
- Pas de panic hors des invariants de programmation ; les erreurs attendues sont des valeurs (`errors.New`, `fmt.Errorf`), jamais des `string` magiques.
- **Interfaces** : déclarées côté consommateur, 1-2 méthodes max — pas de hiérarchie préventive.
- Receveurs : pointeurs seulement si mutation nécessaire.
- Nommage : `CamelCase` exporté, `camelCase` interne, pas de `get`/`set` préfixes (`profile.Load`, pas `profile.GetProfile`).

## Efficacité
- `bytes.Buffer` / `strings.Builder` pour construire, jamais `+=` sur des strings en boucle.
- `map[string]T` pour indexer, mais garde l'ordre déterministe via un slice trié à la sortie si le résultat doit être stable.
- `bufio.Scanner` pour lire ligne à ligne — pas de read-all sur des entrées potentiellement grosses.

## Tests (min 8, comportement + cas limites)
- Table-driven : `tests := []struct{ name string; in…; want… }` + `t.Run`.
- Couvre : nominal, entrée vide, entrée invalide, bornes, erreur attendue (`wantErr`).
- Pas de `time.Sleep` pour synchroniser — déterministe seulement.

## Pièges qui coûtent un cycle
- Oublier `defer file.Close()` (ou fermer deux fois).
- Ombreage de variable (`:=` qui masque `err`) — l'erreur la plus fréquente du compilateur débutant.
- `range` sur une map = ordre aléatoire ; un test qui dépend de l'ordre échouera une fois sur dix.
- Ignorer l'erreur de `strconv`/`os.ReadFile` « pour aller vite ».