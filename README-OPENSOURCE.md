# Sentinel-AI 开源版本部署指南

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────┐
│         Agent 容器                                  │
│                                                         │
│  requests.get('http://api.example.com/data')   │
└─────────────────┬───────────────────────────────────┘
                  │
                  │ DNS 查询
                  ▼
┌─────────────────────────────────────────────────────┐
│         CoreDNS (开源，CNCF 毕业）                 │
│                                                         │
│  api.example.com → 10.0.1 (Nginx IP)        │
│  db.example.com → 10.0.1                        │
└─────────────────┬───────────────────────────────────┘
                  │
                  │ HTTP
                  ▼
┌─────────────────────────────────────────────────────┐
│       Nginx/OpenResty (开源）                      │
│                                                         │
│  Lua 脚本调用 Sentinel-AI API                     │
│  返回决策: ALLOW / REVIEW / BLOCK                │
└─────────────────┬───────────────────────────────────┘
                  │
                  │ API 调用
                  ▼
┌─────────────────────────────────────────────────────┐
│     Sentinel-AI 业务逻辑 (Python)                    │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ OpenAI    │  │ 风险评估  │  │ 审批流  │  │
│  │ 意图分析  │  │ 引擎       │  │          │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
│       │              │              │              │
│       └──────────────┼──────────────┘              │
│                      │                             │
│              返回决策                                │
└────────────────────┼───────────────────────────────────┘
                      │
                      │ 执行决策
                      ▼
              ┌──────────────────────┐
              │   真实后端服务        │
              └──────────────────────┘
```

---

## 📦 组件说明

| 组件 | 开源项目 | 版本 | 用途 |
|------|----------|------|------|
| **CoreDNS** | CoreDNS | 1.11.1 | DNS 劫持，所有域名指向 WAF |
| **Nginx** | OpenResty | Alpine | 反向代理 + Lua 脚本 |
| **Sentinel-API** | 自研 | Python 3.12 | OpenAI 意图分析 + 风险评估 |
| **etcd** | CoreOS | 3.5.9 | 可选，动态 DNS 配置 |

---

## 🚀 快速部署

### 步骤 1: 准备环境

**必需:**
- Docker Desktop
- OpenAI API Key

**可选:**
- etcd（如果需要动态 DNS 管理）

---

### 步骤 2: 配置环境变量

**Windows (PowerShell):**

```powershell
# 复制环境变量模板
copy .env.example .env

# 编辑 .env，设置你的 API Key
notepad .env
```

设置以下内容：
```
OPENAI_API_KEY=sk-xxxxx
OPENAI_MODEL=gpt-4o-mini
```

---

### 步骤 3: 启动服务

**Windows:**

```bash
# 一键启动
start-opensource.bat

# 或手动启动
docker-compose -f docker-compose-opensource.yml up -d
```

**Linux/Mac:**

```bash
# 启动所有服务
docker-compose -f docker-compose-opensource.yml up -d
```

---

### 步骤 4: 配置 Agent DNS

**Docker Agent 容器:**

```yaml
apiVersion: v1
kind: Pod
spec:
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 10.0.0.1  # 指向 CoreDNS
  containers:
  - name: agent
    image: your-agent-image
```

**Kubernetes:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dns-config
data:
  resolv.conf: |
    nameserver 10.0.0.1
```

**物理机/虚拟机:**

```bash
# 修改 /etc/resolv.conf
echo "nameserver 10.0.0.1" > /etc/resolv.conf
```

---

### 步骤 5: 测试

**测试 DNS 解析:**

```bash
nslookup api.example.com 10.0.0.1
# 应返回: 10.0.0.1
```

**测试 WAF 网关:**

```bash
# 安全请求（自动放行）
curl http://localhost:8080/api/users

# 危险请求（需要审批）
curl -X DELETE http://localhost:8080/api/users/123
```

**测试 API:**

```bash
# 健康检查
curl http://localhost:8000/health

# 测试分析
curl -X POST http://localhost:8000/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "method": "DELETE",
    "uri": "/api/users/123",
    "headers": {},
    "body": "{}",
    "client_ip": "10.0.1.5",
    "host": "api.example.com",
    "timestamp": 1700000000000
  }'
```

---

## 📊 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| CoreDNS | 53/udp, 53/tcp | DNS 服务 |
| Nginx WAF | 8080/http, 8443/https | 代理网关 |
| Sentinel-API | 8000/http | 业务逻辑 API |
| etcd | 2379, 2380 | etcd API |

---

## 🔧 高级配置

### 添加新域名

编辑 `coredns/Corefile`:

```coredns
example.com:53 {
    file {
        zonefile /etc/coredns/example.com.db
    }
}

. {
    hosts {
        10.0.0.1 new-domain.com
        10.0.0.1 another-domain.com
    }
    log
    errors
}
```

---

### 修改 OpenAI 模型

编辑 `.env`:

```bash
# 使用不同的模型
OPENAI_MODEL=gpt-3.5-turbo      # 更快，更便宜
OPENAI_MODEL=gpt-4o             # 更智能，更贵
OPENAI_MODEL=gpt-4o-mini        # 平衡，推荐
```

---

### 自定义规则引擎

编辑 `sentinel-api/main.py` 中的 `RiskEngine` 类:

```python
class RiskEngine:
    def __init__(self):
        # 添加你的自定义规则
        self.dangerous_methods = ["DELETE", "PUT"]
        self.dangerous_paths = ["/delete", "/admin"]
        self.dangerous_keywords = ["drop", "truncate"]
```

---

## 📝 日志查看

```bash
# WAF 日志
docker logs -f nginx-waf

# API 日志
docker logs -f sentinel-api

# CoreDNS 日志
docker logs -f coredns
```

---

## 🛠️ 故障排查

### CoreDNS 无法解析

```bash
# 检查 CoreDNS 配置
docker exec coredns cat /etc/coredns/Corefile

# 查看 CoreDNS 日志
docker logs -f coredns

# 测试 DNS 解析
docker exec coredns dig @localhost example.com
```

---

### Nginx 502 Bad Gateway

```bash
# 检查后端连接
docker exec nginx-waf wget -O- http://backend-service:8080/health

# 检查 Sentinel-API 是否运行
curl http://localhost:8000/health

# 查看 Nginx 错误日志
docker logs -f nginx-waf 2>&1 | grep error
```

---

### OpenAI API 错误

```bash
# 检查 API Key
docker exec sentinel-api env | grep OPENAI_API_KEY

# 测试 API 连接
curl -X GET https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# 查看 API 日志
docker logs -f sentinel-api 2>&1 | grep error
```

---

## 🎯 下一步

- [ ] 集成企业微信/钉钉审批
- [ ] 添加 Web 管理界面
- [ ] 实现动态 DNS 更新
- [ ] 添加监控和告警
- [ ] 性能测试和优化

---

## 📞 技术支持

- CoreDNS: https://coredns.io/manual/toc/
- OpenResty: https://openresty.org/
- OpenAI API: https://platform.openai.com/docs

---

**版本:** 2.0.0 (基于开源工具）  
**更新时间:** 2026-02-05
