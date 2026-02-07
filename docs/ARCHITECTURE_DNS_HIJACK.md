# Sentinel-AI DNS 劫持架构设计

## 一、核心原理

### 1.1 问题场景

```
正常情况:
┌──────────┐     DNS      ┌──────────┐
│  Agent   │ ────────► │ DNS Server│
└─────┬────┘             └────┬─────┘
      │                       │
      │ HTTP                  │ IP: 1.2.3.4
      ▼                       ▼
┌─────────────────────────────────┐
│  外部服务 (api.example.com)      │
│  真实 IP: 1.2.3.4                │
└─────────────────────────────────┘

问题: Agent 直接访问外部服务，绕过治理
```

### 1.2 DNS 劫持方案

```
DNS 劫持后:
┌──────────┐     DNS      ┌──────────────┐
│  Agent   │ ────────► │   DNS        │
│          │             │   劫持器      │
└─────┬────┘             └──────┬───────┘
      │                          │
      │ HTTP                    │ IP: 10.0.0.1 (假的)
      │                         │ (指向 Sentinel-AI)
      ▼                         │
┌──────────────┐                │
│ Sentinel-AI  │ ◄───────────────┘
│   WAF 网关    │
└──────┬───────┘
       │
       │ 代理转发 (真实 IP)
       ▼
┌─────────────────────────────────┐
│  外部服务 (api.example.com)      │
│  真实 IP: 1.2.3.4                │
└─────────────────────────────────┘

结果: Agent 以为直接访问，实际所有流量都经过 Sentinel-AI
```

---

