# HTTPS 代理支持 - 测试指南

## ✅ 新功能

Diting v0.2.0 现在支持完整的 HTTPS 代理！

### 核心改进
- ✅ **CONNECT 方法支持** - 处理 HTTPS 隧道
- ✅ **动态目标** - 支持任意 HTTPS 域名
- ✅ **TLS 透传** - 不解密 HTTPS 流量（保护隐私）
- ✅ **风险评估** - 基于目标域名的风险分析
- ✅ **人工审批** - 高风险连接需要审批

---

## 🚀 快速测试

### 1. 启动 Diting

```bash
cd cmd/diting
go run main.go
```

### 2. 配置代理

#### 方式 A: 环境变量（推荐）

```bash
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# 测试 HTTP
curl http://httpbin.org/get

# 测试 HTTPS
curl https://api.github.com/users/octocat
```

#### 方式 B: curl 参数

```bash
# HTTP 请求
curl -x http://localhost:8080 http://httpbin.org/get

# HTTPS 请求
curl -x http://localhost:8080 https://api.github.com/users/octocat
```

#### 方式 C: Python requests

```python
import requests

proxies = {
    'http': 'http://localhost:8080',
    'https': 'http://localhost:8080',
}

# HTTP 请求
response = requests.get('http://httpbin.org/get', proxies=proxies)
print(response.json())

# HTTPS 请求
response = requests.get('https://api.github.com/users/octocat', proxies=proxies)
print(response.json())
```

---

## 📊 测试场景

### 场景 1: 安全域名（自动放行）

```bash
# 这些域名会被自动放行
curl -x http://localhost:8080 https://api.github.com/zen
curl -x http://localhost:8080 https://www.google.com
```

**预期结果**: 
- 风险等级: 低 🟢
- 决策: 自动放行
- 无需人工审批

---

### 场景 2: 未知域名（需要审批）

```bash
# 未知域名会触发审批
curl -x http://localhost:8080 https://api.example.com/data
```

**预期结果**:
- 风险等级: 中 🟡
- LLM 分析: "意图: API 调用。影响: 可能修改数据。建议: 建议审批。"
- 需要人工审批: 输入 y/n

---

### 场景 3: 危险域名（高风险）

```bash
# 包含危险关键词的域名
curl -x http://localhost:8080 https://malware.example.com
```

**预期结果**:
- 风险等级: 高 🔴
- 需要人工审批
- 建议拒绝

---

## 🔍 工作原理

### HTTP 请求流程

```
Client → Diting → 风险评估 → 决策 → 转发 → Target
```

### HTTPS 请求流程（CONNECT 方法）

```
1. Client 发送 CONNECT api.github.com:443
2. Diting 评估风险（基于域名）
3. 如果批准：
   - 返回 "200 Connection Established"
   - 建立 TCP 隧道
   - 双向转发加密数据（不解密）
4. 如果拒绝：
   - 返回 403 Forbidden
```

---

## 🎯 关键特性

### 1. 隐私保护

Diting **不解密** HTTPS 流量，只检查：
- 目标域名
- 连接时间
- 流量大小

**不检查**：
- HTTPS 请求内容
- HTTPS 响应内容
- 加密数据

### 2. 动态目标

支持任意目标域名，无需预配置：
```bash
curl -x http://localhost:8080 https://api.openai.com/v1/models
curl -x http://localhost:8080 https://api.stripe.com/v1/customers
curl -x http://localhost:8080 https://random-api.com/data
```

### 3. 风险评估

基于域名的智能风险评估：
- **低风险**: google.com, github.com, microsoft.com
- **中风险**: 未知域名
- **高风险**: 包含 malware, phishing, hack 等关键词

---

## 🐛 故障排查

### 问题 1: curl 报错 "Proxy CONNECT aborted"

**原因**: 连接被拒绝

**解决**: 
- 检查 Diting 日志
- 确认是否拒绝了审批
- 检查目标域名是否在黑名单

---

### 问题 2: Python requests 报错 "ProxyError"

**原因**: 代理配置错误

**解决**:
```python
# 确保代理格式正确
proxies = {
    'http': 'http://localhost:8080',   # 注意是 http://
    'https': 'http://localhost:8080',  # 不是 https://
}
```

---

### 问题 3: 证书验证失败

**原因**: 某些客户端会验证代理证书

**解决**:
```bash
# curl: 跳过证书验证
curl -k -x http://localhost:8080 https://example.com

# Python: 禁用证书验证
requests.get(url, proxies=proxies, verify=False)
```

---

## 📝 审计日志

HTTPS 连接会记录到 `logs/audit.jsonl`：

```json
{
  "timestamp": "2026-02-08T07:45:00Z",
  "method": "CONNECT",
  "host": "api.github.com:443",
  "path": "/",
  "risk_level": "低",
  "intent_analysis": "",
  "decision": "ALLOW",
  "response_code": 200,
  "duration_ms": 150
}
```

---

## 🎯 下一步

### 已完成 ✅
- [x] CONNECT 方法支持
- [x] HTTPS 隧道建立
- [x] 基于域名的风险评估
- [x] 人工审批流程
- [x] 审计日志

### 待完善 🔄
- [ ] TLS 拦截（可选，用于深度检查）
- [ ] 域名白名单/黑名单配置
- [ ] 连接池优化
- [ ] 性能测试

---

## 🤝 反馈

测试中遇到问题？请提交 Issue：
https://github.com/hulk-yin/diting/issues

---

**版本**: v0.2.0  
**更新时间**: 2026-02-08
