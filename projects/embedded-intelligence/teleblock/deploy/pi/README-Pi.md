# TeleBlock sur Raspberry Pi — image prête à flasher

TeleBlock transforme un Raspberry Pi en **passerelle DNS du réseau** : les TV
continuent de résoudre Netflix et YouTube, leur télémétrie tombe dans le
sinkhole, et le tableau de bord répond sur `http://teleblock.local` dès le
premier boot. Rien à installer sur les TV, aucun compte, aucun cloud.

## En trois commandes

    1. make pi-image                     # binaire ARM64 statique + charge utile
    2. sudo tar -C /media/moi/rootfs -xzf dist/teleblock-pi-rootfs.tar.gz
    3. carte dans le Pi, puis http://teleblock.local

Avant l'étape 2, flashez Raspberry Pi OS Lite (64 bits) avec **Raspberry Pi
Imager** et, dans les réglages avancés, posez le nom d'hôte `teleblock`, votre
Wi-Fi et une clé SSH : c'est ce que le premier boot utilise pour se mettre en
ligne. L'étape 2 écrit sur la partition root de la carte, avant de la remettre
dans le Pi.

## Ce que la charge utile pose dans l'image

    /usr/local/bin/teleblock                          binaire linux/arm64 statique (CGO désactivé)
    /etc/systemd/system/teleblock.service             service systemd, CAP_NET_BIND_SERVICE seulement
    /etc/hostname + /etc/hosts                        nom « teleblock » → teleblock.local en mDNS
    /etc/systemd/resolved.conf.d/teleblock.conf       DNSStubListener=no : libère le port 53
    dist/pi-image/payload.json                        manifeste (chemins, modes, SHA-256)

Le service tourne sans root mais peut ouvrir le **port 53** (port privilégié)
grâce à `AmbientCapabilities=CAP_NET_BIND_SERVICE`, et son état appris vit dans
`/var/lib/teleblock` : il survit aux mises à jour du binaire.

## Faire du Pi la passerelle DNS

Dans l'interface du routeur, réglez le serveur DNS distribué en DHCP sur
l'adresse du Pi. Les TV prennent le changement au prochain bail (ou au
redémarrage). `teleblock.local` reste joignable même sans DNS, grâce à avahi.

## Vérifier

    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...   # cross-compile depuis n'importe quelle machine
    make pi-image && ls dist/pi-image/rootfs/etc/systemd/system/
    ./verify.sh embedded-intelligence/teleblock            # tests unitaires + scène E2E

Le binaire produit est statique : il fonctionne sur Raspberry Pi OS Lite 64
bits sans toolchain croisée ni bibliothèque arm64 sur la machine de build. Le
manifeste `payload.json` permet d'auditer une carte après flash sans la monter
(`sha256sum` sur les fichiers, chemins et modes attendus).
