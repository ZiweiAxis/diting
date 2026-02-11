# 3AF 沙箱 Node Agent 接入设计（基础设施视角）

本文描述 **Docker 环境与本地直接执行** 下如何接入 3AF 策略，实现「不登录机器，通过 3AF 统一授权」的目标。聚焦：Node Agent 职责、与 3AF 的协议、3AF 侧扩展点、部署形态。

**范围说明**：企业常见场景以 **Docker 容器内执行** 与 **宿主机/本机直接执行命令** 为主；云端沙箱（如 E2B）不在本设计核心范围内，协议兼容时可顺带支持。

---

## 1. 目标与约束

- **目标**：授权决策在 3AF 控制平面完成；执行节点（Docker 或本机）只负责执行决策，不在机器上做人工授权。
- **约束**：复用现有 3AF 的 Policy Engine、CHEQ、Audit，不重复造轮子；Node 侧尽量轻量、可插拔。

---

## 2. 核心场景：Docker 与本地直接执行

| 场景 | 说明 | 3AF 对接要点 |
|------|------|----------------|
| **Docker 环境** | Agent 或用户在**容器内**执行命令（如 `docker run` / `docker exec` 触发的进程）。 | Node Agent 部署在容器内（entrypoint/sidecar）或宿主机上拦截容器 exec；Resource 建议用 `docker://<container_id>` 或 `docker://<project>/<service>`；同机时可用 UDS + PeerCred（宿主机 3AF），跨机用 gRPC + SessionToken。 |
| **本地直接执行** | 命令在**宿主机/本机**直接跑（非容器），如 CI 机、开发机、Agent 宿主。 | Node Agent 以 Wrapper（如 `3af-exec`）、进程内 Hook 或 Sidecar 形式拦截 exec；Resource 建议用 `local://<host_id>` 或 `local://<project>`；同机场景最典型，UDS + SO_PEERCRED 无密钥识别进程身份。 |

两者共用同一套 3AF 策略模型（Subject-Action-Resource-Context）与 Exec 鉴权 API/gRPC；仅 Resource 命名与部署形态不同。

---

## 3. 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│  3AF Control Plane (现有 proxy + policy + cheq + audit)          │
│  ┌─────────────┐  ┌──────────────┐  ┌───────┐  ┌──────────────┐  │
│  │ Policy      │  │ CHEQ        │  │ Audit │  │ 新增:        │  │
│  │ Engine      │  │ (审批)      │  │       │  │ /auth/exec   │  │
│  └──────┬──────┘  └──────┬──────┘  └───┬───┘  │ (能力鉴权)   │  │
│         │                │             │      └──────┬───────┘  │
└─────────┼────────────────┼─────────────┼─────────────┼──────────┘
          │                │             │             │
          │  UDS / gRPC    │             │             │
          ▼                ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────────┐
│  执行环境 (数据平面)                                              │
│  ┌─ Docker 容器内 ──────────────────────────────────────────────┐ │
│  │ 3AF Node Agent (entrypoint/sidecar) → 拦截 exec → 问 3AF   │ │
│  └─────────────────────────────────────────────────────────────┘ │
│  ┌─ 本地直接执行 (宿主机/本机) ─────────────────────────────────┐ │
│  │ 3AF Node Agent (wrapper / 进程内 Hook) → 拦截 exec → 问 3AF│ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Node Agent 职责（Docker / 本机侧）

### 4.1 部署形态（三选一或组合）

| 形态 | 说明 | 适用 |
|------|------|------|
| **进程内 Hook** | 在跑 Agent 的进程里，在执行命令前调用 3AF 客户端 | 自研 Agent 运行时、可改代码 |
| **Sidecar** | 沙箱内独立进程，通过本地 socket/HTTP 与「命令执行器」通信，执行器先问 sidecar 再 exec | 不改 Agent 代码，改执行入口 |
| **Wrapper / Exec 替代** | 用 `3af-exec` 替代系统 `exec`，内部先问 3AF 再调真实 exec | 能控制 exec 入口的环境（如容器 entrypoint） |

### 4.2 必须做的事

