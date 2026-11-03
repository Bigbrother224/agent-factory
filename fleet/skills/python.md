# Craft Python — l'usine attend du Python propre, typé et std-lib

## Structure
- Paquet `src/<projet>/` ou module simple selon la taille ; CLI dans `__main__.py` ou `cli.py`.
- `pyproject.toml` compatible `ruff check .` + `pytest -q` — zéro dépendance, std lib.
- `tests/` séparé, fichiers `test_<unit>.py`.

## Idiomes attendus
- Fonctions courtes avec **annotations de types** (`def load(path: str) -> dict[str, str]:`) — `ruff` est configured pour aimer ça, les reviewers aussi.
- Erreurs : exceptions précises (`ValueError` pour l'entrée utilisateur, `OSError`/`FileNotFoundError` pour l'IO) — jamais `except Exception:` muet.
- Dataclimat immuable : `@dataclass(frozen=True)` pour les valeurs de retour structurées plutôt qu'un tuple à 5 éléments.
- Compréhensions quand elles restent lisibles (une clause `if` max) — sinon boucle classique.
- `pathlib.Path` partout, jamais de concaténation de chaînes de chemins.

## Efficacité
- `collections.Counter`, `.get()`, `dict`/`set` pour l'indexation — pas de `list` scannée en boucle.
- `str.join` sur itérateur, pas `+=` en boucle.
- Fichiers : `with open(...) as f:` systématique ; itérer ligne à ligne (`for line in f:`) pour les grosses entrées.

## Tests (min 8, comportement + cas limités)
- `pytest` avec fixtures simples ; `pytest.raises(ValueError, match="...")` pour les erreurs attendues.
- Couvre : nominal, vide, invalide, bornes, chemins inexistants.
- Pas de tests dépendant de la date courante ou du réseau — injecte toujours les paramètres.

## Pièges qui coûtent un cycle
- Argument mutable par défaut (`def f(items=[])`).
- Oublier que `ruff check` échoue sur imports non triés — Lance-le avant de finir.
- `open()` sans encodage explicite (`encoding="utf-8"`) — dépend du locale du runner.
- Divulgation d'erreur brute à l'utilisateur final : catch et reformule au niveau CLI.