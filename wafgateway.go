// Sentinel-AI WAF 网关
// 接收来自 DNS 劫持的流量，进行深度检测和治理

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ==================== 配置 ====================
type Config struct {
	ListenAddr   string            `json:"listen_addr"`
	DNSMapping   map[string]string `json:"dns_mapping"`
	OllamaURL    string            `json:"ollama_url"`
	OllamaModel  string            `json:"ollama_model"`
	LogFile      string            `json:"log_file"`
}

var config = Config{
	ListenAddr: "10.0.0.1:8080",
	DNSMapping: map[string]string{
		"api.example.com":  "1.2.3.4:80",
		"db.example.com":   "5.6.7.8:3306",
		"auth.example.com": "9.10.11.12:443",
	},
	OllamaURL:   "http://localhost:11434",
	OllamaModel: "qwen2.5:7b",
}

// ==================== 请求结构 ====================
type Request struct {
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Host      string              `json:"host"`
	Headers   map[string]string   `json:"headers"`
	Body      string              `json:"body"`
	ClientIP  string              `json:"client_ip"`
	Timestamp time.Time           `json:"timestamp"`
}

// ==================== 风险评估 ====================
type RiskAssessment struct {
	Score    int      `json:"score"`    // 0-100
	Level    string   `json:"level"`    // LOW, MEDIUM, HIGH, CRITICAL
	Reasons  []string `json:"reasons"`
	Decision string   `json:"decision"` // ALLOW, REVIEW, BLOCK
}

// ==================== WAF 网关 ====================
type WAFGateway struct {
	config       Config
	proxy        *httputil.ReverseProxy
	policyEngine *PolicyEngine
	llmAnalyzer  *LLMAnalyzer
	approvalChan chan *Request
}

func NewWAFGateway(config Config) *WAFGateway {
	gw := &WAFGateway{
		config:       config,
		policyEngine: NewPolicyEngine(),
		llmAnalyzer:  NewLLMAnalyzer(config.OllamaURL, config.OllamaModel),
		approvalChan: make(chan *Request, 100),
	}

	// 启动审批处理器
	go gw.approvalProcessor()

	return gw
}

// 策略引擎
type PolicyEngine struct{}

func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{}
}

func (e *PolicyEngine) AssessRisk(req *Request) *RiskAssessment {
	assessment := &RiskAssessment{
		Score:   0,
		Level:   "LOW",
		Reasons: []string{},
	}

	// 1. 方法检查
	dangerousMethods := []string{"DELETE", "PUT", "PATCH"}
	for _, method := range dangerousMethods {
		if req.Method == method {
			assessment.Score += 30
			assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("危险方法: %s", method))
		}
	}

	// 2. 路径检查
	dangerousPaths := []string{"/delete", "/remove", "/drop", "/destroy", "/clear"}
	for _, path := range dangerousPaths {
		if strings.Contains(strings.ToLower(req.URL), path) {
			assessment.Score += 40
			assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("危险路径: %s", path))
		}
	}

	// 3. 敏感操作
	if strings.Contains(strings.ToLower(req.Body), "delete") ||
		strings.Contains(strings.ToLower(req.Body), "drop") ||
		strings.Contains(strings.ToLower(req.Body), "truncate") {
		assessment.Score += 30
		assessment.Reasons = append(assessment.Reasons, "检测到危险操作关键词")
	}

	// 4. 生产环境操作
	if strings.Contains(req.Host, "prod") ||
		strings.Contains(req.Host, "production") {
		assessment.Score += 20
		assessment.Reasons = append(assessment.Reasons, "生产环境操作")
	}

	// 5. 计算风险等级
	if assessment.Score >= 90 {
		assessment.Level = "CRITICAL"
		assessment.Decision = "BLOCK"
	} else if assessment.Score >= 70 {
		assessment.Level = "HIGH"
		assessment.Decision = "REVIEW"
	} else if assessment.Score >= 30 {
		assessment.Level = "MEDIUM"
		assessment.Decision = "ALLOW"
	} else {
		assessment.Level = "LOW"
		assessment.Decision = "ALLOW"
	}

	return assessment
}

// LLM 分析器
type LLMAnalyzer struct {
	baseURL string
	model   string
}

func NewLLMAnalyzer(baseURL, model string) *LLMAnalyzer {
	return &LLMAnalyzer{
		baseURL: baseURL,
		model:   model,
	}
}

func (a *LLMAnalyzer) Analyze(req *Request) (string, error) {
	prompt := fmt.Sprintf(`你是一个安全分析专家。请分析以下 HTTP 请求的风险：

方法: %s
路径: %s
Host: %s
请求体: %s

请简洁回答（50字以内）：
1. 这个操作的意图是什么？
2. 可能造成什么影响？
3. 风险等级（低/中/高）？

只返回分析结果。`, req.Method, req.URL, req.Host, req.Body)

	// 调用 Ollama API
	// TODO: 实现 Ollama API 调用
	return "意图分析: 正常操作", nil
}

// 审批处理器
func (gw *WAFGateway) approvalProcessor() {
	for req := range gw.approvalChan {
		color.Yellow("\n╔════════════════════════════════════════════╗")
		color.Yellow("║              🚨 需要人工审批                  ║")
		color.Yellow("╚════════════════════════════════════════════╝")
		fmt.Printf("\n  方法: %s\n", req.Method)
		fmt.Printf("  URL: %s\n", req.URL)
		fmt.Printf("  Host: %s\n", req.Host)
		fmt.Printf("  客户端: %s\n", req.ClientIP)

		color.Yellow("\n是否批准此操作? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			color.Green("\n  ✓ 操作已批准")
			// TODO: 通知等待的请求
		} else {
			color.Red("\n  ✗ 操作已拒绝")
		}
	}
}

