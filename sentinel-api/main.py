"""
Sentinel-AI 业务逻辑服务
使用 OpenAI API 进行意图分析
"""

from fastapi import FastAPI, Request, HTTPException
from pydantic import BaseModel
from openai import OpenAI
import time
import os
from typing import Optional, List
from datetime import datetime
import uvicorn
import json

# ==================== 配置 ====================
class Config:
    OPENAI_API_KEY: str = os.getenv("OPENAI_API_KEY", "")
    OPENAI_MODEL: str = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
    OPENAI_BASE_URL: Optional[str] = os.getenv("OPENAI_BASE_URL")
    ALLOWED_IPS: List[str] = ["127.0.0.1", "10.0.0.0/16"]
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
        
        # 4. 生产环境操作
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
1. 分析操作的意图（10字以内）
2. 评估可能的影响（20字以内）
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
        prompt = f"""请分析这个请求：

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
        self.pending_requests = {}
    
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

# ==================== 日志记录 ====================
import logging
logging.basicConfig(
    filename=config.LOG_FILE,
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@app.on_event("startup")
async def startup_event():
    logger.info("Sentinel-AI API starting...")
    os.makedirs("logs", exist_ok=True)

@app.post("/analyze", response_model=AnalyzeResponse)
async def analyze_request(req: AnalyzeRequest):
    """分析请求并返回决策"""
    logger.info(f"Analyzing request: {req.method} {req.uri} from {req.client_ip}")
    
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
    logger.info(json.dumps(log_entry, ensure_ascii=False))
    
    return response

@app.post("/approval")
async def handle_approval(approval: ApprovalRequest):
    """处理审批结果"""
    logger.info(f"Approval for {approval.request_id}: {approval.approved} by {approval.approver}")
    
    return approval_manager.approve(
        approval.request_id,
        approval.approved,
        approval.approver,
        approval.reason
    )

@app.get("/health")
async def health():
    """健康检查"""
    return {"status": "healthy", "service": "sentinel-ai-api", "version": "2.0.0"}

@app.get("/")
async def root():
    """根路径"""
    return {
        "service": "Sentinel-AI API",
        "version": "2.0.0",
        "endpoints": {
            "POST /analyze": "分析请求并返回决策",
            "POST /approval": "处理审批结果",
            "GET /health": "健康检查"
        }
    }

# ==================== 主程序 ====================
if __name__ == "__main__":
    import os
    os.makedirs("logs", exist_ok=True)
    
    # 检查 API Key
    if not config.OPENAI_API_KEY:
        print("WARNING: OPENAI_API_KEY not set!")
        print("Set it with: export OPENAI_API_KEY=sk-xxx")
        print("Or create .env file with: OPENAI_API_KEY=sk-xxx")
    
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8000,
        log_level="info",
        access_log=True
    )
