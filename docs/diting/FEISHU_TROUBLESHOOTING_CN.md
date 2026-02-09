# 飞书收不到审批消息 - 排查说明

## 0. WebSocket 模式（若使用）

若使用 **WebSocket**（main_ws_fixed / main_feishu_chat / main_ws）：

- 审批消息发到**当前会话的 chat_id**（你给机器人发消息的会话），**不需要**填 `approval_user_id`。
- **先给应用发一条消息**，终端会打印 `Chat ID` 和 `user_id`；之后高风险请求的审批会发到该会话。
- 若已发过消息且 WebSocket 已连上，终端应出现过「收到飞书消息」和 **Chat ID**。若仍收不到审批，请确认：① 飞书开放平台已开启「长连接」；② 当前运行的是 WebSocket 版（如 `./diting_ws_fixed`），不是 main。

**从 WebSocket 获取 user_id**：再给机器人发任意一条消息，终端会打印「发送者 user_id: xxx」，可用于 main 轮询模式。

---

## 1. ID 类型（main 已修复）

**原因**：`approval_user_id` 填的是 **open_id**（`ou_` 开头），但之前发消息用了 `receive_id_type=user_id`，类型不一致。

**修改**：代码已按前缀自动识别：
- `ou_` 开头 → 使用 `receive_id_type=open_id`
- 否则 → 使用 `receive_id_type=user_id`

重新编译并运行：

```bash
go build -o diting main.go && ./diting
```

## 2. 飞书应用权限

在 [飞书开放平台](https://open.feishu.cn/app) → 你的应用 → 权限管理，确认已开通并生效：

| 权限 | 说明 |
|------|------|
| **im:message:send_as_bot** | 以应用身份发送单聊/群聊消息（发审批消息必需） |
| **im:message:read_as_user**（若用轮询收回复） | 读取用户与应用的会话消息，用于轮询审批回复 |

发布/生效后，在「权限与范围」中确认已勾选并由管理员生效。

## 3. 用户是否与机器人有过会话

部分环境下，应用**先要和用户有过 1v1 会话**才能稳定发消息（用户曾在 IM 里打开该应用或发过一条消息）。可让审批人在飞书中搜索该应用并发「hi」再试。

## 4. 看网关日志

触发一次高风险请求后看终端：

- 若出现 `Approval request sent to Feishu, message_id=xxx` → 接口调用成功，消息已发出；若仍收不到，多为权限或会话问题（见上）。
- 若出现 `Failed to send approval request: ...` → 看后面错误码（如 99991663），对照 [飞书错误码](https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN) 排查。

## 5. 报错「open_id cross app」

**原因**：`approval_user_id` 填的是**其他应用下的 open_id**。飞书规定接收人 ID 必须是**当前应用**下的标识；open_id 按应用隔离，不能跨应用使用。

**处理**：改为使用**本应用下该用户的 user_id**（不要用 open_id），并显式指定 `receive_id_type=user_id`：

- **All-in-One（config.yaml / config.example.yaml）**：在 `delivery.feishu` 下设置  
  `receive_id_type: user_id`，并把 `approval_user_id` 改为该用户的 **user_id**（或环境变量 `DITING_FEISHU_RECEIVE_ID_TYPE=user_id`、`DITING_FEISHU_APPROVAL_USER_ID=<user_id>`）。
- **获取 user_id**：  
  - 若用户已给本应用发过消息：`go run get_feishu_user_id.go`，按提示收事件后终端会打印 **user_id**；  
  - 或 [飞书管理后台](https://open.feishu.cn/app) → 你的应用 → 通讯录/权限；  
  - 或调用 API [通过手机号/邮箱获取 user_id](https://open.feishu.cn/document/server-docs/contact-v3/user/batch_get_id)。

配置为 **user_id + receive_id_type=user_id** 后，即可正常发审批消息。

## 6. 配置检查

**.env 或 config.yaml** 中：

- `DITING_FEISHU_APPROVAL_USER_ID`（或 YAML `delivery.feishu.approval_user_id`）：**推荐填 user_id**（本应用下），避免「open_id cross app」。不要填 chat_id（`oc_xxx`）。
- 飞书投递启用：YAML `delivery.feishu.enabled: true` 或通过 .env 覆盖。