1. **构造能力请求**  
   将「当前要执行的操作」抽象成 3AF 能理解的 **Subject-Action-Resource-Context**（与现有策略模型一致）：
   - **Subject**：Agent 身份（与 L0 一致，如 API Key / workspace_id）。
   - **Action**：能力 ID，如 `exec:net.outbound`、`exec:fs.write`、`exec:tool.docker`。
   - **Resource**：作用范围，如 `docker://<container_id>`、`local://<host_id>` 或 `project-a/dev`。
   - **Context**：可选，如 `command_preview`、`env`。

2. **调用 3AF 鉴权 API**  
   - 唯一入口：`POST /auth/exec`（见下节）。  
   - 请求体：JSON，含上述字段 + `trace_id`（可选，便于与审计关联）。  
   - 响应：`200` + 决策 + 若为 review 则含 `cheq_id` 或需轮询的标识。

3. **根据响应执行策略**  
   - **allow**：放行本次执行。  
   - **deny**：拒绝执行，返回可读错误给调用方，并记录到本地日志（可选再上报 3AF）。  
   - **review**：  
     - 若 3AF 返回「同步等待审批」：Node Agent 轮询 3AF 的 CHEQ 状态接口，直到 approved/rejected/expired，再放行或拒绝。  
     - 若 3AF 返回「异步审批」：Node Agent 先返回「等待审批」给调用方，审批结果通过 Webhook / 回调 / 轮询再触发放行（可 Phase 2）。

4. **身份与网络**  
   - Node Agent 调用 3AF 时需携带 **L0 身份**（与 HTTP 代理一致，如 `Authorization: Bearer <api_key>`），以便 3AF 做 L0 校验和审计关联。  
   - 3AF 的地址可配置（环境变量或本地配置文件），例如 `DITING_3AF_URL=https://3af.company.com`。

---

## 5. 3AF 侧：新增 `/auth/exec` 能力鉴权 API

### 5.1 复用现有组件

- **Policy Engine**：已有 `Evaluate(ctx, *RequestContext) -> Decision`。  
  - 扩展：**RequestContext** 增加「请求类型」或使用统一的 Resource/Action 命名空间，使「HTTP 请求」与「执行能力请求」共用同一套规则。  
  - 推荐：在 3AF 内把「exec 能力请求」映射为统一的 `RequestContext`（见下）。

- **CHEQ**：review 时创建 ConfirmationObject，与现有 HTTP review 流程一致；Node Agent 通过 3AF 提供的「按 cheq_id 查询状态」接口轮询。

- **Audit**：所有 `/auth/exec` 的请求与决策写入同一审计存储，便于追溯「谁在何时在哪个沙箱执行了何种能力」。

### 5.1.1 飞书审批与验证逻辑：与现有一致（必须）

执行层请求进入 **review** 时，必须与现有 HTTP 代理 review 走**同一套** CHEQ + 飞书审批与验证逻辑，不做两套实现、不新增单独验证路径。

| 环节 | 现有一致性要求 | 说明 |
|------|----------------|------|
| **CHEQ 创建** | 同一 `cheq.CreateInput` / `ConfirmationObject` 结构 | TraceID、Resource、Action、Summary（执行层用 command_line 等作摘要）、ExpiresAt、Type（可用 `operation_approval` 或 `exec_approval` 区分展示）；同一 `cheq.Create()` 入口。 |
| **投递** | 同一 `DeliveryProvider.Deliver()` | 飞书卡片/消息、确认人解析（OwnershipResolver）与 HTTP 代理共用；同一配置、同一 Feishu 应用。 |
| **审批入口** | 同一 `/cheq/approve` 与飞书卡片回调 | 确认人通过现有链接（如 `/cheq/approve?id=xxx&approved=true`）或飞书卡片按钮提交；同一 `cheq.Submit()`，同一幂等与状态机。 |
| **状态查询** | 同一 `cheq.GetByID()` | Node Agent 轮询或 3AF 内部等待时，使用与 HTTP 代理相同的查询接口与终态判断（approved/rejected/expired）。 |
| **审计** | 同一 `appendEvidenceWithCHEQ` 与 Evidence 字段 | trace_id、decision、policy_rule_id、decision_reason、CHEQStatus、Confirmer 等与现有一致；仅请求来源为 exec 可通过 action 前缀或请求类型区分。 |
| **E2E 验证** | 与现有飞书审批 E2E 同流程 | 触发 review → 飞书收到待办/卡片 → 同一方式点击批准/拒绝 → 同一方式查审计按 trace_id 校验 decision/confirmer；执行层仅多「请求来自 ExecAuth」的用例，验证逻辑与断言方式不变。 |

