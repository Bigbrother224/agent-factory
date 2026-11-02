# TeleBlock — pare-feu de confidentialité pour téléviseurs connectés

- **Date :** 2026-11-02
- **Scout :** Groupe A (pi-team scout, nemotron-3-super) — digest `.veille/2026-11-02.md` — enrichi après retour porte 1
- **Label :** idea/draft

## 1. Tendance sourcée
- HN front page, 2026-11-02 : « Ask HN: Which TV do you recommend for privacy? » — 489 pts, 612 commentaires (aucune TV respectueuse de la vie privée disponible à l'achat).
- HN front page, 2026-11-02 : « The AI cost of JavaScript frameworks » (399 pts) et l'écosystème unslop (GitHub, 5 547 ★ en <7 jours) — la privacy et la fin du tracking en UI sont des sujets chauds du moment.

## 2. Besoin réel du marché
- **Qui :** foyers et petites structures équipés de smart TV (Samsung, LG, Sony, TCL…), non-techniciens inclus.
- **Preuve :** le fil HN ci-dessus (612 coms) documente l'impossibilité d'acheter une TV « dumb » ; Pi-hole/NextDNS sont cités mais cassent le streaming (Netflix, YouTube) car contenu et télémétrie partagent les mêmes domaines.

## 3. Problème résolu (1 phrase)
Le projet résout : bloquer la télémétrie et la publicité des smart TV **sans interrompre le streaming**, ce que les bloqueurs DNS généralistes ne savent pas faire.

## 4. Différenciation vs solutions existantes (explicite)

| | Pi-hole / NextDNS | TeleBlock |
|---|---|---|
| **Granularité du blocage** | Domaine entier — si la télémétrie et la vidéo Netflix partagent un domaine/CDN, bloquer casse la lecture | Connexion : inspection TLS/SNI + empreinte (JA4) + chemin HTTP → bloque le flux de télémétrie, laisse passer le flux vidéo **du même domaine** |
| **Méthode** | Liste noire statique à maintenir à la main | Profils par marque/modèle (Samsung, LG, Sony, TCL) livrés en données + **mode apprentissage** : 1ʳᵉ semaine en observation (baseline par périphérique), puis application automatique |
| **Configuration** | Réservée aux utilisateurs techniques (DNS, regex, allowlists) | Zéro config : le tableau de bord détecte les TV du réseau, on clique sur sa marque, c'est fini |
| **Feedback utilisateur** | Logs DNS bruts | « Aujourd'hui : 4 217 tentatives de tracking bloquées sur votre Samsung » en langage clair, par TV |
| **Privacy du privacy-tool** | NextDNS = cloud (paradoxe) ; Pi-hole = local mais manuel | **Local-only, zéro compte, zéro télémétrie du pare-feu lui-même**, open-source auditable |

## 5. Installation utilisateur (UX, 3 chemins)

1. **Le plus simple (cible grand public) :** image Raspberry Pi pré-configurée à flasher (Raspberry Pi Imager / balenaEtcher) sur une carte SD → le Pi se branche sur le routeur → le téléphone ouvre `http://teleblock.local` → le tableau de bord détecte les TV du réseau → l'utilisateur clique sur la marque de chaque TV → actif. Le Pi devient la passerelle DNS de la TV (ou de tout le réseau domestique, au choix).
2. **Déjà un Pi-hole ?** script une ligne (`curl -sL teleblock.sh | sh`) ou image Docker Compose : migration importée, TeleBlock remplace Pi-hole sans toucher au reste du réseau.
3. **Avancé :** binaire Go statique sur n'importe quelle machine Linux du réseau (même un vieux laptop).

TV non-détectée : mode « TV générique » + apprentissage.

## 6. Premier contrat technique (pour le groupe B)
- **Stack :** Go pur (démons DNS + classification dans un seul binaire statique), interface web embarquée ; aucune dépendance externe, cross-compile ARM64 (Pi).
- **MVP à livrer (périmètre de l'audition) :** résolveur DNS avec profils par périphérique (map IP→marque) + classifieur SNI/empreinte par flux avec règles par marque + **tests unitaires réels** ; doit passer `./verify.sh` (go vet + go build + go test).
- **V2 :** mode apprentissage, tableau de dashboard, image Pi flashable.
- **Définition de « shippé » (produit final) :** image Pi flashable + docker compose, blocage vérifié sur Samsung + LG avec Netflix et YouTube fonctionnels.
- **Contraintes :** aucun compte cloud ; zéro interruption de streaming ; open-source auditable ; pas de MITM du contenu vidéo (inspection SNI passive + reset TCP ciblé uniquement).