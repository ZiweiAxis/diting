# Diting 快速启动

## 推荐飞书入口（优先）

**飞书审批**推荐使用 **main** 入口编译的 **diting**：审批消息发到配置的审批人，通过轮询收取回复。**无需飞书「长连接」**，也无需公网回调地址。

| 项目 | 说明 |
|------|------|
| **入口文件** | `main.go` |
| **编译产物** | `diting` |
| **配置** | `config.yaml` + `.env`：飞书相关由 `.env` 的 `DITING_FEISHU_*` 覆盖，详见 CONFIG_LAYERS.md |
| **回复** | 审批消息发到审批人单聊；审批人在飞书回复「approve & 请求ID」或「deny & 请求ID」 |

其他入口：
- **main_complete.go**（群聊 + 长连接/回调）：需飞书「事件订阅」或「长连接」及 `chat_id`；未开启长连接会 404。
- **main_feishu_chat.go**（群聊 + 轮询）：需先与机器人建会话获取 `chat_id`，适合群内审批。

---

## 第一步：安装依赖

```bash
cd cmd/diting
go mod tidy
# 如需：
go get github.com/fatih/color
go get github.com/google/uuid
```

## 第二步：编译（推荐飞书入口）

```bash
# 推荐：飞书消息回复审批（无需长连接）。本目录多入口，须指定文件构建。
go build -o diting main.go
```

或构建群聊+长连接版（需飞书开放平台开启长连接）：

```bash
go build -o diting main_complete.go
```

## 第二步 B：配置 config.yaml + .env（飞书必填）

首次使用：`cp config.example.yaml config.yaml`，`cp .env.example .env`，在 `.env` 中填入 `DITING_FEISHU_APP_ID`、`DITING_FEISHU_APP_SECRET`、`DITING_FEISHU_APPROVAL_USER_ID` 等（勿提交 .env）。

确保 `.env` 中飞书相关包含：

```json
"feishu": {
  "enabled": true,
  "app_id": "你的app_id",
  "app_secret": "你的app_secret",
  "approval_user_id": "审批人飞书 user_id",
  "approval_timeout_minutes": 5,
  "use_message_reply": true,
  "poll_interval_seconds": 2
}
```

## 第三步：运行

```bash
./diting
```

应看到类似输出：

```
✓ 配置加载成功
  LLM: Claude Haiku 3.5
  飞书: 消息回复模式
  审批人: <approval_user_id>

✓ 代理服务器启动成功
  监听: http://localhost:8081
```

## 第四步：测试

### 方式 1：测试脚本

```bash
./test.sh
```

### 方式 2：手动测试

#### 低风险请求（自动放行）

```bash
curl -x http://127.0.0.1:8081 https://httpbin.org/get
```

#### 高风险请求（需审批）

```bash
curl -x http://127.0.0.1:8081 -X DELETE https://httpbin.org/delete
```

审批消息会发到 `approval_user_id` 对应的飞书单聊。在飞书中按提示回复，例如：
- 批准：`approve abc12def`（abc12def 为消息中请求 ID 前 8 位）
- 拒绝：`deny abc12def`

（也支持「同意」「拒绝」等，需包含请求 ID。）

## 第五步：飞书配置

### 1. 使用默认 diting（推荐）

- 在飞书开放平台获取 **app_id**、**app_secret**。
- 获取审批人 **user_id**（管理后台或接口），填入 `.env` 的 `DITING_FEISHU_APPROVAL_USER_ID`。
- 无需开启「长连接」或配置公网回调。

### 2. 使用 main_complete（群聊 + 长连接）

- 确保机器人在 `chat_id` 对应群中。
- 飞书开放平台需开启「事件订阅」或「长连接」，否则会 404。
- 在群内发消息后，终端会显示收到的消息；在群内回复「批准」或「拒绝」完成审批。

### 3. 最小验证（3 步）

见项目根目录 `_bmad-output/feishu-approval-minimal-verification.md`。

## 第六步：查看审计日志

```bash
cat logs/audit.jsonl | jq
```

或实时查看：

```bash
tail -f logs/audit.jsonl | jq
```

## 常见用法

### 浏览器代理

将浏览器代理设为：127.0.0.1:8081（HTTP/HTTPS）。

### 命令行

```bash
export http_proxy=http://127.0.0.1:8081
export https_proxy=http://127.0.0.1:8081

curl https://api.example.com
wget https://example.com
```

### Python

```python
import requests
proxies = {
    'http': 'http://127.0.0.1:8081',
    'https': 'http://127.0.0.1:8081',
}
response = requests.get('https://api.example.com', proxies=proxies)
```

## 监控与调试

- 终端会显示：请求、风险评估、审批状态、飞书消息。
- 审计日志：`cat logs/audit.jsonl | jq`，可按 `.status`、`.risk_level` 等过滤。

## 停止 / 重启

- 停止：`Ctrl+C`
- 重启：`./diting`

## 自定义

编辑 `config.yaml` 与 `.env` 可修改：代理端口、风险规则、审批超时、审计日志路径。修改后重启生效。

## 验证清单

- [ ] 服务启动成功（`./diting`）
- [ ] 代理监听 8081，飞书显示「消息回复模式」
- [ ] 低风险（如 GET）自动放行
- [ ] 高风险（如 DELETE）触发审批，飞书收到消息
- [ ] 回复批准/拒绝后请求被放行/拦截
- [ ] 审计日志有记录（见第六步）

最小 3 步验证：见 `_bmad-output/feishu-approval-minimal-verification.md`。

## 需要帮助

完整文档：`README_COMPLETE.md`。飞书收不到消息等排错见 **[FEISHU_TROUBLESHOOTING_CN.md](FEISHU_TROUBLESHOOTING_CN.md)**（[English](FEISHU_TROUBLESHOOTING.md)）。
