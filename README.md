# agent-factory — l'usine à projets

Usine à projets logiciels pilotée par 3 groupes d'agents. Le dépôt est le bus de communication :
issues, PRs et labels sont le protocole. Aucun chat entre agents. Aucun service payant : tout tourne sur ton plan Ollama.

---

## 🧭 Mode d'emploi (pour l'humain — toi)

**Les 2 portes que tu tiens :**
1. **Approuver/refuser une idée** : `gh issue edit <N> -R Bigbrother224/agent-factory --add-label idea/approved` (ou `idea/rejected` + un commentaire avec la raison)
2. **Merger/refuser une PR** : `gh pr merge <N> -R Bigbrother224/agent-factory`

**Voir l'avancement (tableau de bord) :**
```bash
gh issue list -R Bigbrother224/agent-factory --label idea/draft      # idées en attente de ta porte 1
gh issue list -R Bigbrother224/agent-factory --label ship/pr        # projets prêts à merger (porte 2)
gh pr list -R Bigbrother224/agent-factory                           # PRs en cours + CI + revue
gh issue list -R Bigbrother224/agent-factory --label verify/failed  # échecs (rare, ou relances)
```
Sur le web : l'onglet **Issues** triée par label = ton pipeline. Rien d'autre à regarder.

**Parler aux agents — tes commentaires sont pris en compte :**
- **N'importe quel commentaire sur une issue** : lu par le scout pour les idées (rejet + raison → il réitère en tenant compte).
- **`!fix …`** en commentaire d'une issue → la flotte B applique tes instructions au projet et rouvre une PR vérifiée. Exemple : `!fix ajoute un README au projet`.

**Le cycle complet, du début à la fin :**
```
cron 08:30 UTC → scout (veille HN+GitHub) → issue idea/draft
        ↓ ta porte 1 : idea/approved
flotte B : session Aider (glm-5.3) dans le runner GitHub → code + tests → verify.sh (arbitre, max 3 essais)
        ↓ branche fleet/<projet> → PR
groupe C : CI verify.sh + revue lecture-seule (kimi) publiée sur la PR → labels verify/ok + ship/pr
        ↓ ta porte 2 : merge
projet livré dans projects/<projet>/
```

## Qui travaille dans l'usine (agents déployés)

| Groupe | Agent | Modèle | Quand il vit |
|---|---|---|---|
| **A — Scouts** | 1 scout (veille, lecture-seule) | nemotron-3-super | 1 fois/jour (cron 08:30 UTC, machine mac) |
| **B — Exécutants** | 1 session Aider par idée approuvée | glm-5.3 | éphémère : nait à l'approbation, meurt à la PR |
| **C — Vérificateurs** | ① CI verify.sh (arbitre froid, zéro LLM) ② 1 réviseur par PR (lecture-seule) | — ② kimi-k2.7-code | ① chaque PR ② chaque PR, publie sa revue en commentaire |

Aucun agent ne « dort » sur une VM : l'usine est événementielle — un label, un commentaire, un cron, et l'agent adéquat nait, travaille, disparaît. C'est ce qui rend le coût quasi nul.

## Où est le code des projets

Chaque projet livré vit dans **`projects/<projet>/`** (auto-vérifiable : node/go/python détectés par `verify.sh`). Sur ton Mac : `~/Projects/factory/` (miroir local, `git pull` pour synchroniser).

## Comment on garantit que ce n'est pas du code pourri

1. **Arbitre froid** (`verify.sh` + CI) : lint + build + tests réels obligatoires, PR rouge bloquée. Le modèle ne peut pas mentir ici : c'est la CI qui exécute.
2. **Deuxième paire d'yeux** : chaque PR reçoit la revue d'un modèle DIFFÉRENT de celui qui a écrit le code (kimi-k2.7-code), publiée sur la PR — verdict GO/NO-GO + problèmes listés.
3. **Contrat B** (`fleet/CONTRAT-B.md`) : périmètre verrouillé (`projects/<slug>/` uniquement), 8+ tests comportementaux minimum, interdiction de déclarer « vert » sans exécuter.
4. **Limite assumée** : l'auteur des tests est l'agent codeur — un biais possible. Ta porte 2 (2 minutes : la revue kimi + le diff) reste le contrôle final. Pour les projets critiques, un audit pi-team (rôle auditor) peut être ajouté à la demande.

## Protocole des labels

| Label | Signification | Qui le pose |
|---|---|---|
| `idea/draft` | idée candidate, attend ta porte 1 | scout |
| `idea/approved` | la flotte B démarre | **toi** |
| `idea/rejected` | refusée, le scout réitère (mets la raison en commentaire) | **toi** |
| `ship/pr` | PR vérifiée, attend ta porte 2 | workflow |
| `verify/ok` / `verify/failed` | verdict du groupe C | workflow |
| `!fix …` (commentaire) | tes instructions de correction | **toi** |

## Structure du dépôt

```
ideas/          # un md par idée (4 critères, sources datées)
projects/       # LE CODE : un sous-dossier par projet livré
verify.sh       # l'arbitre froid
.github/workflows/
  verify.yml        # CI arbitre (chaque PR)
  fleet.yml         # porte 1 → session Aider → PR
  revue.yml         # réviseur kimi sur chaque PR
  retour-humain.yml # tes !fix → session Aider
fleet/          # contrat B, revue, slugify (outillage de la flotte)
scripts/        # veille.sh (digest HN+GitHub), scout-daily.sh (cron)
audition/       # étape 3 : comparaison Aider/OpenHands + verdict
ETAT.md         # l'état du projet usine
```