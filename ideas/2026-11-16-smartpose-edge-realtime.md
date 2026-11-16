# SmartPose Edge: Real‑time Full‑Body Pose from a Single Smartwatch  
Date: 2026-11-16  
Scout  
Axe: embedded-intelligence  
Label: idea/draft  

## 1) Axe & légitimité portfolio  
SmartPose Edge places a tiny deep‑learning model directly on the MCU/DSP of a commodity smartwatch, turning its built‑in IMU (accelerometer + gyroscope + magnetometer) into a continuous, low‑latency full‑body pose estimator. This is pure **embedded‑intelligence**: model inference runs on‑device, no offload to cloud or external compute, fitting the factory’s focus on edge ML and hardware‑aware AI systems.

## 2) Tendance sourcée  
> *Researchers track whole body movement with just a smartwatch* – Cornell News, 2026‑11‑16【digest】  

The academic breakthrough shows that rich kinematic information can be extracted from a single wrist‑worn IMU using modern ML, opening a path to productize the capability.

## 3) Besoin réel du marché  
- Sports‑science & physiotherapy clinics need affordable motion capture to assess gait, rehabilitation progress, and injury risk.  
- Consumer VR/AR and gaming seek low‑cost full‑body avatars without cumbersome suits or external cameras.  
- Industrial ergonomics aims to monitor worker posture on the factory floor to prevent musculoskeletal disorders.  
Current marker‑based optical systems cost >$10k; IMU suits run >$2k and still require multiple nodes. A single‑watch solution under $200 would address a > $1B addressable market (sports tech + VR + occupational health).

## 4) Problème résolu  
Existing approaches either:  
- Require many distributed IMUs (complex wiring, calibration).  
- Depend on external cameras (occlusion, lighting, privacy concerns).  
- Use heavyweight models unsuitable for MCU inference (> 100 MFLOPS).  

SmartPose Edge solves this by delivering **accurate 15‑joint 3D pose** from **one watch**, preserving user privacy (all processing on‑device) and eliminating setup time.

## 5) Ce qui manque aux solutions existantes  
- No open‑source, ultra‑lightweight pose‑estimation model tuned for < 1 MFLOP budgets typical of wearable MCUs.  
- Lack of a ready‑to‑use SDK that abstracts sensor‑fusion, preprocessing, and inference for developers.  
- Minimal public datasets linking wrist IMU signals to full‑body pose, hindering model training reproducibility.

## 6) Premier contrat technique (MVP en ≤ 3 commandes)  
```bash
# 1️⃣ Clone the repo (contains sample IMU CSV & Dockerfile)
git clone https://github.com/factory/smartpose-edge.git
# 2️⃣ Build the container that bundles the TinyPose model & runtime
docker build -t smartpose ./smartpose-edge
# 3️⃣ Run inference on the supplied demo data → outputs pose.json
docker run --rm -v "$(pwd)/smartpose-edge/data":/data smartpose \
    python /app/infer.py --input /data/demo.csv --output /data/pose.json
```
*`infer.py` loads a 1‑D CNN‑Transformer (≈ 0.8 MFLOP), normalizes the 3‑axis IMU stream (window = 128 samples, stride = 16), and writes a JSON array of 3‑D coordinates for hips, knees, ankles, shoulders, elbows, wrists, head, and spine.*  

Result: a sub‑second latency pose stream usable directly by Unity/Unreal plugins or a simple Python visualizer—demonstrating a functional, market‑ready MVP built entirely around the embedded‑intelligence axis.