这样，执行层与 HTTP 代理在「谁创建 CHEQ、谁投递、谁处理审批、谁写审计、如何 E2E 验证」上完全一致，仅**请求入口**（ExecAuth vs 代理）和 **Summary/Type** 等展示信息不同。

### 5.2 请求/响应契约（建议）

**请求** `POST /auth/exec`

```json
{
  "subject": "agent-api-key-xxx",
  "action": "exec:net.outbound",
  "resource": "docker://abc123 或 local://host-01",
  "context": {
    "command_preview": "curl -s https://api.example.com",
    "workspace_id": "ws-001",
    "env": "dev"
  },
  "trace_id": "optional-trace-id"
}
```

- 请求头：`Authorization: Bearer <api_key>`（L0）；可选 `X-Trace-ID`。
- **action** 建议命名空间：`exec:<capability>`，例如 `exec:net.outbound`、`exec:fs.write`、`exec:process.spawn`。

**响应**

- **200 OK**（允许）
  ```json
  { "decision": "allow", "policy_rule_id": "rule-1", "reason": "" }
  ```
- **200 OK**（需审批，同步等待）
  ```json
  { "decision": "review", "cheq_id": "cheq-xxx", "policy_rule_id": "rule-2", "reason": "requires approval" }
  ```
  Node Agent 轮询 `GET /cheq/status?id=cheq-xxx`（或现有 CHEQ 查询接口），超时后按 expired 处理。
- **403 Forbidden**（拒绝）
  ```json
  { "decision": "deny", "policy_rule_id": "rule-3", "reason": "exec not allowed for this resource" }
  ```

### 5.3 RequestContext 扩展（最小侵入）

现有 `RequestContext` 为 HTTP 设计（Method, TargetURL, Resource, Action）。两种方式二选一或短期并存：

- **方案 A**：增加字段 `RequestType string`（如 `"http"` | `"exec"`），当 `RequestType == "exec"` 时，用 `context` 里的 `action`/`resource` 直接填 `Action`/`Resource`，`TargetURL` 可填 `exec://<resource>` 或空。Policy 规则按 `Resource`/`Action` 匹配，已支持。
- **方案 B**：不改 `RequestContext`，新增类型 `ExecRequestContext`，实现同一 Policy 接口的「适配器」：将 ExecRequestContext 转为 Policy 能接受的 Evaluate 入参（例如仍输出 Subject/Action/Resource 给引擎）。

推荐 **方案 A**：在 `models.RequestContext` 增加 `RequestType` 与可选 `ExecContext`，在 proxy 或新 handler 里从 `POST /auth/exec` 的 body 构造 `RequestContext`，然后走同一套 `policy.Evaluate` + 审计；review 时与现有 CHEQ 流程一致，仅入口从「代理请求」改为「exec 请求」。

---

## 6. 策略模型（L2）如何覆盖 exec

- 现有规则已是 **Resource + Action** 维度，只需在规则集中为「执行能力」定义 Resource/Action。
- 例如：
  - `resource = "docker://*"` 或 `local://*`，`action = "exec:net.outbound"` → deny。
  - `resource = "docker://project-a/*"`，`action = "exec:fs.write"` → review。
  - `resource = "local://project-a/dev"`，`action = "exec:fs.read"` → allow。
- 这样 **不需要为 exec 单独做一套 PDP**，只是扩展了 Action 的取值空间和请求入口。

---

## 7. Node Agent 与 3AF 的交互时序（概要）

1. 工作负载请求在沙箱内执行命令（例如 `curl ...`）。
2. Node Agent 拦截，解析出能力（如 `exec:net.outbound`），构造 `POST /auth/exec` 请求，带 L0 身份。
3. 3AF 做 L0 校验 → Policy.Evaluate(exec RequestContext) → 得到 allow/deny/review。
4. **allow**：3AF 返回 200 allow → Node Agent 放行执行 → 可选：执行结果再上报 3AF 审计（Phase 2）。
5. **deny**：3AF 返回 403 → Node Agent 拒绝执行，返回错误。
6. **review**：3AF 创建 CHEQ，返回 200 review + cheq_id → Node Agent 轮询 CHEQ 状态 → 批准则放行，拒绝/超时则拒绝。

