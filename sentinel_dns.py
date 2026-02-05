#!/usr/bin/env python3
"""
Sentinel-AI DNS 监控模块
企业级智能体零信任治理平台 - DNS 安全治理

功能:
- DNS 查询实时监控
- 威胁情报匹配
- 域名白/黑名单管理
- DGA 域名检测
- DNS 隧道检测

依赖: pip install scapy
运行: sudo python3 sentinel_dns.py
"""

import json
import time
import socket
import threading
import argparse
from datetime import datetime
from collections import defaultdict, deque
from typing import Dict, List, Tuple, Optional

# 尝试导入 Scapy
try:
    from scapy.all import sniff, DNS, DNSQR, IP, UDP, TCP
    SCAPY_AVAILABLE = True
except ImportError:
    SCAPY_AVAILABLE = False
    print("⚠️  Scapy 未安装，将运行模拟模式")
    print("   安装: pip install scapy")
    print()

# ============ 配置 ============
CONFIG = {
    "interface": None,  # None = 监听所有接口
    "port": 53,  # DNS 端口
    "log_file": "logs/dns_events.jsonl",
    "threat_intel": {
        "enabled": True,
        "timeout": 5,  # 威胁情报查询超时 (秒)
    },
    "dga_detection": {
        "enabled": True,
        "threshold": 0.7,  # DGA 检测阈值
    },
    "dns_tunnel_detection": {
        "enabled": True,
        "max_queries_per_minute": 50,  # 每分钟最大查询数
    },
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
    print_colored("║       Sentinel-AI DNS 安全监控模块 v1.0               ║", Colors.CYAN)
    print_colored("║    防止恶意域名、数据外泄、DNS 隧道                  ║", Colors.CYAN)
    print_colored("╚════════════════════════════════════════════════════════╝", Colors.CYAN)
    print()

# ============ 域名分类管理 ============
class DomainClassifier:
    """域名分类器"""
    
    def __init__(self):
        self.whitelist = set()  # 白名单
        self.blacklist = set()  # 黑名单
        self.greylist = set()   # 灰名单（需人工审核）
        
        # 加载默认规则
        self._load_default_rules()
    
    def _load_default_rules(self):
        """加载默认域名规则"""
        # 常见安全域名（白名单）
        self.whitelist.update([
            'google.com', 'microsoft.com', 'apple.com',
            'amazon.com', 'cloudflare.com', 'github.com',
            'docker.io', 'docker.com', 'kubernetes.io',
            'ollama.ai', 'huggingface.co', 'openai.com',
            'python.org', 'pip.org', 'pypi.org',
            'npmjs.com', 'npmjs.org', 'golang.org',
        ])
        
        # 已知恶意域名（黑名单）
        self.blacklist.update([
            'malicious-domain.com', 'phishing-site.net',
            'c2-server.bad', 'crypto-mining.pool',
        ])
    
    def classify(self, domain: str) -> str:
        """分类域名"""
        # 移除 www. 前缀
        domain_clean = domain.lower().replace('www.', '')
        
        # 检查黑名单
        for black in self.blacklist:
            if black in domain_clean:
                return "BLACKLIST"
        
        # 检查白名单
        for white in self.whitelist:
            if white in domain_clean:
                return "WHITELIST"
        
        # 检查灰名单特征
        if self._is_greylist(domain_clean):
            return "GREYLIST"
        
        return "UNKNOWN"
    
    def _is_greylist(self, domain: str) -> bool:
        """判断是否可能是灰名单域名"""
        # 短域名（可能是 DGA）
        if len(domain) < 8:
            return True
        
        # 随机字符比例高
        import string
        random_chars = sum(c in string.digits for c in domain)
        if random_chars / len(domain) > 0.5:
            return True
        
        return False
    
    def add_whitelist(self, domain: str):
        self.whitelist.add(domain.lower())
    
    def add_blacklist(self, domain: str):
        self.blacklist.add(domain.lower())

# ============ 威胁情报模块 ============
class ThreatIntel:
    """威胁情报查询"""
    
    def __init__(self):
        self.cache = {}
        self.cache_ttl = 3600  # 缓存 1 小时
    
    def check(self, domain: str) -> Dict:
        """检查域名威胁情报"""
        # 检查缓存
        if domain in self.cache:
            cached, timestamp = self.cache[domain]
            if time.time() - timestamp < self.cache_ttl:
                return cached
        
        # 查询威胁情报（模拟）
        result = self._query_threat_intel(domain)
        
        # 更新缓存
        self.cache[domain] = (result, time.time())
        
        return result
    
    def _query_threat_intel(self, domain: str) -> Dict:
        """查询威胁情报 API（模拟）"""
        # 实际场景可集成:
        # - VirusTotal API
        # - AlienVault OTX
        # - Cisco Umbrella
        # - IBM X-Force
        
        # 模拟响应
        import random
        
        # 5% 概率返回恶意
        if random.random() < 0.05:
            return {
                "threat": True,
                "category": "C2",
                "confidence": 0.95,
                "sources": ["VirusTotal", "OTX"],
                "last_seen": datetime.now().isoformat()
            }
        
        # 10% 概率返回可疑
        if random.random() < 0.10:
            return {
                "threat": False,
                "suspicious": True,
                "category": "Unknown",
                "confidence": 0.60,
                "sources": [],
            }
        
        return {
            "threat": False,
            "suspicious": False,
            "confidence": 1.0,
            "sources": [],
        }

# ============ DGA 检测 ============
class DGADetector:
    """域名生成算法（DGA）检测"""
    
    def __init__(self, threshold=0.7):
        self.threshold = threshold
    
    def detect(self, domain: str) -> Dict:
        """检测是否为 DGA 域名"""
        # 移除 TLD
        parts = domain.split('.')
        if len(parts) < 2:
            return {"dga": False, "score": 0.0}
        
        main_domain = '.'.join(parts[:-1])
        
        # 计算特征
        entropy = self._calculate_entropy(main_domain)
        length = len(main_domain)
        ratio_digit = sum(c.isdigit() for c in main_domain) / length
        ratio_consonant = self._consonant_ratio(main_domain)
        vowel_ratio = self._vowel_ratio(main_domain)
        
        # DGA 特征评分
        score = 0.0
        
        # 熵值高
        if entropy > 3.5:
            score += 0.3
        
        # 数字比例高
        if ratio_digit > 0.3:
            score += 0.2
        
        # 辅音比例异常
        if ratio_consonant > 0.7:
            score += 0.2
        
        # 元音比例异常
        if vowel_ratio < 0.1 or vowel_ratio > 0.5:
            score += 0.2
        
        # 长度异常
        if length > 20 or length < 6:
            score += 0.1
        
        return {
            "dga": score >= self.threshold,
            "score": score,
            "entropy": entropy,
            "length": length,
            "ratio_digit": ratio_digit,
            "ratio_consonant": ratio_consonant,
            "vowel_ratio": vowel_ratio
        }
    
    def _calculate_entropy(self, s: str) -> float:
        """计算字符串熵"""
        import math
        if not s:
            return 0.0
        
        freq = defaultdict(int)
        for c in s:
            freq[c] += 1
        
        entropy = 0.0
        for count in freq.values():
            p = count / len(s)
            entropy -= p * math.log2(p)
        
        return entropy
    
    def _consonant_ratio(self, s: str) -> float:
        """计算辅音比例"""
        vowels = set('aeiouAEIOU')
        consonants = sum(1 for c in s if c.isalpha() and c not in vowels)
        total = sum(1 for c in s if c.isalpha())
        return consonants / total if total > 0 else 0.0
    
    def _vowel_ratio(self, s: str) -> float:
        """计算元音比例"""
        vowels = set('aeiouAEIOU')
        vowel_count = sum(1 for c in s if c in vowels)
        total = sum(1 for c in s if c.isalpha())
        return vowel_count / total if total > 0 else 0.0

# ============ DNS 隧道检测 ============
class DNSTunnelDetector:
    """DNS 隧道检测"""
    
    def __init__(self, max_queries_per_minute=50):
        self.max_queries_per_minute = max_queries_per_minute
        self.query_counts = defaultdict(deque)
        self.lock = threading.Lock()
    
    def record_query(self, domain: str, client_ip: str):
        """记录 DNS 查询"""
        with self.lock:
            key = f"{client_ip}:{domain.split('.')[0]}"
            now = time.time()
            
            # 添加查询记录
            self.query_counts[key].append(now)
            
            # 清理 1 分钟前的记录
            one_minute_ago = now - 60
            while self.query_counts[key] and self.query_counts[key][0] < one_minute_ago:
                self.query_counts[key].popleft()
    
    def detect_tunnel(self, domain: str, client_ip: str) -> Dict:
        """检测 DNS 隧道"""
        with self.lock:
            key = f"{client_ip}:{domain.split('.')[0]}"
            count = len(self.query_counts[key])
            
            # 检查查询频率
            is_tunnel = count > self.max_queries_per_minute
            
            # 检查子域名层级（隧道通常使用多层子域名）
            subdomain_depth = len(domain.split('.')) - 2
            if subdomain_depth > 5:
                is_tunnel = True
            
            # 检查域名字符（隧道可能使用 Base64 等）
            if self._is_base64_like(domain):
                is_tunnel = True
            
            return {
                "tunnel": is_tunnel,
                "query_count": count,
                "max_allowed": self.max_queries_per_minute,
                "subdomain_depth": subdomain_depth,
            }
    
    def _is_base64_like(self, domain: str) -> bool:
        """检查是否像 Base64 编码"""
        # 移除点和数字
        s = ''.join(c for c in domain if c.isalpha())
        
        # Base64 字符集
        base64_chars = set('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/')
        
        # 检查是否全部是 Base64 字符
        return all(c in base64_chars for c in s) and len(s) > 20

# ============ 决策引擎 ============
class DNSDecisionEngine:
    """DNS 决策引擎"""
    
    def __init__(self):
        self.classifier = DomainClassifier()
        self.threat_intel = ThreatIntel()
        self.dga_detector = DGADetector(threshold=CONFIG["dga_detection"]["threshold"])
        self.tunnel_detector = DNSTunnelDetector(max_queries_per_minute=CONFIG["dns_tunnel_detection"]["max_queries_per_minute"])
    
    def decide(self, query: Dict) -> Dict:
        """决策 DNS 查询"""
        domain = query["domain"]
        client_ip = query["client_ip"]
        
        # 记录查询
        self.tunnel_detector.record_query(domain, client_ip)
        
        decision = {
            "domain": domain,
            "client_ip": client_ip,
            "action": "ALLOW",
            "risk_level": "低",
            "reasons": [],
            "details": {}
        }
        
        # 1. 域名分类
        classification = self.classifier.classify(domain)
        decision["details"]["classification"] = classification
        
        if classification == "BLACKLIST":
            decision["action"] = "BLOCK"
            decision["risk_level"] = "高"
            decision["reasons"].append("域名在黑名单中")
            return decision
        
        if classification == "WHITELIST":
            decision["action"] = "ALLOW"
            decision["risk_level"] = "低"
            decision["reasons"].append("域名在白名单中")
            return decision
        
        # 2. 威胁情报检查
        if CONFIG["threat_intel"]["enabled"]:
            threat_info = self.threat_intel.check(domain)
            decision["details"]["threat_intel"] = threat_info
            
            if threat_info.get("threat", False):
                decision["action"] = "BLOCK"
                decision["risk_level"] = "高"
                decision["reasons"].append(f"威胁情报检测: {threat_info.get('category', 'Unknown')}")
                return decision
            
            if threat_info.get("suspicious", False):
                decision["risk_level"] = "中"
                decision["reasons"].append("威胁情报标记为可疑")
        
        # 3. DGA 检测
        if CONFIG["dga_detection"]["enabled"]:
            dga_result = self.dga_detector.detect(domain)
            decision["details"]["dga"] = dga_result
            
            if dga_result["dga"]:
                decision["action"] = "REVIEW"
                decision["risk_level"] = "高"
                decision["reasons"].append(f"DGA 域名检测 (score: {dga_result['score']:.2f})")
        
        # 4. DNS 隧道检测
        if CONFIG["dns_tunnel_detection"]["enabled"]:
            tunnel_result = self.tunnel_detector.detect_tunnel(domain, client_ip)
            decision["details"]["tunnel"] = tunnel_result
            
            if tunnel_result["tunnel"]:
                decision["action"] = "BLOCK"
                decision["risk_level"] = "高"
                decision["reasons"].append(f"DNS 隧道检测 (查询数: {tunnel_result['query_count']})")
        
        return decision

# ============ 日志记录 ============
class DNSLogger:
    """DNS 日志记录"""
    
    def __init__(self, log_file):
        self.log_file = log_file
        import os
        os.makedirs(os.path.dirname(log_file), exist_ok=True)
    
    def log(self, query: Dict, decision: Dict):
        """记录 DNS 事件"""
        event = {
            "timestamp": datetime.now().isoformat(),
            "query": query,
            "decision": decision
        }
        
        with open(self.log_file, "a") as f:
            f.write(json.dumps(event) + "\n")

# ============ 事件处理 ============
class DNSSentinel:
    """DNS 监控核心"""
    
    def __init__(self):
        self.engine = DNSDecisionEngine()
        self.logger = DNSLogger(CONFIG["log_file"])
        self.running = False
    
    def handle_packet(self, packet):
        """处理 DNS 数据包"""
        try:
            # 解析 DNS 查询
            if packet.haslayer(DNSQR):
                query_name = packet[DNSQR].qname.decode('utf-8').rstrip('.')
                query_type = packet[DNSQR].qtype
                
                client_ip = packet[IP].src
                
                query = {
                    "domain": query_name,
                    "type": query_type,
                    "client_ip": client_ip,
                    "server_ip": packet[IP].dst
                }
                
                # 决策
                decision = self.engine.decide(query)
                
                # 显示
                self._display_event(query, decision)
                
                # 记录
                self.logger.log(query, decision)
        
        except Exception as e:
            pass  # 静默失败
    
    def _display_event(self, query: Dict, decision: Dict):
        """显示事件"""
        domain = query["domain"]
        action = decision["action"]
        risk_level = decision["risk_level"]
        reasons = decision["reasons"]
        
        # 颜色
        colors = {
            "ALLOW": Colors.GREEN,
            "BLOCK": Colors.RED,
            "REVIEW": Colors.YELLOW
        }
        color = colors.get(action, Colors.WHITE)
        
        risk_colors = {
            "高": Colors.RED,
            "中": Colors.YELLOW,
            "低": Colors.GREEN
        }
        risk_color = risk_colors.get(risk_level, Colors.WHITE)
        
        # 图标
        icons = {
            "ALLOW": "✓",
            "BLOCK": "🚫",
            "REVIEW": "⚠️"
        }
        icon = icons.get(action, "?")
        
        # 输出
        print_colored(f"\n[{datetime.now().strftime('%H:%M:%S')}] {icon} DNS 查询", color)
        print(f"   域名: {domain}")
        print(f"   客户端: {query['client_ip']}")
        print_colored(f"   决策: {action} ({risk_level}风险)", risk_color)
        
        if reasons:
            print_colored(f"   原因: {', '.join(reasons)}", Colors.YELLOW)
    
    def start(self):
        """开始监控"""
        if not SCAPY_AVAILABLE:
            self._run_mock()
            return
        
        print_colored("🔍 DNS 监控已启动", Colors.GREEN)
        print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
        print()
        
        # 开始抓包
        sniff(
            filter=f"udp port {CONFIG['port']} or tcp port {CONFIG['port']}",
            prn=self.handle_packet,
            store=False,
            iface=CONFIG["interface"]
        )
    
    def _run_mock(self):
        """模拟模式"""
        print_colored("🔍 运行在模拟模式", Colors.YELLOW)
        print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
        print()
        
        import random
        
        mock_domains = [
            ('google.com', 'safe'),
            ('malicious-domain.com', 'blacklist'),
            ('abc123def456ghi.com', 'dga'),
            ('x123.malicious.bad', 'blacklist'),
            ('a1b2c3d4.e5f6g7h8.i9j0k1l2.mno.com', 'tunnel'),
        ]
        
        try:
            while True:
                domain, reason = random.choice(mock_domains)
                
                query = {
                    "domain": domain,
                    "type": 1,
                    "client_ip": f"192.168.1.{random.randint(1, 255)}",
                    "server_ip": "8.8.8.8"
                }
                
                decision = self.engine.decide(query)
                self._display_event(query, decision)
                self.logger.log(query, decision)
                
                time.sleep(random.uniform(2, 5))
        
        except KeyboardInterrupt:
            print_colored("\n\n监控已停止", Colors.YELLOW)

# ============ 主程序 ============
def main():
    import os
    
    print_header()
    
    # 检查权限
    if os.getuid() != 0:
        print_colored("❌ 错误: 需要 root 权限运行", Colors.RED)
        print_colored("   请使用: sudo python3 sentinel_dns.py", Colors.YELLOW)
        return
    
    # 启动监控
    sentinel = DNSSentinel()
    sentinel.start()

if __name__ == "__main__":
    main()
