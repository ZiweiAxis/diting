# Sentinel-AI eBPF 技术架构文档

## 一、eBPF 技术深度解析

### 1.1 eBPF 是什么？

**eBPF (Extended Berkeley Packet Filter)** 是 Linux 内核的革命性技术，允许在**不修改内核源码**的情况下，在**内核态**安全地执行**用户态程序**。

| 特性 | 传统内核模块 | eBPF |
|------|-------------|------|
| **安全性** | 可能导致内核崩溃 | BPF 验证器保证安全 |
| **开发难度** | 需要深入了解内核 | 类似普通 C 程序 |
| **部署方式** | 重新编译内核 | 热加载，动态更新 |
| **性能** | 高 | 高（零拷贝，内核态执行） |

### 1.2 BPF 验证器工作原理

```
用户态程序
     │
     ▼
┌──────────────────┐
│  eBPF 字节码      │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────────┐
│      BPF 验证器                  │
│  - 语法检查                      │
│  - 内存访问安全验证              │
│  - 循环次数限制 (保证不会死循环) │
│  - 函数调用白名单检查            │
│  - 寄存器使用验证                │
└────────┬────────────────────────┘
         │
         ▼
┌──────────────────┐
│  JIT 编译器      │
│  (Just-In-Time)  │
└────────┬─────────┘
         │
         ▼
   原生机器码
         │
         ▼
┌──────────────────┐
│   Linux 内核      │
│   (执行)          │
└──────────────────┘
```

---

## 二、Sentinel-AI eBPF 架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Agent 应用层                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │ LangChain    │  │ AutoGPT      │  │ OpenClaw     │               │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
└────────────────────────┬────────────────────────────────────────────┘
                         │ 系统调用
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Sentinel-AI 用户态                             │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │                    eBPF 协调器                           │      │
│  │  - 加载 eBPF 程序到内核                                  │      │
│  │  - 接收 perf event ring buffer 事件                      │      │
│  │  - 事件分类与路由                                        │      │
│  └───────────────────┬──────────────────────────────────────┘      │
│                      │                                              │
│  ┌───────────────────┼──────────────────┐                        │
│  │                   ▼                  │                        │
│  │  ┌────────────────────────────┐    │                        │
│  │  │      决策引擎 (LLM)        │    │                        │
│  │  │  - 风险评估                │    │                        │
│  │  │  - 意图分析                │    │                        │
│  │  │  - 策略匹配                │    │                        │
│  │  └────────────────────────────┘    │                        │
│  │                                    │                        │
│  │  ┌────────────────────────────┐    │                        │
│  │  │      审批工作流            │    │                        │
│  │  │  - 企业微信/钉钉推送       │    │                        │
│  │  │  - Web UI 审批界面         │    │                        │
│  │  │  - 自动批准/拒绝规则       │    │                        │
│  │  └────────────────────────────┘    │                        │
│  │                                    │                        │
│  │  ┌────────────────────────────┐    │                        │
│  │  │      审计日志存储          │    │                        │
│  │  │  - 全链路记录              │    │                        │
│  │  │  - 证据链存储              │    │                        │
│  │  └────────────────────────────┘    │                        │
│  └───────────────────┬────────────────┘                        │
└──────────────────────┼────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Linux 内核 (eBPF)                              │
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Exec 监控   │  │  File 监控   │  │  Net 监控    │              │
│  │              │  │              │  │              │              │
│  │  execve()    │  │  unlinkat()  │  │  tcp_v4_     │              │
│  │  execveat()  │  │  renameat()  │  │  connect()   │              │
│  │              │  │              │  │  tcp_v6_     │              │
│  │              │  │              │  │  connect()   │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                 │                 │                        │
│         ▼                 ▼                 ▼                        │
│  ┌──────────────────────────────────────────────────┐              │
│  │          Perf Event Ring Buffer                  │              │
│  │  (内核到用户态的高性能数据传输通道)              │              │
│  └──────────────────────┬───────────────────────────┘              │
└─────────────────────────┼────────────────────────────────────────────┘
                          │
                          ▼
                     用户态程序
                   (读取事件)
