# Development Guide (Sentinel-AI / Diting)

This document defines repository layout, recommended entry points, and code/commit conventions for collaboration and maintenance.

---

## 1. Repository structure

```
sentinel-ai/
├── cmd/diting/           # Diting governance gateway (main app)
│   ├── main.go           # Recommended: Feishu approval (poll + chat_id fallback)
│   ├── main_ollama.go    # Ollama-only, no Feishu
│   ├── main_ws_fixed.go  # Feishu WebSocket (requires Feishu "long connection")
│   ├── config.json       # Runtime config (do not commit secrets)
│   ├── go.mod, go.sum
│   ├── QUICKSTART.md     # Quick start (including recommended entry)
│   └── FEISHU_*.md       # Feishu integration and troubleshooting
├── pkg/                  # Reusable packages (dns, waf, etc.)
├── deployments/          # Docker / deployment configs
├── docs/                 # Project docs and norms
├── scripts/              # Build and test scripts
└── _bmad-output/         # BMAD outputs (phase summaries, checklists, etc.)
```

---

## 2. Recommended entry points and build

| Scenario              | Entry file        | Binary          | Notes |
|-----------------------|-------------------|-----------------|-------|
| Feishu approval       | `main.go`         | `diting`        | Poll for replies; chat_id fallback; no long connection |
| WebSocket approval    | `main_ws_fixed.go`| `diting_ws_fixed` | Requires Feishu "long connection" |
| No Feishu             | `main.go`         | `diting`        | Ollama intent analysis, local approval |

- Build: `go build -o diting main.go` (from `cmd/diting`). This directory has multiple main packages; build by specifying the file, e.g. `go build -o diting-ollama main_ollama.go`.
- Config: see `cmd/diting/QUICKSTART.md` and `config.json` comments; keep secrets in env or local overrides; do not commit keys.

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
