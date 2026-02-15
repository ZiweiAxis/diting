# E2E 与演示可重复执行说明

**对应**：谛听 ISSUE_LIST I-020 — 端到端脚本与文档在干净环境下可重复跑通，耗时与结果可验证。

---

## 1. 前置条件（干净环境）

- **Go 1.21+**：`go version`
- **工作目录**：在 `cmd/diting` 下执行（即本目录）
- **可选**：飞书审批类验证需配置 `.env` 或 `DITING_FEISHU_*` 环境变量，见 [ACCEPTANCE_CHECKLIST.md](../ACCEPTANCE_CHECKLIST.md)

---

## 2. 可重复 E2E 路径（无需飞书）

**脚本**：`./scripts/verify_exec.sh`  
**耗时**：约 30 秒内（构建 + 启动 + 请求 + 退出）  
**结果**：脚本退出码 0 表示通过；非 0 表示失败。

```bash
cd cmd/diting
./scripts/verify_exec.sh
```

**验证内容**：

- 使用临时端口 `18080` 启动 diting，避免与已有服务冲突
- `POST /auth/exec`（allow）
- `GET /auth/sandbox-profile`
- `3af-exec echo ok`
- 结束后自动停止进程

**可重复性**：不依赖飞书、不依赖固定端口；仅需本机 Go 与网络可用（脚本内请求为本地）。

---

## 3. 代理 + 审批测试（需先启动 diting）

**脚本**：`./scripts/test.sh`  
**前提**：在另一终端已执行 `make run` 或 `./bin/diting`（默认监听 8080）。

```bash
cd cmd/diting
# 终端 1
make run
# 终端 2
./scripts/test.sh
# 若 diting 使用非默认端口，可：PROXY_PORT=18080 ./scripts/test.sh
```

**验证内容**：经代理发 GET/POST/DELETE/HTTPS 请求；中/高风险会进入审批（若未配置飞书则请求会挂起至超时）。  
**可重复性**：同一环境、同一配置下多次执行结果一致；端口可通过 `PROXY_PORT` 覆盖。

---

## 4. 飞书审批全流程（需飞书环境）

见 [ACCEPTANCE_CHECKLIST.md](../ACCEPTANCE_CHECKLIST.md) 与 `./scripts/verify_feishu_approval.sh full`。  
依赖飞书应用配置与人工点击，属「演示可重复」而非「全自动可重复」。

---

## 5. 验收标准（I-020）

| 项 | 标准 |
|----|------|
| **无飞书路径** | 在干净环境（仅 Go + 本目录）执行 `./scripts/verify_exec.sh`，退出码 0，耗时可接受（如 < 60s） |
| **代理测试** | 先启动 diting 后执行 `./scripts/test.sh`，脚本无报错，代理返回状态码可查 |
| **文档** | 本文档与 scripts/README 中可发现上述脚本用途与执行顺序 |

---

*与 ISSUE_LIST I-020、scripts/README.md、ACCEPTANCE_CHECKLIST.md 一致。*