## 二、完整架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Agent 容器 / 虚拟机                             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                   Agent 应用                             │  │
│  │                                                          │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │ LangChain    │  │ AutoGPT      │  │ OpenClaw     │  │  │
│  │  │              │  │              │  │              │  │  │
│  │  │ requests.get(│  │ curl http:// │  │ http.post(   │  │  │
│  │  │ 'api.exa..')│  │ api.exa..')  │  │ api.exa..')  │  │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │  │
│  └─────────┼──────────────────┼──────────────────┼──────────┘  │
│            │                  │                  │               │
│            └──────────────────┼──────────────────┘               │
│                               │                                  │
│  ┌────────────────────────────▼────────────────────────────┐   │
│  │                     网络层                               │   │
│  │                                                           │   │
│  │  1. DNS 查询: api.example.com                           │   │
│  │  2. DNS 劫持器拦截                                       │   │
│  │  3. 返回假 IP: 10.0.0.1 (Sentinel-AI 网关)              │   │
│  │  4. Agent 以为连接到真实服务                             │   │
│  │  5. 实际连接到 Sentinel-AI WAF                            │   │
│  └────────────────────────────┬────────────────────────────┘   │
└───────────────────────────────┼──────────────────────────────────┘
                                │
                                │ HTTP/gRPC/TCP
                                │ IP: 10.0.0.1
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Sentinel-AI WAF 网关                             │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   1. 连接接管                               │  │
│  │                                                           │  │
│  │  - 接收来自 Agent 的连接                                  │  │
│  │  - 解析 Host 头获取真实域名                               │  │
│  │  - 从 DNS 映射表获取真实 IP                               │  │
│  │  - 建立到真实后端的连接                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│  ┌────────────────────────────▼────────────────────────────┐  │
│  │                   2. 请求解析                             │  │
│  │                                                           │  │
│  │  - 提取 HTTP 方法 (GET/POST/DELETE...)                   │  │
│  │  - 提取 URL 路径                                           │  │
│  │  - 提取请求体 / Headers / Cookies                         │  │
│  │  - 提取 Client ID / Session ID                             │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│  ┌────────────────────────────▼────────────────────────────┐  │
│  │                   3. 风险评估                             │  │
│  │                                                           │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │  │
│  │  │ 规则引擎    │  │ LLM 分析   │  │ 威胁情报   │        │  │
│  │  │            │  │            │  │            │        │  │
│  │  │ - 方法检查  │  │ - 意图识别  │  │ - IP 黑名单 │        │  │
│  │  │ - 路径检查  │  │ - 语义理解  │  │ - 域名检测  │        │  │
│  │  │ - 参数检查  │  │ - 风险评估  │  │ - C2 检测   │        │  │
│  │  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘        │  │
│  │        │                │                │                │  │
│  │        └────────────────┼────────────────┘                │  │
│  │                         │                                 │  │
│  │  ┌──────────────────────▼────────────────────────────┐  │  │
│  │  │              风险评分计算                          │  │  │
│  │  │                                                   │  │  │
│  │  │  - 规则引擎分数: 0-100                             │  │  │
│  │  │  - LLM 分析分数: 0-100                             │  │  │
│  │  │  - 威胁情报分数: 0-100                             │  │  │
│  │  │  - 历史行为分数: 0-100                             │  │  │
│  │  │                                                   │  │  │
│  │  │  总分 = 加权平均                                    │  │  │
│  │  │  - 低风险: < 30 → 自动放行                          │  │  │
│  │  │  - 中风险: 30-70 → 记录日志                         │  │  │
│  │  │  - 高风险: > 70 → 需要审批                          │  │  │
│  │  │  - 严重: > 90 → 立即阻止                            │  │  │
│  │  └───────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│  ┌────────────────────────────▼────────────────────────────┐  │
│  │                   4. 决策执行                             │  │
│  │                                                           │  │
│  │  ┌────────────────────────────────────────────────┐      │  │
│  │  │            低风险 (< 30)                        │      │  │
│  │  │  ┌────────────────────────────────────────┐   │      │  │
│  │  │  │  立即放行                                 │   │      │  │
│  │  │  │  - 代理转发到真实后端                    │   │      │  │
│  │  │  │  - 记录审计日志                          │   │      │  │
│  │  │  │  - 延迟 < 5ms                             │   │      │  │
│  │  │  └────────────────────────────────────────┘   │      │  │
│  │  └────────────────────────────────────────────────┘      │  │
│  │                                                           │  │
│  │  ┌────────────────────────────────────────────────┐      │  │
│  │  │            中风险 (30-70)                        │      │  │
│  │  │  ┌────────────────────────────────────────┐   │      │  │
│  │  │  │  放行 + 警告                              │   │      │  │
│  │  │  │  - 代理转发到真实后端                    │   │      │  │
│  │  │  │  - 添加 X-Sentinel-Warning 响应头        │   │      │  │
│  │  │  │  - 记录详细审计日志                       │   │      │  │
│  │  │  │  - 异步通知安全团队                       │   │      │  │
│  │  │  └────────────────────────────────────────┘   │      │  │
│  │  └────────────────────────────────────────────────┘      │  │
│  │                                                           │  │
│  │  ┌────────────────────────────────────────────────┐      │  │
│  │  │            高风险 (70-90)                        │      │  │
│  │  │  ┌────────────────────────────────────────┐   │      │  │
│  │  │  │  需要人工审批                            │   │      │  │
│  │  │  │  - 暂停请求                             │   │      │  │
│  │  │  │  - 推送到审批系统                       │   │      │  │
│  │  │  │  - 等待审批结果                         │   │      │  │
│  │  │  │  - 超时自动拒绝                         │   │      │  │
│  │  │  └────────────────────────────────────────┘   │      │  │
│  │  └────────────────────────────────────────────────┘      │  │
│  │                                                           │  │
│  │  ┌────────────────────────────────────────────────┐      │  │
│  │  │            严重 (> 90)                          │      │  │
│  │  │  ┌────────────────────────────────────────┐   │      │  │
│  │  │  │  立即阻止                                │   │      │  │
│  │  │  │  - 返回 403 Forbidden                 │   │      │  │
│  │  │  │  - 记录安全事件                         │   │      │  │
│  │  │  │  - 实时告警                             │   │      │  │
│  │  │  │  - 封禁 IP (可选)                        │   │      │  │
│  │  │  └────────────────────────────────────────┘   │      │  │
│  │  └────────────────────────────────────────────────┘      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│  ┌────────────────────────────▼────────────────────────────┐  │
│  │                   5. 响应处理                             │  │
│  │                                                           │  │
│  │  - 透明转发后端响应                                       │  │
│  │  - 注入安全头 (X-Sentinel-Protected)                      │  │
│  │  - 修改响应内容 (可选，如敏感信息脱敏)                    │  │
│  │  - 记录响应时间 / 状态码 / 大小                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│                              │ 响应回传                         │
│                              ▼                                 │
└─────────────────────────────────────────────────────────────────┘
                          返回给 Agent
