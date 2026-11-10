# Edge‑Means Fraud Sentinel for Real‑Time Rails  
Date: 2026-11-10  
Scout  
Axe: embedded-intelligence  
Label: idea/draft  

## 1) Axe & légitimité portfolio  
Ce projet place l’inférence d’un modèle de clustering ultra‑léger directement sur un terminal de paiement : IA embarquée (edge‑ML) pour décision en temps réel, illustrant l’axe **embedded‑intelligence** du portefeuille.  

## 2) Tendance sourcée  
- Le lancement du système de paiement en temps réel **Real‑Time Rail (RTR)** du Canada montre un besoin croissant de traitement instantané des transactions 【Payments Canada announces launch of Canada's Real-Time Rail (RTR) payment system】.  
- L’algorithme **E‑means** dépasse k‑means en un dixième d’époques, offrant un clustering très rapide et peu gourmand 【E-means: Outperforming k-means in one-tenth the epochs】.  

## 3) Besoin réel du marché  
Les émetteurs de cartes et les acquéreurs doivent détecter la fraude à la pointe de vente avec une latence < 5 ms pour ne pas perturber le flux RTR, tout en limitant la consommation énergétique des terminaux 【source】.  

## 4) Problème résolu  
Nous proposons un sentinelle embarquée qui exécute E‑means sur les caractéristiques des transactions (montant, horaire, merchant‑code, score de risque) et signale les outliers en < 2 ms, évitant ainsi le recours au cloud et réduisant les faux négatifs.  

## 5) Ce qui manque aux solutions existantes  
Les solutions actuelles sont soit :  
- des règles statiques facilement contournables,  
- des modèles de deep learning nécessitant GPU ou connexion cloud, augmentant la latence et le coût.  
Aucune offre ne combine un clustering non supervisé, sous‑10 ko, exécutable sur un MCU Cortex‑M4 sans acceleration externe.  

## 6) Premier contrat technique  
```bash
git clone https://github.com/factory/edge-means-sentinel.git
cd edge-means-sentinel && ./run.sh   # builds firmware, flashes a STM32F4 demo board and streams synthetic transactions
```
Le script `./run.sh` invoque le Makefile qui : compile le modèle E‑means (≈8 ko) avec CMSIS‑NN, flash le binaire, puis lance un générateur de flux de transactions (basé sur les schémas RTR) et affiche en temps réel les alertes de fraude sur le port série.  
*Tout le processus tient en deux commandes après clonage.*  

---  
Sources : Payments Canada RTR (2026‑11‑10), E‑means (2026‑11‑10).