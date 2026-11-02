# Étape 3 — Audition de la flotte B : RÉSULTATS FINAUX

Contrainte verrouillée par l'humain (2026-11-02) : **zéro service tiers payant** — uniquement le plan Ollama existant. Candidats donc tous libres, tous pilotés par l'API Ollama (`https://ollama.com/v1`, clé Trousseau `pi-ollama`), même brief (`BRIEF-B.md`), même modèle (`glm-5.3:cloud`).

## Tableau factuel

| Candidat | PR livrée ? | CI verte ? | Interventions humaines/opérateur | Durée | Coût |
|---|---|---|---|---|---|
| **Aider** (libre) | ✅ **PR #3** | ✅ run 41459426998 | **2** (① format udiff requis — 1ʳᵉ session n'a rien appliqué + hallucination de vérif ; ② relance fix tests) | 1602 s en 3 sessions (~27 min utiles) | plan Ollama seul |
| **OpenHands** (libre, Docker) | ❌ **RIEN** | — | 2 (image dérivée Dockerfile requis car entrypoint descend de root ; sandbox build) | 4417 s (60/60 itérations, état ERROR) | plan Ollama seul (60 itérations consommées) |
| **SWE-agent** (libre) | non évalué | — | installé (`~/.local/bin/sweagent`), run non tenté dans le budget temps du tour | — | — |
| ~~Devin / Codex / Jules~~ | — | — | — | — | **exclus par l'humain : payants** |

## Détails factuels

### Aider 0.86.2 — VAINQUEUR
- Session 1 (glm-5.3-flash, format `whole`) : 463 s, **zéro fichier appliqué** (format d'édition non suivi), et le modèle a **affirmé « résultat vert » sans avoir exécuté verify.sh** — hallucination de vérification détectée par l'arbitre.
- Session 2 (glm-5.3, format `udiff`) : 629 s, 10 fichiers, structure complète (main.go + internal/dns, profile, rules, sni + tests) — mais `go test` ROUGE (`sni_test`).
- Session 3 (relance standard de l'usine `verify/failed`→B) : 510 s avec `--test-cmd ./verify.sh teleblock --auto-test`, **corrigé seul jusqu'au vert**.
- CI GitHub verte sur PR #3 au premier dépôt. Livrable complet : résolveur DNS par profil + parseur ClientHello réel + règles par marque + 24 tests.

### OpenHands — ÉCHEC
- Setup lourd sur macOS : image 1.33 Go + entrypoint qui force l'utilisateur non-root → il fallut une image dérivée (`openhands-audition:local`) avec client docker statique + buildx préinstallés.
- Run réel : 60 itérations (max) atteintes en 4417 s, **aucun fichier créé** dans le workspace. Modèle/agent incapables de converger sur la tâche dans le budget.
- Log complet : `/tmp/audition-openhands.log`.

## VERDICT (go/no-go de l'étape 3)

**La flotte B = Aider**, configuration verrouillée :
```
aider --model openai/glm-5.3:cloud --openai-api-base https://ollama.com/v1 \
  --weak-model openai/glm-5.3-flash:cloud --edit-format udiff --yes-always \
  --test-cmd "./verify.sh <projet>" --auto-test --message "<tâche>"
```
- Gratuit (libre), git-native (commits + PR), auto-test branché sur l'arbitre → boucle `verify/failed` autonome intégrée.
- Point de vigilance usine : toujours brancher `--test-cmd` sur `verify.sh` + l'arbitre CI en bout de chaîne, car le modèle peut **halluciner une vérification** — seul l'arbitre froid dit la vérité.
- OpenHands : écarté (non-convergence). SWE-agent : disponible pour une audition ultérieure si on veut un challenger, une commande prête suffit.