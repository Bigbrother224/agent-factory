# Archon — CI autonome pour architectures CPU conçues par IA

Date: 2026-11-20
Scout: A
Axe: ai-automation-and-systems
Label: idea/draft

## 1) Axe & légitimité portfolio
Pipeline d'agents (génération RTL → vérification formelle → scan sécurité → rapport) qui transforme un fichier Verilog/Chisel issu d'IA en livrable hardware audité. Incarnation directe de ai-automation-and-systems : orchestration CI, agents de vérification, déploiement en une commande.

## 2) Tendance sourcée
Hacker News, 2026-11-20 : « Frostbite: A new CPU architecture discovered, designed and verified by AI » (458 pts, 265 coms) prouve que l'IA conçoit désormais des CPU. Même jour : CVE-2026-121481 OpenSSL (CVSS 9.8) souligne le risque critique de sécurité mémoire en bas niveau.

## 3) Besoin réel du marché
Startups de puces et équipes embarquées utilisent des LLM pour écrire du RTL, mais n'ont pas de pipeline automatisé pour valider fonctionnalité et sécurité avant FPGA/gravure. Une erreur coûte des mois et des millions.

## 4) Problème résolu
Remplace la vérification manuelle et silotée par une CI qui exécute simulation, propriétés formelles et détection de motifs vulnérables sur tout cœur CPU généré par IA, avant intégration.

## 5) Ce qui manque aux solutions existantes
Verilator / SymbiYosys sont des outils isolés, sans orchestration, sans agent de triage ni rapport unifié pour du RTL issu de LLM. Aucun lien entre « conception par IA » et « CI matérielle sécurisée ».

## 6) Premier contrat technique
`pip install archon-ci && archon new --rtl design.sv && archon verify` produit un rapport HTML (simulation Icarus, propriétés SymbiYosys, scan de dépassements tampon). MVP démotrable en <3 commandes, intégrable GitHub Actions.

Sources: Hacker News front page, 2026-11-20 (Frostbite, OpenSSL CVE-2026-121481).