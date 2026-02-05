# 🚀 Sentinel-AI - 5 分钟快速开始

## 第一步: 安装 Python (如果没有)

### Windows
1. 访问: https://www.python.org/downloads/
2. 下载最新版本 (Python 3.8+)
3. 安装时 **务必勾选** "Add Python to PATH"
4. 安装完成后，打开新的命令行窗口

### 验证安装
```bash
python --version
```
应该显示: `Python 3.x.x`

---

## 第二步: 安装依赖

```bash
cd E:\workspace\sentinel-ai
pip install requests
```

---

## 第三步: 启动服务

### 方式 1: 使用启动脚本 (推荐)
```bash
start-python.bat
```

### 方式 2: 直接运行
```bash
python sentinel.py
```

### 预期输出
```
╔════════════════════════════════════════════════════════╗
║         Sentinel-AI 治理网关 MVP v0.1                 ║
║    企业级智能体零信任治理平台 - Python 版本           ║
╚════════════════════════════════════════════════════════╝

⚠️  警告: Ollama 未运行，将使用规则引擎模式
   启动 Ollama: ollama serve
   下载模型: ollama pull qwen2.5:7b

✓ 代理服务器启动成功
  监听地址: http://localhost:8080
  目标地址: http://httpbin.org

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 第四步: 测试功能

### 打开新的命令行窗口

### 测试 1: 安全请求 (自动放行)
```bash
curl http://localhost:8080/get
```

**预期结果:**
- 风险等级: 低 🟢
- 决策: 自动放行
- 无需人工审批

### 测试 2: 危险请求 (需要审批)
```bash
curl -X DELETE http://localhost:8080/delete
```

**预期结果:**
- 风险等级: 高 🔴
- 显示 LLM 意图分析
- 提示输入 y/n 进行审批

**操作:**
- 输入 `n` 拒绝请求
- 输入 `y` 批准请求

---

## 第五步: 查看审计日志

```bash
type logs\audit.jsonl
```

或者查看最后一条:
```bash
powershell -Command "Get-Content logs\audit.jsonl -Tail 1 | ConvertFrom-Json | ConvertTo-Json"
```

---

## 🎯 自动化测试

运行完整测试套件:
```bash
test-auto.bat
```

这会自动测试:
- ✅ GET 请求 (自动放行)
- ✅ HEAD 请求 (自动放行)
- ⚠️ POST 请求 (需要审批)
- ⚠️ DELETE 请求 (需要审批)
- ⚠️ 危险路径 (需要审批)

---

## 📚 下一步

### 1. 阅读完整文档
- [README.md](README.md) - 项目概述
- [INSTALL.md](INSTALL.md) - 详细部署指南
- [TEST.md](TEST.md) - 测试场景
- [DEMO.md](DEMO.md) - 演示脚本

### 2. 安装 Ollama (可选)
启用 AI 意图分析功能:
```bash
# 下载安装: https://ollama.ai/download
ollama serve
ollama pull qwen2.5:7b
```

然后重启 Sentinel-AI，就能看到 AI 分析了！

### 3. 配置真实后端
编辑 `sentinel.py` 第 18 行:
```python
"target_url": "http://your-real-api.com",
```

### 4. 准备演示
按照 [DEMO.md](DEMO.md) 准备 3 分钟演示

---

## ❓ 常见问题

### Q: 端口 8080 被占用怎么办?
**A:** 编辑 `sentinel.py` 第 17 行，改为其他端口:
```python
"proxy_listen": ("0.0.0.0", 8081),
```

### Q: curl 命令不存在?
**A:** Windows 10+ 自带 curl。如果没有:
- 使用 PowerShell: `Invoke-WebRequest`
- 或安装 Git Bash: https://git-scm.com/downloads

### Q: 如何停止服务?
**A:** 在运行 Sentinel-AI 的终端按 `Ctrl+C`

### Q: 审计日志在哪里?
**A:** `logs/audit.jsonl` (JSONL 格式，每行一条记录)

---

## 🎉 完成!

现在你已经有了一个可以工作的 AI Agent 治理网关！

**接下来可以:**
- 🎬 录制演示视频
- 📊 准备 PPT
- 💼 约见投资人/客户
- 🚀 开始下一阶段开发

---

**祝你成功! 🚀**

有问题查看 [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) 或 [INSTALL.md](INSTALL.md)
