# Sentinel-AI 部署指南

## 快速开始 (3 分钟)

### 方式 1: Python 版本 (推荐 - 无需编译)

#### 1. 安装 Python
- 下载: https://www.python.org/downloads/
- 版本要求: Python 3.8+
- 安装时勾选 "Add Python to PATH"

#### 2. 安装依赖
```bash
cd E:\workspace\sentinel-ai
pip install -r requirements.txt
```

#### 3. 启动服务
```bash
# Windows
start-python.bat

# 或直接运行
python sentinel.py
```

#### 4. 测试
```bash
# 新开一个终端
curl http://localhost:8080/get
```

---

### 方式 2: Go 版本 (高性能)

#### 1. 安装 Go
- 下载: https://go.dev/dl/
- 版本要求: Go 1.21+
- 安装后重启终端

#### 2. 编译运行
```bash
cd E:\workspace\sentinel-ai

# Windows
start.bat

# Linux/Mac
chmod +x start.sh
./start.sh
```

---

## 可选: 安装 Ollama (本地 LLM)

### 为什么需要 Ollama?
- 提供 AI 意图分析能力
- 不安装也能运行 (会降级到规则引擎)

### 安装步骤

#### Windows
1. 下载: https://ollama.ai/download
2. 安装后自动启动服务
3. 下载模型:
```bash
ollama pull qwen2.5:7b
```

#### Linux/Mac
```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve &
ollama pull qwen2.5:7b
```

### 验证安装
```bash
curl http://localhost:11434/api/tags
```

---

## 测试验证

### 1. 安全请求 (应该自动放行)
```bash
curl -X GET http://localhost:8080/get
```

**预期输出:**
```
[23:15:30] 收到请求
  方法: GET
  路径: /get
  风险等级: 低 🟢
  决策: 自动放行
  耗时: 5ms
```

### 2. 危险请求 (应该需要审批)
```bash
curl -X DELETE http://localhost:8080/delete
```

**预期输出:**
```
[23:15:35] 收到请求
  方法: DELETE
  路径: /delete
  风险等级: 高 🔴

  🤖 LLM 意图分析:
  意图: 删除数据。影响: 数据不可恢复。建议: 需要审批。

╔════════════════════════════════════════════════════════╗
║                  🚨 需要人工审批                       ║
╚════════════════════════════════════════════════════════╝

  请求: DELETE /delete
  分析: 意图: 删除数据。影响: 数据不可恢复。建议: 需要审批。

  是否批准此操作? (y/n): _
```

输入 `n` 拒绝，输入 `y` 放行。

---

## 配置说明

### Python 版本配置
编辑 `sentinel.py` 中的 `CONFIG` 字典:

```python
CONFIG = {
    "proxy_listen": ("0.0.0.0", 8080),  # 监听地址和端口
    "target_url": "http://httpbin.org",  # 真实后端地址
    "ollama_endpoint": "http://localhost:11434",
    "ollama_model": "qwen2.5:7b",
    "dangerous_methods": ["DELETE", "PUT", "PATCH", "POST"],
    "dangerous_paths": ["/delete", "/remove", "/drop"],
    "auto_approve_methods": ["GET", "HEAD", "OPTIONS"],
}
```

### Go 版本配置
编辑 `main.go` 中的 `config` 变量 (第 24 行):

```go
var config = Config{
    ProxyListen:       ":8080",
    TargetURL:         "http://httpbin.org",
    OllamaEndpoint:    "http://localhost:11434",
    OllamaModel:       "qwen2.5:7b",
    // ...
}
```

---

## 生产部署建议

### 1. 修改目标地址
将 `target_url` 改为你的真实 API 地址:
```python
"target_url": "http://your-api.example.com"
```

### 2. 配置 Agent
修改 Agent 的 API 端点:
```python
# 原来
api_url = "http://your-api.example.com"

# 改为
api_url = "http://localhost:8080"  # 通过 Sentinel-AI 代理
```

### 3. 启用 HTTPS (可选)
使用 Nginx 反向代理:
```nginx
server {
    listen 443 ssl;
    server_name sentinel.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
    }
}
```

### 4. 持久化运行

#### Windows (使用 NSSM)
```bash
# 下载 NSSM: https://nssm.cc/download
nssm install Sentinel-AI "C:\Python\python.exe" "E:\workspace\sentinel-ai\sentinel.py"
nssm start Sentinel-AI
```

#### Linux (使用 systemd)
创建 `/etc/systemd/system/sentinel-ai.service`:
```ini
[Unit]
Description=Sentinel-AI Gateway
After=network.target

[Service]
Type=simple
User=sentinel
WorkingDirectory=/opt/sentinel-ai
ExecStart=/usr/bin/python3 sentinel.py
Restart=always

[Install]
WantedBy=multi-user.target
```

启动服务:
```bash
sudo systemctl enable sentinel-ai
sudo systemctl start sentinel-ai
```

---

## 故障排查

### 问题 1: 端口被占用
```
OSError: [Errno 48] Address already in use
```

**解决方法:**
```bash
# 查找占用端口的进程
netstat -ano | findstr :8080

# 杀死进程
taskkill /PID <进程ID> /F

# 或修改配置使用其他端口
"proxy_listen": ("0.0.0.0", 8081)
```

### 问题 2: Ollama 连接失败
```
⚠️  警告: Ollama 未运行
```

**解决方法:**
```bash
# 检查 Ollama 是否运行
curl http://localhost:11434/api/tags

# 如果没有运行，启动它
ollama serve

# 如果没有安装，下载安装
# https://ollama.ai/download
```

### 问题 3: Python 依赖安装失败
```
ERROR: Could not find a version that satisfies the requirement requests
```

**解决方法:**
```bash
# 升级 pip
python -m pip install --upgrade pip

# 使用国内镜像
pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
```

### 问题 4: 无法访问目标 API
```
Bad Gateway: Connection refused
```

**解决方法:**
- 检查 `target_url` 配置是否正确
- 确认目标 API 可以访问
- 检查防火墙设置

---

## 性能优化

### 1. 使用 Go 版本
Go 版本性能是 Python 版本的 5-10 倍:
- Python: ~200 req/s
- Go: ~2000 req/s

### 2. 禁用 LLM 分析
如果不需要 AI 分析，可以注释掉 LLM 调用代码，纯规则引擎模式延迟 < 5ms。

### 3. 使用更快的 LLM 模型
```bash
# 使用更小的模型
ollama pull qwen2.5:3b  # 更快，但准确度略低

# 或使用量化版本
ollama pull qwen2.5:7b-q4_0
```

---

## 下一步

1. 阅读 [TEST.md](TEST.md) 了解测试场景
2. 阅读 [DEMO.md](DEMO.md) 准备演示
3. 查看 [README.md](README.md) 了解完整功能

---

## 技术支持

- GitHub Issues: (待创建)
- 邮箱: support@sentinel-ai.example.com
- 文档: https://docs.sentinel-ai.example.com
