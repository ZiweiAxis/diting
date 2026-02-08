# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Feishu (Lark) approval: send approval requests to user or chat, poll for approve/deny reply; open_id cross-app fallback to chat_id
- Default entry point: `main.go` builds to `diting` (Feishu approval); `main_ollama.go` for Ollama-only build
- Audit query script: `cmd/diting/query_audit.sh` (--approved, --denied, -n)
- Open-source readiness: `config.example.json`, `SECURITY.md`, `.gitignore` for config

### Changed
- Entry normalization: no _v2 naming; single default binary `diting` from `main.go`
- `main_feishu_v2.go` renamed to `main_feishu_chat.go`
- Docs and scripts updated to reference `diting` / `main.go` consistently

### Documentation
- `docs/DEVELOPMENT.md` - project structure, entry points, Go/Feishu conventions
- `cmd/diting/QUICKSTART.md`, `FEISHU_TROUBLESHOOTING.md` - setup and troubleshooting

## [0.1.0] - 2026-02-05

### Added
- Initial MVP release
- HTTP reverse proxy with intelligent interception
- AI-powered intent analysis using OpenAI/Ollama
- Risk assessment engine with three-tier classification
- Human approval workflow for high-risk operations
- Full audit trail logging in JSONL format
- Dual implementation (Python and Go)
- DNS hijacking support via CoreDNS
- WAF gateway using Nginx/OpenResty
- eBPF kernel-level monitoring (experimental)
- Comprehensive documentation (18 markdown files)
- Docker and Podman deployment configurations
- Quick start scripts for Windows and Linux

### Documentation
- README.md - Project overview
- README-OPENSOURCE.md - Open source deployment guide
- QUICKSTART.md - Quick start guide
- INSTALL.md - Installation guide
- DEPLOYMENT_OPENSOURCE.md - Open source deployment
- ARCHITECTURE_DNS_HIJACK.md - DNS hijacking architecture
- TECHNICAL_EBPF.md - eBPF technical documentation
- TEST.md - Testing documentation
- DEMO.md - Demo script
- PITCH_DECK_GUIDE.md - Business presentation guide

### Features
- Zero-intrusion deployment (no agent modification required)
- AI-driven decision making with rule engine fallback
- Human-in-the-loop for critical operations
- Complete audit trail for compliance
- Multi-language support (Python/Go)
- Container-ready with Docker/Podman
- Open source architecture using CoreDNS and OpenResty

[Unreleased]: https://github.com/hulk-yin/diting/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hulk-yin/diting/releases/tag/v0.1.0
