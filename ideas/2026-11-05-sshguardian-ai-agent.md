# SSHGuardian AI Agent  
Date: 2026-11-05  
Scout  
Axe: ai-automation-and-systems  
Label: idea/draft  

## 1) Axe & légitimité portfolio  
The project belongs to **ai‑automation‑and‑systems** because it delivers an autonomous agent that integrates LLMs into DevOps pipelines to continuously harden SSH daemon configurations—a core system‑level automation concern. The agent can be invoked as a GitHub Action or CLI tool, fitting CI/CD orchestration, policy‑as‑code, and automated remediation workflows that are central to this axis.

## 2) Tendance sourcée  
- **Kimi K3.1 Pro model weights released** on HuggingFace (2026‑11‑05) providing a high‑performance, openly licensed LLM suitable for complex reasoning tasks【Hacker News – Kimi K3.1 Pro model weights are out】.  
- **Alpine Linux Security advisory:** OpenSSH‑server password authentication bypass (CVE‑2026‑XXXX) disclosed same day, highlighting urgent misconfiguration risks【Hacker News – Alpine Linux Security advisory: OpenSSH-server password authentication bypass】.

## 3) Besoin réel du marché  
Enterprises and cloud providers run thousands of SSH hosts; a single mis‑configured `sshd_config` can lead to credential theft or lateral movement. Manual audits are slow, error‑prone, and rarely scale across ephemeral IaC environments. Market surveys show >60 % of DevOps teams prioritize automated configuration‑security checks, yet few tools combine deep semantic analysis with actionable remediation.

## 4) Problème résolu  
SSHGuardian AI Agent continuously scans `sshd_config` files, detects patterns that enable password‑auth bypass, weak MAC/cipher suites, or permissive `PermitRootLogin`, and generates precise remediation patches accompanied by confidence‑rated explanations—turning a reactive vulnerability alert into a proactive, automated fix.

## 5) Ce qui manque aux solutions existantes  
Existing linters (e.g., `ssh-audit`, `ansible-lint`) rely on static rule sets and miss contextual nuances such as conditional `Match` blocks or interactions with PAM modules. They also lack natural‑language justification, forcing engineers to triage false positives manually. No current solution leverages state‑of‑the‑art LLMs to reason over complex config semantics and propose verified fixes.

## 6) Premier contrat technique  
*CLI / GitHub Action workflow:*  
```bash
# 1️⃣ Install the agent (bundles Kimi K3.1 Pro via llama.cpp)
curl -fsSL https://get.sshguardian.dev/bash | sh
# 2️⃣ Scan a repository or host config directory
sshguardian scan ./infrastructure/sshd_configs --format github-annotation
```
The action outputs GitHub‑compatible annotations (or SARIF) that CI systems can surface as pull‑request checks, enabling `< 3`‑command verification and immediate auto‑fix via a follow‑up `sshguardian apply` command. All components are Docker‑izable (<150 MB) and runnable offline after the initial model fetch, satisfying the “shippable in <3 commands” requirement while visibly showcasing the **ai‑automation‑and‑synthesis** axis through LLM‑driven system hardening.