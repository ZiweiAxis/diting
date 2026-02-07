# Diting (谛听)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.8+](https://img.shields.io/badge/python-3.8+-blue.svg)](https://www.python.org/downloads/)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](https://www.docker.com/)

**Enterprise-grade AI Agent Zero-Trust Governance Platform**

**谛听** - A mythical creature in Chinese mythology that can distinguish truth from falsehood, good from evil.

[中文文档](README.md) | [Quick Start](QUICKSTART.md) | [Documentation](docs/)

---

## 🎯 Overview

Diting (谛听) is an enterprise-grade AI security governance platform that builds a zero-trust architecture using open-source tools, enabling AI Agents to run securely, controllably, and compliantly.

Just like the mythical creature Diting that serves as the mount of Ksitigarbha Bodhisattva and can discern truth from lies, this platform acts as a guardian for AI agents, ensuring their operations are safe and trustworthy.

### Key Features

- ✅ **Fully Transparent** - No agent modification required, zero intrusion
- ✅ **Unbypassable** - DNS hijacking + network-layer interception
- ✅ **AI-Driven** - OpenAI intent analysis with intelligent decision-making
- ✅ **Full Audit Trail** - Every operation is traceable for compliance
- ✅ **Human-in-the-Loop** - Manual approval for high-risk operations
- ✅ **Open Source Stack** - Built on CoreDNS + Nginx/OpenResty

---

## 🏗️ Architecture

### Three-Layer Governance Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent Application Layer                  │
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
│                  Data Plane - Interception Layer             │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │         DNS Hijacking (CoreDNS)                   │     │
│  │  api.example.com → 10.0.0.1 (WAF Gateway)        │     │
│  └───────────────────────────────────────────────────┘     │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │      Nginx/OpenResty Gateway (Lua)                │     │
│  │  - Request analysis                                │     │
│  │  - Decision execution                              │     │
│  │  - Cache management                                │     │
│  └───────────────────────────────────────────────────┘     │
│                                                              │
│  ┌───────────────────────────────────────────────────┐     │
│  │      Diting Business Logic (Python/Go)            │     │
│  │  - OpenAI intent analysis                          │     │
│  │  - Risk assessment                                 │     │
│  │  - Approval workflow                               │     │
│  └───────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

- Python 3.8+ or Go 1.21+
- Docker (optional, for containerized deployment)
- OpenAI API Key (or Ollama for local LLM)

### Installation

#### Python Version (Recommended for Quick Start)

```bash
# Clone the repository
git clone https://github.com/hulk-yin/diting.git
cd diting

# Install dependencies
pip install -r requirements.txt

# Start the service
python sentinel.py
```

#### Go Version (High Performance)

```bash
# Clone the repository
git clone https://github.com/hulk-yin/diting.git
cd diting

# Download dependencies
go mod download

# Run the service
go run main.go
```

#### Docker Deployment

```bash
# Start all services
docker-compose up -d

# Or use the open-source stack
docker-compose -f docker-compose-opensource.yml up -d
```

### Testing

```bash
# Safe request (auto-approved)
curl http://localhost:8080/get

# Dangerous request (requires approval)
curl -X DELETE http://localhost:8080/delete

# View audit logs
cat logs/audit.jsonl
```

---

## 📦 Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **DNS Hijacking** | CoreDNS | Route all domains to WAF gateway |
| **WAF Gateway** | Nginx/OpenResty | Reverse proxy with Lua scripting |
| **Business Logic** | Python/Go | AI analysis + risk assessment |
| **LLM** | OpenAI/Ollama | Intent analysis |
| **Storage** | JSONL | Audit trail logging |

---

## 💡 Core Features

### 1. Intelligent Risk Assessment
- HTTP method-based (GET safe, DELETE dangerous)
- URL path-based (/delete, /remove, etc.)
- Request body content analysis
- Three-tier risk classification (low/medium/high)

### 2. AI Intent Analysis
- Integrated with OpenAI/Ollama
- Automatic intent and impact analysis
- Fallback to rule engine when LLM unavailable
- Response time < 2 seconds

### 3. Human Approval Workflow
- Interactive CLI approval
- Full context display
- Approve/deny decisions
- Extensible to enterprise messaging platforms

### 4. Full Audit Trail
- JSONL format logging
- Complete request/response recording
- Decision reasoning and approver tracking
- Post-incident forensics support

### 5. Zero-Intrusion Deployment
- No agent code modification required
- No backend API changes needed
- Only DNS configuration required

---

## 📚 Documentation

- [Quick Start Guide](QUICKSTART.md) - Get started in 5 minutes
- [Installation Guide](INSTALL.md) - Detailed deployment instructions
- [Open Source Deployment](DEPLOYMENT_OPENSOURCE.md) - Deploy with open-source tools
- [Architecture Guide](ARCHITECTURE_DNS_HIJACK.md) - DNS hijacking architecture
- [eBPF Technical Guide](TECHNICAL_EBPF.md) - Kernel-level monitoring
- [Testing Guide](TEST.md) - Test scenarios and cases
- [Demo Script](DEMO.md) - Presentation guide
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

---

## 🛠️ Development

### Project Structure

```
diting/
├── main.go                 # Go implementation
├── sentinel.py             # Python implementation
├── sentinel_dns.py         # DNS hijacking module
├── sentinel_ebpf.py        # eBPF monitoring module
├── wafgateway.go           # WAF gateway
├── coredns/                # CoreDNS configuration
├── nginx/                  # Nginx/OpenResty configuration
├── sentinel-api/           # API service
├── logs/                   # Audit logs
└── docs/                   # Documentation
```

### Running Tests

```bash
# Python
python -m pytest

# Go
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

- [CoreDNS](https://coredns.io/) - DNS server
- [OpenResty](https://openresty.org/) - Web platform
- [OpenAI](https://openai.com/) - AI models
- [Ollama](https://ollama.ai/) - Local LLM runtime

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
