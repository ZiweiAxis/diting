# Diting 治理网关 v2.0.0 - 完整集成版

企业级智能体零信任治理平台，集成飞书长连接审批系统。

## 🎯 功能特性

### ✅ 已实现功能

1. **HTTP/HTTPS 代理服务器**（端口 8081）
   - ✅ HTTP 请求代理
   - ✅ HTTPS CONNECT 隧道
   - ✅ 完整的请求/响应转发

2. **智能风险评估**
   - ✅ 基于请求方法的风险评估
   - ✅ 基于 URL 路径的风险评估
   - ✅ 基于域名的风险评估
   - ✅ 三级风险等级：低/中/高

3. **飞书审批流程**
   - ✅ 飞书长连接（WebSocket）
   - ✅ 实时接收用户回复
   - ✅ 高风险操作自动触发审批
   - ✅ 低风险操作自动放行
   - ✅ 5分钟超时自动拒绝
   - ✅ 支持并发审批请求

4. **消息处理**
   - ✅ 解析飞书文本消息
   - ✅ 识别审批关键词（批准/拒绝）
   - ✅ 自动匹配最近的待审批请求

5. **审计日志**
   - ✅ JSONL 格式日志
   - ✅ 记录所有请求详情
   - ✅ 包含审批状态和耗时

## 📋 配置说明

### config.json

```json
{
  "proxy": {
    "listen": ":8081",
    "timeout_seconds": 30
  },
  "feishu": {
    "enabled": true,
    "app_id": "xxxx",
    "app_secret": "***",
    "chat_id": "xxxx",
    "approval_timeout_minutes": 5
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

- **proxy.listen**: 代理服务器监听地址
- **proxy.timeout_seconds**: 请求超时时间（秒）
- **feishu.chat_id**: 飞书群聊 ID（接收审批消息）
- **feishu.approval_timeout_minutes**: 审批超时时间（分钟）
- **risk.dangerous_methods**: 危险的 HTTP 方法
- **risk.dangerous_paths**: 危险的 URL 路径关键词
- **risk.auto_approve_methods**: 自动批准的方法
- **risk.safe_domains**: 安全域名列表
- **audit.log_file**: 审计日志文件路径

## 🚀 快速开始

### 1. 编译

```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
go build -o diting main_complete.go
```

### 2. 运行

```bash
./diting
```

或指定配置文件：

```bash
./diting config.json
```

### 3. 配置代理

将浏览器或应用的代理设置为：

```
HTTP Proxy: 127.0.0.1:8081
HTTPS Proxy: 127.0.0.1:8081
```

## 📊 风险评估规则

### 低风险（自动放行）
- GET、HEAD、OPTIONS 请求
- 访问安全域名列表中的域名

### 中风险（需要审批）
- 危险方法但非危险路径
- 危险路径但非危险方法
- 未知域名的普通请求

### 高风险（需要审批）
- 危险方法 + 危险路径
- 例如：DELETE /admin/users

## 💬 审批流程

### 1. 触发审批

当检测到高风险操作时，系统会：
1. 暂停请求
2. 发送审批消息到飞书群聊
3. 等待用户回复

### 2. 审批消息格式

```
🚨 高风险操作审批

📋 审批 ID: 12345678
🔗 请求方法: DELETE
🌐 目标 URL: https://api.example.com/admin/users
🏠 主机: api.example.com
⚠️  风险等级: high
⏰ 时间: 2026-02-08 11:24:00

请回复：
✅ 批准 / approve / y
❌ 拒绝 / reject / n

⏱️  5分钟后自动拒绝
```

### 3. 用户回复

在飞书群聊中回复以下任一关键词：

**批准**：
- `批准`
- `approve`
- `y`
- `yes`
- `同意`

**拒绝**：
- `拒绝`
- `reject`
- `n`
- `no`
- `不同意`

### 4. 自动超时

如果 5 分钟内没有回复，系统会：
- 自动拒绝请求
- 返回 403 Forbidden
- 记录审计日志

## 📝 审计日志

日志文件：`logs/audit.jsonl`

每行一条 JSON 记录：

```json
{
  "timestamp": "2026-02-08T11:24:00Z",
  "request_id": "uuid",
  "method": "DELETE",
  "url": "https://api.example.com/admin/users",
  "host": "api.example.com",
  "risk_level": "high",
  "status": "approved",
  "approval_id": "uuid",
  "duration_ms": 1234,
  "client_addr": "127.0.0.1:12345",
  "user_agent": "curl/7.68.0"
}
```

## 🎨 终端输出

系统使用彩色终端输出，便于监控：

- 🔵 **蓝色**：系统信息
- 🟢 **绿色**：成功操作
- 🟡 **黄色**：警告信息
- 🔴 **红色**：错误信息
- ⚪ **白色**：详细信息

## 🔧 并发支持

系统支持并发处理多个审批请求：

- 每个请求独立的审批流程
- 使用 UUID 标识每个审批
- 线程安全的审批管理器
- 自动匹配最近的待审批请求

## 🛡️ 安全特性

1. **零信任架构**：默认拒绝，显式批准
2. **实时审批**：高风险操作必须人工审批
3. **超时保护**：防止审批请求无限等待
4. **完整审计**：所有请求都有日志记录
5. **风险分级**：智能评估请求风险等级

## 📦 依赖项

```go
github.com/fatih/color
github.com/google/uuid
github.com/larksuite/oapi-sdk-go/v3
```

安装依赖：

```bash
go mod tidy
```

## 🐛 故障排查

### 1. 飞书长连接失败

检查：
- App ID 和 App Secret 是否正确
- 网络连接是否正常
- 飞书应用权限是否配置

### 2. 收不到审批消息

检查：
- Chat ID 是否正确
- 机器人是否已加入群聊
- 飞书应用是否有发送消息权限

### 3. 审批回复无效

检查：
- 是否在正确的群聊中回复
- 回复的关键词是否正确
- 是否有待审批的请求

### 4. 代理连接失败

检查：
- 端口 8081 是否被占用
- 防火墙是否允许连接
- 代理配置是否正确

## 📄 许可证

MIT License

## 👥 作者

Diting Team

## 🔗 相关链接

- [飞书开放平台](https://open.feishu.cn/)
- [飞书 SDK 文档](https://github.com/larksuite/oapi-sdk-go)
