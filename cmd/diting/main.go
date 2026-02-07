package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// 配置
type Config struct {
	ProxyListen        string
	OllamaEndpoint     string
	OllamaModel        string
	DangerousMethods   []string
	DangerousPaths     []string
	AutoApproveMethods []string
}

var config = Config{
	ProxyListen:        ":8080",
	OllamaEndpoint:     "http://localhost:11434",
	OllamaModel:        "qwen2.5:7b",
	DangerousMethods:   []string{"DELETE", "PUT", "PATCH", "POST"},
	DangerousPaths:     []string{"/delete", "/remove", "/drop", "/destroy", "/clear"},
	AutoApproveMethods: []string{"GET", "HEAD", "OPTIONS"},
}

// 审计日志结构
type AuditLog struct {
	Timestamp      time.Time `json:"timestamp"`
	Method         string    `json:"method"`
	Host           string    `json:"host"`
	Path           string    `json:"path"`
	Body           string    `json:"body"`
	RiskLevel      string    `json:"risk_level"`
	IntentAnalysis string    `json:"intent_analysis"`
	Decision       string    `json:"decision"`
	Approver       string    `json:"approver"`
	ResponseCode   int       `json:"response_code"`
	Duration       int64     `json:"duration_ms"`
}

// LLM 请求结构
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func main() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v0.2.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - HTTPS 代理支持        ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 检查 Ollama 是否可用
	if !checkOllama() {
		color.Yellow("⚠️  警告: Ollama 未运行，将使用规则引擎模式")
		color.Yellow("   启动 Ollama: ollama serve")
		color.Yellow("   下载模型: ollama pull %s", config.OllamaModel)
		fmt.Println()
	}

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr: config.ProxyListen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				// HTTPS 代理 (CONNECT 方法)
				handleHTTPS(w, r)
			} else {
				// HTTP 代理
				handleHTTP(w, r)
			}
		}),
	}

	color.Green("✓ 代理服务器启动成功")
	color.White("  监听地址: http://localhost%s", config.ProxyListen)
	color.White("  支持协议: HTTP + HTTPS (CONNECT)")
	fmt.Println()
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}

