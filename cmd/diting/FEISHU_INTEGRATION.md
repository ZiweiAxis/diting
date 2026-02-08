# 飞书集成改造方案

## 📋 改造目标

将谛听（Diting）的审批流程从命令行交互改为飞书消息审批，并使用 Claude Haiku 替代 Ollama。

---

## 🔄 主要变更

### 1. LLM 集成：Ollama → Claude Haiku

**变更前**:
```go
OllamaEndpoint: "http://localhost:11434"
OllamaModel: "qwen2.5:7b"
```

**变更后**:
```json
{
  "llm": {
    "provider": "anthropic",
    "base_url": "https://d01bad1e79ad-vip.aicoding.sh",
    "api_key": "aicoding-617126d04e7745e2c593d78665552c7f",
    "model": "claude-haiku-3-5",
    "max_tokens": 1024,
    "temperature": 0.7
  }
}
```

**优势**:
- ✅ 无需本地部署 Ollama
- ✅ Claude Haiku 更便宜（相比 Sonnet）
- ✅ 响应速度快
- ✅ 质量稳定

---

### 2. 审批流程：命令行 → 飞书

**变更前**:
```go
// 命令行交互
fmt.Print("是否批准此操作? (y/n): ")
reader := bufio.NewReader(os.Stdin)
input, _ := reader.ReadString('\n')
```

**变更后**:
```go
// 发送飞书消息
sendFeishuApprovalRequest(requestInfo)
// 等待飞书回调或消息回复
decision := waitForFeishuApproval(requestID, timeout)
```

**飞书消息格式**:

#### 方式 1: 交互式卡片（推荐）
```json
{
  "msg_type": "interactive",
  "card": {
    "header": {
      "title": {
        "content": "🚨 Diting 高风险操作审批",
        "tag": "plain_text"
      },
      "template": "red"
    },
    "elements": [
      {
        "tag": "div",
        "text": {
          "content": "**操作**: DELETE /api/users/123\n**风险等级**: 高 🔴\n**意图分析**: 删除用户数据，不可恢复",
          "tag": "lark_md"
        }
      },
      {
        "tag": "action",
        "actions": [
          {
            "tag": "button",
            "text": {
              "content": "✅ 批准",
              "tag": "plain_text"
            },
            "type": "primary",
            "value": {
              "action": "approve",
              "request_id": "req_123456"
            }
          },
          {
            "tag": "button",
            "text": {
              "content": "❌ 拒绝",
              "tag": "plain_text"
            },
            "type": "danger",
            "value": {
              "action": "reject",
              "request_id": "req_123456"
            }
          }
        ]
      }
    ]
  }
}
```

#### 方式 2: 普通消息（降级方案）
```
🚨 Diting 高风险操作审批

操作: DELETE /api/users/123
风险等级: 高 🔴
意图分析: 删除用户数据，不可恢复

请回复：
- "批准" 或 "approve" 或 "y" 来批准
- "拒绝" 或 "reject" 或 "n" 来拒绝

⏱️ 5分钟内未响应将自动拒绝
```

---

## 🏗️ 技术架构

### 组件交互流程

```
┌─────────────┐
│   Agent     │
│  (任意框架)  │
└──────┬──────┘
       │ HTTP/HTTPS
       ▼
┌─────────────────────────────────────┐
│      Diting 治理网关 (Go)           │
├─────────────────────────────────────┤
│  1. 拦截请求                        │
│  2. 风险评估                        │
│  3. Claude Haiku 意图分析           │
│  4. 高风险 → 发送飞书审批           │
│  5. 等待审批结果                    │
│  6. 执行决策 + 审计日志             │
└────────┬────────────────────────────┘
         │
         ├─→ Claude Haiku API
         │   (意图分析)
         │
         └─→ 飞书 API
             (审批消息)
                 │
                 ▼
         ┌──────────────┐
         │  飞书用户    │
         │  点击按钮    │
         │  或回复消息  │
         └──────────────┘
```

