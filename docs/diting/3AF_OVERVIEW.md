# Diting 3AF — Product Philosophy & Multi-dimensional View

This document explains **3AF** from product design philosophy in multiple dimensions, for consistent messaging, evangelism, and branding.

---

## 1. Name and homophone

- **3AF** = **AI Agent Audit & Firewall**.
- **3AF ≈ Safe**: The name sounds like **Safe**, signalling the product goal—to keep every AI Agent outbound call **safe**, auditable, and under control.

---

## 2. Multi-dimensional view (design philosophy)

### 2.1 Product positioning

| Dimension | Meaning |
|-----------|--------|
| **3AF letter meaning** | AI **A**gent **A**udit & **F**irewall — an audit and firewall layer in front of AI agents. |
| **The “3”** | Three layers of control: Identity (L0) → Policy (L1/L2) → Human confirmation (CHEQ). |
| **Homophone** | 3AF sounds like **Safe** — “make it safe.” |

One-line product shape: an **Audit + Firewall** between agents and external APIs, under zero-trust governance.

### 2.2 Zero-trust dimension

- **Never trust, always verify**: no unverified call is trusted.
- **3AF mapping**:
  - **A (Agent)**: Identify *who* is calling — L0 identity; reject if unknown.
  - **A (Audit)**: Full trail — who did what to which request, when.
  - **F (Firewall)**: Allow/deny/review by policy; no blind forwarding.

### 2.3 Capability dimension (intercept · evaluate · confirm · record)

| Stage | 3AF mapping | Description |
|-------|-------------|-------------|
| **Intercept** | Firewall | Proxy intercepts all outbound requests. |
| **Evaluate** | Firewall + Agent | L1/L2 policy (allow/deny/review) by identity, resource, action. |
| **Confirm** | Audit + human | CHEQ; audit records confirmer, decision, policy_rule_id. |
| **Record** | Audit | Full audit, trace_id decision chain, compliance and forensics. |

### 2.4 Audience and value

- **Security / compliance**: Audit for traceability and evidence; Firewall for control and policy.
- **Platform / ops**: Clear agent identity, hot-reload policy, queryable audit; ops via config + CLI + Feishu/messages.
- **Confirmers**: No separate UI; decisions in Feishu (or similar) — “message finds the person.”

### 2.5 vs WAF / traditional firewall

- **WAF**: Protects *inbound* traffic (user/browser → service).
- **3AF**: Protects *outbound* traffic (AI Agent → external APIs); object is the agent, not the browser; capabilities are **audit + policy + human-in-the-loop**, not just rule blocking.

---

## 3. One-line pitch (EN / 中文)

- **English**: *Diting 3AF is an AI Agent Audit & Firewall — it keeps every agent call **Safe** through identity, policy, human-in-the-loop, and full audit.*
- **中文**: *Diting 3AF 是面向 AI 智能体的审计与防火墙，通过身份、策略、人机协同与全量审计，让每次调用都 **Safe**（3AF 谐音安全）。*

---

## 4. Branding and doc usage

- **Full name**: Diting 3AF (谛听 3AF) — AI Agent Audit & Firewall.
- **Short**: 3AF or Diting 3AF; slogan “3AF = Safe” where appropriate.
- **Technical docs**: Keep using L0/L1/L2, CHEQ, Audit, Firewall; in overviews add one line: product shape is 3AF (Agent Audit & Firewall, homophone Safe).

---

[中文版](3AF_OVERVIEW_CN.md)
