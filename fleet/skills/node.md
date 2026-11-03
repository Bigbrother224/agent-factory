# Craft Node — l'usine attend du Node idiomatique, pas du JavaScript traduit

## Structure
- `src/` = cœur métier pur (aucun `process.argv`, aucun `console.log` hors de la CLI).
- `cli.js` = interface : parsing des arguments, sortie, erreurs utilisateur.
- `test/` = `node --test` (fichiers `*.test.js`).
- `"type": "module"` — ESM partout, `export`/`import`, jamais `require`.
- `package.json` : `"lint": "node --check src/index.js && node --check cli.js"`, `"test": "node --test"`, `"build"` si sortie vérifiable. **Zéro dépendance.**

## Idiomes attendus
- API publique : fonctions nommées exportées, paramètres explicites — pas d'objet `options` vague.
- Erreurs : `throw new Error('message actionnable')` avec message qui dit QUOI et POURQUOI.
- Async : `async/await` — pas de chaînes `.then()` imbriquées.
- Strings : template literals ; itération : `for…of` sur Map/Set quand tu dédupes/indexes.
- Parsing d'options maison simple (`for (const arg of process.argv.slice(2))`) plutôt qu'un mini-framework.

## Efficacité
- Recherche/dédoublonnage : `Map`/`Set`, jamais `Array.includes` dans une boucle (O(n²) criard).
- Accumulation de strings en boucle : table + `join`, pas `+=`.

## Tests (min 8, comportement + cas limites)
- Importe depuis `../src/index.js` — teste le CŒUR, pas la CLI.
- Couvre : chemin nominal, entrée vide, entrée invalide, bornes, erreurs attendues (`assert.throws`).
- `import assert from 'node:assert/strict'` — égalité stricte toujours.

## Pièges qui coûtent un cycle
- Oublier `"type": "module"` → ESM refusé.
- `import` d'un fichier sans extension `.js`.
- Mutation d'arguments reçus (copie avant : `[...arr]`, `{...obj}`).
- `fs.readFileSync` sans `try/catch` sur un chemin fourni par l'utilisateur.