---

## 🔧 实现细节

### 1. Claude Haiku 集成

```go
type ClaudeRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    MaxTokens   int       `json:"max_tokens"`
    Temperature float64   `json:"temperature"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

func analyzeIntentWithClaude(method, path, body string) string {
    prompt := fmt.Sprintf(`分析以下 API 操作的意图和风险：
方法: %s
路径: %s
请求体: %s

请简要说明：
1. 操作意图
2. 潜在影响
3. 是否建议审批`, method, path, body)

    req := ClaudeRequest{
        Model: config.LLM.Model,
        Messages: []Message{
            {Role: "user", Content: prompt},
        },
        MaxTokens:   config.LLM.MaxTokens,
        Temperature: config.LLM.Temperature,
    }

    // 调用 Claude API
    resp := callClaudeAPI(req)
    return resp.Content[0].Text
}
```

### 2. 飞书审批集成

```go
// 审批请求结构
type ApprovalRequest struct {
    RequestID      string    `json:"request_id"`
    Method         string    `json:"method"`
    Path           string    `json:"path"`
    RiskLevel      string    `json:"risk_level"`
    IntentAnalysis string    `json:"intent_analysis"`
    Timestamp      time.Time `json:"timestamp"`
    Status         string    `json:"status"` // pending/approved/rejected/timeout
}

// 全局审批请求映射
var approvalRequests = sync.Map{}

// 发送飞书审批请求
func sendFeishuApprovalRequest(req ApprovalRequest) error {
    // 存储请求
    approvalRequests.Store(req.RequestID, &req)

    // 构建飞书消息
    if config.Feishu.UseInteractiveCard {
        // 发送交互式卡片
        return sendFeishuCard(req)
    } else {
        // 发送普通消息
        return sendFeishuMessage(req)
    }
}

// 等待审批结果
func waitForFeishuApproval(requestID string, timeout time.Duration) string {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // 检查审批状态
            if val, ok := approvalRequests.Load(requestID); ok {
                req := val.(*ApprovalRequest)
                if req.Status == "approved" {
                    return "ALLOW"
                } else if req.Status == "rejected" {
                    return "DENY"
                }
            }

            // 检查超时
            if time.Now().After(deadline) {
                // 超时自动拒绝
                if val, ok := approvalRequests.Load(requestID); ok {
                    req := val.(*ApprovalRequest)
                    req.Status = "timeout"
                    approvalRequests.Store(requestID, req)
                }
                sendFeishuTimeoutNotification(requestID)
                return "DENY"
            }
        }
    }
}

// 处理飞书回调（卡片按钮点击）
func handleFeishuCallback(w http.ResponseWriter, r *http.Request) {
    var callback FeishuCallback
    json.NewDecoder(r.Body).Decode(&callback)

    requestID := callback.Action.Value["request_id"]
    action := callback.Action.Value["action"] // "approve" or "reject"

    if val, ok := approvalRequests.Load(requestID); ok {
        req := val.(*ApprovalRequest)
        if action == "approve" {
            req.Status = "approved"
        } else {
            req.Status = "rejected"
        }
        approvalRequests.Store(requestID, req)
    }

    w.WriteHeader(http.StatusOK)
}

