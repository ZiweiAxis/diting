# Sentinel-AI 基于开源工具的完整配置

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────┐
│              Agent 容器                                  │
│                                                          │
│  requests.get('http://api.example.com/data')        │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ DNS 查询
                 ▼
┌─────────────────────────────────────────────────────────┐
│           CoreDNS (开源，CNCF 毕业)                  │
│                                                          │
│  api.example.com → 10.0.1.1 (Nginx IP)            │
│  db.example.com → 10.0.1.1                             │
│                                                          │
│  插件: sentinel-ai (自定义插件)                      │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ HTTP
                 ▼
┌─────────────────────────────────────────────────────────┐
│            Nginx / OpenResty (开源）                  │
│                                                          │
│  location / {                                          │
│    access_by_lua_block {                                │
│      local http = require "resty.http")     │
│      local httpc = http.new()                           │
│      -- 调用 Sentinel-AI API 分析                      │
│      local res, err = httpc:request_uri(             │
│        "http://sentinel-ai:8000/analyze",            │
│        {                                              │
│          method = "POST",                             │
│          body = cjson.encode({                         │
│            method = ngx.var.request_method,             │
│            uri = ngx.var.request_uri,                 │
│            headers = ngx.req.get_headers(),              │
│            body = ngx.var.request_body               │
│          })                                            │
│        }                                               │
│      )                                                │
│                                                        │
│      -- 根据决策执行                                   │
│      if res.status == 403 then                      │
│        ngx.exit(403)                                   │
│      end                                            │
│    }                                                │
│                                                        │
│    proxy_pass http://backend;                       │
│  }                                                  │
└────────────────┬─────────────────────────────────────────┘
                 │
                 │ API 调用
                 ▼