所有请求在 3AF 侧写审计；无需登录机器即可在 3AF 上查看/配置策略和审批。

---

## 8. 配置与部署要点

- **3AF**  
  - 暴露 `POST /auth/exec` 和（若需要）`GET /cheq/status`；  
  - 策略规则中增加对 `exec:*` 类 Action 的规则；  
  - 保持 L0 的 `allowed_api_keys` 与 Node Agent 使用的凭证一致。

- **Node Agent**  
  - 配置项：`3AF_BASE_URL`、`3AF_API_KEY`（或等效）、可选 `TRACE_ID_HEADER`；  
  - 超时：与 3AF CHEQ 超时对齐，避免长时间阻塞；  
  - 降级：若 3AF 不可用，可配置「失败时默认拒绝」或「默认放行」（建议默认拒绝）。

- **Docker / 本机**  
  - Docker 内：确保容器能访问 3AF（同机 UDS 或跨机 TCP）；  
  - 本机：Node Agent 与 3AF 同机时优先 UDS，跨机时 gRPC + mTLS。

---

## 9. 与「非网络请求页」的关系

- 本设计只解决 **沙箱内命令/能力执行** 的授权与 3AF 对接。  
- 「非网络请求页」上的操作（如按钮、Panel）若最终会触发沙箱内命令，则 **同一命令会经过 Node Agent → 3AF**，无需再登录机器授权。  
- 若希望对「页面按钮」本身做显式授权（在点击时先问 3AF），可在前端或 BFF 增加对 3AF 的调用（同一 `/auth/exec` 或扩展为 `/auth/action`），与本文的 Node Agent 共用一套策略与审计。

---

## 10. 需要 sudo / 提权时的处理与提前授权

执行需要 **sudo（或其它提权）** 的场景很常见。3AF 接管后，原则是 **提前授权**：谁可以在哪类环境里以高权限跑哪些命令，由 3AF 在策略/Profile 里事先决定，而不是在机器上输密码或临时改 sudoers。

### 10.1 能力建模

将「以高权限执行」统一抽象为能力，与其它 exec 能力一致：

- **Action**：`exec:sudo` 或 `exec:privilege.escalation`。
- **Resource**：与普通 exec 一致（如 `docker://...`、`local://...`）。
- **Context**：必须带 `command_line`（即 `sudo` 后面的那条命令），便于策略匹配与审计，例如 `apt-get update`、`systemctl restart nginx`。

Node Agent 在拦截到用户/Agent 执行 `sudo <cmd>` 时，不直接调系统 sudo，而是先按上述构造请求问 3AF；3AF 返回 allow/deny/review 后，再决定是否真正执行 sudo（或通过 3AF 控制的提权路径执行）。

### 10.2 提前授权的两种形态

| 形态 | 说明 | 典型用法 |
|------|------|----------|
| **Profile 预授权（Sudo Hot Cache）** | 在 GetSandboxProfile 时下发一批「免审的 sudo 指令」白名单（可带参数模式）。Node Agent 命中则**本地直接放行**，不发起 ExecAuth RPC，再代为执行 sudo。 | 常见运维命令：`apt-get update`、`systemctl restart <服务>`、挂载/写特定系统路径等；按 Resource/环境区分不同白名单。 |
| **策略预配置** | 在 3AF 策略里预先配置「该 Subject/Resource 允许 `exec:sudo`，且 `command_line` 匹配某模式则 allow」。首次执行时即由策略直接放行，无需人工审批，但**全部记审计**。 | 某项目 dev 环境允许 `docker ...`、`systemctl ...`；prod 仅允许 `systemctl restart app`。 |

两者可并存：先查 Profile 的 Sudo Hot Cache，未命中再走 ExecAuth；策略里可对 `exec:sudo` 做 allow/review/deny。

### 10.3 执行路径（谁真正执行 sudo）

