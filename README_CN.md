# Diting (谛听)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.8+](https://img.shields.io/badge/python-3.8+-blue.svg)](https://www.python.org/downloads/)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](https://www.docker.com/)

**企业级 AI 智能体零信任治理平台**

**谛听** - 中国神话中的神兽，能辨别真假善恶，倾听世间万物之声。

[English](README.md) | [快速开始](QUICKSTART.md)

---

## 🎯 项目概述

Diting（谛听）是一个企业级 AI 安全治理平台，使用开源工具构建零信任架构，让 AI Agent 安全、可控、合规地运行。

正如神话中的谛听作为地藏菩萨的坐骑，能够辨别真假善恶，本平台也作为 AI Agent 的守护者，确保其操作安全可信。

### 核心特性

- ✅ **完全透明** - Agent 无需修改，无感知
- ✅ **无法绕过** - DNS 劫持 + 网络层拦截
- ✅ **AI 驱动** - OpenAI 意图分析，智能决策
- ✅ **全链路审计** - 每个操作可追溯，满足合规要求
- ✅ **人机协同** - 高风险操作人工审批
- ✅ **开源工具** - 基于 CoreDNS + Nginx/OpenResty，稳定可靠

---

## 🏗️ 架构

### 三层治理架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Agent 应用层                          │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │  LangChain   │  │  AutoGPT     │  │  OpenClaw    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────┐
│                  数据面 - 拦截层                             │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │         DNS 劫持 (CoreDNS)                        │     │
│  │  api.example.com → 10.0.0.1 (WAF 网关)          │     │
│  └───────────────────────────────────────────────────┘     │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │      Nginx/OpenResty 网关 (Lua)                   │     │
│  │  - 请求分析                                        │     │
│  │  - 决策执行                                        │     │
│  │  - 缓存管理                                        │     │
│  └───────────────────────────────────────────────────┘     │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │      Diting 业务逻辑 (Python/Go)                  │     │
│  │  - OpenAI 意图分析                                 │     │
│  │  - 风险评估                                        │     │
│  │  - 审批工作流                                      │     │
│  └───────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

### 前置要求

- Python 3.8+ 或 Go 1.21+
- Docker（可选，用于容器化部署）
- OpenAI API Key（或使用 Ollama 本地 LLM）

### 安装

#### Python 版本（推荐快速开始）

```bash
# 克隆仓库
git clone https://github.com/hulk-yin/diting.git
cd diting

# 安装依赖
pip install -r requirements.txt

# 启动服务
python sentinel.py
```

#### Go 版本（高性能）

```bash
# 克隆仓库
git clone https://github.com/hulk-yin/diting.git
cd diting

# 下载依赖
go mod download

# 运行服务
go run main.go
```

#### Docker 部署

```bash
# 启动所有服务
docker-compose up -d

# 或使用开源技术栈
docker-compose -f docker-compose-opensource.yml up -d
```

### 测试

```bash
# 安全请求（自动放行）
curl http://localhost:8080/get

# 危险请求（需要审批）
curl -X DELETE http://localhost:8080/delete

# 查看审计日志
cat logs/audit.jsonl
```

---

## 📦 组件说明

| 组件 | 技术 | 用途 |
|------|------|------|
| **DNS 劫持** | CoreDNS | 将所有域名路由到 WAF 网关 |
| **WAF 网关** | Nginx/OpenResty | 反向代理 + Lua 脚本 |
| **业务逻辑** | Python/Go | AI 分析 + 风险评估 |
| **LLM** | OpenAI/Ollama | 意图分析 |
| **存储** | JSONL | 审计日志 |

---

## 💡 核心功能

### 1. 智能风险评估
- 基于 HTTP 方法（GET 安全，DELETE 危险）
- 基于 URL 路径（/delete、/remove 等）
- 请求体内容分析
- 三级风险分类（低/中/高）

### 2. AI 意图分析
- 集成 OpenAI/Ollama
- 自动分析操作意图和影响
- LLM 不可用时降级到规则引擎
- 响应时间 < 2 秒

### 3. 人工审批流程
- 交互式命令行审批
- 完整上下文展示
- 批准/拒绝决策
- 可扩展到企业消息平台

### 4. 全链路审计
- JSONL 格式日志
- 完整请求/响应记录
- 决策理由和审批人追踪
- 支持事后取证分析

### 5. 零侵入部署
- 无需修改 Agent 代码
- 无需修改后端 API
- 仅需配置 DNS

---

## 📚 文档

- [快速开始指南](QUICKSTART.md) - 5 分钟上手
- [安装指南](INSTALL.md) - 详细部署说明
- [开源部署](DEPLOYMENT_OPENSOURCE.md) - 使用开源工具部署
- [架构指南](ARCHITECTURE_DNS_HIJACK.md) - DNS 劫持架构
- [eBPF 技术指南](TECHNICAL_EBPF.md) - 内核级监控
- [测试指南](TEST.md) - 测试场景和用例
- [演示脚本](DEMO.md) - 演示指南
- [贡献指南](CONTRIBUTING.md) - 如何贡献

---

## 🛠️ 开发

### 项目结构

```
diting/
├── python/                 # Python 实现
│   ├── sentinel.py         # 主服务
│   ├── sentinel_dns.py     # DNS 劫持
│   └── sentinel_ebpf.py    # eBPF 监控
│
├── cmd/diting/             # Go 主应用
│   └── main.go             # 入口点
│
├── pkg/                    # Go 包
│   ├── dns/                # DNS 劫持
│   ├── waf/                # WAF 网关
│   └── ebpf/               # eBPF 监控
│
├── deployments/            # 部署配置
│   ├── docker/             # Docker Compose
│   ├── coredns/            # CoreDNS 配置
│   └── nginx/              # Nginx 配置
│
├── docs/                   # 文档
└── scripts/                # 工具脚本
```

详细架构请查看 [STRUCTURE.md](docs/STRUCTURE.md)。

### 运行测试

```bash
# Python
python -m pytest

# Go
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

- [CoreDNS](https://coredns.io/) - DNS 服务器
- [OpenResty](https://openresty.org/) - Web 平台
- [OpenAI](https://openai.com/) - AI 模型
- [Ollama](https://ollama.ai/) - 本地 LLM 运行时

---

## 📞 联系方式

- GitHub Issues: [https://github.com/hulk-yin/diting/issues](https://github.com/hulk-yin/diting/issues)

---

## 🌟 Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=hulk-yin/diting&type=Date)](https://star-history.com/#hulk-yin/diting&Date)

---

## 🐉 关于名字

**谛听（Diting）** 是中国佛教神话中的神兽，地藏菩萨的坐骑。它拥有辨别真假善恶的超凡能力，能够倾听世间万物之声。这完美体现了我们平台的使命：以智慧和精准治理 AI Agent 的行为。

---

**用 ❤️ 打造 by Diting 团队**
