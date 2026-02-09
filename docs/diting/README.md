# Diting 文档（集中存放）

本目录为 **cmd/diting** 的参考与过程性文档，从 cmd/diting 迁出以便目录简洁。  
**规划与验收** 以仓库根目录 **\_bmad-output/** 为准（BMAD）。

**说明**：过程性文档已迁到仓库根目录 **\_process_docs/**（该目录已加入 `.gitignore`，不提交），例如 `_process_docs/diting/` 下的 TEST_REPORT、DELIVERY、FEISHU_WEBSOCKET_FIX 等。今后此类文档请统一放在 `_process_docs/`。

---

## 快速开始 / 验收

| 文档 | 用途 |
|------|------|
| [QUICKSTART.md](QUICKSTART.md) / [QUICKSTART_CN.md](QUICKSTART_CN.md) | 快速启动（中/英） |
| [QUICKSTART_DOCKER.md](QUICKSTART_DOCKER.md) | 容器 15 分钟快速开始 |
| [VERIFY_CARD.md](VERIFY_CARD.md) | 交互卡片验证 |
| [FEISHU_LONG_CONNECTION_CARD.md](FEISHU_LONG_CONNECTION_CARD.md) | 长连接 + 卡片 |
| [ACCEPTANCE_ENV_AND_LLM.md](ACCEPTANCE_ENV_AND_LLM.md) | .env/LLM 验收报告（过程） |

## 飞书

| 文档 | 用途 |
|------|------|
| [FEISHU_SETUP.md](FEISHU_SETUP.md) | 飞书应用与审批配置 |
| [FEISHU_APPROVAL_FLOW_CN.md](FEISHU_APPROVAL_FLOW_CN.md) | 审批流程说明 |
| [FEISHU_GET_USER_ID_CN.md](FEISHU_GET_USER_ID_CN.md) | 获取 user_id |
| [FEISHU_TROUBLESHOOTING_CN.md](FEISHU_TROUBLESHOOTING_CN.md) / [FEISHU_TROUBLESHOOTING.md](FEISHU_TROUBLESHOOTING.md) | 飞书问题排查 |
| [FEISHU_INTEGRATION.md](FEISHU_INTEGRATION.md) | 飞书集成改造方案 |
| [FEISHU_WEBSOCKET_FIX.md](FEISHU_WEBSOCKET_FIX.md) | WebSocket 修复（过程） |

## 交付 / 历史 README

| 文档 | 用途 |
|------|------|
| [DELIVERY.md](DELIVERY.md) | 交付总结与文件清单 |
| [README_FEISHU.md](README_FEISHU.md) / [README_COMPLETE.md](README_COMPLETE.md) / [README_QUICK_START.md](README_QUICK_START.md) | 历史 README 变体 |
| [TEST_REPORT.md](TEST_REPORT.md) | 测试报告（过程） |

## 其他

| 文档 | 用途 |
|------|------|
| [DNS_MODE.md](DNS_MODE.md) | Proxy 与 DNS 双接入说明 |

---

日常在 **cmd/diting** 只需看：**README.md、CONFIG_LAYERS.md、DEV_WATCH.md、ACCEPTANCE_CHECKLIST.md、DOCS_INDEX.md、MAIN_ENTRIES.md**；其余按需到本目录或 \_bmad-output 查阅。
