# Diting（All-in-One）

本目录为 Diting 治理网关实现。**推荐入口**：All-in-One（策略 + CHEQ + 飞书投递 + 审计），使用 **config.yaml + .env**。

---

## 快速开始

```bash
# 构建（推荐入口，产物在 bin/diting）
make build
# 或：go build -o bin/diting ./cmd/diting_allinone

# 运行
make run
# 或：./bin/diting
```

构建产物统一放在本目录下 **bin/**，勿在 cmd/diting 根目录生成 diting、diting_* 等二进制。

首次使用：`cp config.example.yaml config.yaml`，`cp .env.example .env`，在 `.env` 中填写 `DITING_FEISHU_APP_ID`、`DITING_FEISHU_APP_SECRET`、`DITING_FEISHU_APPROVAL_USER_ID` 等。详见 [CONFIG_LAYERS.md](CONFIG_LAYERS.md)。

---

## 本目录文档（最少必要）

| 文档 | 用途 |
|------|------|
| [CONFIG_LAYERS.md](CONFIG_LAYERS.md) | 配置：config.yaml + .env |
| [DEV_WATCH.md](DEV_WATCH.md) | 本地 Watch 模式（make watch） |
| [ACCEPTANCE_CHECKLIST.md](ACCEPTANCE_CHECKLIST.md) | 闭环验收检查单 |
| [MAIN_ENTRIES.md](MAIN_ENTRIES.md) | 入口说明（已清理多余 main_*.go） |
| [DOCS_INDEX.md](DOCS_INDEX.md) | 文档索引与 docs/diting、_bmad-output 说明 |

---

## 为什么会有很多文档？与 BMAD 的关系

- **规划 / Epic / 验收** 应使用 **BMAD**，产出在仓库 **\_bmad-output/**（PRD、epics、architecture、acceptance 等）。
- **cmd/diting 下曾有很多 .md**，是开发过程中**临时写的**（飞书对接、排错、快速开始、交付总结等），没有按 BMAD 收敛，导致文档堆积。
- **本次整理**：已将大部分迁到 **docs/diting/**，本目录只保留上述最少必要文档。**过程性文档**（验收报告、修复记录、交付总结等）统一放在仓库 **\_process_docs/**（已 gitignore，不提交）；**临时性工具/脚本**（如获取飞书 user_id 的独立脚本）放在 **\_temp/**（已 gitignore，不提交），后续此类内容请都放对应目录。

更多参考与历史文档见 [docs/diting/README.md](../../docs/diting/README.md)；规划与验收见 **\_bmad-output/**。
