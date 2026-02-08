# 提交建议（项目整理与开发规范）

以下为本次「项目整理 + 完善开发规范」后的提交建议；执行前请确认 `config.json` 不含敏感信息或已改为 `config.example.json` 再提交。

---

## 建议分批提交

### 1. 规范与文档（推荐先提交）

```bash
git add .gitignore CONTRIBUTING.md README.md docs/DEVELOPMENT.md docs/DELIVERY.md docs/COMMIT_GUIDANCE.md
git commit -m "docs: add development norms and update contributing

- Add docs/DEVELOPMENT.md (project structure, entry points, Go/Feishu conventions)
- Update CONTRIBUTING.md with link to DEVELOPMENT and no-secrets reminder
- Update .gitignore for cmd/diting binaries and local logs
- Update docs/DELIVERY.md with Feishu phase summary"
```

### 2. 飞书审批与 Diting 入口

```bash
git add cmd/diting/main.go cmd/diting/main_ollama.go cmd/diting/main_ws_fixed.go cmd/diting/main_feishu_chat.go cmd/diting/main_ws.go \
  cmd/diting/feishu_listener.go cmd/diting/feishu_websocket.go cmd/diting/get_feishu_user_id.go \
  cmd/diting/QUICKSTART.md cmd/diting/FEISHU_TROUBLESHOOTING.md cmd/diting/FEISHU_*.md cmd/diting/DELIVERY.md \
  cmd/diting/query_audit.sh cmd/diting/run-feishu-verification.sh cmd/diting/diagnose_feishu.sh \
  cmd/diting/go.mod cmd/diting/go.sum
# 若 config 已脱敏或使用 example：
# git add cmd/diting/config.json
git commit -m "feat(diting): Feishu approval with chat_id fallback and dev tooling

- main: open_id cross app fallback to chat_id send/poll; add ChatID to config
- WebSocket builds: main_ws_fixed, main_feishu_chat; log user_id on message receive
- get_feishu_user_id.go for resolving user_id; query_audit.sh for audit log
- QUICKSTART, FEISHU_TROUBLESHOOTING, run-feishu-verification.sh"
```

### 3. 其他 main_* 与测试脚本（可选）

按需 `git add` 其余 `cmd/diting/main_*.go`、`test*.sh` 等，再单独 commit（例如 `chore(diting): add alternate Feishu/main variants and scripts`）。

---

## 注意

- **勿提交**：`cmd/diting/config.json` 若含 `app_secret`、`api_key` 等；可提交 `config.example.json` 占位。
- **可选忽略**：`.cursor/`、`_bmad/`、`_bmad-output/` 若仅本地使用，可在 `.gitignore` 中保留或按团队约定是否纳入版本控制。
