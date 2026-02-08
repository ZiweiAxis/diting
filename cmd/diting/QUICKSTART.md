# Diting Quick Start

## Recommended Feishu entry (preferred)

**Feishu approval** is best used with the **main** entry built as **diting**: messages go to the configured approver and replies are polled. **No Feishu "long connection"** and no public callback URL are required.

| Item | Description |
|------|-------------|
| **Entry file** | `main.go` |
| **Binary** | `diting` |
| **Config** | In `config.json`, `feishu` must include: `enabled`, `app_id`, `app_secret`, **`approval_user_id`** (approver user_id), `approval_timeout_minutes`, **`use_message_reply`: true**, **`poll_interval_seconds`** (e.g. 2) |
| **Reply** | Approval messages go to the approver's DM; approver replies in Feishu with "approve & requestID" or "deny & requestID" |

Other entries:
- **main_complete.go** (group chat + long connection/callback): requires Feishu "event subscription" or "long connection" and `chat_id`. Returns 404 if long connection is not enabled.
- **main_feishu_chat.go** (group chat + poll): requires a session with the bot to get `chat_id`; suitable for in-group approval.

---

## Step 1: Install dependencies

```bash
cd cmd/diting
go mod tidy
# If needed:
go get github.com/fatih/color
go get github.com/google/uuid
```

## Step 2: Build (recommended Feishu entry)

```bash
# Recommended: Feishu message-reply approval (no long connection). Multiple mains in this dir; build by file.
go build -o diting main.go
```

Or build the group+long-connection variant (requires Feishu "long connection"):

```bash
go build -o diting main_complete.go
```

## Step 2b: Configure config.json (required for Feishu)

First time: copy `config.example.json` to `config.json`, then fill in your `app_id`, `app_secret`, etc. (do not commit `config.json` with secrets).

Ensure the Feishu section in `config.json` includes:

```json
"feishu": {
  "enabled": true,
  "app_id": "your_app_id",
  "app_secret": "your_app_secret",
  "approval_user_id": "approver Feishu user_id",
  "approval_timeout_minutes": 5,
  "use_message_reply": true,
  "poll_interval_seconds": 2
}
```

## Step 3: Run

```bash
./diting
```

You should see output similar to:

```
✓ Config loaded
  LLM: Claude Haiku 3.5
  Feishu: message reply mode
  Approver: <approval_user_id>

✓ Proxy server started
  Listen: http://localhost:8081
```

## Step 4: Test

### Option 1: Test script

```bash
./test.sh
```

### Option 2: Manual test

#### Low-risk request (auto-approved)

```bash
curl -x http://127.0.0.1:8081 https://httpbin.org/get
```

#### High-risk request (requires approval)

```bash
curl -x http://127.0.0.1:8081 -X DELETE https://httpbin.org/delete
```

Approval messages are sent to the Feishu DM of `approval_user_id`. In Feishu, reply as prompted, e.g.:
- Approve: `approve abc12def` (abc12def = first 8 chars of request ID in the message)
- Deny: `deny abc12def`

(Replies like "agree" / "reject" are also supported if they include the request ID.)

## Step 5: Feishu setup

### 1. Using default diting (recommended)

- Get **app_id** and **app_secret** from Feishu open platform.
- Get the approver's **user_id** (admin console or API) and set `approval_user_id` in `config.json`.
- No need to enable "long connection" or configure a public callback.

### 2. Using main_complete (group + long connection)

- Ensure the bot is in the group for `chat_id`.
- Feishu open platform must enable "event subscription" or "long connection", or you get 404.
- After sending a message in the group, the terminal shows received messages; reply "approve" or "deny" in the group.

### 3. Minimal verification (3 steps)

See repo root `_bmad-output/feishu-approval-minimal-verification.md`.

## Step 6: View audit logs

```bash
cat logs/audit.jsonl | jq
```

Or tail:

```bash
tail -f logs/audit.jsonl | jq
```

## Common usage

### Browser proxy

Set browser proxy to:
- HTTP/HTTPS Proxy: 127.0.0.1:8081

### Command line

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

## Monitoring and debugging

- Terminal shows: incoming requests, risk results, approval status, Feishu messages.
- Audit logs: `cat logs/audit.jsonl | jq`, filter by `.status`, `.risk_level`, etc.

## Stop / restart

- Stop: `Ctrl+C`
- Restart: `./diting`

## Customization

Edit `config.json` for: proxy port, risk rules, approval timeout, audit log path. Restart to apply.

## Verification checklist

- [ ] Service starts (`./diting`)
- [ ] Proxy listens on 8081; Feishu shows "message reply mode"
- [ ] Low-risk (e.g. GET) auto-approved
- [ ] High-risk (e.g. DELETE) triggers approval; Feishu receives message
- [ ] Approve/deny reply allows/blocks request
- [ ] Audit log has entries (Step 6)

Minimal 3-step verification: see `_bmad-output/feishu-approval-minimal-verification.md`.

## Help

Full docs: `README_COMPLETE.md`.
