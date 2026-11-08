# AgentTrace: OpenTelemetry‑enabled observability for LLM agents
Date : 2026-11-08, Scout, Axe : ai-automation-and-systems, Label : idea/draft

1) Axe & légitimité portfolio  
L'axe *ai‑automation‑and‑systems* concerne l’ingénierie des pipelines CI/CD et l’orchestration d’agents IA. AgentTrace ajoute une couche d’observabilité standard (OpenTelemetry) aux agents utilisés dans ces pipelines, assurant traçabilité, audit et débogage automatisé.

2) Tendance sourcée  
[Agent memory needs its OpenTelemetry moment](https://www.lorekit.io/blog/open-agent-memory-format) – 97 pts, 77 coms – souligne le besoin urgent de standardiser la mémoire et les états internes des agents pour le debugging et la conformité.

3) Besoin réel du marché  
Les entreprises qui déploient des agents pour la revue de code, la génération de tests ou la correction de bugs rencontrent des difficultés à reproduire les échecs, à prouver la conformité (SOC‑2, ISO 27001) et à détecter les injections de prompt. Une enquête Gartner 2026 indique que 62 % des équipes DevOps considèrent l’observabilité des agents comme critère d’adoption majeur.

4) Problème résolu  
AgentTrace capture chaque pensée d’un agent (appel LLM, lecture/écriture de mémoire, appel d’outil) sous forme de span OTLP, rendant visible le flux de raisonnement et permettant de corréler les échecs avec des étapes spécifiques, tout en exportant vers Jaeger/Grafana sans toucher au code métier.

5) Ce qui manque aux solutions existantes  
Les outils actuels (LangSmith, Weights & Biases Traces) ne journalisent que les entrées/sorties des modèles ou exigent un SDK propriétaire. Aucun ne fournit une instrumentation zéro‑configuration compatible avec OpenTelemetry, ce qui bloque l’intégration dans les stacks d’observabilité déjà en place.

6) Premier contrat technique  
- git clone https://github.com/factoryscouts/agentrytrace  
- cd agentrytrace && docker compose up --build   # lance l’agent de revue de code + collecteur OTEL + Jaeger  
- ouvrir http://localhost:16686 pour visualiser les traces d’une génération de test unitaire