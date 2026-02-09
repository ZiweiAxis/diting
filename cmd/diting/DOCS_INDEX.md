# Diting 文档索引

收敛说明：飞书与验收相关以以下为主，其余为历史或专项参考，可按需查阅。

---

## 主要文档（优先看）

| 文档 | 用途 |
|------|------|
| **ACCEPTANCE_CHECKLIST.md** | 闭环验收检查单（策略→卡片→长连接点击→放行/拒绝） |
| **run_acceptance.sh** | 验收脚本：`start` 启动服务 / `trigger` 触发审批 / `stop` 停止 |
| **VERIFY_CARD.md** | 交互卡片验证步骤（发卡片、点按钮、200340 排查） |
| **FEISHU_LONG_CONNECTION_CARD.md** | 长连接 + 卡片：只走 WebSocket（card.action.trigger），不填 request_url |
| **QUICKSTART_DOCKER.md** | 容器 15 分钟快速开始（Dockerfile.diting + 运行与验证） |
| **DNS_MODE.md** | 双接入：Proxy 与 DNS 模式说明与验证 |
| **config.acceptance.yaml** | 验收配置（策略、CHEQ、飞书、use_card_delivery / use_long_connection） |
| **README.md** | 项目入口与构建说明 |

---

## 参考 / 历史（按需）

| 文档 | 说明 |
|------|------|
| FEISHU_SETUP.md | 飞书应用与权限配置 |
| FEISHU_APPROVAL_FLOW_CN.md | 审批流程说明 |
| FEISHU_GET_USER_ID_CN.md | 获取 user_id（HTTP 回调方式示例） |
| FEISHU_TROUBLESHOOTING_CN.md | 飞书问题排查（中文） |
| QUICKSTART_CN.md / QUICKSTART.md | 快速开始 |
| DELIVERY.md | 投递机制说明 |
| README_QUICK_START.md / README_FEISHU.md / README_COMPLETE.md | 其他 README 变体 |

过程性文档（FEISHU_INTEGRATION、FEISHU_WEBSOCKET_FIX、TEST_REPORT 等）需时再查即可。
