# EdgePhoneAI: Turn any Android phone into a plug‑and‑play edge AI accelerator  
Date: 2026-11-15  
Scout  
Axe: embedded-intelligence  
Label: idea/draft  

### 1) Axe & légitimité portfolio  
EdgePhoneAI belongs to the **embedded‑intelligence** axis because it repurposes ubiquitous mobile hardware (CPU, GPU, DSP, NPU) as a ready‑to‑run inference platform for TinyML and ONNX models, demonstrating real‑world system engineering around device drivers, containerisation and hardware acceleration without designing new silicon.

### 2) Tendance sourcée  
The Hacker News post *“Why get an expensive Pi when I have an old phone?”* (15 Nov 2026, 597 points, 315 comments) highlights the growing desire to reuse existing smartphones as low‑cost compute nodes instead of buying dedicated SBCs【https://so.long.thanks.fish/why-pi-when-i-have-an-old-phone/】.  

### 3) Besoin réel du marché  
Developers and hobbyists need cheap, instantly available edge AI hardware for prototyping sensor‑fusion, vision, and audio projects. Global shipments of single‑board computers grew only 4 % YoY while >2 bn smartphones are retired each year, representing a massive untapped compute reservoir that remains inaccessible due to complex toolchains.

### 4) Problème résolu  
Today turning a phone into an edge AI node requires manually flashing custom ROMs, compiling cross‑toolchains, and configuring vendor‑specific drivers—a barrier that prevents rapid experimentation. EdgePhoneAI eliminates this friction by delivering a pre‑built, Android‑compatible container that automatically detects and uses the device’s GPU/NNP delegates for TensorFlow Lite and ONNX Runtime.

### 5) Ce qui manque aux solutions existantes  
Existing approaches (Termux, UserLAnd, Linux Deploy) provide generic Linux environments but lack:  
- Zero‑config hardware acceleration (GPU/DSP delegates);  
- One‑click model deployment and benchmarking;  
- Automatic cleanup/revert to stock state.  
EdgePhoneAI fills these gaps with a unified script that bundles the runtime, delegates, and a model‑loader UI.

### 6) Premier contrat technique  
```bash
# 1️⃣ Install host prerequisites (once)
sudo apt-get install -y android-tools-adb docker.io

# 2️⃣ Deploy and run a model (example MobilenetV2)
./edgephoneai deploy mobilenet_v2.tflite   # pushes container, triggers adb push, starts inference
```
The script performs:  
- `adb reboot bootloader && fastboot oem unlock` (if needed, user‑confirmed);  
- Launches a privileged Docker container using `docker run --privileged -v /dev/bus/usb:/dev/bus/usb`;  
- Inside the container, loads `libtensorflowlite_gpu_delegate.so` (or vendor NNP delegate) and runs a benchmark, reporting latency and FPS.  
All steps complete in under three commands, yielding a demonstrable MVP that showcases embedded‑intelligence edge AI on everyday smartphones.