// 处理飞书消息回复
func handleFeishuMessage(message FeishuMessage) {
    // 解析消息内容
    content := strings.ToLower(strings.TrimSpace(message.Content))
    
    // 查找待审批的请求
    approvalRequests.Range(func(key, value interface{}) bool {
        req := value.(*ApprovalRequest)
        if req.Status == "pending" {
            // 匹配审批关键词
            if content == "批准" || content == "approve" || content == "y" {
                req.Status = "approved"
                approvalRequests.Store(key, req)
                sendFeishuConfirmation(message.UserID, "✅ 已批准操作")
                return false
            } else if content == "拒绝" || content == "reject" || content == "n" {
                req.Status = "rejected"
                approvalRequests.Store(key, req)
                sendFeishuConfirmation(message.UserID, "❌ 已拒绝操作")
                return false
            }
        }
        return true
    })
}
```

---

## 📊 配置说明

### config.json 完整配置

```json
{
  "proxy": {
    "listen": ":8081",
    "timeout_seconds": 30
  },
  "llm": {
    "provider": "anthropic",
    "base_url": "https://d01bad1e79ad-vip.aicoding.sh",
    "api_key": "aicoding-617126d04e7745e2c593d78665552c7f",
    "model": "claude-haiku-3-5",
    "max_tokens": 1024,
    "temperature": 0.7
  },
  "feishu": {
    "enabled": true,
    "approval_user_id": "ou_c06d8e07a92b69d09889a055cb6725bc",
    "approval_timeout_minutes": 5,
    "use_interactive_card": true,
    "fallback_to_message": true
  },
  "risk": {
    "dangerous_methods": ["DELETE", "PUT", "PATCH", "POST"],
    "dangerous_paths": ["/delete", "/remove", "/drop", "/destroy", "/clear", "/admin", "/production"],
    "auto_approve_methods": ["GET", "HEAD", "OPTIONS"],
    "safe_domains": ["api.github.com", "httpbin.org"]
  },
  "audit": {
    "log_file": "logs/audit.jsonl",
    "enabled": true
  }
}
```

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `proxy.listen` | 代理监听地址 | `:8081` |
| `proxy.timeout_seconds` | 请求超时时间 | `30` |
| `llm.model` | Claude 模型 | `claude-haiku-3-5` |
| `llm.max_tokens` | 最大生成 token 数 | `1024` |
| `feishu.approval_user_id` | 审批人飞书 ID | 必填 |
| `feishu.approval_timeout_minutes` | 审批超时时间（分钟） | `5` |
| `feishu.use_interactive_card` | 是否使用交互式卡片 | `true` |
| `feishu.fallback_to_message` | 卡片失败时降级到消息 | `true` |

---

## 🧪 测试场景

### 场景 1: 低风险操作（自动放行）
```bash
curl -x http://localhost:8081 http://httpbin.org/get
```
**预期**: 自动放行，无需审批

### 场景 2: 高风险操作（飞书审批）
```bash
curl -x http://localhost:8081 -X DELETE http://httpbin.org/delete
```
**预期**:
1. Diting 发送飞书审批消息
2. 用户点击"批准"或"拒绝"按钮
3. Diting 执行相应操作

### 场景 3: 审批超时
```bash
curl -x http://localhost:8081 -X DELETE http://httpbin.org/delete
# 5分钟内不响应
```
**预期**:
1. 5分钟后自动拒绝
2. 发送超时通知到飞书

---

## 📝 待办事项

- [x] 创建配置文件 config.json
- [ ] 改造 main.go 集成 Claude Haiku
- [ ] 实现飞书审批逻辑
- [ ] 添加飞书回调处理
- [ ] 实现审批超时机制
- [ ] 编写单元测试
- [ ] 更新文档

---

## 🚀 部署步骤

1. **配置文件**
   ```bash
   cp config.json.example config.json
   # 修改 feishu.approval_user_id 为你的飞书 ID
   ```

2. **编译运行**
   ```bash
   go build -o diting main.go
   ./diting
   ```

3. **配置飞书回调**
   - 在飞书开放平台配置回调 URL
   - 设置事件订阅（接收消息）

4. **测试验证**
   ```bash
   # 测试低风险操作
   curl -x http://localhost:8081 http://httpbin.org/get
   
   # 测试高风险操作
   curl -x http://localhost:8081 -X DELETE http://httpbin.org/delete
   ```

---

**状态**: 🚧 开发中  
**预计完成时间**: 2026-02-08  
**负责人**: OpenClaw AI Assistant
