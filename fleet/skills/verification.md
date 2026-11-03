# Vérification avant de déclarer fini — preuves avant affirmations

> Source : skill `verification-before-completion` de obra/superpowers (MIT), condensé pour la flotte.

## LA LOI

```
AUCUNE DÉCLARATION DE « FINI / VERT / RÉSOLU » SANS COMMANDE EXÉCUTÉE ET SORTIE OBSERVÉE
```

Les affirmations sans preuve sont des hallucinations. L'arbitre `./verify.sh` est le
seul juge — ta confiance n'est pas une preuve.

## La checklist de fin de session (dans cet ordre)

1. **Exécuter** `./verify.sh <axe>/<projet>` — et REGARDER la sortie jusqu'au verdict
   final (vert/rouge), pas juste l'absence de crash.
2. **Chaque exigence de la tranche** : retrouve-la dans la roadmap, montre comment
   elle est satisfaite (fichier + test qui la protège). Une exigence sans test = pas finie.
3. **Chaque test compte** : minimum 8 tests réels (contrat B), comportement + cas limites.
4. **Zéro reste** : pas de TODO, pas de code commenté « pour plus tard », pas de
   placeholder, pas de console.log oublié.
5. **Le diff relit le brief** : rien de construit hors de la spécification,
   rien de la spécification manquant.
6. **Le README portfolio** (si la tranche l'exige) : démo en < 3 commandes, sortie attendue.

## Interdits

- « Ça devrait marcher » — devrait n'est pas un état.
- « Les tests passaient tout à l'heure » — ré-exécute, montre.
- Déclarer vert sur la foi de la session précédente : la re-vérification coûte quelques
  secondes, une régression mergée coûte un cycle complet.
- Mettre à jour la roadmap ([x]) AVANT que l'arbitre soit vert.

## Si tu ne peux pas vérifier

Dis-le explicitement : « NON VÉRIFIÉ + pourquoi + ce qui manque ». Un échec honnête
est corrigeable au prochain cycle ; un succès inventé pollue tout le pipeline.