// 处理 HTTPS 请求 (CONNECT 方法)
func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 打印请求信息
	color.Cyan("\n[%s] 收到 HTTPS 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", color.YellowString(r.Method))
	fmt.Printf("  目标: %s\n", color.WhiteString(r.Host))

	// 风险评估 (基于目标域名)
	riskLevel := assessRiskHTTPS(r.Host)
	fmt.Printf("  风险等级: %s\n", colorizeRisk(riskLevel))

	// 创建审计日志
	audit := AuditLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.Host,
		Path:      "/",
		RiskLevel: riskLevel,
	}

	// 决策逻辑
	var decision string
	var intentAnalysis string

	if riskLevel == "低" {
		decision = "ALLOW"
		color.Green("  决策: 自动放行")
	} else {
		// LLM 意图分析
		intentAnalysis = analyzeIntentHTTPS(r.Host)
		fmt.Printf("\n  🤖 LLM 意图分析:\n")
		color.Cyan("  %s", intentAnalysis)
		fmt.Println()

		// 人工审批
		decision = humanApprovalHTTPS(r.Host, intentAnalysis)
	}

	audit.IntentAnalysis = intentAnalysis
	audit.Decision = decision

	// 执行决策
	if decision == "ALLOW" {
		color.Green("\n  ✓ 连接已放行")

		// 劫持连接
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}

		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer clientConn.Close()

		// 连接到目标服务器
		targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
		if err != nil {
			color.Red("  ✗ 连接目标失败: %v", err)
			audit.ResponseCode = 502
			saveAuditLog(audit)
			return
		}
		defer targetConn.Close()

		// 返回 200 Connection Established
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		// 双向转发数据
		go io.Copy(targetConn, clientConn)
		io.Copy(clientConn, targetConn)

		audit.ResponseCode = 200
	} else {
		color.Red("\n  ✗ 连接已拒绝")
		w.WriteHeader(http.StatusForbidden)
		response := map[string]interface{}{
			"error":   "连接被 Diting 拒绝",
			"reason":  intentAnalysis,
			"policy":  "需要管理员审批",
			"contact": "请联系安全管理员",
		}
		json.NewEncoder(w).Encode(response)
		audit.ResponseCode = 403
		audit.Approver = "DENIED"
	}

	// 记录耗时
	duration := time.Since(startTime).Milliseconds()
	audit.Duration = duration
	fmt.Printf("  耗时: %dms\n", duration)

	// 保存审计日志
	saveAuditLog(audit)

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// 处理 HTTP 请求
func handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 打印请求信息
	color.Cyan("\n[%s] 收到 HTTP 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", color.YellowString(r.Method))
	fmt.Printf("  URL: %s\n", color.WhiteString(r.URL.String()))

	// 读取请求体
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	bodyStr := string(bodyBytes)
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200] + "..."
	}

	// 风险评估
	riskLevel := assessRisk(r, bodyStr)
	fmt.Printf("  风险等级: %s\n", colorizeRisk(riskLevel))

	// 创建审计日志
	audit := AuditLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.Host,
		Path:      r.URL.Path,
		Body:      bodyStr,
		RiskLevel: riskLevel,
	}

	// 决策逻辑
	var decision string
	var intentAnalysis string

	if riskLevel == "低" {
		decision = "ALLOW"
		color.Green("  决策: 自动放行")
	} else {
		// 调用 LLM 分析意图
		intentAnalysis = analyzeIntent(r, bodyStr)
		fmt.Printf("\n  🤖 LLM 意图分析:\n")
		color.Cyan("  %s", intentAnalysis)
		fmt.Println()

		// 人工审批
		decision = humanApproval(r, intentAnalysis)
	}

	audit.IntentAnalysis = intentAnalysis
	audit.Decision = decision

	// 执行决策
	if decision == "ALLOW" {
		color.Green("\n  ✓ 请求已放行")

		// 转发请求
		proxyRequest(w, r)
		audit.ResponseCode = 200
	} else {
		color.Red("\n  ✗ 请求已拒绝")
		w.WriteHeader(http.StatusForbidden)
		response := map[string]interface{}{
			"error":   "操作被 Diting 拒绝",
			"reason":  intentAnalysis,
			"policy":  "需要管理员审批",
			"contact": "请联系安全管理员",
		}
		json.NewEncoder(w).Encode(response)
		audit.ResponseCode = 403
		audit.Approver = "DENIED"
	}

	// 记录耗时
	duration := time.Since(startTime).Milliseconds()
	audit.Duration = duration
	fmt.Printf("  耗时: %dms\n", duration)

	// 保存审计日志
	saveAuditLog(audit)

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// 转发 HTTP 请求
func proxyRequest(w http.ResponseWriter, r *http.Request) {
	// 创建新的请求
	targetURL := r.URL
	if targetURL.Scheme == "" {
		targetURL.Scheme = "http"
	}
	if targetURL.Host == "" {
		targetURL.Host = r.Host
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 复制请求头
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// 发送请求
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	io.Copy(w, resp.Body)
}

// HTTPS 风险评估
func assessRiskHTTPS(host string) string {
	hostLower := strings.ToLower(host)

	// 检查危险域名
	dangerousDomains := []string{"malware", "phishing", "hack", "exploit"}
	for _, domain := range dangerousDomains {
		if strings.Contains(hostLower, domain) {
			return "高"
		}
	}

	// 检查常见安全域名
	safeDomains := []string{"google.com", "github.com", "microsoft.com", "apple.com"}
	for _, domain := range safeDomains {
		if strings.Contains(hostLower, domain) {
			return "低"
		}
	}

	return "中"
}

// HTTP 风险评估
func assessRisk(r *http.Request, body string) string {
	// 自动放行的方法
	for _, method := range config.AutoApproveMethods {
		if r.Method == method {
			return "低"
		}
	}

	// 危险方法
	for _, method := range config.DangerousMethods {
		if r.Method == method {
			return "高"
		}
	}

	// 危险路径
	for _, path := range config.DangerousPaths {
		if strings.Contains(strings.ToLower(r.URL.Path), path) {
			return "高"
		}
	}

	// 检查请求体中的危险关键词
	dangerousKeywords := []string{"delete", "drop", "truncate", "remove", "destroy"}
	bodyLower := strings.ToLower(body)
	for _, keyword := range dangerousKeywords {
		if strings.Contains(bodyLower, keyword) {
			return "中"
		}
	}

	return "中"
}

// HTTPS 意图分析
func analyzeIntentHTTPS(host string) string {
	prompt := fmt.Sprintf(`你是一个企业安全分析专家。请分析以下 HTTPS 连接请求的意图和风险：

目标域名: %s

请简洁回答（50字以内）：
1. 这个域名的用途是什么？
2. 可能存在什么风险？
3. 是否应该批准？

只返回分析结果，不要解释。`, host)

	// 尝试调用 Ollama
	if checkOllama() {
		reqBody := OllamaRequest{
			Model:  config.OllamaModel,
			Prompt: prompt,
			Stream: false,
		}

		jsonData, _ := json.Marshal(reqBody)
		resp, err := http.Post(
			config.OllamaEndpoint+"/api/generate",
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err == nil && resp.StatusCode == 200 {
			var ollamaResp OllamaResponse
			json.NewDecoder(resp.Body).Decode(&ollamaResp)
			resp.Body.Close()
			if ollamaResp.Response != "" {
				return strings.TrimSpace(ollamaResp.Response)
			}
		}
	}

	// 降级到规则引擎
	if strings.Contains(host, "api") {
		return "意图: API 调用。影响: 可能修改数据。建议: 建议审批。"
	}
	return "意图: HTTPS 连接。影响: 未知。建议: 建议审批。"
}

// HTTP 意图分析
func analyzeIntent(r *http.Request, body string) string {
	prompt := fmt.Sprintf(`你是一个企业安全分析专家。请分析以下 API 请求的意图和风险：

请求方法: %s
请求路径: %s
请求体: %s

请简洁回答（50字以内）：
1. 这个操作的意图是什么？
2. 可能造成什么影响？
3. 是否应该批准？

只返回分析结果，不要解释。`, r.Method, r.URL.Path, body)

	// 尝试调用 Ollama
	if checkOllama() {
		reqBody := OllamaRequest{
			Model:  config.OllamaModel,
			Prompt: prompt,
			Stream: false,
		}

		jsonData, _ := json.Marshal(reqBody)
		resp, err := http.Post(
			config.OllamaEndpoint+"/api/generate",
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err == nil && resp.StatusCode == 200 {
			var ollamaResp OllamaResponse
			json.NewDecoder(resp.Body).Decode(&ollamaResp)
			resp.Body.Close()
			if ollamaResp.Response != "" {
				return strings.TrimSpace(ollamaResp.Response)
			}
		}
	}

	// 降级到规则引擎
	if r.Method == "DELETE" {
		return "意图: 删除数据。影响: 数据不可恢复。建议: 需要审批。"
	}
	if strings.Contains(r.URL.Path, "production") {
		return "意图: 操作生产环境。影响: 可能影响业务。建议: 需要审批。"
	}
	return "意图: 修改数据。影响: 中等风险。建议: 建议审批。"
}

// HTTPS 人工审批
func humanApprovalHTTPS(host string, analysis string) string {
	color.Yellow("\n╔════════════════════════════════════════════════════════╗")
	color.Yellow("║                  🚨 需要人工审批                       ║")
	color.Yellow("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  连接: HTTPS %s\n", host)
	fmt.Printf("  分析: %s\n\n", analysis)
	color.Yellow("  是否批准此连接? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		return "ALLOW"
	}
	return "DENY"
}

// HTTP 人工审批
func humanApproval(r *http.Request, analysis string) string {
	color.Yellow("\n╔════════════════════════════════════════════════════════╗")
	color.Yellow("║                  🚨 需要人工审批                       ║")
	color.Yellow("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("\n  请求: %s %s\n", r.Method, r.URL.Path)
	fmt.Printf("  分析: %s\n\n", analysis)
	color.Yellow("  是否批准此操作? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		return "ALLOW"
	}
	return "DENY"
}

func checkOllama() bool {
	resp, err := http.Get(config.OllamaEndpoint + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func saveAuditLog(audit AuditLog) {
	// 简单的文件日志
	logFile := "logs/audit.jsonl"
	os.MkdirAll("logs", 0755)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	jsonData, _ := json.Marshal(audit)
	f.Write(jsonData)
	f.WriteString("\n")
}

func colorizeRisk(level string) string {
	switch level {
	case "高":
		return color.RedString("高 🔴")
	case "中":
		return color.YellowString("中 🟡")
	default:
		return color.GreenString("低 🟢")
	}
}
