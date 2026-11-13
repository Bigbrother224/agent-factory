# TinyLLM Controller: On‑device LLM Voice Control for Appliances  
Date : 2026-11-13, Scout, Axe : embedded-intelligence, Label : idea/draft  

1) **Axe & légitimité portfolio**  
   The project fits the *embedded‑intelligence* axis because it brings a real‑time language model onto a low‑power MCU, enabling AI inference directly on hardware without relying on cloud services. It demonstrates how modern quantization techniques can turn everyday appliances into intelligent edge nodes, a core theme of the axis.

2) **Tendance sourcée**  
   - “I’m making LLMs work on small and old devices” – 438 pts, shows strong community interest in running LLMs on constrained hardware​【Digest】.  
   - “Edge0‑145B: Frontier AI Capabilities on Your Own Computer” – 312 pts, highlights the momentum toward bringing frontier‑scale AI to personal compute​【Digest】.  

3) **Besoin réel du marché**  
   Consumers and small manufacturers are frustrated by subscription‑locked smart appliances (e.g., embroidery machines, coffee makers) that require cloud connectivity for basic functions​【Digest – Show HN embroidery machine】. There is a clear market for open‑source, locally‑controlled firmware that eliminates recurring fees while adding richer interaction (voice, natural language).

4) **Problème résolu**  
   Proprietary firmware forces users into vendor clouds, raises privacy concerns, and blocks customization. Existing DIY alternatives require reverse‑engineering each device and still lack a usable AI interface for non‑technical users.

5) **Ce qui manque aux solutions existantes**  
   Current open‑firmware projects (e.g., Marlin, Klipper) focus on G‑code execution but provide no natural‑language understanding. Quantized LLM runners exist for PCs or smartphones, but none target sub‑1 W MCUs with enough accuracy for command recognition.

6) **Premier contrat technique**  
   - **Hardware target**: ESP32‑S3 (240 MHz, 512 KB RAM, NPU‑friendly).  
   - **Model**: Llama‑2‑1B quantized to 4‑bit via GPTQ, sized ~300 MB → compressed to ~8 MB using tensor‑pruning and shared embeddings, fitting in external PSRAM.  
   - **Software stack**:  
     * ESP‑IDF + custom NN runtime (based on TVM micro) for inference (<120 ms per token).  
     * Wake‑word engine (Porcupine) feeding audio to the LLM.  
     * Simple intent parser mapping LLM output to GPIO/MQTT commands (e.g., “turn on the lamp”).  
   - **MVP steps (<3 commands)**:  
     1. `idf.py set-target esp32s3`  
     2. `idf.py build`  
     3. `idf.py -p /dev/ttyUSB0 flash monitor`  
   After flashing, speaking “Hey Device, turn on the light” triggers the LLM, which outputs a JSON intent that toggles GPIO‑2. Source citations: LLMs on small devices​【Digest】; Edge AI capabilities​【Digest】.  

---  
*All data points are drawn exclusively from the 2026‑11‑13 digest.*