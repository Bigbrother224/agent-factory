# Roadmap — TeleBlock (issue #2, axe embedded-intelligence)

État du MVP actuel : résolveur DNS avec profils par appareil (`internal/dns`, `internal/profile`, `internal/rules`), parsing/classification SNI passive (`internal/sni`), tests unitaires, binaire Go (`main.go`), passant `./verify.sh`.
Définition de « shippé » (issue #2) : image Pi flashable + docker compose, blocage vérifié sur Samsung + LG avec Netflix et YouTube fonctionnels.

| Tranche | Objectif |
|---|---|
| T1 | Empreinte JA4 + blocage TCP ciblé (fin de l'inspection SNI passive) |
| T2 | Mode apprentissage / observation par appareil |
| T3 | Tableau de bord web embarqué |
| T4 | README portfolio anglais (contrat B règle 6) |
| T5 | Docker Compose (migration Pi-hole / déploiement avancé) |
| T6 | Image Raspberry Pi flashable & cross-compile ARM64 |
| T7 | Validation Samsung + LG : Netflix et YouTube fonctionnels, télémétrie bloquée |
| T8 | Définition de shippé atteinte |

- [x] T1 : inspection TLS passive sur port 443 — extraire le SNI en direct via TCP proxy/redirect iptables, appliquer le verdict du Set de marque et envoyer un reset TCP ciblé sur les flux bloqués (jamais du contenu vidéo décrypté) — fichiers : `internal/tcp/` (nouveaux), `main.go`, tests dans `internal/tcp/*_test.go` — vérif: `./verify.sh teleblock` (le package `internal/tcp` passe go vet/build/test).
- [ ] T2 : empreinte JA4 par ClientHello — calculer l'empreinte (version TLS + suites + extensions) et l'exposer pour classer les flux inconnus vers le verdict Observe — fichiers : `internal/sni/ja4.go` (ou `internal/ja4/`), `internal/ja4/ja4_test.go` — vérif: `./verify.sh teleblock`.
- [ ] T3 : mode apprentissage par appareil — chaque TV démarre en observation (~1 semaine), consigne les domaines vus et leur verdict proposé, puis applique automatiquement le profil à la fin de la période ; état persisté localement (JSON/SQLite, zéro cloud) — fichiers : `internal/observe/`, `internal/observe/observe_test.go`, `internal/profile/` (états) — vérif: `./verify.sh teleblock`.
- [ ] T4 : tableau de bord web local — interface HTTP embarquée `http://teleblock.local` : détection des TV du réseau (scan ARP/DNS), assignation de marque en un clic, compteur « N tentatives de tracking bloquées aujourd'hui » par TV en langage clair ; zéro compte, zéro télémétrie — fichiers : `internal/web/` (handlers + templates), `internal/web/web_test.go`, `main.go` — vérif: `./verify.sh teleblock`.
- [ ] T5 : README anglais niveau portfolio — why (problème HN, différenciation vs Pi-hole/NextDNS), demo (captures/dashboard), how (les 3 chemins d'installation), tests (ce que couvre verify.sh), licence open-source ; lisible en 2 min — fichiers : `projects/embedded-intelligence/teleblock/README.md` — vérif: `./verify.sh teleblock` + relecture humaine porte 2 (règle 6).
- [ ] T6 : déploiement Docker Compose — image multi-arch, bind port 53 + redirection, volume pour l'état, compose compatible « déjà un Pi-hole » (script une ligne `curl | sh` d'import) — fichiers : `deploy/docker-compose.yml`, `deploy/Dockerfile`, `deploy/install.sh` — vérif: `docker compose config -q` + `./verify.sh teleblock`.
- [ ] T7 : image Raspberry Pi flashable — cross-compile Go ARM64 statique, image de boot pré-configurée (Raspberry Pi Imager), le Pi devient passerelle DNS, `teleblock.local` accessible au premier boot — fichiers : `deploy/pi/` (scripts image, systemd unit), `Makefile`, `deploy/pi/README-Pi.md` — vérif: `GOOS=linux GOARCH=arm64 go build ./...` + `./verify.sh teleblock`.
- [ ] T8 : définition de shippé atteinte — blocage effectivement vérifié sur une TV Samsung et une TV LG réelles avec Netflix et YouTube fonctionnels pendant la lecture, télémétrie bloquée confirmée au dashboard, procédure d'installation T5/T6/T7 suivie de bout en bout — fichiers : `docs/validation.md` (compte-rendu de tests Samsung/LG daté avec résultats), mises à jour mineures du README si besoin — vérif: `./verify.sh teleblock` + `docker compose up` en local ok + compte-rendu `docs/validation.md` relu par la porte humaine 2, labels `ship/pr` et `verify/ok` posés.