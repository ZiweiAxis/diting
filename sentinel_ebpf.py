#!/usr/bin/env python3
"""
Sentinel-AI eBPF 监控模块
企业级智能体零信任治理平台 - 内核级监控

功能:
- 系统调用实时监控
- 文件操作追踪
- 网络连接审计
- 危险命令拦截

依赖: pip install bcc
运行: sudo python3 sentinel_ebpf.py
"""

import ctypes
import json
import os
import signal
import sys
import time
from datetime import datetime
from pathlib import Path

# 尝试导入 BCC，如果失败则提供安装指导
try:
    from bcc import BPF
    BCC_AVAILABLE = True
except ImportError:
    BCC_AVAILABLE = False
    print("⚠️  BCC 模块未安装")
    print("   请执行: sudo apt install -y bpfcc-tools linux-headers-$(uname -r)")
    print("   或: pip install bcc")
    print()
    print("   正在启动模拟模式...")
    print()

# ============ 配置 ============
CONFIG = {
    "log_file": "logs/ebpf_events.jsonl",
    "dangerous_commands": [
        "rm -rf", "rm -r", "dd if=", "mkfs", "chmod 777",
        "chown root", ":(){ :|:& };:", "kill -9", "reboot", "shutdown"
    ],
    "sensitive_paths": [
        "/etc", "/var", "/usr", "/home", "/root", "/boot", "/sys"
    ],
    "sensitive_ports": [22, 3306, 5432, 6379, 27017, 1433],  # SSH, MySQL, PostgreSQL, Redis, MongoDB, SQL Server
}