```

### 2.2 监控点设计

| 监控点 | eBPF 挂载点 | 捕获内容 | 风险等级 |
|--------|-------------|---------|---------|
| **命令执行** | `tracepoint/syscalls/sys_enter_execve` | 完整命令行参数 | 高 |
| **文件删除** | `tracepoint/syscalls/sys_enter_unlinkat` | 文件路径 | 高 |
| **文件修改** | `tracepoint/syscalls/sys_enter_renameat` | 原路径/新路径 | 中 |
| **网络连接** | `kprobe/tcp_v4_connect` | 目标 IP/端口 | 中 |
| **敏感系统调用** | `kprobe/mknod`, `kprobe/chmod` | 设备文件/权限 | 高 |

---

## 三、技术实现细节

### 3.1 Exec 监控代码

```c
// 监控 execve 系统调用
SEC("tracepoint/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx)
{
    struct event_t e = {};
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    // 获取进程信息
    e.pid = pid;
    e.uid = bpf_get_current_uid_gid();
    e.timestamp = bpf_ktime_get_ns();
    e.type = TYPE_EXEC;
    
    // 获取进程名
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取命令行参数 (最多读取 256 字节)
    bpf_probe_read_user_str(e.argv, sizeof(e.argv), 
                            (void *)ctx->args[0]);
    
    // 发送到用户态
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}
```

### 3.2 文件删除监控代码

```c
// 监控 unlinkat 系统调用
SEC("tracepoint/syscalls/sys_enter_unlinkat")
int trace_unlinkat(struct trace_event_raw_sys_enter *ctx)
{
    struct event_t e = {};
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    e.pid = pid;
    e.uid = bpf_get_current_uid_gid();
    e.timestamp = bpf_ktime_get_ns();
    e.type = TYPE_UNLINK;
    
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取文件路径
    char *filename = (char *)ctx->args[1];
    bpf_probe_read_user_str(e.filename, sizeof(e.filename), filename);
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}
```

### 3.3 网络连接监控代码

```c
// 监控 TCP 连接
SEC("kprobe/tcp_v4_connect")
int kprobe_tcp_v4_connect(struct pt_regs *ctx, struct sock *sk)
{
    struct event_t e = {};
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    
    e.pid = pid;
    e.timestamp = bpf_ktime_get_ns();
    e.type = TYPE_CONNECT;
    
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取目标端口
    u16 dport = 0;
    bpf_probe_read_kernel(&dport, sizeof(dport), 
                           &sk->__sk_common.skc_dport);
    e.port = bpf_ntohs(dport);
    
    // 获取目标地址
    u32 daddr = 0;
    bpf_probe_read_kernel(&daddr, sizeof(daddr), 
                           &sk->__sk_common.skc_daddr);
    e.addr_v4 = daddr;
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}
```

---

## 四、性能优化

### 4.1 性能对比

| 指标 | 传统 strace | eBPF | Sentinel-AI |
|------|-------------|------|-------------|
| **延迟** | ~1000x 慢 | < 1% 开销 | ~5% 开销 |
| **吞吐量** | 受限 | 无限制 | ~10k events/s |
| **CPU 使用** | 高 | 极低 | 低 |
| **内存占用** | 高 | < 10MB | ~50MB |

### 4.2 优化策略

1. **事件过滤**
   - 只监控指定 PID 范围
   - 只监控特定 UID
   - 只监控特定网络命名空间

2. **批量处理**
   - 使用 perf_event_batch
   - 减少 sysenter/sysexit

3. **数据压缩**
   - 使用哈希表去重
   - 只发送变更

---

## 五、安全加固

### 5.1 防逃逸设计

```
┌────────────────────────────────────────────────────┐
│                  Agent 容器                          │
│  ┌──────────────┐                                   │
│  │  Agent 进程   │                                   │
│  └──────┬───────┘                                   │
└─────────┼──────────────────────────────────────────┘
          │
          │ eBPF 监控 (无法绕过)
          ▼
┌────────────────────────────────────────────────────┐
│                Linux 内核                           │
│                                                     │
│  ┌──────────────────────────────────────────┐     │
│  │         eBPF 程序 (在内核态运行)           │     │
│  │  - 即使 Agent 容器逃逸，仍在内核层监控     │     │
│  │  - 无法被用户态程序修改或删除              │     │
│  └──────────────────────────────────────────┘     │
└────────────────────────────────────────────────────┘
```

### 5.2 零信任原则

1. **默认拒绝** - 所有操作默认需要审批
2. **最小权限** - 只授予必要的权限
3. **持续验证** - 每次操作都重新评估
4. **审计追踪** - 所有操作可追溯

---

## 六、部署指南

### 6.1 K8s DaemonSet 部署

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: sentinel-ebpf
  namespace: sentinel-system
spec:
  selector:
    matchLabels:
      app: sentinel-ebpf
  template:
    metadata:
      labels:
        app: sentinel-ebpf
    spec:
      hostPID: true  # 共享 PID 命名空间
      hostNetwork: true  # 共享网络
      tolerations:
      - effect: NoSchedule
        operator: Exists
      containers:
      - name: sentinel
        image: sentinel-ai/ebpf:latest
        securityContext:
          privileged: true  # 特权模式
          capabilities:
            add:
            - SYS_ADMIN
            - SYS_PTRACE
            - NET_ADMIN
        volumeMounts:
        - name: sys
          mountPath: /sys
          readOnly: true
        - name: proc
          mountPath: /proc
          readOnly: true
        - name: modules
          mountPath: /lib/modules
          readOnly: true
        - name: src
          mountPath: /usr/src
          readOnly: true
      volumes:
      - name: sys
        hostPath:
          path: /sys
      - name: proc
        hostPath:
          path: /proc
      - name: modules
        hostPath:
          path: /lib/modules
      - name: src
        hostPath:
          path: /usr/src
```

### 6.2 系统要求

| 要求 | 版本/配置 |
|------|----------|
| **内核版本** | >= 4.10 (推荐 >= 5.10) |
| **BPF 支持** | CONFIG_BPF=y |
| **cgroup v2** | 必需 |
| **CPU** | x86_64 / ARM64 |
| **内存** | >= 4GB |

---

## 七、故障排查

### 7.1 常见问题

**Q1: eBPF 程序无法加载**
```bash
# 检查 BPF 支持
cat /proc/config.gz | gunzip | grep BPF

# 检查内存限制
ulimit -l  # 应该 >= 8388608
```

**Q2: 权限不足**
```bash
# 确认 root 权限
whoami  # 应该是 root

# 或配置 capabilities
sudo setcap cap_sys_admin+ep /path/to/sentinel-ai
```

**Q3: 性能问题**
```bash
# 查看 eBPF 统计信息
sudo bpftool prog show
sudo bpftool perf show

# 查看事件丢失
cat /sys/kernel/debug/tracing/trace_pipe
```

---

## 八、参考资料

- [BPF and XDP Reference Guide](https://docs.cilium.io/en/stable/bpf/)
- [Linux Kernel eBPF Documentation](https://www.kernel.org/doc/html/latest/bpf/)
- [BCC - Tools for BPF-based Linux IO analysis](https://github.com/iovisor/bcc)
- [Cilium - eBPF-based Networking](https://cilium.io/)

---

## 九、后续计划

- [ ] 支持更多监控点 (socket, tracepoints)
- [ ] 实现 cgroup 级别的拦截
- [ ] WebAssembly 驱动的策略引擎
- [ ] 跨平台支持 (macOS, Windows eBPF)
- [ ] AI 自适应策略学习
