# Commit guidance (project cleanup and development norms)

Suggested commits after "project cleanup + development norms"; ensure `config.json` contains no secrets (or use `config.example.json`) before committing.

---

## Suggested batch commits

### 1. Norms and documentation (recommended first)

```bash
git add .gitignore CONTRIBUTING.md README.md docs/DEVELOPMENT.md docs/DELIVERY.md docs/COMMIT_GUIDANCE.md
git commit -m "docs: add development norms and update contributing

- Add docs/DEVELOPMENT.md (project structure, entry points, Go/Feishu conventions)
- Update CONTRIBUTING.md with link to DEVELOPMENT and no-secrets reminder
- Update .gitignore for cmd/diting binaries and local logs
- Update docs/DELIVERY.md with Feishu phase summary"
```

### 2. Feishu approval and Diting entry

```bash
git add cmd/diting/main.go cmd/diting/main_ollama.go cmd/diting/main_ws_fixed.go cmd/diting/main_feishu_chat.go cmd/diting/main_ws.go \
  cmd/diting/feishu_listener.go cmd/diting/feishu_websocket.go cmd/diting/get_feishu_user_id.go \
  cmd/diting/QUICKSTART.md cmd/diting/FEISHU_TROUBLESHOOTING.md cmd/diting/FEISHU_*.md cmd/diting/DELIVERY.md \
  cmd/diting/query_audit.sh cmd/diting/run-feishu-verification.sh cmd/diting/diagnose_feishu.sh \
  cmd/diting/go.mod cmd/diting/go.sum
git commit -m "feat(diting): Feishu approval with chat_id fallback and dev tooling

- main: open_id cross app fallback to chat_id send/poll; add ChatID to config
- WebSocket builds: main_ws_fixed, main_feishu_chat; log user_id on message receive
- get_feishu_user_id.go for resolving user_id; query_audit.sh for audit log
- QUICKSTART, FEISHU_TROUBLESHOOTING, run-feishu-verification.sh"
```

### 3. Other main_* and test scripts (optional)

Add remaining `cmd/diting/main_*.go`, `test*.sh`, etc. as needed, then commit separately (e.g. `chore(diting): add alternate Feishu/main variants and scripts`).

---

## Notes

- **Do not commit**: `cmd/diting/config.json` if it contains `app_secret`, `api_key`, etc.; you may commit `config.example.json` as a template.
- **Optional ignores**: Add `.cursor/`, `_bmad/`, `_bmad-output/` to `.gitignore` if they are for local use only, or follow team policy on version control.
