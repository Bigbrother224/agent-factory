# Portfolio & Usine — Bigbrother224

Ce dépôt est **à la fois un portfolio vivant et l'usine logicielle qui le construit** :
des groupes d'agents autonomes y produisent des projets réels, testés et revus,
un axe de spécialité à la fois. Aucun service payant : tout tourne sur plan Ollama.

---

## 🎯 Le portfolio — 3 axes de spécialité

### [AI Automation & Systems](projects/ai-automation-and-systems/)
*Agents, orchestration, pipelines, sandboxing, CI — l'ingénierie système derrière les systèmes IA autonomes.*

### [Embedded Intelligence](projects/embedded-intelligence/)
*L'IA qui descend dans le matériel : embarqué, edge ML, capteurs, contraintes réelles.*
- **[TeleBlock](projects/embedded-intelligence/teleblock/)** — pare-feu de confidentialité pour smart TVs : bloque télémétrie/pub **sans casser le streaming** (classification TLS/SNI par flux, pas par domaine — ce que Pi-hole ne sait pas faire). Go, binaire statique, Raspberry Pi.

### [Experimental Computing](projects/experimental-computing/)
*Informatique expérimentale poussée jusqu'à la démonstration : paradigmes, calcul non conventionnel.*

*Chaque projet shippe : tests réels passant en CI, un README anglais niveau portfolio (why / demo < 3 commandes / how it works), et une revue de code publique.*

---

## ⚙️ L'usine (comment ce portfolio se construit)

**Une équipe résidente bosse en continu sur le chantier actif :**

La boucle chantier (launchd, toutes les 45 min sur le Mac + auto-merge GitHub) :
le **planificateur** découpe le projet approuvé en tranches (ROADMAP.md), un
**crew pi-team** (codeur + vérifieur, flash) implémente une tranche par cycle,
l'arbitre vérifie, la **revue kimi** joue le tech lead (ses retours nourrissent
le cycle suivant), et l'auto-merge GitHub fusionne les tranches vertes+GO.
À la fin du roadmap : label `ship/pr` → porte 2 humaine. **L'usine construit
pendant que tu ne regardes pas.**

**3 groupes d'agents, événementiels — zéro VM qui dort :**

| Groupe | Agent | Quand |
|---|---|---|
| **A — Scouts** | veille HN+GitHub (digest sourcé) → idée alignée sur UN axe | 1×/jour, cron 08:30 UTC |
| **B — Bâtisseurs** | Aider (glm-5.3) construit le projet dans le runner GitHub | à chaque idée approuvée |
| **C — Vérificateurs** | arbitre CI (`verify.sh`) + réviseur lecture-seule (kimi) | chaque PR |

**Les 2 portes humaines :**
1. Approuver une idée : `gh issue edit <N> -R Bigbrother224/agent-factory --add-label idea/approved` (l'issue doit porter un label `axe/*`)
2. Merger la PR (après CI verte + revue GO) : `gh pr merge <N> -R Bigbrother224/agent-factory`

**Promotion portfolio** : un projet bien exécuté devient un repo public autonome
(avec sa CI) : `scripts/publier.sh <axe>/<projet>`. Garde-fous : arbitre vert +
README anglais niveau portfolio obligatoires — l'usine produit ce README via le
chantier avant que la publication soit possible.

**Voir l'avancement :**
```bash
gh issue list -R Bigbrother224/agent-factory --label idea/draft   # idées en attente (porte 1)
gh issue list -R Bigbrother224/agent-factory --label ship/pr      # prêtes à merger (porte 2)
gh pr list -R Bigbrother224/agent-factory                        # PRs + CI + revues
```

**Parler aux agents :** `!fix …` en commentaire d'une issue → la flotte applique tes instructions au projet.

**Qualité (pourquoi ce n'est pas du code jetable) :** CI = lint+build+**tests réels** obligatoires (PR rouge bloquée) · chaque PR relue publiquement par un second modèle (verdict GO/NO-GO) · contrat de périmètre strict · README portfolio obligatoire.

## Protocole des labels

| Label | Posé par | Signification |
|---|---|---|
| `idea/draft` | scout | idée en attente de la porte 1 |
| `axe/*` | scout | axe portfolio du projet (les 3 ci-dessus) |
| `idea/approved` / `idea/rejected` | **humain** | la flotte démarre / le scout réitère (raison en commentaire) |
| `ship/pr` | workflow | PR vérifiée, attend la porte 2 |
| `verify/ok` / `verify/failed` | workflow | verdict du groupe C |

## Structure

```
projects/<axe>/<projet>/   ← LE PORTFOLIO
ideas/                     # idées sourcées (template 6 sections)
verify.sh                  # arbitre froid : lint + build + tests
.github/workflows/         # fleet (construit), revue (kimi), retour-humain (!fix), verify (CI)
fleet/                     # contrat B, revue, ciblage (outillage flotte)
scripts/                   # veille + cron scout quotidien
ETAT.md                    # l'état de l'usine
```