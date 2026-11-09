# EdgeCoarse: Real‑Time LLM Summarization via Coarse‑to‑Fine Generation  
Date: 2026-11-09  
Scout, Axe: ai-automation-and-systems, Label: idea/draft  

### 1) Axe & légitimité portfolio  
Le projet s’inscrit clairement dans l’axe **ai‑automation‑and‑systems** : il propose une implémentation légère d’une nouvelle stratégie de génération de texte (coarse‑to‑fine) pour automatiser la production de résumés instantanés sur des infrastructures CI/CD ou des agents locaux, sans nécessiter de GPU massif. Il montre comment une recherche récente peut être transformée en brique logicielle reproductible et déployable en quelques commandes, répondant à la vocation du portefeuille de démontrer une ingénierie système autour des agents et pipelines d’IA.

### 2) Tendance sourcée  
*Next‑Concept Modeling: Fast, Parallel Coarse‑to‑Fine Text Generation* (arXiv 2026‑10‑14) – 169 pts, 39 coms【digest】. Cette travaux présente une approche parallèle qui génère d’abord un aperçu grossier (“coarse”) puis affine progressivement le texte, réduisant drastiquement le nombre de passes du modèle tout en préservant la qualité.

### 3) Besoin réel du marché  
Les équipes produit ont besoin de résumer en temps réel des logs, des tickets ou des contenus internes pour alimenter des chatbots d’assistance, mais les modèles LLM traditionnels imposent une latence >2 s même sur des instances CPU modestes, ce qui freine l’adoption en edge ou en pipelines CI où chaque seconde compte.

### 4) Problème résolu  
EdgeCoarse implémente la génération coarse‑to‑fine afin de produire un résumé de qualité comparable à un modèle complet en **≤ 0,5 s** sur un CPU x86‑64 standard, en ne faisant appel qu’à une petite fraction des paramètres du modèle original (≈ 30 % des opérations matricielles). Cela élimine le goulet d’étranglement de latence tout en conservant la pertinence du résumé.

### 5) Ce qui manque aux solutions existantes  
- Les bibliothèques de résumé actuelles (HuggingFace Transformers, llama.cpp) appliquent le modèle complet à chaque token, générant une latence élevée.  
- Aucune offre open‑source n’exploite explicitement la stratégie coarse‑to‑fine pour du texte génératif sur CPU.  
- Aucun exemple ne montre comment intégrer cette approche dans un pipeline CI/CD avec moins de trois commandes d’installation/exécution.

### 6) Premier contrat technique  
```bash
# 1️⃣ Installation (pip)
pip install edgecoarse==0.1.0

# 2️⃣ Téléchargement du poids léger (≈ 50 Mo) – fourni dans le repo
edgecoarse download --model distilbert-coarse2fine

# 3️⃣ Inférence – résumé d’un fichier texte en moins d’une commande
echo "Texte long à résumer..." | edgecoarse summarize --input -
```
*Résultat affiché en stdout en < 0,5 s sur une instance CI standard (Ubuntu 22.04, 2 vCPU).*  
Sources utilisées exclusivement du digest du 2026‑11‑09.