- **机器侧**：真正执行「以 root 跑命令」的仍是 OS（sudo 或 setuid helper）。3AF 不替代内核，只决定「这次是否允许」。
- **推荐做法**：  
  - 业务进程/Agent **不**直接拥有 NOPASSWD sudo；  
  - 由 **Node Agent**（或 3AF 指定的 privileged helper）在收到 3AF allow 后，代表业务侧调用 sudo 执行**已被 3AF 放行的那条命令**。  
  - 这样 sudoers 只需对 Node Agent 所在用户做最小 NOPASSWD（或通过 helper），且该用户仅被 Node Agent 用于执行 3AF 已授权的命令，避免整机大开。

### 10.4 常见需要 sudo 的场景（预授权示例）

| 场景 | 可预授权方式 |
|------|--------------|
| 安装包 | Profile Sudo Hot Cache：`apt-get update`, `apt-get install -y <包名>` 或策略模式 `apt-get install -y *`。 |
| 启停服务 | Hot Cache 或策略：`systemctl start/stop/restart <服务名>`。 |
| 写系统路径 | 策略：允许 `exec:sudo` 且 `command_line` 匹配 `cp ... /etc/...` 或限定路径。 |
| 挂载 / Docker | 策略或 Hot Cache：`mount ...`、`docker run ...`（按环境区分）。 |

### 10.5 系统提示「需要 sudo 权限，要求输入密码」时的接管与无密码执行

**场景**：用户或 Agent 执行某条命令时，系统发现需要 root 权限，会提示「需要 sudo 权限，请输入密码」。希望由 3AF 接管授权后，**执行时直接完成，不再弹出密码提示**。

**思路**：在系统有机会弹密码**之前**就完成「提取、判断、授权」，并由 **Node Agent（或 privileged helper）代为执行**；实际以高权限跑命令的进程在 sudoers 中配置为 NOPASSWD，因此执行阶段不会向用户索要密码。

#### 10.5.1 3AF 如何「提取判断」并接管授权

| 入口 | 说明 | 提取与判断方式 |
|------|------|----------------|
| **用户显式执行 `sudo <cmd>`** | 命令里已经带了 sudo。 | Node Agent 在 **exec 层拦截**到要执行的是 `sudo`，解析出后面的 `<cmd>`，构造 `action=exec:sudo`、`context.command_line=<cmd>` 问 3AF；先查 Profile 的 Sudo Hot Cache，命中则本地放行，未命中则 ExecAuth。授权通过后，由 Node Agent **代为**调用 `sudo <cmd>`（Node 侧已配置 NOPASSWD），原用户进程不再自己调 sudo，故**不会看到密码提示**。 |
| **用户执行未带 sudo 的命令，但该命令会触发系统要密码** | 例如直接执行 `apt-get install foo`、`systemctl start nginx`，系统或子进程会提示需要密码。 | Node Agent 维护一份「**可能需提权的命令**」列表（可配置，如 `apt-get`、`apt`、`systemctl`、`mount`、`docker` 等）。在这些命令 **exec 之前**拦截，先问 3AF：视为「等效 exec:sudo」请求（action 可为 `exec:sudo`，command_line 为整条命令）。3AF 允许后，**不由用户进程直接执行原命令**，而是由 Node Agent 代为执行 `sudo <原命令>`（或通过 helper），这样实际跑的是已配置 NOPASSWD 的 Node/helper，**执行时不会向用户弹密码**；用户/Agent 只得到「命令已获授权并执行完成」的结果。 |

即：**提取** = 在 exec 前识别「这是 sudo」或「这是可能需提权的命令」；**判断** = 3AF 按 Profile/策略做 allow/deny/review；**接管** = 授权通过后由 Node Agent/helper 代为以 NOPASSWD 身份执行，用户侧不再接触密码提示。

#### 10.5.2 执行时为何不再提示输入密码

- 实际以 root 权限执行命令的是 **Node Agent 或 3AF 指定的 privileged helper**，在 sudoers 里仅对该用户/helper 配置 **NOPASSWD**（且仅允许执行 3AF 已放行的命令）。
- 用户/Agent 进程**不**直接调用 sudo，也不持有密码；它只发起「要执行某条命令」的请求，经 Node Agent 问 3AF、3AF 放行后，由 Node Agent 代为执行。因此从用户视角：授权在 3AF 完成，执行时**直接完成，不再提示输入密码**。

