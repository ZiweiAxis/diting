# 闭环验收检查单

按本单执行，验证：**策略 review → 飞书收到交互卡片 → 长连接回传点击 → 请求放行/拒绝** 全流程。

---

## 前置条件

- [ ] **飞书开放平台**：应用已选「使用长连接接收事件」，订阅 **card.action.trigger**；未配置或已清空 HTTP 回调地址（避免 200340）。
- [ ] **环境变量**：在 `cmd/diting` 目录下已配置飞书（任选其一）：
  - 复制 `.env.example` 为 `.env`，填入 `DITING_FEISHU_APP_ID`、`DITING_FEISHU_APP_SECRET`、`DITING_FEISHU_APPROVAL_USER_ID`（或 `DITING_FEISHU_CHAT_ID`）；
  - 或当前 shell 已 `export DITING_FEISHU_APP_ID=...` 等。

---

## 执行步骤

### 1. 释放 8080 并启动服务

```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
./run_acceptance.sh start
```

或手动：

```bash
# 若 8080 被占用，先结束占用进程
pkill -f diting_allinone   # 或 kill <PID>
# 启动（会加载 .env，主配置为 config.yaml 或 config.example.yaml）
go run ./cmd/diting_allinone/
# 或： ./bin/diting
```

**验收点**：终端出现  
`[diting] 飞书投递已启用，审批人将收到待确认消息`  
以及（数秒内）`[diting] 飞书长连接已建立，等待卡片交互事件...`。无 200340 相关报错。

### 2. 触发待审批请求

在**另一终端**执行：

```bash
curl -s -X POST "http://localhost:8080/admin" -H "Host: example.com" -d '{}'
```

此时请求会挂起，等待审批（最多约 120 秒，见 config.example.yaml 中 cheq.timeout_seconds）。

### 3. 在飞书中操作

- 打开飞书（或配置的群聊），应收到一条 **「Diting 待确认」** 的**交互卡片**（带「批准」「拒绝」按钮）。
- **点击「批准」** 或 **「拒绝」**。
- **验收点**：点击后**不出现 200340**；运行 diting 的终端出现类似  
  `[diting] 飞书长连接卡片审批: id=xxx approved=true`（或 `approved=false`）。

### 4. 验证请求结果

- 若点击了**批准**：步骤 2 的 `curl` 应返回 **200**（或上游 2xx），请求放行。
- 若点击了**拒绝**：`curl` 应返回 **403**（或相应拒绝响应）。

### 5. 审计（可选）

```bash
# 查看最近一条审计
./query_audit.sh -n 1
# 或按 trace_id
./query_audit.sh --trace-id <trace_id>
```

---

## 通过标准

- 飞书收到带「批准」「拒绝」的交互卡片。
- 点击按钮后无 200340，审批结果经长连接回传。
- 批准后请求放行（200），拒绝后请求被拒（403）。
- 审计中可查到对应 trace 的 allow/deny 记录。

全部满足即**闭环验收通过**。
