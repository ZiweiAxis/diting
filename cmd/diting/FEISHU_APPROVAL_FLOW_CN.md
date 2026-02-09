# 如何看到飞书审批流程

**与飞书的对接方式**：当前 All-in-One 与飞书对接为 **REST API 发消息** + **网关审批链接**完成批准/拒绝，**不依赖飞书回调**。若你需要从飞书接收事件（如用户发消息），应使用飞书 **长连接（WebSocket）**，不是 HTTP 回调。

验收时若**未设置飞书相关环境变量**，网关会使用**占位投递**（不往飞书发消息），因此看不到飞书审批。按下面步骤配置后，review 请求会发到飞书，您可在飞书内看到消息并完成审批。

**重要**：飞书消息**不是在启动时发送的**，而是当**有请求命中「需人工确认」（review）策略时**才会创建待确认并往飞书发消息。所以需要先触发一次 review 请求（见下方步骤 3）。

---

## 1. 必须配置的环境变量

在**启动 diting 之前**在同一终端执行（或写入 `.env` 后 `source .env`）：

```bash
# 必填：飞书应用凭证（从飞书开放平台获取）
export DITING_FEISHU_APP_ID=xxxx
export DITING_FEISHU_APP_SECRET=***

# 必填：审批人飞书 open_id（谁要收到待确认消息）
export DITING_FEISHU_APPROVAL_USER_ID=xxxx
```

- `DITING_FEISHU_APP_ID` / `DITING_FEISHU_APP_SECRET`：未设置时不会启用飞书投递，只会用占位（不发消息）。
- `DITING_FEISHU_APPROVAL_USER_ID`：不设置时飞书 API 会报「无 receive_id」；也可用配置里的 `ownership.static_map` 指定确认人（见下）。

可选（群聊兜底）：

```bash
export DITING_FEISHU_CHAT_ID=xxxx   # 无审批人时发到群
```

---

## 2. 用验收配置启动

```bash
cd cmd/diting

# 1）设置上述环境变量后
# 2）启动上游（可选，用于代理转发）
# python3 -c "import http.server,socketserver; socketserver.TCPServer(('',8081), http.server.BaseHTTPRequestHandler).serve_forever()" &

# 3）启动网关（会看到 “[diting] 飞书投递已启用，审批人将收到待确认消息”）
./bin/diting
```

若未配置飞书，启动时会打印：

`[diting] 飞书未配置 app_id/app_secret，使用占位投递（不发飞书）。设置 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET 后可见飞书审批流程`

---

## 3. 触发一次 review 并在飞书审批（必做才会收到飞书消息）

1. 保持 diting 在**当前终端**运行（已出现「飞书投递已启用」即可）。
2. 在**另一个终端**发会走 review 的请求（当前规则下访问 `/admin` 会走 review）：
   ```bash
   curl -v http://127.0.0.1:8080/admin
   ```
   该请求会**阻塞**，直到有人批准或超时。**此时**网关会往飞书发一条待确认消息。

2. **飞书**：您（或配置的审批人）会收到一条文本消息，内容包含：
   - 待确认请求
   - TraceID、ID、摘要
   - 提示「请在网关或 CLI 完成批准/拒绝」

3. **批准方式任选其一**：
   - 在浏览器打开（把 `<id>` 换成消息里的 ID）：
     - 批准：`http://localhost:8080/cheq/approve?id=<id>&approved=true`
     - 拒绝：`http://localhost:8080/cheq/approve?id=<id>&approved=false`
   - 或用 curl：
     ```bash
     curl "http://127.0.0.1:8080/cheq/approve?id=<id>&approved=true"
     ```

4. 批准后，步骤 1 里阻塞的 `curl http://127.0.0.1:8080/admin` 会返回 200 并得到上游响应（若已起 8081 上游）。

---

## 4. 用 static_map 指定审批人（可选）

若不想用环境变量 `DITING_FEISHU_APPROVAL_USER_ID`，可在配置里用 `ownership.static_map` 指定「谁审批哪类资源」：

```yaml
ownership:
  static_map:
    "*": ["你的飞书 open_id"]           # 默认审批人
    "/admin": ["你的飞书 open_id"]      # 管理路径审批人
```

只要 **app_id / app_secret** 通过环境变量配置好，且 **approval_user_id 或 static_map** 中至少有一个能提供接收人，就会走飞书投递并看到飞书审批流程。

---

## 5. 小结

| 现象 | 原因 | 处理 |
|------|------|------|
| 没收到飞书消息 | 未设置 `DITING_FEISHU_APP_ID` / `DITING_FEISHU_APP_SECRET` | 设置后重启 diting |
| 没收到飞书消息 | 未设置审批人（`DITING_FEISHU_APPROVAL_USER_ID` 或 `static_map`） | 设置其一后重启 |
| 启动时提示「使用占位投递」 | 同上，未启用飞书 | 按上文配置环境变量后重启 |

配置正确时，启动会看到 **「飞书投递已启用，审批人将收到待确认消息」**，此时触发 review 即可在飞书看到审批流程。
