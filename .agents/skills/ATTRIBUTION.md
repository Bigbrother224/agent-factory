# Attribution — skills externes curés pour l'usine

Skills installés le 2026-11-03, sélectionnés pour équiper les agents de l'usine
(crew du chantier, flotte B, scout, réviseur) contre le vibecoding.

| Skill | Source | Licence | Usage dans l'usine |
|---|---|---|---|
| `frontend-design` | [anthropics/skills](https://github.com/anthropics/skills) — skills/frontend-design | **Apache 2.0** (LICENSE.txt joint) | L'étalon design anti-vibecoding ; adapté pour les briefs dans `fleet/skills/frontend-design.md` |
| `test-driven-development` (+ writing-good-tests.md) | [obra/superpowers](https://github.com/obra/superpowers) — skills/test-driven-development | MIT | Discipline du codeur du chantier ; condensé dans `fleet/skills/tdd.md` |
| `systematic-debugging` | obra/superpowers — skills/systematic-debugging | MIT | Cause racine avant correctif ; condensé dans `fleet/skills/debugging.md` |
| `verification-before-completion` | obra/superpowers — skills/verification-before-completion | MIT | Preuves avant déclarations ; condensé dans `fleet/skills/verification.md` |
| `factory-ideation`, `factory-selection` | rédigés maison (2e agent) | — | Craft du groupe A (scout/sélectionneur) — voir ideas/README.md |

## Pourquoi ceux-là

- **obra/superpowers (311k ⭐)** : la référence méthodologique — des lois de fer qui
  compensent exactement les faiblesses des modèles flash (affirmations non prouvées,
  fixes de symptôme, code avant test).
- **anthropics/skills** (Apache 2.0 pour le skill frontend-design) : le skill officiel `frontend-design` nomme littéralement les
  « tells du design généré par IA » (kit SaaS-cards, fond crème + terracotta, eyebrows
  ALL-CAPS, « → » sur les boutons) — l'antidote direct au « on voit trop que c'est vibecodé ».

Les fichiers `.agents/skills/` sont les textes canoniques (découverts par tout agent pi
travaillant dans ce repo). Les `fleet/skills/` sont leurs adaptations-condensés injectées
dans les briefs des sessions Aider (chantier + flotte), qui ne découvrent pas les skills
automatiquement.

## Mise à jour

Ces skills évoluent en amont. Pour resynchroniser :
`gh api repos/<owner>/<repo>/contents/skills/<skill>/SKILL.md --jq .content | base64 -d`