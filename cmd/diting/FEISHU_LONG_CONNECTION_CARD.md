# 飞书长连接 + 卡片交互验证

## 长连接回传（推荐本地/内网）

**卡片点击只走长连接**：在飞书后台选择「使用长连接接收事件」并订阅 **card.action.trigger**，用户点击卡片按钮后，飞书通过 WebSocket 推送该事件，程序在长连接侧解析并调用审批，**不配置、不走 HTTP 回调**。发卡片时不会填 `request_url`，避免 200340。

若需公网 HTTP 回调（非长连接），再配置请求地址并填 `request_url`。

## 长连接 + 卡片验证步骤

1. **飞书开放平台**
   - 应用：企业自建。
   - 「事件与回调」→ 选择 **「使用长连接接收事件」**，保存。
   - 订阅事件中勾选 **card.action.trigger**（卡片回传），点击只走长连接，不走 HTTP 回调。

2. **配置**
   - `config.acceptance.yaml` 或 YAML 中：
     - `delivery.feishu.use_card_delivery: true` → 审批消息改为**交互卡片**（带「批准」「拒绝」按钮）。
     - `delivery.feishu.use_long_connection: true` → 启动时建立飞书 WebSocket 长连接。
   - 或环境变量：`DITING_FEISHU_USE_CARD_DELIVERY=true`、`DITING_FEISHU_USE_LONG_CONNECTION=true`。
   - 飞书 `app_id`、`app_secret`、`approval_user_id` 或 `chat_id` 照常配置。

3. **启动**
   ```bash
   cd cmd/diting && go run ./cmd/diting_allinone/ -config config.acceptance.yaml
   ```
   日志中应出现：`飞书长连接已建立` / `飞书长连接已启动（卡片交互事件将在此处理）`。

4. **触发审批**
   - 通过网关发一条会触发 CHEQ 的请求，使飞书收到**一条带「批准」「拒绝」按钮的卡片**。

5. **点击卡片按钮**
   - 在飞书中点击「批准」或「拒绝」。
   - 飞书按 **card.action.trigger** 经 WebSocket 推送给本程序，程序解析 `action.value` 后调用 `cheq.Submit(id, approved)` 完成审批。
   - 控制台应看到：`飞书长连接卡片审批: id=xxx approved=true`。**不经过 HTTP 回调。**

## 小结

- **长连接**：`use_long_connection: true`，飞书后台选「使用长连接接收事件」并订阅 **card.action.trigger**。
- **卡片**：`use_card_delivery: true`，审批消息为带按钮的卡片；长连接模式下不填 `request_url`，点击只走 WebSocket 回传。
- 验证时：两者都设为 true，发卡片后在飞书点击按钮，仅通过长连接完成审批，不走 HTTP 回调。
