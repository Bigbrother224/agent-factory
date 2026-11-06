# AgentGate — Proxy egress et identité pour agents de codage IA

Date : 2026-11-06
Scout : A
Axe : ai-automation-and-systems
Label : idea/draft

1) Axe & légitimité portfolio
Illustre l'axe ai-automation-and-systems : proxy egress local, moteur de politiques de sortie, pipeline d'anonymisation des PII et intégration avec les agents CLI autonomes. Preuve de génie système, sécurité applicative et orchestration d'agents.

2) Tendance sourcée
Claude Code transmet l'adresse email utilisateur à des sites web tiers (GitHub issue Anthropic, 1 424 pts, 06/11/2026). Parallèlement, des bots IA submergent les bibliothèques universitaires (CBC, 06/11/2026). Les agents actuels n'ont ni identité bornée, ni contrôle de fuite de données.

3) Besoin réel du marché
Entreprises et développeurs adoptent les agents de codage mais craignent l'exfiltration de PII, clés et identifiants vers des endpoints inconnus. Les sites web ont besoin de distinguer agents légitimes de scrapers malveillants.

4) Problème résolu
Un agent CLI peut lire l'environnement et le poster n'importe où sans préavis. AgentGate intercepte tout le trafic sortant de l'agent, masque les PII (emails, tokens, clés), applique une whitelist de sorties et émet un jeton d'identité éphémère pour attestation côté serveur.

5) Ce qui manque aux solutions existantes
Les proxy génériques (mitmproxy, Burp) ignorent le contexte agent (prompt, outil appelé, session). Aucun outil open-source n'associe filtrage PII contextuel et attestation d'identité pour agents CLI locaux.

6) Premier contrat technique
`curl -L install.agentgate.dev | sh` puis `agentgate init` puis `agentgate run -- claude-code`. Démonstration < 3 commandes : requête sortante avec email interceptée, masquée automatiquement, journal d'audit affiché.

Sources : GitHub Anthropic claude-code issue #107757 (06/11/2026), CBC.ca AI bots libraries (06/11/2026).