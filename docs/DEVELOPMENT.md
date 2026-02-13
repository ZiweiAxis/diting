# Development Guide (Diting)

This document defines repository layout, recommended entry points, and code/commit conventions for collaboration and maintenance.

---

## 1. Repository structure

```
diting/
├── cmd/diting_allinone/  # 推荐主入口：All-in-One（策略/CHEQ/飞书/审计）
│   └── main.go           # make build → bin/diting
├── cmd/diting/           # 备用入口与文档（main.go、main_ollama.go 等）
│   ├── config.yaml, go.mod, go.sum
│   ├── QUICKSTART.md, ACCEPTANCE_CHECKLIST.md, MAIN_ENTRIES.md
│   └── FEISHU_*.md       # Feishu integration and troubleshooting
├── pkg/                  # Reusable packages (dns, waf, etc.)
├── deployments/          # Docker / deployment configs
├── docs/                 # Project docs and norms
├── scripts/              # Build and test scripts
└── _bmad-output/         # BMAD outputs (phase summaries, checklists, etc.)
```

---

## 2. Recommended entry points and build

| Scenario              | Entry file               | Binary   | Notes |
|-----------------------|--------------------------|----------|-------|
| **All-in-One（推荐）** | `cmd/diting_allinone/main.go` | `bin/diting` | 策略/CHEQ/飞书/审计一体；`make build` / `make run` |
| Feishu（备用）        | `cmd/diting/main.go`     | `diting` | Poll + chat_id fallback；详见 MAIN_ENTRIES.md |
| WebSocket（备用）     | `cmd/diting/main_ws_fixed.go` | `diting_ws_fixed` | 需飞书「长连接」 |
| No Feishu（备用）     | `cmd/diting/main.go`     | `diting` | Ollama 意图分析，本地审批 |

- Build: **推荐主入口** 仓库根目录 `make build` 或 `go build -o bin/diting ./cmd/diting_allinone`（产出 `bin/diting`）；备用入口见 `cmd/diting/MAIN_ENTRIES.md`。
- Config: see `cmd/diting/QUICKSTART.md` and `config.yaml` / `config.example.yaml`; keep secrets in env or local overrides; do not commit keys.
- **Watch 模式（本地开发）**：修改代码或配置后自动重新编译并重启，无需每次手敲命令。安装 [air](https://github.com/air-verse/air)：`go install github.com/air-verse/air@latest`；在仓库根目录执行 `make watch`，或在 `cmd/diting` 下执行 `air`。配置见 `cmd/diting/.air.toml`。

---

## 3. Go code conventions

- Follow [Effective Go](https://go.dev/doc/effective_go) and `go fmt`.
- Before commit: `go build ./...`, `go vet ./...` (and `go test ./...` if tests exist).
- Exported symbols must have comments; when adding a new main_*.go, document purpose and recommended use in README/QUICKSTART.
- Dependencies: managed via `go mod` only; do not commit `vendor/` unless required by the project.

---

## 4. Feishu and approval

- **Config**: Prefer **user_id** under this app for `approval_user_id` (avoids open_id cross-app). If only `chat_id` is set, main will fall back to sending to that chat and polling it when user send fails.
- **Logging**: Approval decisions go to the audit JSONL; keep logs readable for `query_audit.sh` and later analysis.
- **New entries**: If you add a main_*.go, document how it differs from the recommended entry in QUICKSTART or FEISHU_TROUBLESHOOTING.

---

## 5. Commits and branches

- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/) (e.g. `feat:`, `fix:`, `docs:`). **Use English for commit messages.**
- **Branches**: Use `feature/xxx` or `fix/xxx` for development; keep main buildable and docs aligned with the recommended entry.
- **Before commit**: Ensure no binaries (e.g. `diting`), local logs, or keys in `config.json` are committed; use `.env` or `config.local.json` for secrets and keep them in `.gitignore`.

---

## 6. Documentation and artifacts

- User-facing: `README.md`, `README_CN.md`, `cmd/diting/QUICKSTART.md`.
- Development and troubleshooting: `CONTRIBUTING.md`, `docs/DEVELOPMENT.md` (this file), `cmd/diting/FEISHU_TROUBLESHOOTING.md`.
- BMAD phase artifacts: `_bmad-output/` (summaries, checklists, roadmaps); sync important conclusions to `docs/` or `cmd/diting/DELIVERY.md`.

---

*Last updated: 2026-02-08, aligned with post–Feishu-approval validation.*