┌─────────────────────────────────────────────────────────┐
│         Sentinel-AI 业务逻辑服务 (Python）              │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ OpenAI 分析  │  │ 风险评估      │  │ 审批工作流  │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
│                                                          │
│  返回决策: ALLOW / REVIEW / BLOCK                   │
└─────────────────────────────────────────────────────────┘
```

---

## 📦 部署文件

### 1. CoreDNS 配置

**文件:** `coredns/Corefile`

```coredns
example.com:53 {
    etcd {
        # 从 etcd 读取域名映射
        path /skydns
        endpoint http://etcd:2379
    }
    
    # 或使用文件存储
    file {
        zonefile /etc/coredns/example.com.db
    }
    
    # 日志
    log
    
    # 错误处理
    errors
    
    # 默认返回 Nginx IP
    # 所有 example.com 域名都指向 10.0.1.1
}

# 自定义 DNS 响应（劫持所有域名）
. {
    hosts {
        # 所有域名都指向 Sentinel-AI WAF
        10.0.1.1 api.example.com
        10.0.1.1 db.example.com
        10.0.1.1 auth.example.com
        # 通配符（如果支持）
        10.0.1.1 *.example.com
    }
    log
    errors
}
```

---

### 2. Nginx/OpenResty 配置

**文件:** `nginx/nginx.conf`

```nginx
worker_processes auto;
events {
    worker_connections 1024;
}

http {
    # 上游后端（真实服务）
    upstream backend_real {
        # 可以通过 Sentinel-AI API 动态更新
        server 1.2.3.4:80;
        server 5.6.7.8:3306;
        server 9.10.11.12:443;
        keepalive 32;
    }

    # Sentinel-Ai API（业务逻辑）
    upstream sentinel_api {
        server sentinel-ai:8000;
        keepalive 16;
    }

    server {
        listen 8080;
        server_name _;

        # 客户端请求体大小
        client_max_body_size 10M;

        # Sentinel-AI 分析 Lua 脚本
        init_by_lua_block {
            require "resty.http"
            require "resty.core"
            require "cjson.safe"
        }

        # 默认 location（所有请求先分析）
        location / {
            # 缓存 Sentinel-AI 决策
            access_by_lua_block {
                local cache_key = ngx.var.request_method .. ":" .. ngx.var.uri .. ":" .. ngx.var.remote_addr
                
                -- 检查缓存
                local cached = ngx.shared.decision_cache:get(cache_key)
                if cached then
                    ngx.log(ngx.INFO, "Using cached decision: ", cached)
                    
                    if cached == "BLOCK" then
                        ngx.status = 403
                        ngx.say('{"error":"Blocked by Sentinel-AI (cached)"}')
                        ngx.exit(403)
                    elseif cached == "ALLOW" then
                        -- 继续到 proxy_pass
                    else
                        -- REVIEW 状态
                        ngx.status = 202
                        ngx.say('{"message":"Request pending approval"}')
                        ngx.exit(202)
                    end
                end

                -- 调用 Sentinel-AI API 分析
                local httpc = http.new()
                local res, err = httpc:request_uri(
                    "http://sentinel-ai:8000/analyze",
                    {
                        method = "POST",
                        body = cjson.encode({
                            method = ngx.var.request_method,
                            uri = ngx.var.request_uri,
                            headers = ngx.req.get_headers(),
                            body = ngx.var.request_body,
                            client_ip = ngx.var.remote_addr,
                            host = ngx.var.host,
                            timestamp = ngx.now() * 1000
                        }),
                        headers = {
                            ["Content-Type"] = "application/json"
                        },
                        timeout = 2000  -- 2秒超时
                    }
                )

                if not res then
                    ngx.log(ngx.ERR, "Sentinel-AI API error: ", err)
                    -- 出错时默认放行
                    return
                end

                -- 解析响应
                local decision = cjson.decode(res.body)
                local action = decision.action
                local risk_score = decision.risk_score
                local risk_level = decision.risk_level

                ngx.log(ngx.INFO, "Sentinel-AI decision: ", action, " (", risk_score, ")", risk_level)

                -- 缓存决策（5分钟）
                ngx.shared.decision_cache:set(cache_key, action, 300)

                -- 执行决策
                if action == "BLOCK" then
                    ngx.status = 403
                    ngx.header["Content-Type"] = "application/json"
                    ngx.header["X-Sentinel-Blocked"] = "true"
                    ngx.header["X-Sentinel-Reason"] = decision.reason
                    ngx.header["X-Sentinel-Risk-Level"] = risk_level
                    ngx.header["X-Sentinel-Risk-Score"] = tostring(risk_score)
                    
                    ngx.say(cjson.encode({
                        error = "Request blocked by Sentinel-AI WAF",
                        reason = decision.reason,
                        risk_score = risk_score,
                        risk_level = risk_level,
                        request_id = decision.request_id,
                        timestamp = decision.timestamp
                    }))
                    ngx.exit(403)

                elseif action == "REVIEW" then
                    ngx.status = 202
                    ngx.header["Content-Type"] = "application/json"
                    ngx.header["X-Sentinel-Pending"] = "true"
                    
                    ngx.say(cjson.encode({
                        message = "Request pending approval",
                        request_id = decision.request_id,
                        expires_in = decision.expires_in
                    }))
                    ngx.exit(202)

                else
                    -- ALLOW: 继续到 proxy_pass
                    ngx.header["X-Sentinel-Protected"] = "true"
                    ngx.header["X-Sentinel-Risk-Level"] = risk_level
                    ngx.header["X-Sentinel-Risk-Score"] = tostring(risk_score)
                    ngx.header["X-Sentinel-Request-ID"] = decision.request_id
                end
            }

            # 代理到真实后端
            proxy_pass http://backend_real;
            
            # 传递原始 Host
            proxy_set_header Host $http_host;
            
            # 传递真实 IP
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            
            # 超时
            proxy_connect_timeout 30s;
            proxy_send_timeout 30s;
            proxy_read_timeout 30s;
        }
    }

    # Sentinel-AI 决策更新 API（用于动态更新后端）
    location /internal/sentinel/update {
        internal;
        
        content_by_lua_block {
            local http = require "resty.http"
            local cjson = require "cjson.safe"
            
            -- 只允许本地访问
            if ngx.var.remote_addr ~= "127.0.0.1" then
                ngx.status = 403
                ngx.say('{"error":"Forbidden"}')
                ngx.exit(403)
            end

            -- 解析请求体
            ngx.req.read_body()
            local body_data = ngx.var.request_body
            local update = cjson.decode(body_data)
            
            -- 更新 upstream
            -- TODO: 动态更新 backend_real 的服务器列表
            
            ngx.say('{"status":"ok"}')
        }
    }
    }

    # 健康检查
    location /health {
        access_log off;
        return 200 '{"status":"healthy"}';
    }
}

# 共享内存（用于缓存）
lua_shared_dict decision_cache 10m;  # 10MB
```

---

### 3. Sentinel-AI 业务逻辑服务 (Python + OpenAI）

**文件:** `sentinel-api/main.py`

```python
"""
Sentinel-AI 业务逻辑服务
使用 OpenAI API 进行意图分析
"""