// 解析请求
func ParseRequest(r *http.Request) *Request {
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return &Request{
		Method:    r.Method,
		URL:       r.URL.String(),
		Host:      r.Host,
		Headers:   headersToMap(r.Header),
		Body:      string(bodyBytes),
		ClientIP:  getClientIP(r),
		Timestamp: time.Now(),
	}
}

func headersToMap(headers http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		result[k] = strings.Join(v, ", ")
	}
	return result
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// 处理 HTTP 请求
func (gw *WAFGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 打印请求信息
	color.Cyan("\n[%s] 收到请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", r.Method)
	fmt.Printf("  Host: %s\n", r.Host)
	fmt.Printf("  URL: %s\n", r.URL.String())
	fmt.Printf("  客户端: %s\n", getClientIP(r))

	// 解析请求
	req := ParseRequest(r)

	// 1. 获取真实域名
	realHost := req.Host
	realAddr := gw.config.DNSMapping[realHost]

	// 如果不在映射表，可能是直接 IP 访问
	if realAddr == "" {
		color.Yellow("  ⚠️  域名不在映射表，可能是直接 IP 访问")
		// 尝试解析 Host
		host, port, _ := net.SplitHostPort(req.Host)
		if port == "" {
			port = "80"
		}
		realAddr = net.JoinHostPort(host, port)
	}

	fmt.Printf("  真实后端: %s\n", realAddr)

	// 2. 风险评估
	assessment := gw.policyEngine.AssessRisk(req)

	// 显示风险评估
	riskColors := map[string]string{
		"LOW":       "\033[92m",
		"MEDIUM":    "\033[93m",
		"HIGH":      "\033[91m",
		"CRITICAL":  "\033[95m",
	}
	riskIcons := map[string]string{
		"LOW":       "🟢",
		"MEDIUM":    "🟡",
		"HIGH":      "🔴",
		"CRITICAL":  "🚨",
	}

	riskColor := riskColors[assessment.Level]
	riskIcon := riskIcons[assessment.Level]

	fmt.Printf("  风险等级: %s%s %s\033[0m (%d分)\n", riskColor, riskIcon, assessment.Level, assessment.Score)
	for _, reason := range assessment.Reasons {
		fmt.Printf("    - %s\n", reason)
	}

	// 3. 决策执行
	if assessment.Decision == "BLOCK" {
		// 立即阻止
		color.Red("\n  ✗ 请求已阻止")

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Sentinel-Blocked", "true")
		w.Header().Set("X-Sentinel-Reason", strings.Join(assessment.Reasons, "; "))

		response := map[string]interface{}{
			"error":        "Request blocked by Sentinel-AI WAF",
			"reason":       assessment.Reasons,
			"risk_score":   assessment.Score,
			"risk_level":   assessment.Level,
			"request_id":   generateRequestID(),
			"timestamp":    time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)

		// 记录审计日志
		gw.logAudit(req, assessment, "BLOCK", 403)
		return
	}

	if assessment.Decision == "REVIEW" {
		// 需要审批
		color.Yellow("\n  ⚠️  需要人工审批")

		// 发送到审批队列
		gw.approvalChan <- req

		// 暂时返回等待
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Sentinel-Pending", "true")

		response := map[string]interface{}{
			"message": "Request pending approval",
			"request_id": generateRequestID(),
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)

		gw.logAudit(req, assessment, "REVIEW", 202)
		return
	}

	// 自动放行
	color.Green("\n  ✓ 自动放行")

	// 代理到真实后端
	target, err := url.Parse("http://" + realAddr)
	if err != nil {
		color.Red("  解析后端地址失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义 Director，添加安全头
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Sentinel-Protected", "true")
		req.Header.Set("X-Sentinel-Risk-Level", assessment.Level)
	}

	proxy.ServeHTTP(w, r)

	// 记录审计日志
	duration := time.Since(startTime)
	gw.logAudit(req, assessment, "ALLOW", 200)
	color.Cyan("  耗时: %dms", duration.Milliseconds())
}

// 记录审计日志
func (gw *WAFGateway) logAudit(req *Request, assessment *RiskAssessment, decision string, statusCode int) {
	audit := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339Nano),
		"request":     req,
		"assessment": assessment,
		"decision":    decision,
		"status_code": statusCode,
		"request_id":  generateRequestID(),
	}

	data, _ := json.Marshal(audit)
	// TODO: 写入日志文件
	log.Println(string(data))
}

// 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// ==================== 主程序 ====================
func main() {
	// 打印标题
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Sentinel-AI WAF 网关 v1.0                    ║")
	color.Cyan("║    接收 DNS 劫持流量，进行深度检测和治理              ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 显示配置
	color.Yellow("配置:")
	fmt.Printf("  监听地址: %s\n", config.ListenAddr)
	fmt.Printf("  Ollama: %s (%s)\n", config.OllamaURL, config.OllamaModel)
	fmt.Printf("  DNS 映射 (%d):\n", len(config.DNSMapping))
	for domain, addr := range config.DNSMapping {
		color.Red("    %s → %s", domain, addr)
	}
	fmt.Println()

	// 创建 WAF 网关
	gateway := NewWAFGateway(config)

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: gateway,
	}

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	color.Green("✓ WAF 网关启动成功")
	fmt.Printf("  监听地址: %s\n", config.ListenAddr)
	fmt.Println()
	color.Yellow("所有 DNS 劫持的流量都将经过此网关")
	fmt.Println()
	color.White("测试:")
	fmt.Printf("  curl -H 'Host: api.example.com' http://%s/api/test\n", config.ListenAddr)
	fmt.Println()

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}
