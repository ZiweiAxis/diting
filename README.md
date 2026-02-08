# Diting (谛听)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](https://www.docker.com/)

**Enterprise-grade AI Agent Zero-Trust Governance Platform**

**谛听 (Diting)** - A mythical creature in Chinese mythology that can distinguish truth from falsehood, good from evil. This platform acts as a guardian for AI agents, ensuring their operations are safe and trustworthy.

**Status:** MVP / concept validation — suitable for trial and feedback; not yet production-ready.

[中文文档](README_CN.md) | [Quick Start](#-quick-start) | [Security](SECURITY.md)

---

## 🎯 Overview

Diting is an enterprise-grade AI security governance platform that intercepts and governs AI Agent API calls through intelligent reverse proxy, enabling AI Agents to run securely, controllably, and compliantly.

### Key Features

- ✅ **Dynamic API Proxy** - Intercepts any external API calls from AI agents
- ✅ **Zero Intrusion** - No agent code modification required
- ✅ **AI-Driven Analysis** - OpenAI/Ollama intent analysis with intelligent decision-making
- ✅ **Risk Assessment** - Three-tier risk classification (low/medium/high)
- ✅ **Human-in-the-Loop** - Manual approval for high-risk operations
- ✅ **Full Audit Trail** - Every operation is traceable for compliance
- ✅ **High Performance** - Built with Go, handles 2000+ req/s

---

## 🏗️ Architecture

### Simple & Powerful

```
┌─────────────────────────────────────────────────────────┐
│                   AI Agent                               │
│                                                          │
│  requests.get('https://api.openai.com/chat')        │
│  requests.post('https://api.github.com/repos')      │
│  requests.delete('https://api.stripe.com/data')     │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ All HTTP/HTTPS requests
                 ▼
┌─────────────────────────────────────────────────────────┐
│              Diting Governance Gateway                   │
│                                                          │
│  1. Intercept all API calls                             │
│  2. Risk assessment (method/path/content)               │
│  3. AI intent analysis (Ollama/OpenAI)                  │
│  4. Human approval (high-risk only)                     │
│  5. Audit logging (full trail)                          │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ Forward (if approved)
                 ▼
┌─────────────────────────────────────────────────────────┐
│              External APIs                               │
│                                                          │
│  OpenAI, GitHub, Stripe, any SaaS APIs...           │
└─────────────────────────────────────────────────────────┘
```

**Why Go?**
- Native support for dynamic reverse proxy
- Automatic DNS resolution and connection pooling
- Built-in HTTPS/TLS handling
- High performance (2000+ req/s)
- Single binary deployment

---

## 🚀 Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- Docker (optional, for containerized deployment)
- Ollama (optional, for local LLM analysis)

### Installation

#### Option 1: Run from Source

```bash
# Clone the repository
git clone https://github.com/hulk-yin/diting.git
cd diting/cmd/diting

# Download dependencies
go mod download

# Run the service
go run main.go
```

#### Option 2: Build Binary

```bash
cd diting/cmd/diting

# Build
go build -o diting main.go

# Run
./diting
```

#### Option 2b: Feishu approval (recommended for human-in-the-loop)

For high-risk operations with Feishu approval (no public URL or Feishu "long connection" required):

```bash
cd diting/cmd/diting

# Build Feishu message-reply version
go build -o diting main.go

# Configure config.json (feishu.approval_user_id, use_message_reply: true, poll_interval_seconds)
# Then run
./diting
```

See **[cmd/diting/QUICKSTART.md](cmd/diting/QUICKSTART.md)** for config and minimal verification steps.

#### Option 3: Docker Deployment

```bash
cd diting/deployments/docker
docker-compose up -d
```

### Testing

```bash
# Configure your AI agent to use Diting as proxy
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Safe request (auto-approved)
curl http://localhost:8080/get

# Dangerous request (requires approval)
curl -X DELETE http://localhost:8080/delete

# View audit logs
cat logs/audit.jsonl
```

---

## 📦 Project Structure

```
diting/
├── cmd/diting/             # Main application
│   ├── main.go             # Entry point
│   ├── go.mod              # Go module
│   └── README.md
│
├── pkg/                    # Reusable packages (future)
│   ├── dns/                # DNS utilities
│   ├── waf/                # WAF utilities
│   └── ebpf/               # eBPF monitoring (future)
│
├── deployments/            # Deployment configs
│   ├── docker/             # Docker Compose
│   └── kubernetes/         # K8s manifests (future)
│
├── docs/                   # Documentation
│   ├── QUICKSTART.md
│   ├── INSTALL.md
│   └── ...
│
└── scripts/                # Utility scripts
```

---

## 💡 Core Features

### 1. Dynamic API Proxy

Unlike traditional reverse proxies (Nginx) that require fixed upstream configuration, Diting dynamically handles any external API:

```go
// Automatically handles any target
requests.get('https://api.openai.com/chat')      // ✅ Works
requests.post('https://api.github.com/repos')    // ✅ Works
requests.delete('https://random-api.com/data')   // ✅ Works
```

### 2. Intelligent Risk Assessment

- **HTTP Method**: GET (safe) vs DELETE (dangerous)
- **URL Path**: `/delete`, `/remove`, `/drop` (high risk)
- **Request Body**: Dangerous keywords detection
- **Three-tier**: Low / Medium / High

### 3. AI Intent Analysis

- Integrated with Ollama (local LLM) or OpenAI
- Automatic intent and impact analysis
- Fallback to rule engine when LLM unavailable
- Response time < 2 seconds

### 4. Human Approval Workflow

- Interactive CLI approval for high-risk operations
- Full context display (method, path, analysis)
- Approve/deny decisions
- Extensible to enterprise messaging platforms

### 5. Full Audit Trail

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

## 📚 Documentation

- [Quick Start Guide](docs/QUICKSTART.md) - Get started in 5 minutes
- [Installation Guide](docs/INSTALL.md) - Detailed deployment instructions
- [Architecture Guide](docs/ARCHITECTURE_DNS_HIJACK.md) - System architecture
- [Testing Guide](docs/TEST.md) - Test scenarios and cases
- [Demo Script](docs/DEMO.md) - Presentation guide
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

---

## 🛠️ Development

### Building

```bash
cd cmd/diting
go build -o diting main.go
```

### Cross-compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o diting-linux main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o diting.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o diting-mac main.go
```

### Running Tests

```bash
go test ./...
```

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Go](https://golang.org/) - Programming language
- [Ollama](https://ollama.ai/) - Local LLM runtime
- [OpenAI](https://openai.com/) - AI models

---

## 📞 Contact

- GitHub Issues: [https://github.com/hulk-yin/diting/issues](https://github.com/hulk-yin/diting/issues)

---

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=hulk-yin/diting&type=Date)](https://star-history.com/#hulk-yin/diting&Date)

---

## 🐉 About the Name

**Diting (谛听)** is a divine creature in Chinese Buddhist mythology, known as the mount of Ksitigarbha Bodhisattva. It possesses the supernatural ability to distinguish truth from falsehood, good from evil, and can hear all sounds in the world. This perfectly embodies our platform's mission: to discern and govern AI agent behaviors with wisdom and precision.

---

**Made with ❤️ by the Diting Team**