from fastapi import FastAPI, Request, HTTPException
from pydantic import BaseModel
from openai import OpenAI
import time
from typing import Optional, List
import uvicorn
from datetime import datetime

# ==================== 配置 ====================
class Config:
    OPENAI_API_KEY: str = "sk-xxx"  # 你的 OpenAI API Key
    OPENAI_MODEL: str = "gpt-4o-mini"  # 或 gpt-4o, gpt-3.5-turbo
    OPENAI_BASE_URL: Optional[str] = None  # 可选，用于自定义端点
    ALLOWED_IPS: List[str] = ["127.0.0.1", "10.0.0.0/16"]  # 允许的 IP
    LOG_FILE: str = "logs/sentinel-api.log"

config = Config()

# ==================== OpenAI 客户端 ====================
client = OpenAI(
    api_key=config.OPENAI_API_KEY,
    base_url=config.OPENAI_BASE_URL
)

# ==================== 数据模型 ====================
class AnalyzeRequest(BaseModel):
    method: str
    uri: str
    headers: dict
    body: str
    client_ip: str
    host: str
    timestamp: int

class AnalyzeResponse(BaseModel):
    action: str  # ALLOW, REVIEW, BLOCK
    risk_score: int
    risk_level: str  # LOW, MEDIUM, HIGH, CRITICAL
    reason: str
    request_id: str
    timestamp: str
    llm_analysis: Optional[str] = None
    rule_violations: List[str] = []

class ApprovalRequest(BaseModel):
    request_id: str
    approved: bool
    approver: str
    reason: Optional[str] = None

# ==================== 风险评估引擎 ====================
class RiskEngine:
    """规则 + AI 驱动的风险评估"""
    
    def __init__(self):
        # 危险方法
        self.dangerous_methods = ["DELETE", "PUT", "PATCH", "POST"]
        
        # 危险路径
        self.dangerous_paths = ["/delete", "/remove", "/drop", "/destroy", "/clear"]
        
        # 危险关键词
        self.dangerous_keywords = ["delete", "drop", "truncate", "remove", "destroy"]
        
        # 生产环境标识
        self.prod_indicators = ["prod", "production", "live", "master"]
    
    def assess(self, req: AnalyzeRequest) -> tuple[int, str, List[str]]:
        """返回 (分数, 等级, 违规列表)"""
        score = 0
        violations = []
        
        # 1. 方法检查
        if req.method in self.dangerous_methods:
            score += 30
            violations.append(f"危险方法: {req.method}")
        
        # 2. 路径检查
        for path in self.dangerous_paths:
            if path in req.uri.lower():
                score += 40
                violations.append(f"危险路径: {path}")
        
        # 3. 敏感操作
        body_lower = req.body.lower()
        for keyword in self.dangerous_keywords:
            if keyword in body_lower:
                score += 30
                violations.append(f"检测到关键词: {keyword}")
        
        # 4. 生产环境
        for indicator in self.prod_indicators:
            if indicator in req.host.lower():
                score += 20
                violations.append(f"生产环境操作: {indicator}")
        
        # 5. 计算风险等级
        if score >= 90:
            level = "CRITICAL"
        elif score >= 70:
            level = "HIGH"
        elif score >= 30:
            level = "MEDIUM"
        else:
            level = "LOW"
        
        return score, level, violations

