# Diting 文档索引

**本目录**只保留运行与验收必需的最少文档；其余已迁到 **docs/diting/**。  
**规划 / Epic / 验收** 以仓库 **\_bmad-output/** 为准（BMAD 产出）。

---

## 本目录（优先看）

| 文档 | 用途 |
|------|------|
| **README.md** | 入口：构建、运行、配置与 BMAD 说明 |
| **CONFIG_LAYERS.md** | config.yaml + .env 关系与首次使用 |
| **DEV_WATCH.md** | Watch 模式（make watch / air） |
| **ACCEPTANCE_CHECKLIST.md** | 闭环验收检查单（策略→飞书卡片→放行/拒绝） |
| **MAIN_ENTRIES.md** | 入口说明（推荐 All-in-One，已清理多余 main_*.go） |
| **run_acceptance.sh**、**trigger_review.sh**、**query_audit.sh** | 验收与审计脚本（本目录） |
| **config.example.yaml**、**.env.example** | 配置模板（复制为 config.yaml、.env 后本地使用，不提交） |
| **policy_rules.example.yaml** | 策略规则示例（可选，见 policy.rules_path） |
| **scripts/** | 可选脚本：install-dev-deps.sh、test.sh、test_feishu.sh、run-with-feishu.sh；临时验证脚本已迁 _temp/diting/，见 scripts/README.md |

---

## 本目录不提交/仅本地

以下由 .gitignore 忽略，不提交：**config.yaml**、**.env**、**bin/**（构建产物）、**data/**（审计/CHEQ 数据）、**tmp/**（air 临时）。仓库级 **\_process_docs/**、**\_temp/** 见下表。

---

## 其他文档位置

| 位置 | 内容 |
|------|------|
| **docs/diting/** | 快速开始、飞书配置与排查、交付总结、历史 README、过程性报告等 |
| **\_bmad-output/** | PRD、Epics、Architecture、Acceptance（BMAD） |
| **\_process_docs/** | 过程性文档（不提交）：验收报告、修复记录、交付总结等，今后此类文档统一放此目录 |
| **\_temp/** | 临时性工具/脚本（不提交）：如获取飞书 user_id 的独立脚本等，与过程文档同方式、统一放此目录 |

详见 [docs/diting/README.md](../../docs/diting/README.md)。