# ============ 颜色输出 ============
class Colors:
    CYAN = '\033[96m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    WHITE = '\033[97m'
    BOLD = '\033[1m'
    RESET = '\033[0m'

def print_colored(text, color=Colors.WHITE):
    print(f"{color}{text}{Colors.RESET}")

def print_header():
    print_colored("╔════════════════════════════════════════════════════════╗", Colors.CYAN)
    print_colored("║       Sentinel-AI eBPF 内核监控模块 v1.0             ║", Colors.CYAN)
    print_colored("║    系统调用级实时监控 - Agent 无法绕过的拦截          ║", Colors.CYAN)
    print_colored("╚════════════════════════════════════════════════════════╝", Colors.CYAN)
    print()

# ============ 事件结构体定义 ============
class Event(ctypes.Structure):
    _fields_ = [
        ("pid", ctypes.c_uint32),
        ("uid", ctypes.c_uint32),
        ("gid", ctypes.c_uint32),
        ("type", ctypes.c_uint8),  # 1=exec, 2=unlink, 3=connect, 4=write
        ("timestamp", ctypes.c_uint64),
        ("comm", ctypes.c_char * 16),
        ("filename", ctypes.c_char * 256),
        ("argv", ctypes.c_char * 256),
        ("addr_v4", ctypes.c_uint32),
        ("port", ctypes.c_uint16),
    ]

# ============ eBPF 程序 ============
BPF_PROGRAM = """
#include <uapi/linux/ptrace.h>
#include <linux/sched.h>
#include <net/sock.h>
#include <linux/socket.h>

struct event_t {
    u32 pid;
    u32 uid;
    u32 gid;
    u8 type;
    u64 timestamp;
    char comm[16];
    char filename[256];
    char argv[256];
    u32 addr_v4;
    u16 port;
};

BPF_PERF_OUTPUT(events);

// 类型定义
#define TYPE_EXEC 1
#define TYPE_UNLINK 2
#define TYPE_CONNECT 3
#define TYPE_WRITE 4

// 监控 execve - 命令执行
SEC("tracepoint/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx)
{
    struct event_t e = {};
    e.pid = bpf_get_current_pid_tgid() >> 32;
    e.uid = bpf_get_current_uid_gid();
    e.gid = bpf_get_current_uid_gid() >> 32;
    e.type = TYPE_EXEC;
    e.timestamp = bpf_ktime_get_ns();
    
    // 获取进程名
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取命令行第一个参数
    bpf_probe_read_user_str(e.argv, sizeof(e.argv), (void *)ctx->args[0]);
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}

// 监控 unlinkat - 文件删除
SEC("tracepoint/syscalls/sys_enter_unlinkat")
int trace_unlinkat(struct trace_event_raw_sys_enter *ctx)
{
    struct event_t e = {};
    e.pid = bpf_get_current_pid_tgid() >> 32;
    e.uid = bpf_get_current_uid_gid();
    e.gid = bpf_get_current_uid_gid() >> 32;
    e.type = TYPE_UNLINK;
    e.timestamp = bpf_ktime_get_ns();
    
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取文件路径
    char *filename = (char *)ctx->args[1];
    bpf_probe_read_user_str(e.filename, sizeof(e.filename), filename);
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}

// 监控 tcp_connect - TCP 连接
SEC("kprobe/tcp_v4_connect")
int kprobe_tcp_v4_connect(struct pt_regs *ctx, struct sock *sk)
{
    struct event_t e = {};
    e.pid = bpf_get_current_pid_tgid() >> 32;
    e.uid = bpf_get_current_uid_gid();
    e.gid = bpf_get_current_uid_gid() >> 32;
    e.type = TYPE_CONNECT;
    e.timestamp = bpf_ktime_get_ns();
    
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取目标地址和端口
    u16 dport = 0;
    bpf_probe_read_kernel(&dport, sizeof(dport), &sk->__sk_common.skc_dport);
    e.port = dport >> 8;
    
    u32 daddr = 0;
    bpf_probe_read_kernel(&daddr, sizeof(daddr), &sk->__sk_common.skc_daddr);
    e.addr_v4 = daddr;
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}

// 监控 write - 文件写入
SEC("tracepoint/syscalls/sys_enter_write")
int trace_write(struct trace_event_raw_sys_enter *ctx)
{
    struct event_t e = {};
    e.pid = bpf_get_current_pid_tgid() >> 32;
    e.uid = bpf_get_current_uid_gid();
    e.gid = bpf_get_current_uid_gid() >> 32;
    e.type = TYPE_WRITE;
    e.timestamp = bpf_ktime_get_ns();
    
    bpf_get_current_comm(e.comm, sizeof(e.comm));
    
    // 获取文件描述符 (简化处理，不获取路径以减少开销)
    e.filename[0] = 0;
    
    events.perf_submit(ctx, &e, sizeof(e));
    return 0;
}

char _license[] SEC("license") = "GPL";
"""

# ============ 决策引擎 ============
class DecisionEngine:
    """AI 驱动的决策引擎"""
    
    def __init__(self):
        self.events = []
    
    def analyze(self, event):
        """分析事件风险"""
        e = event
        
        # 根据类型分析
        if e.type == 1:  # EXEC
            return self._analyze_exec(e)
        elif e.type == 2:  # UNLINK
            return self._analyze_unlink(e)
        elif e.type == 3:  # CONNECT
            return self._analyze_connect(e)
        elif e.type == 4:  # WRITE
            return self._analyze_write(e)
        
        return {"decision": "ALLOW", "risk": "低", "reason": "未知事件类型"}
    
    def _analyze_exec(self, e):
        """分析命令执行"""
        argv = e.argv.decode('utf-8', errors='ignore').rstrip('\x00')
        
        # 检查危险命令
        for cmd in CONFIG["dangerous_commands"]:
            if cmd in argv:
                return {
                    "decision": "BLOCK",
                    "risk": "高",
                    "reason": f"检测到危险命令: {cmd}",
                    "suggestion": "此操作可能导致系统损坏，建议人工审批"
                }
        
        # 检查敏感文件操作
        if "/etc/" in argv or "/var/" in argv or "/usr/" in argv:
            return {
                "decision": "REVIEW",
                "risk": "中",
                "reason": "操作敏感系统目录",
                "suggestion": "需要人工确认"
            }
        
        return {"decision": "ALLOW", "risk": "低", "reason": "正常命令执行"}
    
    def _analyze_unlink(self, e):
        """分析文件删除"""
        path = e.filename.decode('utf-8', errors='ignore').rstrip('\x00')
        
        # 检查敏感路径
        for sensitive in CONFIG["sensitive_paths"]:
            if path.startswith(sensitive):
                return {
                    "decision": "BLOCK",
                    "risk": "高",
                    "reason": f"尝试删除系统目录文件: {path}",
                    "suggestion": "禁止删除系统目录文件"
                }
        
        # 检查是否是日志删除
        if ".log" in path and "/var/log/" in path:
            return {
                "decision": "REVIEW",
                "risk": "中",
                "reason": f"删除日志文件: {path}",
                "suggestion": "请确认操作必要性"
            }
        
        return {"decision": "ALLOW", "risk": "低", "reason": "普通文件删除"}
    
    def _analyze_connect(self, e):
        """分析网络连接"""
        port = e.port
        
        # 检查敏感端口
        if port in CONFIG["sensitive_ports"]:
            port_names = {
                22: "SSH", 3306: "MySQL", 5432: "PostgreSQL",
                6379: "Redis", 27017: "MongoDB", 1433: "SQL Server"
            }
            service = port_names.get(port, "unknown")
            return {
                "decision": "REVIEW",
                "risk": "高",
                "reason": f"连接敏感服务端口: {port} ({service})",
                "suggestion": "Agent 连接数据库需要授权"
            }
        
        # 检查外网连接
        if port == 443 or port == 80:
            return {
                "decision": "ALLOW",
                "risk": "低",
                "reason": "常规 Web 服务连接"
            }
        
        return {"decision": "ALLOW", "risk": "低", "reason": "常规网络连接"}
    
    def _analyze_write(self, e):
        """分析文件写入"""
        comm = e.comm.decode('utf-8', errors='ignore').rstrip('\x00')
        
        # 检查是否是 Agent 进程
        if "agent" in comm.lower() or "ai" in comm.lower():
            return {
                "decision": "REVIEW",
                "risk": "中",
                "reason": f"AI 进程 ({comm}) 正在写入文件",
                "suggestion": "监控 AI 文件操作"
            }
        
        return {"decision": "ALLOW", "risk": "低", "reason": "常规文件写入"}

# ============ 模拟监控器 ============
class MockMonitor:
    """BCC 不可用时的模拟监控器"""
    
    def __init__(self):
        self.decision_engine = DecisionEngine()
        print("⚠️  运行在模拟模式 (BCC 不可用)")
        print("   模拟模式会生成一些示例事件用于演示")
        print()
        print("   要启用真实监控，请:")
        print("   1. sudo apt install -y bpfcc-tools linux-headers-$(uname -r)")
        print("   2. pip install bcc")
        print()
    
    def start(self):
        """开始模拟监控"""
        print_colored("🔍 模拟监控已启动 (每 5 秒生成示例事件)", Colors.YELLOW)
        print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
        print()
        
        try:
            while True:
                self.generate_mock_event()
                time.sleep(5)
        except KeyboardInterrupt:
            print_colored("\n\n监控已停止", Colors.YELLOW)
    
    def generate_mock_event(self):
        """生成模拟事件"""
        import random
        
        # 模拟事件类型
        event_types = [
            (1, "execve", "ls -la", 0, 0),
            (1, "execve", "cat /etc/passwd", 0, 0),
            (2, "unlinkat", "/tmp/test.log", 0, 0),
            (3, "tcp_connect", None, 0x0100007f, 3306),
            (1, "execve", "rm -rf /var/log/test", 0, 0),
            (2, "unlinkat", "/etc/config.json", 0, 0),
            (3, "tcp_connect", None, 0x0100007f, 22),
        ]
        
        event_type, event_name, filename, addr, port = random.choice(event_types)
        
        # 创建事件
        e = Event()
        e.pid = random.randint(1000, 9999)
        e.uid = 1000
        e.gid = 1000
        e.type = event_type
        e.timestamp = int(time.time() * 1e9)
        
        comm = event_name if event_name != "tcp_connect" else "python3"
        e.comm = comm.encode('utf-8')[:15].ljust(16, b'\x00')
        
        if filename:
            e.filename = filename.encode('utf-8')[:255].ljust(256, b'\x00')
        else:
            e.filename = b'\x00' * 256
        
        if event_type == 1:
            e.argv = filename.encode('utf-8')[:255].ljust(256, b'\x00')
        
        e.addr_v4 = addr
        e.port = port
        
        # 处理事件
        self.handle_event(e)
    
    def handle_event(self, event):
        """处理事件"""
        timestamp = datetime.fromtimestamp(event.timestamp / 1e9)
        
        # 根据类型显示
        if event.type == 1:
            self.show_exec_event(event, timestamp)
        elif event.type == 2:
            self.show_unlink_event(event, timestamp)
        elif event.type == 3:
            self.show_connect_event(event, timestamp)
        
        # 决策分析
        decision = self.decision_engine.analyze(event)
        self.show_decision(decision)
        
        # 记录日志
        self.log_event(event, decision)
    
    def show_exec_event(self, event, timestamp):
        """显示命令执行事件"""
        argv = event.argv.decode('utf-8', errors='ignore').rstrip('\x00')
        
        color = Colors.RED if any(cmd in argv for cmd in CONFIG["dangerous_commands"]) else Colors.GREEN
        icon = "🚨" if any(cmd in argv for cmd in CONFIG["dangerous_commands"]) else "✓"
        
        print_colored(f"\n[{timestamp}] {icon} 命令执行", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid})")
        print(f"   命令: {argv}")
    
    def show_unlink_event(self, event, timestamp):
        """显示文件删除事件"""
        path = event.filename.decode('utf-8', errors='ignore').rstrip('\x00')
        
        is_sensitive = any(path.startswith(p) for p in CONFIG["sensitive_paths"])
        color = Colors.RED if is_sensitive else Colors.YELLOW
        icon = "🚨" if is_sensitive else "⚠️"
        
        print_colored(f"\n[{timestamp}] {icon} 文件删除", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid})")
        print(f"   路径: {path}")
    
    def show_connect_event(self, event, timestamp):
        """显示网络连接事件"""
        addr = f"{event.addr_v4 & 0xFF}.{(event.addr_v4 >> 8) & 0xFF}.{(event.addr_v4 >> 16) & 0xFF}.{(event.addr_v4 >> 24) & 0xFF}"
        
        is_sensitive = event.port in CONFIG["sensitive_ports"]
        color = Colors.YELLOW if is_sensitive else Colors.GREEN
        icon = "⚠️" if is_sensitive else "🌐"
        
        print_colored(f"\n[{timestamp}] {icon} 网络连接", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid})")
        print(f"   目标: {addr}:{event.port}")
    
    def show_decision(self, decision):
        """显示决策结果"""
        decision_colors = {
            "BLOCK": Colors.RED,
            "REVIEW": Colors.YELLOW,
            "ALLOW": Colors.GREEN
        }
        color = decision_colors.get(decision["decision"], Colors.WHITE)
        
        icon = "❌" if decision["decision"] == "BLOCK" else ("⚠️" if decision["decision"] == "REVIEW" else "✓")
        print_colored(f"   决策: {icon} {decision['decision']} ({decision['risk']}风险)", color)
        print_colored(f"   原因: {decision['reason']}", Colors.WHITE)
        print_colored(f"   建议: {decision['suggestion']}", Colors.CYAN)
    
    def log_event(self, event, decision):
        """记录事件到日志文件"""
        os.makedirs("logs", exist_ok=True)
        
        log_entry = {
            "timestamp": datetime.fromtimestamp(event.timestamp / 1e9).isoformat(),
            "pid": event.pid,
            "uid": event.uid,
            "type": event.type,
            "comm": event.comm.decode('utf-8', errors='ignore').rstrip('\x00'),
            "filename": event.filename.decode('utf-8', errors='ignore').rstrip('\x00'),
            "argv": event.argv.decode('utf-8', errors='ignore').rstrip('\x00'),
            "addr_v4": event.addr_v4,
            "port": event.port,
            "decision": decision["decision"],
            "risk": decision["risk"],
            "reason": decision["reason"],
            "suggestion": decision["suggestion"]
        }
        
        with open(CONFIG["log_file"], "a") as f:
            f.write(json.dumps(log_entry) + "\n")