#### 10.5.3 小结

- **系统会提示需要 sudo 并要输入密码**时：通过 **exec 前拦截**（显式 `sudo <cmd>` 或「可能需提权的命令」列表）+ **3AF 提前/实时授权** + **Node Agent 代为以 NOPASSWD 执行**，实现 3AF 接管授权且执行时无密码提示。
- 预授权（Profile Sudo Hot Cache、策略预配置）使已登记的命令在首次或后续执行时无需人工审批，仍可全部记审计。

---

## 11. 沙箱边界是否可由 3AF 接管？

**结论：可以。** 3AF 接管的是「边界的决策与下发」；真正在进程上执行边界的仍是 OS/内核，3AF 不替代内核，只决定「用哪套边界」。

### 11.1 能接管什么 / 不能接管什么

| 维度 | 能否由 3AF 接管 | 说明 |
|------|------------------|------|
| **边界由谁决定** | ✅ 可以 | 3AF 按 Subject/Resource/Context 决定「该沙箱用哪条 profile」并下发给节点。 |
| **边界长什么样** | ✅ 可以 | 网络是否开放、可写路径、允许的 syscall 集合等，由 3AF 侧策略/配置定义，节点只消费。 |
| **边界在谁身上执行** | ❌ 不能 | 实际限制（cgroups、namespaces、seccomp、mount）仍由节点上的内核/运行时执行，3AF 不跑在内核里。 |

即：**3AF = 控制平面（决定并下发 profile）；节点/内核 = 数据平面（按 profile 施加边界）。**

### 11.2 实现思路：3AF 下发「沙箱 Profile」

1. **在 3AF 侧定义并存储「沙箱 Profile」**  
   - 每条 profile 描述一类边界，例如：  
     - `network`: true/false  
     - `fs_writable_paths`: []string  
     - `allowed_syscalls`: 白名单或预设名（如 `default`, `restricted`）  
     - `max_memory_mb`, `readonly_root` 等（与 Docker / 本机能力对齐）。

2. **策略决定「谁用哪条 profile」**  
   - 按 Resource（如 `docker://project-a/dev`、`local://host-01`）、Subject、Context 映射到 profile_id。  
   - 与现有 L2 规则一致：可以是静态配置表，或由 Policy Engine 扩展一条「sandbox_profile(resource) -> profile_id」的查询。

3. **节点/Node Agent 拉取并应用 profile**  
  - **创建/启动执行环境时**：节点（或编排器）请求 3AF，例如 `GET /auth/sandbox-profile?resource=docker://project-a/dev` 或 `resource=local://host-01`，带 L0 身份或同机 PeerCred。  
  - 3AF 返回该 resource 对应的 profile（JSON：network、fs、syscalls 等）。  
  - 节点用该 profile 配置环境：Docker 用 `--network=none`、`--read-only`、`--cap-drop` 等；本机可据此做命名空间/权限约束或仅做策略缓存。
   - **可选**：沙箱已存在时，Node Agent 定期拉取 profile，若变更则告警或按策略决定是否重建沙箱（Phase 2）。

4. **执行时仍走 /auth/exec**  
   - 边界由 profile 定好；单次「是否允许执行某条命令」仍走 `POST /auth/exec`（allow/deny/review）。  
   - 这样：**边界 = 3AF 下发的 profile；单次操作 = 3AF 的 exec 鉴权。** 两层都归 3AF 管，节点只执行。

### 11.3 小结

- **沙箱边界上的 OS/内核能力** 可以被 **3AF 接管**，含义是：  
  - 边界的**定义与选择**在 3AF；  
  - 边界的**执行**仍在节点内核/运行时。  
- 实现上：新增「沙箱 profile」模型 + 按 resource 下发 profile 的接口（如 `GET /auth/sandbox-profile`），节点在创建/配置沙箱时拉取并应用，即可做到「边界也由 3AF 统一管，无需登录机器改沙箱配置」。

---

## 12. 方案逻辑审查：风险与缓解

在将方案作为「Docker/混合环境安全内核」落地时，需明确以下风险与实现约束。

### 12.1 PeerCred 的信任边界

