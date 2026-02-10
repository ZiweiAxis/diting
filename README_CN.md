# Diting 3AF (谛听 3AF)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](https://www.docker.com/)

**Diting 3AF：AI Agent 审计与防火墙（企业级智能体零信任治理平台）**

**谛听** - 中国神话中的神兽，能辨别真假善恶，倾听世间万物之声。本平台作为 AI Agent 的守护者，确保其操作安全可信。

**状态：** MVP / 概念验证 — 适合试用与反馈；尚未达到生产就绪。

[English](README.md) | [快速开始](#-快速开始) | [安全](SECURITY.md)

---

## 🎯 项目概述

Diting 3AF（谛听 3AF）是一个企业级 AI 安全治理平台，本质上是面向 AI Agent 的**审计与防火墙（AI Agent Audit & Firewall）**：通过智能反向代理拦截和治理 AI Agent 的 API 调用，让 AI Agent 安全、可控、合规地运行。

### 核心特性

- ✅ **动态 API 代理** - 拦截 AI Agent 的任意外部 API 调用
- ✅ **零侵入** - 无需修改 Agent 代码
- ✅ **AI 驱动** - OpenAI/Ollama 意图分析，智能决策
- ✅ **风险评估** - 三级风险分类（低/中/高）
- ✅ **人机协同** - 高风险操作人工审批
- ✅ **全链路审计** - 每个操作可追溯，满足合规要求
- ✅ **高性能** - Go 语言构建，处理 2000+ req/s

### 3AF 是什么？—— 多维度解读

**3AF** 即 **AI Agent Audit & Firewall**（AI 智能体审计与防火墙）。读音谐音 **Safe**（安全）—— 目标就是让 AI 智能体的每一次出网调用都**安全、可审计、可控**。

| 维度 | 含义 |
|------|------|
| **产品名** | **3**AF = **A**I **A**gent **A**udit & **F**irewall：在智能体与外部 API 之间的审计与防火墙。 |
| **设计** | **3** 层控制：L0 身份 → L1/L2 策略 → CHEQ 人工确认；**A**udit 全量审计；**F**irewall 按策略放行/拒绝。 |
| **使命** | **Safe**（谐音）：让智能体流量 **Safe** —— 可追溯（审计）、可管控（防火墙）、身份可知（Agent）。 |
| **零信任** | 永不信任、始终校验：识别 Agent（A）、执行策略（F）、全程留痕（A）。 |

更多：[3AF — 产品理念与多维度解读](docs/diting/3AF_OVERVIEW_CN.md)（[English](docs/diting/3AF_OVERVIEW.md)）。

---

## 🏗️ 架构

### 简洁而强大

```
┌─────────────────────────────────────────────────────────┐
│                   AI Agent                               │
│                                                          │
│  requests.get('https://api.openai.com/chat')        │
│  requests.post('https://api.github.com/repos')      │
│  requests.delete('https://api.stripe.com/data')     │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ 所有 HTTP/HTTPS 请求
                 ▼
┌─────────────────────────────────────────────────────────┐
│              Diting 治理网关                             │
│                                                          │
│  1. 拦截所有 API 调用                                    │
│  2. 风险评估（方法/路径/内容）                           │
│  3. AI 意图分析（Ollama/OpenAI）                        │
│  4. 人工审批（仅高风险）                                 │
│  5. 审计日志（完整追踪）                                 │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ 转发（如果批准）
                 ▼
┌─────────────────────────────────────────────────────────┐
│              外部 API                                    │
│                                                          │
│  OpenAI, GitHub, Stripe, 任意 SaaS API...           │
└─────────────────────────────────────────────────────────┘
```

**为什么选择 Go？**
- 原生支持动态反向代理
- 自动 DNS 解析和连接池管理
- 内置 HTTPS/TLS 处理
- 高性能（2000+ req/s）
- 单一二进制部署

---

## 🚀 快速开始

### 前置要求

- Go 1.21+（从源码构建）
- Docker（可选，容器化部署）
- Ollama（可选，本地 LLM 分析）

### 安装

#### 方式 1：从源码运行

```bash
# 克隆仓库
git clone https://github.com/hulk-yin/diting.git
cd diting/cmd/diting

# 下载依赖
go mod download

# 运行服务
go run main.go
```

#### 方式 2：构建二进制

```bash
cd diting/cmd/diting

# 构建
go build -o diting main.go

# 运行
./diting
```

#### 方式 2b：飞书审批（推荐用于人机协同）

高风险操作需飞书审批时（无需公网回调或飞书「长连接」）：

```bash
cd diting/cmd/diting

# 构建飞书消息回复审批版
go build -o diting main.go

# 配置 config.json（feishu.approval_user_id、use_message_reply: true、poll_interval_seconds）
# 然后运行
./diting
```

配置与最小验证步骤见 **[cmd/diting/QUICKSTART_CN.md](cmd/diting/QUICKSTART_CN.md)**（[English](cmd/diting/QUICKSTART.md)）。

#### 方式 3：Docker 部署

```bash
cd diting/deployments/docker
docker-compose up -d
```

### 测试

```bash
# 配置 AI Agent 使用 Diting 作为代理
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# 安全请求（自动放行）
curl http://localhost:8080/get

# 危险请求（需要审批）
curl -X DELETE http://localhost:8080/delete

# 查看审计日志
cat logs/audit.jsonl
```

---

## 📦 项目结构

```
diting/
├── cmd/diting/             # 主应用
│   ├── main.go             # 入口点
│   ├── go.mod              # Go 模块
│   └── README.md
│
├── pkg/                    # 可复用包（未来）
│   ├── dns/                # DNS 工具
│   ├── waf/                # WAF 工具
│   └── ebpf/               # eBPF 监控（未来）
│
├── deployments/            # 部署配置
│   ├── docker/             # Docker Compose
│   └── kubernetes/         # K8s 清单（未来）
│
├── docs/                   # 文档
│   ├── QUICKSTART.md
│   ├── INSTALL.md
│   └── ...
│
└── scripts/                # 工具脚本
```

---

## 💡 核心功能

### 1. 动态 API 代理

与传统反向代理（Nginx）需要固定上游配置不同，Diting 动态处理任意外部 API：

```go
// 自动处理任意目标
requests.get('https://api.openai.com/chat')      // ✅ 支持
requests.post('https://api.github.com/repos')    // ✅ 支持
requests.delete('https://random-api.com/data')   // ✅ 支持
```

### 2. 智能风险评估

- **HTTP 方法**：GET（安全）vs DELETE（危险）
- **URL 路径**：`/delete`、`/remove`、`/drop`（高风险）
- **请求体**：危险关键词检测
- **三级分类**：低 / 中 / 高

### 3. AI 意图分析

- 集成 Ollama（本地 LLM）或 OpenAI
- 自动分析操作意图和影响
- LLM 不可用时降级到规则引擎
- 响应时间 < 2 秒

### 4. 人工审批流程

- 高风险操作交互式 CLI 审批
- 完整上下文展示（方法、路径、分析）
- 批准/拒绝决策
- 可扩展到企业消息平台

### 5. 全链路审计

```json
{
  "timestamp": "2026-02-08T00:20:00Z",
  "method": "DELETE",
  "path": "/api/users/123",
  "risk_level": "高",
  "intent_analysis": "意图: 删除用户数据...",
  "decision": "ALLOW",
  "approver": "admin",
  "duration_ms": 1850
}
```

---

## 📚 文档

- [快速开始指南（中文）](cmd/diting/QUICKSTART_CN.md) / [Quick Start (EN)](cmd/diting/QUICKSTART.md) - 5 分钟上手
- [飞书排错（中文）](cmd/diting/FEISHU_TROUBLESHOOTING_CN.md) - 收不到审批消息时排查
- [安装指南](docs/INSTALL.md) - 详细部署说明
- [架构指南](docs/ARCHITECTURE_DNS_HIJACK.md) - 系统架构
- [测试指南](docs/TEST.md) - 测试场景和用例
- [演示脚本](docs/DEMO.md) - 演示指南
- [贡献指南](CONTRIBUTING.md) - 如何贡献

---

## 🛠️ 开发

### 构建

```bash
cd cmd/diting
go build -o diting main.go
```

### 交叉编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o diting-linux main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o diting.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o diting-mac main.go
```

### 运行测试

```bash
go test ./...
```

---

## 🤝 贡献

我们欢迎贡献！请查看我们的[贡献指南](CONTRIBUTING.md)了解详情。

### 如何贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

---

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## 🙏 致谢

- [Go](https://golang.org/) - 编程语言
- [Ollama](https://ollama.ai/) - 本地 LLM 运行时
- [OpenAI](https://openai.com/) - AI 模型

---

## 📞 联系方式

- GitHub Issues: [https://github.com/hulk-yin/diting/issues](https://github.com/hulk-yin/diting/issues)

---

## 🌟 Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=hulk-yin/diting&type=Date)](https://star-history.com/#hulk-yin/diting&Date)

---

## 🐉 关于名字

**谛听（Diting）** 是中国佛教神话中的神兽，地藏菩萨的坐骑。它拥有辨别真假善恶的超凡能力，能够倾听世间万物之声。这完美体现了我们平台的使命：以智慧和精准治理 AI Agent 的行为。

**3AF** 谐音 **Safe**（安全）：我们希望通过审计（可追溯）、防火墙（可控）、智能体身份（可知）让每一次调用都 **Safe**。

---

**用 ❤️ 打造 by Diting 团队**
