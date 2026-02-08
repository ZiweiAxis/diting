# Not receiving Feishu approval messages – troubleshooting

## 0. WebSocket mode (if you use it)

If you use **WebSocket** (main_ws_fixed / main_feishu_chat / main_ws):

- Approval messages are sent to the **chat_id** of the current session (the chat where you messaged the bot). You **do not** need to set `approval_user_id`.
- **Send one message to the app first**; the terminal will print `Chat ID` and `user_id`. After that, high-risk requests send approval to that chat.
- If you already sent a message and WebSocket is connected, the terminal should have shown "received Feishu message" and the **Chat ID**. If you still do not get approval: (1) ensure Feishu open platform has "long connection" enabled; (2) ensure you are running the WebSocket binary (e.g. `./diting_ws_fixed`), not main.

**Getting user_id from WebSocket**: Send any message to the bot again; the terminal prints "sender user_id: xxx", which you can use for main poll mode.

---

## 1. ID type (fixed in main)

**Cause**: `approval_user_id` was set to **open_id** (prefix `ou_`) but the code used `receive_id_type=user_id`, so the type did not match.

**Change**: The code now infers type from prefix:
- `ou_` → `receive_id_type=open_id`
- Otherwise → `receive_id_type=user_id`

Rebuild and run:

```bash
go build -o diting main.go && ./diting
```

## 2. Feishu app permissions

In [Feishu Open Platform](https://open.feishu.cn/app) → your app → Permission management, ensure these are enabled and applied:

| Permission | Purpose |
|------------|---------|
| **im:message:send_as_bot** | Send DM/group messages as the app (required for approval messages) |
| **im:message:read_as_user** (if using poll for replies) | Read user–app conversation for polling approval replies |

After publishing, confirm in "Permissions & scope" that they are checked and applied by admin.

## 3. User must have chatted with the bot

In some setups, the app can send messages reliably only **after the user has had at least one 1v1 conversation** (user opened the app in IM or sent one message). Have the approver search for the app in Feishu and send "hi", then try again.

## 4. Check gateway logs

After triggering one high-risk request, check the terminal:

- If you see `Approval request sent to Feishu, message_id=xxx` → the API call succeeded; the message was sent. If the user still does not see it, check permissions or conversation (above).
- If you see `Failed to send approval request: ...` → read the error (e.g. 99991663) and check [Feishu error codes](https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN).

## 5. Error "open_id cross app"

**Cause**: `approval_user_id` is an **open_id from another app**. Feishu requires the recipient ID to be under **this app**; open_id is per-app and cannot be used across apps.

**Fix**: Use the **user_id for this app** (not open_id):

- **If the user has already sent a message to this app**: use the repo tool to print user_id:
  ```bash
  cd cmd/diting
  go run get_feishu_user_id.go
  ```
  In Feishu open platform → Event subscription → set request URL to `http://your-public-url/feishu/event` (e.g. expose port 9000 with ngrok). Send one message to the app; the terminal prints **user_id**. Copy it to `config.json` → `feishu.approval_user_id`.
- Or: In [Feishu admin](https://open.feishu.cn/app) → your app → Directory/permissions → get user user_id;
- Or: Use Feishu API to [get user_id by phone/email](https://open.feishu.cn/document/server-docs/contact-v3/user/batch_get_id).

After setting **user_id** in config, the code uses `receive_id_type=user_id` and approval messages work.

## 6. Config checklist

In `config.json`:

- `feishu.approval_user_id`: **use user_id** (for this app) to avoid "open_id cross app". Do not use chat_id (`oc_xxx`).
- `feishu.enabled`: `true`.
