# 验证交互卡片

按以下步骤验证飞书**交互卡片**（带「批准」「拒绝」按钮的审批消息）。

## 1. 配置

- 已启用交互卡片：
  - 在 config.yaml / config.example.yaml 中已设置 `delivery.feishu.use_card_delivery: true`，或环境变量 `DITING_FEISHU_USE_CARD_DELIVERY=true`。
- 飞书：`app_id`、`app_secret`、`approval_user_id` 或 `chat_id` 由 **.env** 的 DITING_* 提供。
- 策略会触发 review：例如 `policy_rules.example.yaml` 中 `resource: "/admin"` 的规则会走 `decision: review`。

## 2. 启动服务

```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
# 若用 config.json 注入飞书配置，会先读 config.json 再读 YAML
go run ./cmd/diting_allinone/
```

或先编译再运行：

```bash
go build -o diting_allinone ./cmd/diting_allinone/
./bin/diting
```

确认日志中有「飞书投递已启用」且无报错。

## 3. 触发待审批请求

发一条会命中「需 review」策略的请求，例如：

```bash
curl -s -X POST "http://localhost:8080/admin" -H "Host: example.com" -d '{}'
```

此时网关会挂起等待审批，并**向飞书发送一条消息**。

## 4. 在飞书中查看

- 打开飞书（或对应群聊），应收到一条 **「Diting 待确认」** 的**交互卡片**：
  - 标题：Diting 待确认
  - 正文：TraceID、ID、摘要、批准/拒绝链接
  - 两个按钮：**批准**、**拒绝**
- 若收到的是纯文本（没有卡片和按钮），请确认：
  - `use_card_delivery` 为 `true`（YAML 或 `DITING_FEISHU_USE_CARD_DELIVERY=true`）；
  - 使用的是当前 All-in-One 程序（带 `sendCard` 的版本）。

## 5. 验证按钮（可选）

- **方式 A：长连接**  
  - 在配置中设置 `use_long_connection: true`（或 `DITING_FEISHU_USE_LONG_CONNECTION=true`），飞书后台选择「使用长连接接收事件」。  
  - 重启服务后，在飞书中点击卡片上的「批准」或「拒绝」，审批结果会经长连接回调到本程序，请求会随之放行或拒绝。

- **方式 B：HTTP 回调**  
  - 飞书后台配置卡片回调地址为：`https://你的域名/feishu/card`（需公网可访问）。  
  - 点击按钮后，飞书会 POST 到该地址，本程序已实现 `POST /feishu/card` 会解析并调用 `cheq.Submit`。

完成以上步骤即表示**交互卡片**从发送到展示已验证通过；若需完整走通审批，再按 5 选一种回调方式即可。

---

### 点击按钮报错 200340

**原因**：飞书会向「卡片里的 request_url」或「应用后台配置的回调地址」发 HTTP 回调；该地址不可达（如 localhost）则报 200340。

**处理**：

1. **程序侧**：`use_long_connection: true` 时，卡片已显式设 `request_url: ""`，表示不走 HTTP 回调，只走长连接。请确认运行用的是带该逻辑的版本，且 config 中 use_long_connection: true 或环境变量 DITING_FEISHU_USE_LONG_CONNECTION=true。
2. **飞书后台**：若仍 200340，请到 **飞书开放平台 → 该应用 → 事件与回调**（或「回调订阅」）中，查看是否配置了「将回调发送至开发者服务器」的请求地址。若已配置且为本地/不可达地址，请**清空或删除该请求地址**，只保留「使用长连接接收事件」，这样卡片点击只会通过 WebSocket 推送，不再尝试 HTTP 回调。