- **问题**：若 PeerCred 作为 Request 字段由客户端填充，被入侵的 Agent 可伪造 UID/PID 发送给 3AF，从而冒充其他身份。
- **修正**：**当使用 UDS 时，服务端必须忽略 Request 中的 PeerCred，强制通过 `getsockopt(SO_PEERCRED)`（或等价 syscall）从连接中提取。** Request 中的该字段仅用于跨机调试或存根；同机身份以内核提供的凭证为准。跨机时身份由 mTLS 证书或 SessionToken 决定。

### 12.2 Hot Cache 的更新与失效

- **问题**：管理员在 3AF 后台紧急封禁某命令后，Node Agent 本地的 Hot Cache 若长期有效，仍会本地放行。
- **修正**：  
  - **配置热推**：在 AuthStream 中增加服务端主动推送 `ProfileUpdate`（即 `SandboxProfile`）的能力；Node 收到后刷新本地 Hot Cache（或使当前缓存失效并重新拉取 Profile）。  
  - **TTL**：为 Hot Cache 设置较短 TTL（如 5 分钟），到期后重新拉取 GetSandboxProfile 或依赖下一次 AuthStream 的 profile_update。  
  - **版本比对**：SandboxProfile 携带 `version`（或哈希），Node 可定期用 GetSandboxProfile 比对版本，若变化则刷新缓存。

### 12.3 Sudo 命令模式的匹配风险

- **问题**：若 `command_line_pattern` 仅做简单字符串匹配，易被绕过（如 `apt-get install` vs `apt-get  install` 多空格、或参数重排）。
- **修正**：Node Agent 侧应对命令行做 **参数解析（Tokenization）** 后再与 pattern 匹配；或约定 pattern 使用 **Glob 规范**（如 `systemctl restart *`），匹配前对 command_line 做规范化（如合并连续空白、规范引号）。Proto 中已在 `SudoHotCacheEntry` 的注释中写明该实现约束。

---

## 13. 实现要点（Implementation Notes）

以下为后端与 Node Agent 实现时的关键约束与示例。

### 13.1 服务端如何安全获取 PeerCred（Go）

使用 UDS 时，**不得信任客户端请求体中的 PeerCred**，应从连接层提取。示例（伪代码，实际需结合 `net.UnixConn.SyscallConn()` 与平台 syscall）：

```go
// 从 context 中提取 UDS 的 SO_PEERCRED（仅同机 UDS 有效）
import "google.golang.org/grpc/peer"

func getPeerCredFromCtx(ctx context.Context) (*PeerCred, error) {
    p, ok := peer.FromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("no peer info")
    }
    if p.Addr.Network() != "unix" {
        // TCP：无 PeerCred，依赖 mTLS 或 SessionToken
        return nil, nil
    }
    // 通过 net.UnixConn 获取 fd，再 GetsockoptUcred(SO_PEERCRED)
    // ucred, _ := syscall.GetsockoptUcred(fd, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
    return &PeerCred{Uid: ucred.Uid, Gid: ucred.Gid, Pid: int64(ucred.Pid)}, nil
}
```

在 GetSandboxProfile / ExecAuth 的 gRPC Interceptor 或 Handler 中调用上述逻辑，将得到的 PeerCred 注入到策略上下文；Request 中的 PeerCred 字段不参与信任决策。

### 13.2 3af-exec（Node Agent）Sudo 执行逻辑

以 wrapper 形态（如 `3af-exec sudo apt-get update`）为例：

1. **用户输入**：`3af-exec sudo apt-get update`
2. **查本地 Sudo Hot Cache**（来自上次 GetSandboxProfile 或 AuthStream profile_update）  
   - 若 `apt-get update` 与某条 `command_pattern` 匹配（经 Tokenization/Glob 规范化后）→ **HIT**
3. **HIT**：Client 直接以已配置 NOPASSWD 的身份执行 `sudo apt-get update`，**不请求 3AF**，不弹密码。
4. **MISS**：  
   - Client 发起 `ExecAuth`（action=`exec:sudo`，command_line=`apt-get update`）；  
   - 服务端返回 ALLOW（或 REVIEW 通过）；  
   - Client 再执行 `sudo apt-get update`（仍通过 Node/helper 的 NOPASSWD）。

预授权不仅指定「允许 sudo」，Proto 中 `SudoHotCacheEntry.run_as_user` 可指定以何用户执行（默认 root，也可 www-data 等）。

