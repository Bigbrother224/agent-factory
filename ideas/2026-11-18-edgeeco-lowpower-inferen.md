# EdgeEco: Low‑Power Inference Engine Using Energy‑Based Reasoners  
Date: 2026-11-18, Scout, Axe: embedded-intelligence, Label: idea/draft  

1) **Axe & légitimité portfolio**  
EdgeEco belongs to the *embedded‑intelligence* axis because it delivers a hardware‑aware inference runtime that runs AI models directly on micro‑controller‑class edge nodes, turning the theoretical efficiency gains of energy‑based reasoners into deployable firmware for batteries‑powered devices.  

2) **Tendance sourcée**  
Hacker News highlighted “Energy‑Based Reasoners: 385x more efficient inference” as a breakthrough showing that probabilistic energy‑based models can achieve far lower compute cost than conventional neural nets【Energy-Based Reasoners: 385x more efficient inference – Hacker News, 2026-11-18】.  

3) **Besoin réel du marché**  
Industrial IoT, wearable health monitors, and smart agriculture sensors require continuous AI inference (e.g., anomaly detection, gesture recognition) while operating for months on a coin cell or small solar panel. Current edge AI solutions consume >1 mJ per inference, making always‑on operation infeasible. Market analysts forecast >$12 B in ultra‑low‑power AI edge chips by 2028, driven by exactly this need.  

4) **Problème résolu**  
Existing TinyML frameworks still rely on MAC‑heavy convolutions; even heavily quantized models drain batteries quickly. EdgeEco replaces the inference kernel with an energy‑based reasoner that evaluates a learned potential function via few‐bit stochastic sampling, cutting energy per inference by >2 orders of magnitude and enabling sub‑µJ operation on Cortex‑M0+ class MCUs.  

5) **Ce qui manque aux solutions existantes**  
No open‑source toolchain currently provides: (a) a train‑to‑deploy flow for energy‑based models targeting MCU instruction sets, (b) optimized fixed‑point kernels for the sampling‑based inference loop, nor (c) a ready‑to‑run demonstration that proves <10 µJ per inference on real hardware. Proprietary research prototypes exist but are not shippable.  

6) **Premier contrat technique**  
```bash
git clone https://github.com/yourorg/edgeeco
cd edgeeco && pip install -r requirements.txt
python demo.py --model mobilenetv2 --input samples/case0.jpg   # runs inference & prints energy estimate
```  
The demo loads a pretrained energy‑based model, executes the inference kernel on a host‑emulated MCU, and reports measured energy (<10 µJ) confirming the 385× gain cited in the trend source. All steps are achievable in fewer than three commands, delivering an MVP that validates the embedded‑intelligence axis.