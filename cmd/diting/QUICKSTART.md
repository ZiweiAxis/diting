# Diting 快速启动指南

## 🎯 推荐飞书入口（优先使用）

**飞书审批**推荐使用 **main** 入口编译的 **diting**：消息发到指定审批人、轮询回复，**无需飞书开放平台开启「长连接」**，也无需公网回调地址。

| 项目 | 说明 |
|------|------|
| **入口文件** | `main.go` |
| **编译产物** | `diting` |
| **配置要求** | `config.json` 中 `feishu` 需包含：`enabled`、`app_id`、`app_secret`、**`approval_user_id`**（审批人 user_id）、`approval_timeout_minutes`、**`use_message_reply`: true**、**`poll_interval_seconds`**（如 2） |
| **回复方式** | 审批消息发到审批人单聊；审批人在飞书回复「approve & 请求ID」或「deny & 请求ID」 |

其他入口说明：
- **main_complete.go**（群聊 + 长连接/回调）：需飞书「事件订阅」或「长连接」可用，且配置 `chat_id`。若开放平台未开启长连接，会 404。
- **main_feishu_chat.go**（群聊 + 轮询）：需先与机器人建会话以获取 chat_id，适合群内审批。

---

## 📦 第一步：安装依赖

```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
go mod tidy
# 如需
go get github.com/fatih/color
go get github.com/google/uuid
```

## 🔧 第二步：编译（推荐飞书入口）

```bash
# 推荐：飞书消息回复审批（无需长连接）。本目录多入口，须指定文件构建。
go build -o diting main.go
```

或使用群聊+长连接版（需飞书开放平台开启长连接）：

```bash
go build -o diting main_complete.go
```

## ⚙️ 第二步 B：配置 config.json（飞书必填）

首次使用：复制 `config.example.json` 为 `config.json`，再填入自己的 app_id、app_secret 等（勿提交含密钥的 config.json）。

确保 `config.json` 中飞书段包含以下字段（用默认 `diting` 时必填）：

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

## ▶️ 第三步：运行

```bash
./diting
```

你应该看到类似的输出：

```
✓ 配置加载成功
  LLM: Claude Haiku 3.5
  飞书: 消息回复模式
  审批人: <approval_user_id>

✓ 代理服务器启动成功
  监听地址: http://localhost:8081
```

## 🧪 第四步：测试

### 方法 1：使用测试脚本

```bash
./test.sh
```

### 方法 2：手动测试

#### 测试低风险请求（自动放行）

```bash
curl -x http://127.0.0.1:8081 https://httpbin.org/get
```

#### 测试高风险请求（需要审批）

```bash
curl -x http://127.0.0.1:8081 -X DELETE https://httpbin.org/delete
```

审批消息会发到 `approval_user_id` 对应的飞书单聊。在飞书中按消息提示回复，例如：
- 批准：`approve abc12def`（abc12def 为消息中的请求 ID 前 8 位）
- 拒绝：`deny abc12def`

（也支持回复「同意」「拒绝」等，需包含请求 ID。）

## 📱 第五步：配置飞书

### 1. 使用默认 diting（推荐）

- 在飞书开放平台获取应用的 **app_id**、**app_secret**。
- 获取审批人的 **user_id**（在飞书管理后台或通过接口查询），填入 `config.json` 的 `approval_user_id`。
- 无需开启「长连接」或配置公网回调。

### 2. 使用 main_complete（群聊 + 长连接）

- 确保机器人已加入 `chat_id` 对应群聊。
- 飞书开放平台需开启「事件订阅」或「长连接」，否则会 404。
- 在群聊中发送消息后，终端会显示收到的消息；审批时在群内回复 `批准` 或 `拒绝`。

### 3. 最小验证（3 步）

见项目根目录 `_bmad-output/feishu-approval-minimal-verification.md`。

## 📊 第六步：查看审计日志

```bash
cat logs/audit.jsonl | jq
```

或者实时监控：

```bash
tail -f logs/audit.jsonl | jq
```

## 🎯 常见使用场景

### 场景 1：浏览器代理

在浏览器中配置代理：
- HTTP Proxy: 127.0.0.1:8081
- HTTPS Proxy: 127.0.0.1:8081

### 场景 2：命令行工具

```bash
export http_proxy=http://127.0.0.1:8081
export https_proxy=http://127.0.0.1:8081

# 然后使用任何命令行工具
curl https://api.example.com
wget https://example.com
```

### 场景 3：Python 脚本

```python
import requests

proxies = {
    'http': 'http://127.0.0.1:8081',
    'https': 'http://127.0.0.1:8081',
}

response = requests.get('https://api.example.com', proxies=proxies)
```

## 🔍 监控和调试

### 实时监控终端输出

Diting 会实时显示：
- 收到的请求
- 风险评估结果
- 审批状态
- 飞书消息

### 查看审计日志

```bash
# 查看所有日志
cat logs/audit.jsonl | jq

# 查看被拒绝的请求
cat logs/audit.jsonl | jq 'select(.status == "rejected")'

# 查看高风险请求
cat logs/audit.jsonl | jq 'select(.risk_level == "high")'

# 统计请求数量
cat logs/audit.jsonl | wc -l
```

## 🛑 停止服务

按 `Ctrl+C` 停止服务。

## 🔄 重启服务

```bash
./diting
```

## 📝 自定义配置

编辑 `config.json` 来自定义：
- 代理端口
- 风险评估规则
- 审批超时时间
- 审计日志路径

修改后重启服务即可生效。

## ✅ 验证清单

- [ ] 服务启动成功（`./diting`）
- [ ] 代理监听 8081，飞书显示「消息回复模式」
- [ ] 低风险请求（如 GET）自动放行
- [ ] 高风险请求（如 DELETE）触发审批、飞书收到消息
- [ ] 回复批准/拒绝后请求被放行/拦截
- [ ] 审计日志有记录（见第六步）

**最小 3 步验证**：见 `_bmad-output/feishu-approval-minimal-verification.md`。

## 🆘 需要帮助？

查看完整文档：`README_COMPLETE.md`
