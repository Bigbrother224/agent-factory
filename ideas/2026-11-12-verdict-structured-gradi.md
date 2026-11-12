# Verdict — Structured Grading Agent (Gemini Decisions API)

Date : 2026-11-12
Scout : A
Axe : ai-automation-and-systems
Label : idea/draft

1) Axe & légitimité portfolio
Illustre ai-automation-and-systems : pipeline d'agents de décision, orchestration de prompts structurés, schéma de sortie JSON imposé, et CI de cohérence des rubriques. Ce n'est pas un chatbot, mais un moteur de décision agente intégré à un workflow reproductible.

2) Tendance sourcée
HN front page 2026-11-12 : "Gemini Decisions API" (530 pts) et "Socratic Tutor" de Jane Street (534 pts) confirment le passage des agents conversationnels aux agents de décision structurée (verdict, rubrique, traçabilité).

3) Besoin réel du marché
Éditeurs EdTech, organismes de certification et RH doivent noter des productions ouvertes (dissertations, code, rédactions) à grande échelle, avec justification auditable, coût prévisible et conformité IA Act. Le marché de l'évaluation automatisée croît, mais les outils actuels renvoient des notes "boîte noire".

4) Problème résolu
Verdict transforme une copie et une rubrique en note justifiée, référence texte, score de confiance et JSON stable. Démo en moins de 3 commandes sur un lot de copies, sans appel LLM non structuré.

5) Ce qui manque aux solutions existantes
Les LLM-as-judge retournent du texte libre difficilement parsable ; aucun outil dominant n'offre schéma de décision formel, traçabilité par critère et promotion de rubrique via CI/CD.

6) Premier contrat technique
pip install verdict-agent && verdict init --rubric rubric.yaml && verdict grade submissions/ --api gemini-decisions --out results.json
MVP : CLI Python, appel Gemini Decisions API, schéma JSON de décision, cache des réponses, rapport HTML. Démonstrable en une commande après installation.

Sources : Hacker News front page, 2026-11-12 (Gemini Decisions API, 530 pts ; Socratic Tutor, 534 pts).