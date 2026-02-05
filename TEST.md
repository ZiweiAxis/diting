# Sentinel-AI 测试脚本

## 测试场景 1: 安全查询 (自动放行)

### 测试命令
```bash
curl -X GET http://localhost:8080/get
```

### 预期结果
- 风险等级: 低 🟢
- 决策: 自动放行
- 无需人工审批

---

## 测试场景 2: 危险删除 (需要审批)

### 测试命令
```bash
curl -X DELETE http://localhost:8080/delete
```

### 预期结果
- 风险等级: 高 🔴
- LLM 意图分析: "意图: 删除数据。影响: 数据不可恢复。建议: 需要审批。"
- 提示人工审批
- 输入 `y` 放行，输入 `n` 拒绝

---

## 测试场景 3: 修改生产数据 (需要审批)

### 测试命令
```bash
curl -X PUT http://localhost:8080/api/production/config \
  -H "Content-Type: application/json" \
  -d '{"setting": "value"}'
```

### 预期结果
- 风险等级: 高 🔴
- 路径包含 "production" 关键词
- 需要人工审批

---

## 测试场景 4: 带危险关键词的请求

### 测试命令
```bash
curl -X POST http://localhost:8080/api/cleanup \
  -H "Content-Type: application/json" \
  -d '{"action": "delete", "target": "old_logs"}'
```

### 预期结果
- 风险等级: 中 🟡 或 高 🔴
- 请求体包含 "delete" 关键词
- 需要人工审批

---

## 测试场景 5: 批量测试

### 使用 PowerShell 批量测试
```powershell
# 安全请求 (应该全部自动放行)
@("GET", "HEAD", "OPTIONS") | ForEach-Object {
    Write-Host "`n测试 $_ 请求..." -ForegroundColor Cyan
    curl -X $_ http://localhost:8080/anything
    Start-Sleep -Seconds 2
}

# 危险请求 (应该全部需要审批)
@("DELETE", "PUT", "PATCH") | ForEach-Object {
    Write-Host "`n测试 $_ 请求..." -ForegroundColor Yellow
    curl -X $_ http://localhost:8080/anything
    Start-Sleep -Seconds 2
}
```

---

## 查看审计日志

### 查看所有日志
```bash
cat logs/audit.jsonl
```

### 格式化查看最后一条日志
```bash
# Linux/Mac
tail -n 1 logs/audit.jsonl | jq .

# Windows PowerShell
Get-Content logs/audit.jsonl -Tail 1 | ConvertFrom-Json | ConvertTo-Json
```

### 统计决策分布
```bash
# Linux/Mac
cat logs/audit.jsonl | jq -r .decision | sort | uniq -c

# Windows PowerShell
Get-Content logs/audit.jsonl | ForEach-Object {
    ($_ | ConvertFrom-Json).decision
} | Group-Object | Select-Object Count, Name
```

---

## 性能测试

### 使用 Apache Bench
```bash
# 测试 100 个安全请求的性能
ab -n 100 -c 10 http://localhost:8080/get
```

### 预期性能指标
- 低风险请求: < 10ms
- 高风险请求 (含 LLM): < 2000ms
- 吞吐量: > 100 req/s (低风险)

---

## 集成测试: 模拟 Agent 行为

### Python 测试脚本
```python
import requests
import time

# 配置代理
proxy_url = "http://localhost:8080"

# 模拟 Agent 的一系列操作
operations = [
    ("GET", "/api/users", None, "查询用户列表"),
    ("POST", "/api/users", {"name": "test"}, "创建用户"),
    ("DELETE", "/api/users/123", None, "删除用户"),
    ("PUT", "/api/production/config", {"key": "value"}, "修改生产配置"),
]

for method, path, data, desc in operations:
    print(f"\n{'='*60}")
    print(f"操作: {desc}")
    print(f"请求: {method} {path}")
    
    try:
        if method == "GET":
            resp = requests.get(proxy_url + path)
        elif method == "POST":
            resp = requests.post(proxy_url + path, json=data)
        elif method == "PUT":
            resp = requests.put(proxy_url + path, json=data)
        elif method == "DELETE":
            resp = requests.delete(proxy_url + path)
        
        print(f"状态码: {resp.status_code}")
        if resp.status_code == 403:
            print("❌ 操作被拒绝")
            print(resp.json())
        else:
            print("✓ 操作成功")
    except Exception as e:
        print(f"错误: {e}")
    
    time.sleep(2)
```

---

## 故障测试

### 测试 Ollama 离线降级
1. 停止 Ollama 服务
2. 发送危险请求
3. 验证系统降级到规则引擎模式

### 测试超时处理
1. 修改代码添加审批超时逻辑
2. 发送危险请求后不输入
3. 验证 5 分钟后自动拒绝

---

## 压力测试

### 并发请求测试
```bash
# 使用 wrk 工具
wrk -t4 -c100 -d30s http://localhost:8080/get
```

### 预期结果
- 系统应该能处理 > 1000 req/s
- 无崩溃或内存泄漏
- 所有请求都有审计日志
