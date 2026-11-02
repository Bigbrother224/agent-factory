# Étape 3 — Audition de la flotte B : résultats

Date : 2026-11-02 · Issue testée : #2 (TeleBlock, `idea/approved`) · Brief identique : `BRIEF-B.md`

## Tableau factuel

| Candidat | PR livrée ? | CI verte 1er coup ? | Interventions humaines | Durée | Coût/session | Statut |
|---|---|---|---|---|---|---|
| **Codex (OpenAI)** | — | — | 1 (blocage compte) | 0 s (refus immédiat) | — | ❌ **BLOQUÉ** : le compte ChatGPT refuse tous les modèles |
| **Devin (Cognition)** | — | — | — | — | — | ❌ **BLOQUÉ** : ni CLI ni clé API Cognition sur la machine |
| **Jules (Google)** | — | — | — | — | — | ❌ **BLOQUÉ** : rien d'installé, aucun accès API |

## Codex — diagnostic complet (réel)

- CLI 0.146.0 → **mise à jour 0.166.0** (faite ce jour) : le problème persiste, donc ce n'est pas la version.
- Auth : mode ChatGPT, handshake OK (`HTTP 101 Switching Protocols`), tokens du 2026-10-31.
- Erreur serveur, identique pour tous les modèles testés (`gpt-6-astra`, `gpt-5.2-codex`, `gpt-5.2`, `gpt-5.1-codex`, `gpt-6`, `gpt-6.1`, `gpt-5.5`, `gpt-5.3-codex`, `gpt-5.2-codex-max`) :
  > `The '<modèle>' model is not supported when using Codex with a ChatGPT account.`
- Dernière session Codex réellement fonctionnelle : **juin 2026** (sessions `~/.codex/sessions/2026/06/…`). Plus aucun usage depuis.
- Conclusion : l'accès Codex du compte ChatGPT a cessé entre juin et aujourd'hui (abonnement Plus/Pro non renouvelé, ou changement de plan). Rien de résolvable côté machine.

## Ce qu'il faut pour débloquer (une seule de ces trois options)

1. **Codex** : vérifier que l'abonnement ChatGPT couvre Codex, puis `codex logout && codex login` (auth navigateur — à toi de le faire).
2. **Devin** : créer une clé API sur api.devin.ai (facturé à l'ACU) et la fournir.
3. **Jules** : accès Google (jules.google) + clé API.

## Reprise de l'audition (quand une option est débloquée)

```bash
cd ~/Projects/factory
git checkout -b audition/<candidat>-teleblock
codex exec --skip-git-repo-check -s workspace-write "$(cat audition/BRIEF-B.md)"   # ou l'équivalent Devin/Jules
./verify.sh teleblock   # l'arbitre juge le livrable
git add -A && git commit -m "audition <candidat> : teleblock MVP" && git push -u origin audition/<candidat>-teleblock
gh pr create --fill
```

Puis remplir le tableau ci-dessus avec les métriques réelles : PR livrée ? CI verte du 1er coup (`gh pr checks`) ? interventions humaines (comptées) ? durée ? coût ?

## Décision en attente (porte humaine)

L'audition ne peut pas désigner de gagnant sans au moins un candidat fonctionnel. Le go/no-go de l'étape 3 est suspendu au déblocage d'une des trois options ci-dessus.