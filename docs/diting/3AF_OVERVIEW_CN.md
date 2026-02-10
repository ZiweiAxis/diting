# Diting 3AF — 产品理念与多维度解读

本文从产品设计理念出发，对 **3AF** 做多维度解释，便于对外沟通、布道与品牌统一。

---

## 1. 名称与谐音

- **3AF** = **AI Agent Audit & Firewall**（AI 智能体审计与防火墙）。
- **3AF ≈ Safe**：读音谐音 **Safe**，直接传达产品目标——让 AI Agent 的每一次出网调用都 **Safe**（安全、可审计、可控）。

---

## 2. 多维度解读（设计理念）

### 2.1 产品定位维度

| 维度 | 英文 | 中文 |
|------|------|------|
| **3AF 字母含义** | AI **A**gent **A**udit & **F**irewall | 面向 AI 智能体的**审计**与**防火墙** |
| **3 的体现** | Three layers: Identity → Policy → Human confirmation | 三层控制：身份 → 策略 → 人工确认 |
| **谐音** | 3AF sounds like **Safe** | 3AF 谐音 **Safe**，即「安全」 |

产品形态一句话：在 Agent 与外部 API 之间插入一道**审计 + 防火墙**，实现零信任治理。

### 2.2 零信任维度

- **Never trust, always verify**：不信任任何未校验的调用。
- **3AF 与零信任的对应**：
  - **A (Agent)**：明确「谁在调用」—— L0 身份识别，未识别则拒绝。
  - **A (Audit)**：全程留痕，可追溯「谁在何时对哪条请求做了何种决策」。
  - **F (Firewall)**：按策略放行/拒绝/进入人工确认，不越权、不盲放。

### 2.3 能力维度（拦·评·确·记）

| 环节 | 3AF 对应 | 说明 |
|------|----------|------|
| **拦** | Firewall | 代理拦截所有出网请求，不放过未治理流量。 |
| **评** | Firewall + Agent | L1/L2 策略评估，结合 Agent 身份与资源/动作做 allow/deny/review。 |
| **确** | Audit + 人 | CHEQ 人工确认；审计记录 confirmer、decision、policy_rule_id。 |
| **记** | Audit | 全量审计，trace_id 决策链，满足合规与溯源。 |

### 2.4 受众与价值维度

- **安全/合规**：Audit 满足「可追溯、可举证」；Firewall 满足「可控、可配置策略」。
- **平台/运维**：Agent 身份清晰、策略可热加载、审计可查，运维界面是配置 + CLI + 飞书/消息。
- **业务/确认人**：不需要打开传统软件界面，在飞书等习惯通道里收到审批卡片即可决策，体验是「消息找到人」。

### 2.5 与 WAF / 传统防火墙的区分

- **WAF**：主要防护「人/浏览器 → 服务」的入站流量。
- **3AF**：防护「AI Agent → 外部 API」的**出站**流量，对象是智能体而非用户浏览器；能力是**审计 + 策略 + 人机协同**，而非仅规则拦截。

---

## 3. 一句话对外表述（中英）

- **英文**：*Diting 3AF is an AI Agent Audit & Firewall — it keeps every agent call **Safe** through identity, policy, human-in-the-loop, and full audit.*
- **中文**：*Diting 3AF 是面向 AI 智能体的审计与防火墙，通过身份、策略、人机协同与全量审计，让每次调用都 **Safe**（3AF 谐音安全）。*

---

## 4. 品牌与文档中的用法建议

- **全称**：Diting 3AF（谛听 3AF）— AI Agent Audit & Firewall。
- **简称**：3AF 或 Diting 3AF；可强调「3AF = Safe」用于 slogan 或副标题。
- **技术文档**：继续使用 L0/L1/L2、CHEQ、Audit、Firewall 等术语；在概述或 README 中可加一句「产品形态即 3AF：Agent Audit & Firewall，谐音 Safe」。

---

[English](3AF_OVERVIEW.md)
