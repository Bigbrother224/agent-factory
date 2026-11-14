# PhoneLLM Edge Inference Kit  
Date: 2026-11-14  
Scout  
Axe: embedded-intelligence  
Label: idea/draft  

---  

### 1) Axe & légitimité portfolio  
Ce projet réalise de l’inférence de modèles de langage sur appareil (edge‑AI) en exploitant la bibliothèque **ggml**, optimisée pour le CPU / NPU des smartphones anciens, illustrant clairement l’axe *embedded‑intelligence* du portefeuille.

### 2) Tendance sourcée  
- **Ggml v1.0.0** vient d’être publié (GitHub, 2026‑11‑14) : une implémentation légère de tensors permettant l’exécution rapide de LLM quantifiés sur CPU [source].  
- Le fil “Why get an expensive Pi when I have an old phone?” montre l’intérêt croissant à réutiliser le hardware mobile pour des charges lourdes [source].

### 3) Besoin réel du marché  
Les développeurs et les PME recherchent des assistants IA hors ligne, respectueux de la vie privée, pouvant fonctionner sans connexion cloud ni GPU coûteux, notamment sur flotte de dispositifs mobiles déjà déployés (tablettes, téléphones d’entreprise).

### 4) Problème résolu  
Permettre l’exécution locale d’un modèle LLM de 7 B paramètres quantifié en 4 bits sur un téléphone Android ≥ 8.0, avec une latence inférieure à 200 ms/token, sans installer de dépendances lourdes ni nécessiter de root.

### 5) Ce qui manque aux solutions existantes  
- Les frameworks classiques (TensorFlow Lite, PyTorch Mobile) ne supportent pas efficacement les nouveaux formats GGML quantifiés.  
- Les solutions de type “llama.cpp” nécessitent une compilation manuelle et ne sont pas packagées pour une installation en une commande sur Android.  
- Aucun kit ne fournit un binaire prêt‑à‑l’emploi accompagné d’un script de lancement simplifié.

### 6) Premier contrat technique  
- **Entrée** : texte UTF‑8 (prompt) passé via argument CLI ou pipe.  
- **Sortie** : texte généré, token par token, affiché sur stdout.  
- **Performance** : ≥ 15 tok/s sur un Snapdragon 888 (équivalent à un téléphone de 2022) avec modèle Llama‑2‑7B‑Chat‑GGML‑q4_0.  
- **Interface** : `./run.sh "<prompt>"` produit la réponse en temps réel.  
- **Déploiement** : `git clone https://github.com/yourorg/phone-llm-edge && ./setup.sh` (installe les dépendances termux et télécharge le binaire ggml + modèle) puis `./run.sh "Bonjour, comment ça va ?"` pour vérifier le fonctionnement.  

Sources: Ggml v1.0.0 release (GitHub, 2026‑11‑14) ; “Why get an expensive Pi when I have an old phone?” (HN, 2026‑11‑14).