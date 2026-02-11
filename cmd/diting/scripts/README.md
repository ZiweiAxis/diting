# scripts 说明

本目录为 Diting 可选脚本：开发依赖安装、验收等。**临时性验证/调试脚本**已迁至 **\_temp/diting/**（不提交）。

---

## 本目录脚本（日常/验收用）

| 脚本 | 用途 |
| --- | --- |
| **install-dev-deps.sh** | 安装开发依赖：go mod tidy、air（watch 模式）。在 cmd/diting 下执行。 |
| **test.sh** | 用 curl 经代理(8081)测 GET/POST/DELETE/HTTPS，验证自动放行与需审批流程。 |
| **test_feishu.sh** | 检查 config.yaml、.env、bin/diting，提示飞书集成测试步骤并可选启动 diting。 |
| **run-with-feishu.sh** | 检查 DITING_FEISHU_* 环境变量后启动 `./bin/diting`，用于验收飞书审批。 |
| **verify_feishu_approval.sh** | 飞书审批双路径验证：原有审理（POST /admin）+ 新逻辑（POST /auth/exec exec:sudo）；`full` 为一键两次飞书点击。 |
| **verify_exec.sh** | 执行层验证（无需飞书）：POST /auth/exec、GET /auth/sandbox-profile、3af-exec echo。 |

---

## 已迁至 _temp/diting/（临时工具，不提交）

以下为临时性验证/调试用，已统一放入 **\_temp/diting/**：test_api.sh、test_api2.sh、diagnose_feishu.sh、quick_fix.sh、run-feishu-verification.sh。需用时在 _temp/diting 下执行，注意部分脚本含硬编码密钥，仅本地使用。

---

运行本目录脚本前请在 **cmd/diting** 下执行（或 `cd cmd/diting`），以便找到 config、.env、bin/diting。
