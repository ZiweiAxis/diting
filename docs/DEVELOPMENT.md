# 项目开发规范 (Sentinel-AI / Diting)

本文档约定仓库结构、推荐入口、代码与提交规范，便于协作与维护。

---

## 1. 仓库结构

```
sentinel-ai/
├── cmd/diting/           # Diting 治理网关（主应用）
│   ├── main.go           # 原始入口（Ollama，无飞书）
│   ├── main.go           # 推荐：飞书审批（轮询 + chat_id 回退）
│   ├── main_ws_fixed.go  # 飞书 WebSocket 长连接（需开放平台开启）
│   ├── config.json       # 运行时配置（勿提交敏感信息）
│   ├── go.mod, go.sum
│   ├── QUICKSTART.md     # 快速启动（含推荐入口）
│   └── FEISHU_*.md       # 飞书集成与排错
├── pkg/                  # 可复用包（dns, waf 等）
├── deployments/          # Docker / 部署配置
├── docs/                 # 项目文档与规范
├── scripts/              # 构建与测试脚本
└── _bmad-output/         # BMAD 产出（阶段总结、验证清单等）
```

---

## 2. 推荐入口与构建

| 场景           | 入口文件        | 构建产物      | 说明 |
|----------------|-----------------|---------------|------|
| 飞书审批（推荐） | `main.go`       | `diting`      | 轮询回复，支持 chat_id 回退，无需长连接 |
| WebSocket 审批 | `main_ws_fixed.go` | `diting_ws_fixed` | 需飞书开放平台开启「长连接」 |
| 无飞书         | `main.go`       | `diting`      | Ollama 意图分析，本地审批 |

- 构建：`go build -o diting main.go`（在 `cmd/diting` 下）。本目录存在多个 main 入口，须按单文件构建，例如：`go build -o diting-ollama main_ollama.go`。
- 配置：见 `cmd/diting/QUICKSTART.md` 与 `config.json` 注释；敏感信息用环境变量或本地覆盖，勿提交密钥。

---

## 3. Go 代码规范

- 遵循官方 [Effective Go](https://go.dev/doc/effective_go) 与 `go fmt`。
- 提交前执行：`go build ./...`、`go vet ./...`（若有测试则 `go test ./...`）。
- 导出符号需有注释；新增 main_*.go 时在 README/QUICKSTART 中说明用途与推荐场景。
- 第三方依赖：仅通过 `go mod` 管理，不提交 `vendor/`（除非项目明确要求）。

---

## 4. 飞书与审批相关

- **配置**：`approval_user_id` 建议使用本应用下的 **user_id**（避免 open_id cross app）；若仅配置 `chat_id`，main 会在发用户失败时回退到 chat 发送并轮询该 chat。
- **日志**：审批决策写入 `audit` 配置的 JSONL；可读性优先，便于 `query_audit.sh` 或后续分析。
- **新增入口**：若新增 main_*.go，需在 QUICKSTART 或 FEISHU_TROUBLESHOOTING 中说明与推荐入口的差异。

---

## 5. 提交与分支

- **Commit**：采用 [Conventional Commits](https://www.conventionalcommits.org/)（如 `feat:`, `fix:`, `docs:`）。
- **分支**：功能开发使用 `feature/xxx` 或 `fix/xxx`；主分支保持可构建、文档与推荐入口一致。
- **提交前**：确认未提交二进制（如 `diting`）、本地日志、`config.json` 中的密钥；敏感配置使用 `.env` 或 `config.local.json` 并已加入 `.gitignore`。

---

## 6. 文档与产出

- 用户面向：`README.md`、`README_CN.md`、`cmd/diting/QUICKSTART.md`。
- 开发与排错：`CONTRIBUTING.md`、`docs/DEVELOPMENT.md`（本文档）、`cmd/diting/FEISHU_TROUBLESHOOTING.md`。
- BMAD 阶段产出：`_bmad-output/` 下阶段总结、验证清单、下一步路线图；重要结论可同步到 `docs/` 或 `cmd/diting/DELIVERY.md`。

---

*最后更新：2026-02-08，与飞书审批验证通过后的仓库状态一致。*