---

## 14. 小结

- **Node Agent**：在 Docker 或本机侧拦截执行请求，抽象为能力，调用 3AF `/auth/exec`（或 gRPC ExecAuth），按决策放行或拒绝，review 时轮询 CHEQ 或走 AuthStream 长连接。  
- **3AF**：新增 `/auth/exec`，将 exec 请求映射为现有 RequestContext + Policy + CHEQ + Audit，不登录机器即可完成授权与审计。  
- **策略**：沿用现有 L2 Resource/Action 规则，扩展 `exec:*` 能力集即可。  
- **沙箱边界**：可由 3AF 接管「边界的决策与下发」—— 通过 `GET /auth/sandbox-profile` 等接口按 resource 下发 profile，节点在创建/配置沙箱时拉取并应用；边界的具体执行仍在 OS/内核。  
- **sudo/提权**：需要 sudo 时由 3AF **提前授权**：Profile 可下发「Sudo Hot Cache」免审指令集，策略可预配置 `exec:sudo` 的 allow/review；Node Agent 在 exec 前拦截（显式 `sudo <cmd>` 或「可能需提权的命令」列表），先查预授权再问 3AF，通过后由 Node Agent/helper 代为以 NOPASSWD 执行，**执行时不再提示输入密码**。

下一步可实现：3AF 侧 `POST /auth/exec` handler + RequestContext 扩展，以及最小 Node Agent：Docker 内（entrypoint/sidecar）与本机（wrapper/进程内 Hook）各一例，验证整条链路；若需 3AF 管边界，再增加 sandbox profile 模型与 `/auth/sandbox-profile` 接口。

---

## 15. 架构完整优化建议（摘要）

在「同机/跨机」混合部署与「接管命令行执行」的深层授权场景下，可采用以下优化，提升性能、安全与可维护性。**技术契约以 gRPC + `proto/3af_exec.proto` 为统一基石。**

| 维度 | 优化要点 | 说明 |
|------|----------|------|
| **1. 通信层** | 双模传输 | **同机**：gRPC over Unix Domain Socket，绕过 TCP/IP，微秒级延迟；**跨机**：gRPC over TCP + mTLS，Protobuf 压缩 + 双向认证。 |
| **2. 授权机制** | 无密钥身份 | **同机**：3AF 通过 UDS `SO_PEERCRED` 获取 UID/PID/ContainerID，Node 无需存 API Key；**跨机**：Profile 创建时下发短期 SessionToken，仅该沙箱生命周期有效。 |
| **3. 执行层** | 能力网格 | **Hot Cache**：GetSandboxProfile 时下发免审指令集（如 ls, pwd），本地放行无 RPC；**语义化拦截**：对 curl/pip/rm 等做语义解析，上报 exec:net.outbound / exec:pkg.install 等，策略更贴近业务。 |
| **4. 审批流** | 长连接推送 | gRPC 双向流：Node 发起请求后保持连接进入 Waiting；管理员在 3AF 通过后，服务端经同一流 Push 结果，消除轮询延迟与资源损耗。 |
| **5. 核心模型** | RequestContext 3.0 | 统一 HTTP 与 Exec：Subject（User/App ID 或 Agent/Sandbox ID）、Action（GET/POST 或 exec:read/exec:write）、Resource、Context（User-Agent/IP 或 command_line, working_dir, parent_process）。 |
| **6. 可靠性** | 故障逃生 | **策略下沉**：3AF 定期向 Node 同步加密的「基础策略副本」；**Fail-Open / Fail-Close**：在 Profile 中按 Agent 等级配置，3AF 不可用时的默认行为。 |
| **7. 落地** | 契约与 Local Proxy | 定义 `3af_exec.proto`；同机部署 3AF Local Proxy，聚合 UDS 并转发远程 gRPC；Docker 镜像内 3af-node-agent 作 entrypoint/sidecar 接管 exec；本机用 wrapper（如 `3af-exec`）或进程内 Hook 接管 PATH/exec。 |

详见仓库内 `cmd/diting/proto/3af_exec.proto` 及本目录下对双模传输、PeerCred、SessionToken、双向流与 RequestContext 3.0 的 message 定义。
