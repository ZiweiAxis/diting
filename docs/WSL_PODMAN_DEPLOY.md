# WSL/Podman 部署指南

## 🚀 快速启动（5 分钟）

### 方法 1: 使用 Podman (推荐）

```bash
# 1. 复制项目到 WSL
cd /mnt/e/workspace/sentinel-ai

# 2. 使用 Podman 创建容器（类似 Docker）
podman run -d --name sentinel-coredns \
    --network bridge \
    -p 53:53/udp -p 53:53/tcp \
    -v $(pwd)/coredns:/etc/coredns \
    coredns/coredns:1.11.1 \
    -conf /etc/coredns/Corefile

podman run -d --name sentinel-nginx \
    --network bridge \
    -p 8080:8080 -p 8443:8443 \
    -v $(pwd)/nginx:/etc/nginx \
    -v $(pwd)/logs:/var/log/nginx \
    openresty/openresty:alpine-fat \
    nginx

podman run -d --name sentinel-api \
    --network bridge \
    -p 8000:8000 \
    -v $(pwd)/logs:/app/logs \
    -e OPENAI_API_KEY=$OPENAI_API_KEY \
    python:3.12-slim \
    python -m uvicorn main:app --host 0.0.0.0 --port 8000
```

### 方法 2: 使用 Docker Desktop + WSL

```bash
# 1. 确保 Docker Desktop 的 WSL 2 集成已启用

# 2. 在 WSL 中运行
cd /mnt/e/workspace/sentinel-ai

# 3. 启动服务
docker-compose -f docker-compose-opensource.yml up -d
```

---

## 📋 WSL/Podman 注意事项

| 问题 | 说明 | 解决方案 |
|------|------|----------|
| hostNetwork 不支持 | podman 不支持 hostNetwork | 使用自定义网络 `--network bridge` |
| 端口映射 | Windows 防火墙可能阻止 | 开放端口 53, 8080, 8000, 8443 |
| 路径访问 | Windows 路径需要转换 | 使用 `/mnt/e/` 而不是 `E:\` |
| 权限问题 | 需要管理员权限 | 使用 sudo 或管理员 PowerShell |

---

## 🌐 在 WSL 中使用本地 Docker 套接

```bash
# Docker Desktop 的 Docker 命令行会在 WSL 中自动可用
# 检查
docker --version

# 运行 docker-compose
docker-compose -f docker-compose-opensource.yml up -d
```

---

## 🔍 检查服务

```bash
# CoreDNS
dig @localhost api.example.com

# Nginx WAF
curl http://localhost:8080/health

# Sentinel-AI API
curl http://localhost:8000/health
```

---

## 🛠️ 故障排查

### CoreDNS 无法解析

```bash
# 检查 CoreDNS 日志
podman logs sentinel-coredns

# 测试 DNS 解析
dig @localhost -p 53 example.com

# 检查配置
cat coredns/Corefile
```

### Nginx 无法启动

```bash
# 检查 Nginx 日志
podman logs sentinel-nginx

# 检查配置
nginx -t
```

### API 无法连接

```bash
# 检查 Sentinel-AI API 日志
podman logs sentinel-api

# 测试 API 连接
curl -v http://localhost:8000/health

# 检查环境变量
podman exec sentinel-api env | grep OPENAI
```

---

## 📊 WSL/Podman 架构

```
Windows 主机
     │
     ├─ Docker Desktop (WSL 2 集成)
     │   └─ WSL 2
     │       ├─ Podman
     │       │   └─ Sentinel-AI 容器
     │              │
     │          Docker Network
     │         │
     └──── 10.0.0.1 (Windows 主机)
```

---

## ✅ 完成清单

- [ ] CoreDNS 配置文件已创建
- [ ] Nginx 配置文件已创建
- [ ] Sentinel-AI API 已创建
- [ ] docker-compose.yml 已创建
- [ ] .env.example 已创建
- [ ] 启动脚本已创建
- [ ] WSL/Podman 部署指南已创建

---

**状态:** 100% 完成

**下一步:** 运行 `start-podman.bat` 或使用上面的命令启动服务