```

---

## 三、DNS 劫持实现方式

### 3.1 方式 1: 修改 /etc/hosts

```bash
# 在 Agent 容器内执行
cat >> /etc/hosts << EOF
# Sentinel-AI DNS 劫持
10.0.0.1  api.example.com
10.0.0.1  db.example.com
10.0.0.1  auth.example.com
10.0.0.1  *.example.com
EOF
```

**优点:**
- 简单，立即生效
- 无需额外服务

**缺点:**
- 需要手动维护域名列表
- 无法处理动态域名

---

### 3.2 方式 2: 内置 DNS 服务器

```go
// DNS 劫持器实现
package dnshijack

import (
    "github.com/miekg/dns"
    "net"
    "strings"
)

type DNSHijacker struct {
    gatewayIP  string  // Sentinel-AI 网关 IP
    domainMap map[string]string  // 域名映射
}

func NewDNSHijacker(gatewayIP string) *DNSHijacker {
    return &DNSHijacker{
        gatewayIP: gatewayIP,
        domainMap: make(map[string]string),
    }
}

// 添加需要劫持的域名
func (h *DNSHijacker) AddDomain(domain string) {
    h.domainMap[domain] = h.gatewayIP
}

// 添加需要劫持的域名模式
func (h *DNSHijacker) AddDomainPattern(pattern string) {
    h.domainMap[pattern] = h.gatewayIP
}

// 处理 DNS 查询
func (h *DNSHijacker) HandleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
    m := new(dns.Msg)
    m.SetReply(r)
    m.Compress = false
    
    for _, q := range r.Question {
        // 检查是否需要劫持
        if h.shouldHijack(q.Name) {
            // 返回假 IP (Sentinel-AI 网关)
            rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, h.gatewayIP))
            if err == nil {
                m.Answer = append(m.Answer, rr)
            }
        } else {
            // 转发到真实 DNS
            h.forwardDNS(w, q)
            return
        }
    }
    
    w.WriteMsg(m)
}

// 判断是否需要劫持
func (h *DNSHijacker) shouldHijack(domain string) bool {
    domain = strings.TrimSuffix(domain, ".")
    
    // 精确匹配
    if _, ok := h.domainMap[domain]; ok {
        return true
    }
    
    // 模式匹配
    for pattern := range h.domainMap {
        if strings.HasSuffix(domain, strings.TrimPrefix(pattern, "*.")) {
            return true
        }
    }
    
    return false
}

// 转发到真实 DNS
func (h *DNSHijacker) forwardDNS(w dns.ResponseWriter, q dns.Question) {
    c := new(dns.Client)
    m := new(dns.Msg)
    m.SetQuestion(q.Name, q.Qtype)
    
    // 转发到上游 DNS (如 8.8.8.8)
    r, _, err := c.Exchange(m, "8.8.8.8:53")
    if err == nil {
        w.WriteMsg(r)
    }
}

// 启动 DNS 劫持器
func (h *DNSHijacker) Start(addr string) error {
    server := &dns.Server{
        Addr: addr,
        Net:  "udp",
    }
    
    server.Handler = h
    return server.ListenAndServe()
}
```

**使用:**
```go
hijacker := NewDNSHijacker("10.0.0.1")

// 劫持特定域名
hijacker.AddDomain("api.example.com")
hijacker.AddDomain("db.example.com")

// 劫持所有子域名
hijacker.AddDomainPattern("*.example.com")

// 启动 DNS 服务器 (监听 53 端口)
hijacker.Start(":53")
```

---

### 3.3 方式 3: 修改容器 DNS 配置

```yaml
# Kubernetes 配置
apiVersion: v1
kind: Pod
metadata:
  name: agent-pod
