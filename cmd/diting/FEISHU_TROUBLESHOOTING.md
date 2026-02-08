# 飞书收不到审批消息 - 排查说明

## 0. WebSocket 模式（你当前用的）

**若你用的是 WebSocket 长连接**（main_ws_fixed / main_feishu_chat / main_ws）：

- 审批消息是发到**当前会话的 chat_id**（你给机器人发消息的那条会话），**不需要**填 `approval_user_id`。
- 使用前**先给应用发一条消息**，终端会打印 `Chat ID` 和 `user_id`；之后触发高风险请求时，审批会发到该会话。
- 若你已发过一条消息，只要 WebSocket 已连上，终端里应出现过「收到飞书消息」和对应的 **Chat ID**；下次高风险请求就会发到该会话。若仍收不到审批，请确认：① 飞书开放平台已开启「长连接」；② 当前运行的是 WebSocket 版（如 `./diting_ws_fixed`），不是 main。

**从 WebSocket 拿到 user_id**：再给机器人发任意一条消息，终端会打印「发送者 user_id: xxx」，可复制用于 main 轮询模式。

---

## 1. ID 类型已修复（main）

**原因**：配置里 `approval_user_id` 填的是 **open_id**（`ou_` 开头），但之前发消息用了 `receive_id_type=user_id`，类型不一致会导致发错或失败。

**修改**：代码已改为根据前缀自动识别：
- `ou_` 开头 → 使用 `receive_id_type=open_id`
- 否则 → 使用 `receive_id_type=user_id`

请重新编译并启动后再试：
```bash
go build -o diting main.go && ./diting
```

## 2. 飞书应用权限

在 [飞书开放平台](https://open.feishu.cn/app) → 你的应用 → 权限管理，确认已开通并生效：

| 权限 | 说明 |
|------|------|
| **im:message:send_as_bot** | 以应用身份发送单聊/群聊消息（发审批消息必需） |
| **im:message:read_as_user**（若用轮询收回复） | 读取用户与应用的会话消息，用于轮询审批回复 |

发布/生效后，在「权限与范围」里确认已勾选并已让管理员生效。

## 3. 用户是否与机器人有过会话

部分飞书环境下，应用**先要和用户有过 1v1 会话**才能稳定发消息（用户曾在 IM 里点进过该应用或发过一条消息）。可让审批人在飞书里搜索该应用并发一条「hi」再试。

## 4. 看网关日志

触发一次高风险请求后看终端：

- 若出现 `Approval request sent to Feishu, message_id=xxx` → 说明接口调用成功，消息应已发出；若仍收不到，多半是权限或会话问题（见上）。
- 若出现 `Failed to send approval request: ...` → 看后面的错误信息（如 99991663 等），对照 [飞书错误码](https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN) 排查。

## 5. 报错「open_id cross app」

**原因**：`approval_user_id` 填的是**其他应用下的 open_id**。飞书规定：发消息时用的接收人 ID 必须是**当前应用**下的标识；open_id 按应用隔离，不能跨应用使用。

**处理**：改为使用**本应用下该用户的 user_id**（不是 open_id）：

- **你已给应用发过一条消息时**：用本仓库自带工具打印 user_id：
  ```bash
  cd cmd/diting
  go run get_feishu_user_id.go
  ```
  飞书开放平台 → 事件订阅 → 请求地址填 `http://你的公网地址/feishu/event`（本地可用 ngrok 暴露 9000 端口）。再给应用发一条消息，终端会打印 **user_id**，复制到 `config.json` 的 `feishu.approval_user_id`。
- 或：在 [飞书管理后台](https://open.feishu.cn/app) → 你的应用 → 权限/通讯录 → 获取用户 user_id；
- 或：调用飞书 API [通过手机号/邮箱获取 user_id](https://open.feishu.cn/document/server-docs/contact-v3/user/batch_get_id)。

配置里填 **user_id** 后，代码会使用 `receive_id_type=user_id` 发送，即可正常发审批消息。

## 6. 配置检查

`config.json` 里：

- `feishu.approval_user_id`：**推荐填 user_id**（本应用下），避免「open_id cross app」。不要填成 chat_id（`oc_xxx`）。
- `feishu.enabled` 为 `true`。