# ==================== OpenAI 意图分析 ====================
class OpenAIAnalyzer:
    """使用 OpenAI API 分析意图"""
    
    def __init__(self):
        self.system_prompt = """你是一个企业安全分析专家。请分析以下 HTTP 请求的风险。

任务:
1. 分析操作的意图（5字以内）
2. 评估可能的影响（10字以内）
3. 判断风险等级（LOW/MEDIUM/HIGH/CRITICAL）

返回格式（JSON）:
{
  "intent": "操作意图",
  "impact": "影响描述",
  "risk_level": "LOW/MEDIUM/HIGH/CRITICAL",
  "suggestion": "建议"
}

只返回 JSON，不要其他内容。"""
    
    async def analyze(self, req: AnalyzeRequest) -> dict:
        """调用 OpenAI API 分析"""
        prompt = f"""请分析这个请求:

方法: {req.method}
URL: {req.uri}
Host: {req.host}
客户端 IP: {req.client_ip}
请求体: {req.body[:200]}

{self.system_prompt}"""

        try:
            response = client.chat.completions.create(
                model=config.OPENAI_MODEL,
                messages=[
                    {"role": "system", "content": self.system_prompt},
                    {"role": "user", "content": prompt}
                ],
                temperature=0.1,  # 低温度，更确定
                max_tokens=100,
                timeout=5
            )
            
            import json
            result = json.loads(response.choices[0].message.content)
            
            # 映射风险等级到分数
            risk_scores = {
                "LOW": 10,
                "MEDIUM": 50,
                "HIGH": 80,
                "CRITICAL": 100
            }
            score = risk_scores.get(result.get("risk_level", "MEDIUM"), 50)
            
            return {
                "intent": result.get("intent", ""),
                "impact": result.get("impact", ""),
                "risk_level": result.get("risk_level", "MEDIUM"),
                "risk_score": score,
                "suggestion": result.get("suggestion", "")
            }
        
        except Exception as e:
            # OpenAI 调用失败，使用规则引擎
            print(f"OpenAI error: {e}")
            return {
                "intent": "AI 分析失败",
                "impact": "使用规则引擎",
                "risk_level": "MEDIUM",
                "risk_score": 50,
                "suggestion": "请检查 OpenAI API Key"
            }

# ==================== 审批管理 ====================
class ApprovalManager:
    """审批请求管理"""
    
    def __init__(self):
        self.pending_requests = {}  # request_id -> request
    
    def create_request(self, req: AnalyzeRequest, decision: AnalyzeResponse):
        """创建审批请求"""
        import uuid
        request_id = str(uuid.uuid4())
        
        self.pending_requests[request_id] = {
            "request": req,
            "decision": decision,
            "created_at": datetime.now(),
            "status": "pending"
        }
        
        # TODO: 推送到企业微信/钉钉
        # push_to_approval_system(request_id, req, decision)
        
        return request_id
    
    def approve(self, request_id: str, approved: bool, approver: str, reason: str = None):
        """处理审批结果"""
        if request_id not in self.pending_requests:
            raise HTTPException(status_code=404, detail="Request not found")
        
        request_data = self.pending_requests[request_id]
        request_data["approved"] = approved
        request_data["approver"] = approver
        request_data["reason"] = reason
        request_data["approved_at"] = datetime.now()
        request_data["status"] = "approved" if approved else "rejected"
        
        # TODO: 通知 Nginx 更新缓存
        # notify_nginx(request_id, approved)
        
        return request_data

# ==================== FastAPI 应用 ====================
app = FastAPI(title="Sentinel-AI API", version="2.0.0")
risk_engine = RiskEngine()
openai_analyzer = OpenAIAnalyzer()
approval_manager = ApprovalManager()

@app.post("/analyze", response_model=AnalyzeResponse)
async def analyze_request(req: AnalyzeRequest):
    """分析请求并返回决策"""
    
    # 1. 规则引擎评估
    rule_score, rule_level, violations = risk_engine.assess(req)
    
    # 2. OpenAI 意图分析
    ai_analysis = await openai_analyzer.analyze(req)
    ai_score = ai_analysis["risk_score"]
    ai_level = ai_analysis["risk_level"]
    
    # 3. 综合评分
    # 规则 60% + AI 40%
    final_score = int(rule_score * 0.6 + ai_score * 0.4)
    
    # 4. 确定最终决策
    if final_score >= 90:
        action = "BLOCK"
        final_level = "CRITICAL"
    elif final_score >= 70:
        action = "REVIEW"
        final_level = "HIGH"
    else:
        action = "ALLOW"
        final_level = ai_level
    
    # 5. 生成响应
    import uuid
    response = AnalyzeResponse(
        action=action,
        risk_score=final_score,
        risk_level=final_level,
        reason=f"规则分数:{rule_score}, AI分数:{ai_score}, 违规:{violations}",
        request_id=str(uuid.uuid4()),
        timestamp=datetime.now().isoformat(),
        llm_analysis=f"{ai_analysis['intent']} | {ai_analysis['impact']}",
        rule_violations=violations
    )
    
    # 6. 如果需要审批，创建审批请求
    if action == "REVIEW":
        approval_manager.create_request(req, response)
    
    # 7. 记录日志
    log_entry = {
        "timestamp": response.timestamp,
        "request": req.dict(),
        "analysis": {
            "rule_score": rule_score,
            "ai_score": ai_score,
            "final_score": final_score
        },
        "decision": response.dict()
    }
    # TODO: 写入日志/数据库
    
    return response

