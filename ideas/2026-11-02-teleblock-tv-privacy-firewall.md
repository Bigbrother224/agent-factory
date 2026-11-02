# TeleBlock — pare-feu de confidentialité pour téléviseurs connectés

- **Date :** 2026-11-02
- **Scout :** Groupe A (pi-team scout, nemotron-3-super) — digest `.veille/2026-11-02.md`
- **Label :** idea/draft

## 1. Tendance sourcée
- HN front page, 2026-11-02 : « Ask HN: Which TV do you recommend for privacy? » — 489 pts, 612 commentaires (aucune TV respectueuse de la vie privée disponible à l'achat).
- HN front page, 2026-11-02 : « The AI cost of JavaScript frameworks » (399 pts) et l'écosystème unslop (GitHub, 5 547 ★ en <7 jours) — la privacy et la fin du tracking en UI sont des sujets chauds du moment.

## 2. Besoin réel du marché
- **Qui :** foyers et petites structures équipés de smart TV (Samsung, LG, Sony, TCL…).
- **Preuve :** le fil HN ci-dessus (612 coms) documente l'impossibilité d'acheter une TV « dumb » ; Pi-hole/NextDNS sont cités mais cassent le streaming (Netflix, YouTube) car contenu et télémétrie partagent les mêmes domaines.

## 3. Problème résolu (1 phrase)
Le projet résout : bloquer la télémétrie et la publicité des smart TV **sans interrompre le streaming**, ce que les bloqueurs DNS généralistes ne savent pas faire.

## 4. Ce qui manque aux solutions existantes
- **Pi-hole / NextDNS :** filtrage par domaine uniquement ; casse le streaming quand contenu et traçage partagent un domaine ; configuration réservée aux utilisateurs techniques.
- **Notre différenciateur :** passerelle locale (Raspberry Pi) qui classe le trafic TV en « contenu » (passe) vs « télémétrie/pub » (bloqué) par empreintes TLS/SNI et signatures d'apps, avec liste pré-configurée par modèle de TV et mode apprentissage.

## Premier contrat technique (pour le groupe B)
- **Stack :** Go (DNS/DHCP + classification TLS/SNI) + interface web de configuration ; image Raspberry Pi flashable.
- **Définition de « shippé » :** tableau de bord web fonctionnel, blocage vérifié sur 2 modèles de TV (Samsung + LG), Netflix et YouTube toujours fonctionnels.
- **Contraintes :** aucun compte cloud ; zéro interruption de streaming ; open-source auditable.