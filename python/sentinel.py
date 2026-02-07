"""
Sentinel-AI MVP - Python 版本
企业级智能体零信任治理平台

无需编译，直接运行！
"""

import json
import time
import requests
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse
import threading
import os

# ============ 配置 ============
CONFIG = {
    "proxy_listen": ("0.0.0.0", 8080),
    "target_url": "http://httpbin.org",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_model": "qwen2.5:7b",
    "dangerous_methods": ["DELETE", "PUT", "PATCH", "POST"],
    "dangerous_paths": ["/delete", "/remove", "/drop", "/destroy", "/clear"],
    "auto_approve_methods": ["GET", "HEAD", "OPTIONS"],
}

# ============ 颜色输出 ============
class Colors:
    CYAN = '\033[96m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    RED = '\033[91m'
    WHITE = '\033[97m'
    RESET = '\033[0m'
    BOLD = '\033[1m'

def print_colored(text, color):
    print(f"{color}{text}{Colors.RESET}")

def print_header():
    print_colored("╔════════════════════════════════════════════════════════╗", Colors.CYAN)
    print_colored("║         Sentinel-AI 治理网关 MVP v0.1                 ║", Colors.CYAN)
    print_colored("║    企业级智能体零信任治理平台 - Python 版本           ║", Colors.CYAN)
    print_colored("╚════════════════════════════════════════════════════════╝", Colors.CYAN)
    print()

# ============ Ollama 检查 ============
def check_ollama():
    try:
        resp = requests.get(f"{CONFIG['ollama_endpoint']}/api/tags", timeout=2)
        return resp.status_code == 200
    except:
        return False

# ============ 风险评估 ============
def assess_risk(method, path, body):
    # 自动放行的方法
    if method in CONFIG["auto_approve_methods"]:
        return "低"
    
    # 危险方法
    if method in CONFIG["dangerous_methods"]:
        return "高"
    
    # 危险路径
    for dangerous_path in CONFIG["dangerous_paths"]:
        if dangerous_path in path.lower():
            return "高"
    
    # 检查请求体中的危险关键词
    dangerous_keywords = ["delete", "drop", "truncate", "remove", "destroy"]
    body_lower = body.lower()
    for keyword in dangerous_keywords:
        if keyword in body_lower:
            return "中"
    
    return "中"

def colorize_risk(level):
    if level == "高":
        return f"{Colors.RED}高 🔴{Colors.RESET}"
    elif level == "中":
        return f"{Colors.YELLOW}中 🟡{Colors.RESET}"
    else:
        return f"{Colors.GREEN}低 🟢{Colors.RESET}"

# ============ LLM 意图分析 ============
def analyze_intent(method, path, body):
    prompt = f"""你是一个企业安全分析专家。请分析以下 API 请求的意图和风险：

请求方法: {method}
请求路径: {path}
请求体: {body}

请简洁回答（50字以内）：
1. 这个操作的意图是什么？
2. 可能造成什么影响？
3. 是否应该批准？

只返回分析结果，不要解释。"""

    # 尝试调用 Ollama
    if check_ollama():
        try:
            resp = requests.post(
                f"{CONFIG['ollama_endpoint']}/api/generate",
                json={
                    "model": CONFIG["ollama_model"],
                    "prompt": prompt,
                    "stream": False
                },
                timeout=10
            )
            if resp.status_code == 200:
                result = resp.json().get("response", "").strip()
                if result:
                    return result
        except Exception as e:
            print_colored(f"  LLM 调用失败: {e}", Colors.YELLOW)
    
    # 降级到规则引擎
    if method == "DELETE":
        return "意图: 删除数据。影响: 数据不可恢复。建议: 需要审批。"
    if "production" in path:
        return "意图: 操作生产环境。影响: 可能影响业务。建议: 需要审批。"
    return "意图: 修改数据。影响: 中等风险。建议: 建议审批。"

# ============ 人工审批 ============
def human_approval(method, path, analysis):
    print_colored("\n╔════════════════════════════════════════════════════════╗", Colors.YELLOW)
    print_colored("║                  🚨 需要人工审批                       ║", Colors.YELLOW)
    print_colored("╚════════════════════════════════════════════════════════╝", Colors.YELLOW)
    print(f"\n  请求: {method} {path}")
    print(f"  分析: {analysis}\n")
    
    response = input(f"{Colors.YELLOW}  是否批准此操作? (y/n): {Colors.RESET}").strip().lower()
    return "ALLOW" if response in ["y", "yes"] else "DENY"

# ============ 审计日志 ============
def save_audit_log(audit_data):
    os.makedirs("logs", exist_ok=True)
    with open("logs/audit.jsonl", "a", encoding="utf-8") as f:
        f.write(json.dumps(audit_data, ensure_ascii=False) + "\n")

# ============ HTTP 代理处理器 ============
class ProxyHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.handle_request()
    
    def do_POST(self):
        self.handle_request()
    
    def do_PUT(self):
        self.handle_request()
    
    def do_DELETE(self):
        self.handle_request()
    
    def do_PATCH(self):
        self.handle_request()
    
    def do_HEAD(self):
        self.handle_request()
    
    def do_OPTIONS(self):
        self.handle_request()
    
    def handle_request(self):
        start_time = time.time()
        
        # 打印请求信息
        print_colored(f"\n[{datetime.now().strftime('%H:%M:%S')}] 收到请求", Colors.CYAN)
        print(f"  方法: {Colors.YELLOW}{self.command}{Colors.RESET}")
        print(f"  路径: {Colors.WHITE}{self.path}{Colors.RESET}")
        
        # 读取请求体
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length).decode('utf-8') if content_length > 0 else ""
        body_display = body[:200] + "..." if len(body) > 200 else body
        
        # 风险评估
        risk_level = assess_risk(self.command, self.path, body)
        print(f"  风险等级: {colorize_risk(risk_level)}")
        
        # 创建审计日志
        audit = {
            "timestamp": datetime.now().isoformat(),
            "method": self.command,
            "path": self.path,
            "body": body_display,
            "risk_level": risk_level,
            "intent_analysis": "",
            "decision": "",
            "approver": "",
            "response_code": 0,
            "duration_ms": 0
        }
        
        # 决策逻辑
        decision = ""
        intent_analysis = ""
        
        if risk_level == "低":
            decision = "ALLOW"
            print_colored("  决策: 自动放行", Colors.GREEN)
        else:
            # LLM 意图分析
            intent_analysis = analyze_intent(self.command, self.path, body)
            print(f"\n  🤖 LLM 意图分析:")
            print_colored(f"  {intent_analysis}", Colors.CYAN)
            
            # 人工审批
            decision = human_approval(self.command, self.path, intent_analysis)
        
        audit["intent_analysis"] = intent_analysis
        audit["decision"] = decision
        
        # 执行决策
        if decision == "ALLOW":
            print_colored("\n  ✓ 请求已放行", Colors.GREEN)
            
            # 转发请求到真实后端
            try:
                target_url = CONFIG["target_url"] + self.path
                headers = dict(self.headers)
                headers.pop('Host', None)
                
                resp = requests.request(
                    method=self.command,
                    url=target_url,
                    headers=headers,
                    data=body if body else None,
                    timeout=30
                )
                
                # 返回响应
                self.send_response(resp.status_code)
                for key, value in resp.headers.items():
                    if key.lower() not in ['transfer-encoding', 'connection']:
                        self.send_header(key, value)
                self.end_headers()
                self.wfile.write(resp.content)
                
                audit["response_code"] = resp.status_code
            except Exception as e:
                print_colored(f"  转发失败: {e}", Colors.RED)
                self.send_error(502, f"Bad Gateway: {e}")
                audit["response_code"] = 502
        else:
            print_colored("\n  ✗ 请求已拒绝", Colors.RED)
            
            # 返回 403
            self.send_response(403)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            
            error_response = {
                "error": "操作被 Sentinel-AI 拒绝",
                "reason": intent_analysis,
                "policy": "需要管理员审批",
                "contact": "请联系安全管理员"
            }
            self.wfile.write(json.dumps(error_response, ensure_ascii=False).encode('utf-8'))
            
            audit["response_code"] = 403
            audit["approver"] = "DENIED"
        
        # 记录耗时
        duration = int((time.time() - start_time) * 1000)
        audit["duration_ms"] = duration
        print(f"  耗时: {duration}ms")
        
        # 保存审计日志
        save_audit_log(audit)
        
        print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
    
    def log_message(self, format, *args):
        # 禁用默认日志
        pass

# ============ 主程序 ============
def main():
    print_header()
    
    # 检查 Ollama
    if not check_ollama():
        print_colored("⚠️  警告: Ollama 未运行，将使用规则引擎模式", Colors.YELLOW)
        print_colored("   启动 Ollama: ollama serve", Colors.YELLOW)
        print_colored(f"   下载模型: ollama pull {CONFIG['ollama_model']}", Colors.YELLOW)
        print()
    
    # 启动服务器
    server = HTTPServer(CONFIG["proxy_listen"], ProxyHandler)
    
    print_colored("✓ 代理服务器启动成功", Colors.GREEN)
    print_colored(f"  监听地址: http://localhost:{CONFIG['proxy_listen'][1]}", Colors.WHITE)
    print_colored(f"  目标地址: {CONFIG['target_url']}", Colors.WHITE)
    print()
    print_colored("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", Colors.CYAN)
    print()
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print_colored("\n\n✓ 服务器已停止", Colors.GREEN)
        server.shutdown()

if __name__ == "__main__":
    main()
