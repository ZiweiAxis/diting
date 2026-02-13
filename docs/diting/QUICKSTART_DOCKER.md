# Diting All-in-One 容器 15 分钟快速开始

**所需环境**：已安装 Docker。飞书为可选（仅验证探针与代理时可不配置）。

从仓库根目录执行以下步骤，约 **15 分钟**内可完成「构建镜像 → 运行容器 → 验证探针与代理」。

---

## 1. 构建镜像（约 2 分钟）

```bash
# 在仓库根目录（diting）
docker build -f deployments/docker/Dockerfile.diting -t diting:latest .
```

---

## 2. 准备配置与环境变量

**方式 A：使用示例配置 + 环境变量（最快）**

复制环境变量示例并填写飞书等敏感项：

```bash
cp deployments/docker/.env.example.diting deployments/docker/.env.diting
# 编辑 .env.diting，填写 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID 等
```

**方式 B：挂载自己的 YAML 配置**

将本机 `config.yaml`（含 `policy.rules_path`、`cheq.persistence_path`、`delivery.feishu` 等）放到某目录，例如 `./myconfig/config.yaml`，运行容器时挂载为 `/app/config.yaml` 并设置 `CONFIG_PATH=/app/config.yaml`。

---

## 3. 运行容器

```bash
# 使用 .env.diting 中的环境变量，挂载数据目录以便持久化审计与 CHEQ
docker run -d --name diting \
  -p 8080:8080 \
  -v $(pwd)/data/diting:/app/data \
  --env-file deployments/docker/.env.diting \
  diting:latest
```

若使用自己的配置文件（方式 B）：

```bash
docker run -d --name diting \
  -p 8080:8080 \
  -v $(pwd)/data/diting:/app/data \
  -v $(pwd)/myconfig/config.yaml:/app/config.yaml \
  -e CONFIG_PATH=/app/config.yaml \
  --env-file deployments/docker/.env.diting \
  diting:latest
```

---

## 4. 验证

- **探针**  
  ```bash
  curl -s http://localhost:8080/healthz
  curl -s http://localhost:8080/readyz
  ```
  应返回 `ok` / `ready`。

- **代理与审批**（需上游可达且已配置飞书）  
  - 触发一条需审批的请求，例如：  
    `curl -s -X POST http://localhost:8080/admin -H "Host: example.com" -d '{}'`  
  - 在飞书收到待确认消息或交互卡片后，点击批准/拒绝或访问 `/cheq/approve?id=xxx&approved=true`，原请求应放行。

- **审计**  
  `curl -s "http://localhost:8080/debug/audit?trace_id=<X-Trace-ID>"` 可查看该请求的审计链（需配置了 `audit.path` 且已产生记录）。

---

## 5. 停止与清理

```bash
docker stop diting
docker rm diting
```

---

## 说明

- 默认配置使用 `config.example.yaml`，其中 `policy.rules_path` 为空时策略恒放行；若要走 review 流程，请挂载含 `policy.rules_path`（如 `policy_rules.example.yaml`）的配置，并设置 `cheq.persistence_path`、`delivery.feishu` 等。
- 飞书交互卡片 + 长连接：在配置中设置 `use_card_delivery: true`、`use_long_connection: true`，并在飞书后台选择「使用长连接接收事件」并订阅 `card.action.trigger`。详见 `VERIFY_CARD.md`、`FEISHU_LONG_CONNECTION_CARD.md`。
