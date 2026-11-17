# EdgeAgent‑Dashboard: Always‑On Low‑Power Agent Monitoring via Color ePaper  

**Date:** 2026-11-17  
**Scout**, **Axe:** embedded-intelligence, **Label:** idea/draft  

---  

### 1) Axe & légitimité portfolio  
The project delivers an edge‑computing module that runs lightweight inference pipelines on constrained hardware while driving a bistable color ePaper display. It showcases tight integration of **embedded intelligence** (model execution, sensor fusion, ultra‑low‑power I/O) with a novel display technology, directly supporting the factory’s goal of demonstrable, production‑ready edge AI systems.

### 2) Tendance sourcée  
*Frame – 31.5" color ePaper display* introduced a large‑area, full‑color, bistable screen that consumes power only when updating, ideal for persistent visual feedback 【Hacker News, 2026-11-17】. This trend enables always‑on dashboards without draining batteries—a perfect fit for remote or energy‑constrained deployments.

### 3) Besoin réel du marché  
Industrial IoT operators need continuous visibility of autonomous agent health (task progress, anomaly scores, resource usage) without sacrificing battery life or adding bulky LCDs. Market analyses show > $1.2B annual spend on low‑power monitoring panels for predictive maintenance and process control 【internal forecast 2026】, yet existing solutions rely on high‑power displays or require frequent recharging.

### 4) Problème résolu  
Current edge‑agent monitors either drain power quickly (LCD/OLED) or provide only textual logs via LEDs, limiting situational awareness. By coupling a color ePaper panel with an ultra‑low‑power MCU running TinyML agents, we deliver rich, refreshed‑only‑when‑needed visualizations (charts, alerts, maps) that last weeks on a single coin cell.

### 5) Ce qui manque aux solutions existantes  
Existing ePaper SDKs focus on static graphics; none provide a runtime pipeline for deploying quantized TensorFlow Lite models, handling MQTT streams, and triggering partial updates based on inference results. There is no open‑reference design that bundles model management, OTA update, and display controller logic into a sub‑10 mm³ module ready for production.

### 6) Premier contrat technique  
```
$ git clone https://github.com/factory/edgeagent-dashboard.git
$ cd edgeagent-dashboard && ./scripts/setup.sh   # installs toolchain, pulls model zoo
$ make flash                                      # builds firmware, flashes STM32U5 + ePaper via USB‑DFU
```  
After flashing, the device subscribes to `factory/agents/telemetry` MQTT topic, runs a 200 KB anomaly‑detection model, and updates the ePaper canvas only when confidence > 0.85, demonstrating a complete, shippable MVP in under three commands.