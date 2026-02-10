# Diting Quick Start

## 推荐入口（唯一）

**All-in-One**：策略 + CHEQ + 飞书投递 + 审计，使用 **config.yaml + .env**。飞书支持文本消息与**交互卡片 + 长连接**（点击批准/拒绝即可，无需公网回调）。

| 项目 | 说明 |
|------|------|
| **入口** | `cmd/diting_allinone/main.go`，见 [MAIN_ENTRIES.md](../../cmd/diting/MAIN_ENTRIES.md) |
| **二进制** | `bin/diting`（`make build` 生成） |
| **配置** | `config.yaml` + `.env`；飞书等敏感项填在 `.env` 的 `DITING_FEISHU_*`，见 [CONFIG_LAYERS.md](../../cmd/diting/CONFIG_LAYERS.md) |
| **审批** | 飞书收到待确认消息或交互卡片；批准/拒绝可通过卡片按钮或 `/cheq/approve?id=xxx&approved=true|false` |

---

## 所需环境

- **Go**：1.21 或以上（`go version` 检查）。
- **容器路径**：若使用 Docker 快速开始，需安装 Docker；见 [QUICKSTART_DOCKER.md](QUICKSTART_DOCKER.md)。
- **飞书（可选）**：仅验证探针与代理可达时可不配置飞书；需审批流程时再配置 `.env` 中的 `DITING_FEISHU_APP_ID`、`DITING_FEISHU_APP_SECRET`、`DITING_FEISHU_APPROVAL_USER_ID`。

按下列步骤，约 **15 分钟**内可完成「构建 → 运行 → /healthz 与代理可达」的验证。

---

## Step 1: 安装依赖

```bash
cd cmd/diting
go mod tidy
# 可选：安装 air（watch 模式）
./scripts/install-dev-deps.sh
```

---

## Step 2: 构建

```bash
# 在 cmd/diting 下
make build
# 或：go build -o bin/diting ./cmd/diting_allinone
```

---

## Step 3: 配置（首次使用）

```bash
cp config.example.yaml config.yaml
cp .env.example .env
# 可选（飞书审批）：编辑 .env，填写 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID
# 仅验证探针与代理时可不填飞书，使用 config.example.yaml 即可启动
```

---

## Step 4: 运行

```bash
make run
# 或：./bin/diting
```

终端会输出监听地址（默认 :8080）、飞书是否启用等。

---

## Step 5: 验证

- **探针**：`curl -s http://localhost:8080/healthz`、`curl -s http://localhost:8080/readyz`
- **低风险（自动放行）**：`curl -x http://127.0.0.1:8080 https://httpbin.org/get`
- **需审批**：例如 `curl -x http://127.0.0.1:8080 -X DELETE https://httpbin.org/delete`，飞书收到消息或卡片后点击批准/拒绝，或访问 `http://localhost:8080/cheq/approve?id=<request_id>&approved=true`
- **审计**：`curl -s "http://localhost:8080/debug/audit?trace_id=<X-Trace-ID>"`

验收脚本：`./scripts/test.sh`（在 cmd/diting 下执行）。最小 3 步验证见仓库根 `_bmad-output/feishu-approval-minimal-verification.md`。

---

## 容器化 15 分钟快速开始

若用 Docker，见 **[QUICKSTART_DOCKER.md](QUICKSTART_DOCKER.md)**：构建镜像 → 运行容器 → 验证探针与代理。

---

## 常用

- **浏览器代理**：HTTP/HTTPS 代理设为 127.0.0.1:8080（或 config 中 `proxy.listen_addr` 的端口）
- **审计日志**：`cat data/audit.jsonl | jq` 或按 `audit.path` 配置的路径查看
- **停止**：`Ctrl+C`；重启：`make run` 或 `./bin/diting`

---

## 接入检查清单与验证流量经 Diting

**接入检查清单**

- **Proxy 方式**：将 Agent 或终端的 HTTP(S) 代理指向 Diting，例如 `export HTTP_PROXY=http://127.0.0.1:8080`、`export HTTPS_PROXY=http://127.0.0.1:8080`（端口与 config 中 `proxy.listen_addr` 一致）。确保目标流量走该代理。
- **DNS 方式**：将业务域名解析到运行 Diting 的机器（或网关）；详见 [DNS_MODE.md](DNS_MODE.md)（Corefile、hosts 等）。

**如何验证流量经 Diting**

1. **curl 经代理**：`curl -x http://127.0.0.1:8080 https://httpbin.org/get`，若返回正常则请求经 Diting 转发。
2. **审计**：请求中带 `X-Trace-ID`（或使用响应/日志中的 trace_id），然后 `curl -s "http://localhost:8080/debug/audit?trace_id=<trace_id>"`，能查到记录即表示该请求经 Diting 处理。
3. **审计日志文件**：若配置了 `audit.path`，直接查看该路径下的 JSONL，确认有对应时间戳与 trace_id 的记录。

排障时先确认代理/DNS 已生效，再查审计与 trace_id 定位请求是否经 Diting。

---

## 帮助

- 配置分层： [CONFIG_LAYERS.md](../../cmd/diting/CONFIG_LAYERS.md)
- 入口与备用： [MAIN_ENTRIES.md](../../cmd/diting/MAIN_ENTRIES.md)
- 文档索引： [DOCS_INDEX.md](../../cmd/diting/DOCS_INDEX.md)