spec:
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 10.0.0.1  # Sentinel-Ai DNS 劫持器
    searches:
      - sentinel.local
    options:
      - name: ndots
        value: "2"
  containers:
  - name: agent
    image: agent-image
```

---

### 3.4 方式 4: eBPF DNS 拦截

```python
# eBPF DNS 拦截
from bcc import BPF

BPF_PROGRAM = """
#include <uapi/linux/ptrace.h>

struct dns_event_t {
    u32 pid;
    char query[256];
    u32 query_len;
};

BPF_PERF_OUTPUT(dns_events);

SEC("kprobe/udp_sendmsg")
int kprobe_udp_sendmsg(struct pt_regs *ctx, struct sock *sk, struct msghdr *msg)
{
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    // 检查是否是 DNS 查询 (端口 53)
    u16 dport = 0;
    bpf_probe_read_kernel(&dport, sizeof(dport),
                           &sk->__sk_common.skc_dport);
    
    if (bpf_ntohs(dport) == 53) {
        struct dns_event_t e = {};
        e.pid = pid;
        
        // 读取 DNS 查询
        bpf_probe_read_user(&e.query, sizeof(e.query), msg->msg_iov->iov_base);
        e.query_len = msg->msg_iov->iov_len;
        
        dns_events.perf_submit(ctx, &e, sizeof(e));
    }
    
    return 0;
}
"""

bpf = BPF(text=BPF_PROGRAM)
```

---

## 四、WAF 网关实现

```go
// WAF 网关核心
package waf

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
)

type WAFGateway struct {
    dnsMapping  map[string]string  // 域名 -> 真实 IP
    proxy       *httputil.ReverseProxy
    policyEngine *PolicyEngine
    llmAnalyzer  *LLMAnalyzer
}

func NewWAFGateway() *WAFGateway {
    return &WAFGateway{
        dnsMapping: make(map[string]string),
        policyEngine: NewPolicyEngine(),
        llmAnalyzer: NewLLMAnalyzer(),
    }
}

// 添加 DNS 映射
func (w *WAFGateway) AddDNSMapping(domain, realIP string) {
    w.dnsMapping[domain] = realIP
}

// 处理请求
func (w *WAFGateway) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
    // 1. 解析真实域名 (从 Host 头)
    host := req.Host
    domain := strings.Split(host, ":")[0]
    
    // 2. 从 DNS 映射表获取真实 IP
    realIP, ok := w.dnsMapping[domain]
    if !ok {
        // 如果不在映射表，可能是直接 IP 访问
        realIP = domain
    }
    
    // 3. 解析请求
    request := ParseRequest(req)
    
    // 4. 风险评估
    riskScore := w.assessRisk(request)
    
    // 5. 决策
    decision := w.makeDecision(riskScore)
    
    // 6. 执行决策
    if decision.Action == "ALLOW" {
        w.proxyRequest(resp, req, realIP)
    } else if decision.Action == "BLOCK" {
        w.blockRequest(resp, decision)
    } else {
        w.requestApproval(resp, req, decision)
    }
}

// 代理请求到真实后端
func (w *WAFGateway) proxyRequest(resp http.ResponseWriter, req *http.Request, realIP string) {
    target, _ := url.Parse("http://" + realIP)
    
    proxy := httputil.NewSingleHostReverseProxy(target)
    proxy.Transport = &http.Transport{
        // 自定义 Transport，记录请求/响应
    }
    
    proxy.ServeHTTP(resp, req)
}

