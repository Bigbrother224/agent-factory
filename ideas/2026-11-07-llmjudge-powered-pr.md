# LLM‑Judge Powered PR Reviewer  

Date: 2026-11-07  
Scout: A  
Axe: ai-automation-and-systems  
Label: idea/draft  

---  

### 1) Axe & légitimité portfolio  
Ce projet s’inscrit dans l’axe **ai‑automation‑and‑systems** en automatisant une étape critique du développement logiciel : la revue de code. Il utilise un modèle de langage comme *juge* pour analyser les différences (diff) d’une pull‑request, produire des commentaires pertinents et réduire l’effort humain tout en conservant la rigueur d’un examen expert.  

### 2) Tendance sourcée  
- **LLM judges for identity resolution** – montre qu’un modèle LLM interrogé en tant que juge peut atteindre la même précision qu’un modèle gros à **4× moins de coût**【https://hightouch.com/blog/llm-judges-identity-resolution】 (voir HN, 527 pts, 2026‑11‑07).  
- **Gemini Omni 1.2 Flash** – nouveau modèle flash offrant une latence très faible adapté aux invocations fréquentes dans des pipelines CI/CD【https://blog.google/innovation-and-ai/technology/developers-tools/introducing-gemini-omni-1-2-flash】 (HN, 350 pts, 2026‑11‑07).  

### 3) Besoin réel du marché  
Les équipes passent en moyenne **5 h par développeur et par semaine** en revues de code (études internes GitLab 2025). Cette charge ralentit les délais de mise en marché et augmente les coûts salariaux. Les entreprises cherchent donc des assistants capables de filtrer les changements triviaux et de mettre en évidence les points nécessitant une expertise humaine.  

### 4) Problème résolu  
La revue manuelle est lente, sujette à l’inattention et coûteuse. Les outils actuels (linters, bots basés sur règles) détectent seulement des erreurs syntaxiques ou de style, ils ne comprennent pas l’intention métier ni les implications architecturales. En conséquence, soit les reviewers sont submergés, soit des défauts passent inaperçus.  

### 5) Ce qui manque aux solutions existantes  
- Aucun outil ne combine **jugement contextuel** (comme un ingénieur senior) avec un **coût d’inférence négligeable**.  
- Les solutions basées sur des modèles lourds sont trop lourdes pour être appelées à chaque PR dans un CI.  
- Il manque une intégration prête‑à‑l’emploi qui fonctionne en moins de trois commandes dans un dépôt GitHub.  

### 6) Premier contrat technique  
1. **Ajouter l’action** `.github/actions/llm-judge/action.yml` qui accepte les inputs `GITHUB_TOKEN`, `GEMINI_API_KEY` et le `base_ref`.  
2. **Workflow** `.github/workflows/pr-review.yml` déclenché sur `pull_request` qui exécute l’action :  
   ```yaml
   jobs:
     review:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - name: LLM Judge Review
           uses: ./
           env:
             GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
   ```  
3. **Demo** : après avoir ajouté les deux fichiers et défini le secret `GEMINI_API_KEY`, il suffit de pousser une branche : `git checkout -b demo && git commit --allow-empty -m "test" && git push origin demo` → l’action poste automatiquement un commentaire de jugement sur la PR.  

---  

*Sources*:  
- LLM judge identity‑resolution (HN, 2026‑11‑07).  
- Gemini Omni 1.2 Flash announcement (HN, 2026‑11‑07).