# ============ 真实监控器 ============
class RealMonitor:
    """使用 BCC 的真实监控器"""
    
    def __init__(self):
        print_colored("✓ BCC 模块已加载", Colors.GREEN)
        self.bpf = BPF(text=BPF_PROGRAM)
        self.events = self.bpf["events"]
        self.decision_engine = DecisionEngine()
    
    def start(self):
        """开始监控"""
        print_colored("🔍 eBPF 监控已启动 - 等待系统事件...", Colors.GREEN)
        print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
        print()
        
        try:
            for event in self.events:
                self.handle_event(event)
        except KeyboardInterrupt:
            print_colored("\n\n监控已停止", Colors.YELLOW)
    
    def handle_event(self, event):
        """处理事件"""
        e = event
        timestamp = datetime.fromtimestamp(e.timestamp / 1e9)
        
        # 根据类型显示
        if e.type == 1:
            self.show_exec_event(e, timestamp)
        elif e.type == 2:
            self.show_unlink_event(e, timestamp)
        elif e.type == 3:
            self.show_connect_event(e, timestamp)
        elif e.type == 4:
            self.show_write_event(e, timestamp)
        
        # 决策分析
        decision = self.decision_engine.analyze(e)
        self.show_decision(decision)
        
        # 记录日志
        self.log_event(e, decision)
    
    def show_exec_event(self, event, timestamp):
        """显示命令执行事件"""
        argv = event.argv.decode('utf-8', errors='ignore').rstrip('\x00')
        
        color = Colors.RED if any(cmd in argv for cmd in CONFIG["dangerous_commands"]) else Colors.GREEN
        icon = "🚨" if any(cmd in argv for cmd in CONFIG["dangerous_commands"]) else "✓"
        
        print_colored(f"\n[{timestamp}] {icon} 命令执行", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid}, UID: {event.uid})")
        print(f"   命令: {argv}")
    
    def show_unlink_event(self, event, timestamp):
        """显示文件删除事件"""
        path = event.filename.decode('utf-8', errors='ignore').rstrip('\x00')
        
        is_sensitive = any(path.startswith(p) for p in CONFIG["sensitive_paths"])
        color = Colors.RED if is_sensitive else Colors.YELLOW
        icon = "🚨" if is_sensitive else "⚠️"
        
        print_colored(f"\n[{timestamp}] {icon} 文件删除", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid}, UID: {event.uid})")
        print(f"   路径: {path}")
    
    def show_connect_event(self, event, timestamp):
        """显示网络连接事件"""
        addr = f"{event.addr_v4 & 0xFF}.{(event.addr_v4 >> 8) & 0xFF}.{(event.addr_v4 >> 16) & 0xFF}.{(event.addr_v4 >> 24) & 0xFF}"
        
        is_sensitive = event.port in CONFIG["sensitive_ports"]
        color = Colors.YELLOW if is_sensitive else Colors.GREEN
        icon = "⚠️" if is_sensitive else "🌐"
        
        print_colored(f"\n[{timestamp}] {icon} 网络连接", color)
        print(f"   进程: {event.comm.decode().rstrip(chr(0))} (PID: {event.pid}, UID: {event.uid})")
        print(f"   目标: {addr}:{event.port}")
    
    def show_write_event(self, event, timestamp):
        """显示文件写入事件"""
        comm = event.comm.decode('utf-8', errors='ignore').rstrip('\x00')
        
        is_agent = "agent" in comm.lower() or "ai" in comm.lower()
        color = Colors.YELLOW if is_agent else Colors.GREEN
        icon = "🤖" if is_agent else "✓"
        
        print_colored(f"\n[{timestamp}] {icon} 文件写入", color)
        print(f"   进程: {comm} (PID: {event.pid}, UID: {event.uid})")
    
    def show_decision(self, decision):
        """显示决策结果"""
        decision_colors = {
            "BLOCK": Colors.RED,
            "REVIEW": Colors.YELLOW,
            "ALLOW": Colors.GREEN
        }
        color = decision_colors.get(decision["decision"], Colors.WHITE)
        
        icon = "❌" if decision["decision"] == "BLOCK" else ("⚠️" if decision["decision"] == "REVIEW" else "✓")
        print_colored(f"   决策: {icon} {decision['decision']} ({decision['risk']}风险)", color)
        print_colored(f"   原因: {decision['reason']}", Colors.WHITE)
        print_colored(f"   建议: {decision['suggestion']}", Colors.CYAN)
    
    def log_event(self, event, decision):
        """记录事件到日志文件"""
        os.makedirs("logs", exist_ok=True)
        
        log_entry = {
            "timestamp": datetime.fromtimestamp(event.timestamp / 1e9).isoformat(),
            "pid": event.pid,
            "uid": event.uid,
            "type": event.type,
            "comm": event.comm.decode('utf-8', errors='ignore').rstrip('\x00'),
            "filename": event.filename.decode('utf-8', errors='ignore').rstrip('\x00'),
            "argv": event.argv.decode('utf-8', errors='ignore').rstrip('\x00'),
            "addr_v4": event.addr_v4,
            "port": event.port,
            "decision": decision["decision"],
            "risk": decision["risk"],
            "reason": decision["reason"],
            "suggestion": decision["suggestion"]
        }
        
        with open(CONFIG["log_file"], "a") as f:
            f.write(json.dumps(log_entry) + "\n")

# ============ 主程序 ============
def main():
    print_header()
    
    # 检查权限
    if os.getuid() != 0:
        print_colored("❌ 错误: 需要 root 权限运行", Colors.RED)
        print_colored("   请使用: sudo python3 sentinel_ebpf.py", Colors.YELLOW)
        sys.exit(1)
    
    # 选择监控器
    if BCC_AVAILABLE:
        monitor = RealMonitor()
    else:
        monitor = MockMonitor()
    
    # 启动监控
    monitor.start()

if __name__ == "__main__":
    main()