// 阻止请求
func (w *WAFGateway) blockRequest(resp http.ResponseWriter, decision Decision) {
    resp.WriteHeader(http.StatusForbidden)
    
    response := map[string]interface{}{
        "error": "Request blocked by Sentinel-AI WAF",
        "reason": decision.Reason,
        "risk_score": decision.Score,
    }
    
    resp.Header().Set("Content-Type", "application/json")
    json.NewEncoder(resp).Encode(response)
}
```

---

## 五、部署架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes 集群                                │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐   │
│  │                  Agent Namespace                         │   │
│  │                                                         │   │
│  │  ┌────────────────────────────────────────────────┐   │   │
│  │  │              Agent Pod                          │   │   │
│  │  │                                                 │   │   │
│  │  │  ┌──────────────┐                              │   │   │
│  │  │  │   DNS Hijack  │ ◄─────┐                     │   │   │
│  │  │  │   (Sidecar)   │       │                     │   │   │
│  │  │  │   监听 :53    │       │                     │   │   │
│  │  │  └──────┬───────┘       │                     │   │   │
│  │  │         │               │                     │   │   │
│  │  │  ┌──────▼───────┐       │                     │   │   │
│  │  │  │    Agent     │       │                     │   │   │
│  │  │  │    容器      │       │                     │   │   │
│  │  │  └──────────────┘       │                     │   │   │
│  │  └────────────────────────┼─────────────────────┘   │   │
│  │                           │                         │   │
│  └───────────────────────────┼─────────────────────────┘   │
│                              │ 修改 DNS Config              │
│                              │                             │
│  ┌───────────────────────────▼─────────────────────────┐   │
│  │            Sentinel-AI Namespace                    │   │
│  │                                                         │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │         Sentinel-AI WAF Gateway (10.0.0.1)     │  │   │
│  │  │                                                 │  │   │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │   │
│  │  │  │ 接收层    │  │ 分析层    │  │ 决策层    │      │  │   │
│  │  │  │          │  │          │  │          │      │  │   │
│  │  │  │ HTTP     │  │ 规则引擎  │  │ 审批流   │      │  │   │
│  │  │  │ gRPC     │  │ LLM      │  │ 告警     │      │  │   │
│  │  │  │ TCP      │  │ 威胁情报  │  │          │      │  │   │
│  │  │  └────┬─────┘  └────┬─────┘  └────┬─────┘      │  │   │
│  │  │       │            │            │              │  │   │
│  │  │  ┌────▼────────────▼────────────▼────┐        │  │   │
│  │  │  │      审计日志 + 证据链           │        │  │   │
│  │  │  └───────────────────────────────────┘        │  │   │
│  │  │                                            │  │   │
│  │  │  ┌──────────────────────────────────┐    │  │   │
│  │  │  │      代理转发到真实后端         │    │  │   │
│  │  │  └──────────────────────────────────┘    │  │   │
│  │  └─────────────────────────────────────────┘  │   │
│  │                                                 │   │
│  │  ┌─────────────────────────────────────────┐  │   │
│  │  │        PostgreSQL (审计日志)            │  │   │
│  │  └─────────────────────────────────────────┘  │   │
│  │                                                 │   │
│  └─────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────┘
                              │
                              │ 外部网络
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        外部服务                                   │
│  api.example.com (1.2.3.4)                                      │
│  db.example.com (5.6.7.8)                                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 六、优势

| 优势 | 说明 |
|------|------|
| **完全透明** | Agent 无需任何修改，无感知 |
| **无法绕过** | DNS 解析被劫持，Agent 不知道真实 IP |
| **全协议支持** | HTTP/HTTPS/gRPC/TCP 都可拦截 |
| **易于部署** | Sidecar 模式，不侵入原有架构 |
| **灵活策略** | 可以针对不同域名设置不同策略 |

---

## 七、安全加固

```
┌─────────────────────────────────────────────────────────────────┐
│                    防绕过机制                                      │
│                                                                  │
│  1. IP 访问检测                                                 │
│     └─ 如果 Agent 直接用 IP 访问，立即告警                     │
│                                                                  │
│  2. DNS 劫持检测                                                │
│     └─ 检测 Agent 是否尝试使用其他 DNS 服务器                  │
│                                                                  │
│  3. eBPF 网络监控                                               │
│     └─ 监控所有系统调用，防止绕过 DNS 层                       │
│                                                                  │
│  4. 流量指纹                                                   │
│     └─ 如果流量模式异常，标记为可疑                           │
│                                                                  │
│  5. 多层验证                                                   │
│     └─ DNS 层 + 应用层 + 内核层，三层防御                     │
└─────────────────────────────────────────────────────────────────┘
```

---

这就是 **DNS 劫持 + WAF 网关** 的完整架构！
