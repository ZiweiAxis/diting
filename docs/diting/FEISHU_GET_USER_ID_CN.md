# 如何拿到审批人的 open_id / user_id

飞书对接接收事件有两种方式：**长连接（WebSocket）** 与 **HTTP 回调**。若你已用长连接，在建立长连接后从「收消息」事件里即可拿到发送者的 open_id / user_id，无需本页监听器。

无法从飞书后台直接看到、且未用长连接时，可**启动本机 HTTP 回调监听服务**，等用户给应用发一条消息后，终端会打印该用户的 **open_id** 和 **user_id**。

---

## 1. 启动监听服务

在项目目录下执行（需本机已安装 Go）：

```bash
cd cmd/diting
go run feishu_id_listener.go
```

会看到类似：

```
========================================
  飞书 open_id / user_id 监听服务
========================================
  本机: http://0.0.0.0:9000/feishu/event
  飞书开放平台 → 事件订阅 → 选择 HTTP 回调 → 请求地址填: http://<你的公网地址>/feishu/event
  本地需用 ngrok 等暴露端口: 9000
  配置好后，给应用发一条消息，此处会打印发送者的 open_id 与 user_id。
  （若你使用长连接而非 HTTP 回调，可从长连接收消息事件中直接获取 ID，无需本服务。）
========================================
```

---

## 2. 把本机端口暴露到公网（否则飞书无法访问）

飞书只会请求「公网可访问」的 URL，本地 `localhost:9000` 不行，需要内网穿透。

**用 ngrok（示例）：**

```bash
# 安装后执行（保持运行）
ngrok http 9000
```

会得到一个公网地址，例如：`https://xxxx.ngrok.io`。

---

## 3. 在飞书开放平台配置事件订阅（HTTP 回调方式）

1. 打开 [飞书开放平台](https://open.feishu.cn/app) → 进入你的应用（与 .env 里 DITING_FEISHU_APP_ID 一致）。
2. 左侧 **事件订阅** → 若你使用 **长连接**，无需配置请求地址，从长连接事件即可获取 user_id；若使用 **HTTP 回调**，在 **请求地址** 填：  
   `https://xxxx.ngrok.io/feishu/event`（把 `xxxx.ngrok.io` 换成你的 ngrok 地址）。
3. 点击 **保存**（HTTP 回调时），飞书会发一次验证请求；监听服务终端应出现：`[OK] 飞书请求地址验证已通过`。
4. 在 **订阅事件** 里勾选 **接收消息**（`im.message.receive_v1`），保存。

---

## 4. 让审批人给应用发一条消息

在飞书里找到你的应用（搜索应用名或从「工作台」进入），**以要当审批人的账号**给应用发任意一条消息（例如「hi」）。

---

## 5. 在监听服务终端查看 ID

终端会打印类似：

```
========================================
  收到飞书消息，发送者 ID（本应用下）
========================================
  open_id:  ou_xxxxxxxxxx
  user_id:  xxxxxxxxx
----------------------------------------
  建议：为避免 open_id cross app(99992361)，请使用 user_id
  1. .env 中设置 DITING_FEISHU_APPROVAL_USER_ID = 上面的 user_id
  2. 设置 DITING_FEISHU_RECEIVE_ID_TYPE=user_id
========================================
```

把 **user_id** 填到 **.env** 的 `DITING_FEISHU_APPROVAL_USER_ID`，并设置 `DITING_FEISHU_RECEIVE_ID_TYPE=user_id`，即可用该用户收审批消息。

---

## 可选：改监听端口

默认 9000，可通过环境变量改：

```bash
PORT=9090 go run feishu_id_listener.go
```

此时 ngrok 需对应暴露 9090：`ngrok http 9090`。