@app.post("/approval")
async def handle_approval(approval: ApprovalRequest):
    """处理审批结果"""
    return approval_manager.approve(
        approval.request_id,
        approval.approved,
        approval.approver,
        approval.reason
    )

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "sentinel-ai-api"}

# ==================== 主程序 ====================
if __name__ == "__main__":
    import os
    os.makedirs("logs", exist_ok=True)
    
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8000,
        log_level="info"
    )
```

---

### 4. Docker Compose 部署

**文件:** `docker-compose-opensource.yml`

```yaml
version: '3.8'

services:
  # CoreDNS
  coredns:
    image: coredns/coredns:1.11.1
    container_name: coredns
    ports:
      - "53:53/udp"
      - "53:53/tcp"
    volumes:
      - ./coredns:/etc/coredns
    networks:
      - sentinel-net
    command: -conf /etc/coredns/Corefile

  # Nginx/OpenResty
  nginx:
    image: openresty/openresty:alpine
    container_name: nginx-waf
    ports:
      - "8080:8080"
    volumes:
      - ./nginx:/etc/nginx
    depends_on:
      - coredns
      - sentinel-api
    networks:
      - sentinel-net
    restart: unless-stopped

  # Sentinel-AI API
  sentinel-api:
    build: ./sentinel-api
    container_name: sentinel-api
    ports:
      - "8000:8000"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_MODEL=gpt-4o-mini
    networks:
      - sentinel-net
    restart: unless-stopped

  # etcd (可选，用于动态域名管理）
  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: etcd
    ports:
      - "2379:2379"
      - "2380:2380"
    environment:
      - ETCD_AUTO_COMPACTION_MODE=revision
      - ETCD_QUOTA_BACKEND_BYTES=4294967296
    networks:
      - sentinel-net

networks:
  sentinel-net:
    driver: bridge
```

---

## 🚀 快速部署

### 步骤 1: 准备目录

```bash
cd E:\workspace\sentinel-ai
mkdir -p coredns nginx sentinel-api logs
```

### 步骤 2: 创建配置文件

```bash
# CoreDNS 配置已在上面
cp coredns/Corefile.example coredns/Corefile

# Nginx 配置已在上面
cp nginx/nginx.conf.example nginx/nginx.conf

# Sentinel-AI API
cp sentinel-api/main.py.example sentinel-api/main.py
```

### 步骤 3: 配置环境变量

```bash
# Windows (PowerShell）
$env:OPENAI_API_KEY="sk-xxx"

# Linux/Mac
export OPENAI_API_KEY="sk-xxx"

# 或在 .env 文件中
echo "OPENAI_API_KEY=sk-xxx" > .env
```

### 步骤 4: 启动服务

```bash
# 使用 Docker Compose
docker-compose -f docker-compose-opensource.yml up -d

# 或逐个启动
docker run -d --name coredns -p 53:53/udp -v $(pwd)/coredns:/etc/coredns coredns/coredns:1.11.1
docker run -d --name nginx-waf -p 8080:8080 -v $(pwd)/nginx:/etc/nginx openresty/openresty:alpine
docker run -d --name sentinel-api -p 8000:8000 --env OPENAI_API_KEY=$OPENAI_API_KEY python:3.12-slim
```

### 步骤 5: 配置 Agent DNS

```bash
# 在 Agent 容器中
echo "nameserver 10.0.0.1" > /etc/resolv.conf

# 或使用 K8s DNS ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: dns-config
data:
  resolv.conf: |
    nameserver 10.0.0.1
```

### 步骤 6: 测试

```bash
# 测试 DNS 解析
nslookup api.example.com 10.0.0.1

# 测试 WAF
curl http://10.0.0.1:8080/api/test

# 测试完整链路（在 Agent 中）
curl http://api.example.com/api/users
```

---

## 📊 对比总结

| 组件 | 方案 | 优势 |
|------|------|------|
| **DNS** | CoreDNS | CNCF 毕业、稳定、插件丰富 |
| **代理** | Nginx/OpenResty | 高性能、社区支持、Lua 脚本 |
| **AI** | OpenAI API | 强大模型、免维护、快速迭代 |

---

## 🎯 下一步

- [ ] 创建 K8s 部署配置
- [ ] 集成企业微信/钉钉审批
- [ ] 添加监控和告警
- [ ] 性能测试和优化

---

**版本:** 2.0 (基于开源工具）  
**更新时间:** 2026-02-05
