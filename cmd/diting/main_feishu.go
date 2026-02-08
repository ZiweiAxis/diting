package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// 配置结构
type AppConfig struct {
	Proxy  ProxyConfig  `json:"proxy"`
	LLM    LLMConfig    `json:"llm"`
	Feishu FeishuConfig `json:"feishu"`
	Risk   RiskConfig   `json:"risk"`
	Audit  AuditConfig  `json:"audit"`
}

type ProxyConfig struct {
	Listen         string `json:"listen"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type LLMConfig struct {
	Provider    string  `json:"provider"`
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type FeishuConfig struct {
	Enabled               bool   `json:"enabled"`
	AppID                 string `json:"app_id"`
	AppSecret             string `json:"app_secret"`
	ApprovalUserID        string `json:"approval_user_id"`
	ApprovalTimeoutMinutes int    `json:"approval_timeout_minutes"`
	UseInteractiveCard    bool   `json:"use_interactive_card"`
	UseMessageReply       bool   `json:"use_message_reply"`
	PollIntervalSeconds   int    `json:"poll_interval_seconds"`
}

type RiskConfig struct {
	DangerousMethods   []string `json:"dangerous_methods"`
	DangerousPaths     []string `json:"dangerous_paths"`
	AutoApproveMethods []string `json:"auto_approve_methods"`
	SafeDomains        []string `json:"safe_domains"`
}

type AuditConfig struct {
	LogFile string `json:"log_file"`
	Enabled bool   `json:"enabled"`
}

// 审计日志
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

// 审批请求
type ApprovalRequest struct {
	RequestID      string    `json:"request_id"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	Host           string    `json:"host"`
	RiskLevel      string    `json:"risk_level"`
	IntentAnalysis string    `json:"intent_analysis"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status"` // pending/approved/rejected/timeout
	MessageID      string    `json:"message_id"`
}

// 全局变量
var (
	config           AppConfig
	approvalRequests = sync.Map{}
	feishuToken      string
	feishuTokenMutex sync.RWMutex
)

func main() {
	// 加载配置
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 打印启动信息
	printBanner()

	// 创建日志目录
	os.MkdirAll("logs", 0755)

	// 启动 HTTP 服务器
	server := &http.Server{
		Addr: config.Proxy.Listen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleHTTPS(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
	}

	color.Green("✓ 代理服务器启动成功")
	color.White("  监听地址: http://localhost%s", config.Proxy.Listen)
	color.White("  支持协议: HTTP + HTTPS (CONNECT)")
	fmt.Println()
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}

func printBanner() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v0.3.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书审批集成          ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  LLM: %s", config.LLM.Model)
	if config.Feishu.Enabled {
		color.White("  飞书: 消息回复模式")
		color.White("  审批人: %s", config.Feishu.ApprovalUserID)
	}
	fmt.Println()
}

func loadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

// HTTP 代理处理
func handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	color.Cyan("\n[%s] 收到 HTTP 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", color.YellowString(r.Method))
	fmt.Printf("  URL: %s\n", color.WhiteString(r.URL.String()))

	// 读取请求体
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 风险评估
	riskLevel := assessRisk(r.Method, r.URL.Path, string(bodyBytes))
	fmt.Printf("  风险等级: %s\n", colorizeRisk(riskLevel))

	// 创建审计日志
	audit := AuditLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.URL.Host,
		Path:      r.URL.Path,
		Body:      string(bodyBytes),
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
		intentAnalysis = analyzeIntent(r.Method, r.URL.Path, string(bodyBytes))
		fmt.Printf("\n  🤖 意图分析:\n")
		color.Cyan("  %s", intentAnalysis)
		fmt.Println()

		// 飞书审批
		if config.Feishu.Enabled {
			decision = requestFeishuApproval(r.Method, r.URL.String(), r.URL.Host, riskLevel, intentAnalysis)
		} else {
			decision = "DENY"
			color.Red("  决策: 自动拒绝（飞书未启用）")
		}
	}

	audit.IntentAnalysis = intentAnalysis
	audit.Decision = decision

	// 执行决策
	if decision == "ALLOW" {
		color.Green("\n  ✓ 请求已放行")
		// 转发请求
		proxyHTTPRequest(w, r, &audit)
	} else {
		color.Red("\n  ✗ 请求已拒绝")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":  "请求被 Diting 拒绝",
			"reason": intentAnalysis,
		})
		audit.ResponseCode = 403
		audit.Approver = "DENIED"
	}

	audit.Duration = time.Since(startTime).Milliseconds()
	fmt.Printf("  耗时: %dms\n", audit.Duration)
	saveAuditLog(audit)
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// HTTPS 代理处理
func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	color.Cyan("\n[%s] 收到 HTTPS 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", color.YellowString(r.Method))
	fmt.Printf("  目标: %s\n", color.WhiteString(r.Host))

	// 风险评估
	riskLevel := assessRiskHTTPS(r.Host)
	fmt.Printf("  风险等级: %s\n", colorizeRisk(riskLevel))

	audit := AuditLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.Host,
		Path:      "/",
		RiskLevel: riskLevel,
	}

	var decision string
	var intentAnalysis string

	if riskLevel == "低" {
		decision = "ALLOW"
		color.Green("  决策: 自动放行")
	} else {
		intentAnalysis = fmt.Sprintf("HTTPS 连接到未知域名: %s", r.Host)
		if config.Feishu.Enabled {
			decision = requestFeishuApproval("CONNECT", r.Host, r.Host, riskLevel, intentAnalysis)
		} else {
			decision = "DENY"
		}
	}

	audit.IntentAnalysis = intentAnalysis
	audit.Decision = decision

	if decision == "ALLOW" {
		color.Green("\n  ✓ 连接已放行")
		proxyHTTPSConnection(w, r, &audit)
	} else {
		color.Red("\n  ✗ 连接已拒绝")
		w.WriteHeader(http.StatusForbidden)
		audit.ResponseCode = 403
	}

	audit.Duration = time.Since(startTime).Milliseconds()
	fmt.Printf("  耗时: %dms\n", audit.Duration)
	saveAuditLog(audit)
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// 风险评估
func assessRisk(method, path, body string) string {
	// 检查是否为安全方法
	for _, m := range config.Risk.AutoApproveMethods {
		if method == m {
			return "低"
		}
	}

	// 检查危险方法
	for _, m := range config.Risk.DangerousMethods {
		if method == m {
			return "高"
		}
	}

	// 检查危险路径
	for _, p := range config.Risk.DangerousPaths {
		if strings.Contains(strings.ToLower(path), p) {
			return "高"
		}
	}

	return "中"
}

func assessRiskHTTPS(host string) string {
	// 检查安全域名
	for _, domain := range config.Risk.SafeDomains {
		if strings.Contains(host, domain) {
			return "低"
		}
	}
	return "中"
}

// 意图分析
func analyzeIntent(method, path, body string) string {
	if !config.Feishu.Enabled || config.LLM.APIKey == "" {
		return fmt.Sprintf("规则引擎: %s %s 操作需要审批", method, path)
	}

	prompt := fmt.Sprintf(`分析以下 API 操作的意图和风险：
方法: %s
路径: %s
请求体: %s

请简要说明（50字以内）：
1. 操作意图
2. 潜在影响
3. 是否建议审批`, method, path, body)

	req := map[string]interface{}{
		"model": config.LLM.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": config.LLM.MaxTokens,
	}

	reqBody, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", config.LLM.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", config.LLM.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Sprintf("LLM 分析失败，降级到规则引擎: %s %s 操作需要审批", method, path)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if text, ok := content[0].(map[string]interface{})["text"].(string); ok {
			return text
		}
	}

	return fmt.Sprintf("规则引擎: %s %s 操作需要审批", method, path)
}

// 飞书审批
func requestFeishuApproval(method, path, host, riskLevel, intentAnalysis string) string {
	requestID := fmt.Sprintf("req_%d", time.Now().Unix())

	req := ApprovalRequest{
		RequestID:      requestID,
		Method:         method,
		Path:           path,
		Host:           host,
		RiskLevel:      riskLevel,
		IntentAnalysis: intentAnalysis,
		Timestamp:      time.Now(),
		Status:         "pending",
	}

	approvalRequests.Store(requestID, &req)

	// 发送飞书消息
	message := fmt.Sprintf(`🚨 Diting 高风险操作审批

操作: %s %s
风险等级: %s
意图分析: %s

请回复：
✅ "批准" 或 "approve" 或 "y" 来批准
❌ "拒绝" 或 "reject" 或 "n" 来拒绝

⏱️ %d分钟内未响应将自动拒绝
请求ID: %s`, method, path, riskLevel, intentAnalysis, config.Feishu.ApprovalTimeoutMinutes, requestID)

	messageID, err := sendFeishuMessage(config.Feishu.ApprovalUserID, message)
	if err != nil {
		color.Red("  ✗ 发送飞书消息失败: %v", err)
		return "DENY"
	}

	req.MessageID = messageID
	approvalRequests.Store(requestID, &req)

	color.Yellow("  ⏳ 等待飞书审批...")

	// 等待审批
	timeout := time.Duration(config.Feishu.ApprovalTimeoutMinutes) * time.Minute
	decision := waitForApproval(requestID, timeout)

	if decision == "ALLOW" {
		color.Green("  ✓ 审批通过")
	} else if decision == "DENY" {
		color.Red("  ✗ 审批拒绝")
	} else {
		color.Red("  ✗ 审批超时，自动拒绝")
	}

	return decision
}

func waitForApproval(requestID string, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(time.Duration(config.Feishu.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if val, ok := approvalRequests.Load(requestID); ok {
				req := val.(*ApprovalRequest)
				if req.Status == "approved" {
					return "ALLOW"
				} else if req.Status == "rejected" {
					return "DENY"
				}
			}

			// 轮询飞书消息
			checkFeishuMessages(requestID)

			if time.Now().After(deadline) {
				if val, ok := approvalRequests.Load(requestID); ok {
					req := val.(*ApprovalRequest)
					req.Status = "timeout"
					approvalRequests.Store(requestID, req)
				}
				sendFeishuMessage(config.Feishu.ApprovalUserID, fmt.Sprintf("⏱️ 审批超时，请求 %s 已自动拒绝", requestID))
				return "DENY"
			}
		}
	}
}

// 飞书 API
func getFeishuToken() (string, error) {
	feishuTokenMutex.RLock()
	if feishuToken != "" {
		feishuTokenMutex.RUnlock()
		return feishuToken, nil
	}
	feishuTokenMutex.RUnlock()

	feishuTokenMutex.Lock()
	defer feishuTokenMutex.Unlock()

	reqBody, _ := json.Marshal(map[string]string{
		"app_id":     config.Feishu.AppID,
		"app_secret": config.Feishu.AppSecret,
	})

	resp, err := http.Post("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if token, ok := result["tenant_access_token"].(string); ok {
		feishuToken = token
		return token, nil
	}

	return "", fmt.Errorf("获取 token 失败")
}

func sendFeishuMessage(userID, content string) (string, error) {
	token, err := getFeishuToken()
	if err != nil {
		return "", err
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"receive_id": userID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":"%s"}`, strings.ReplaceAll(content, "\n", "\\n")),
	})

	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=user_id",
		bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if data, ok := result["data"].(map[string]interface{}); ok {
		if msgID, ok := data["message_id"].(string); ok {
			return msgID, nil
		}
	}

	return "", fmt.Errorf("发送消息失败")
}

func checkFeishuMessages(requestID string) {
	// 简化版：检查最近的消息
	// 实际应该获取与机器人的对话消息列表
	// 这里省略复杂的消息轮询逻辑
}

// 代理转发
func proxyHTTPRequest(w http.ResponseWriter, r *http.Request, audit *AuditLog) {
	client := &http.Client{Timeout: time.Duration(config.Proxy.TimeoutSeconds) * time.Second}

	proxyReq, _ := http.NewRequest(r.Method, r.URL.String(), r.Body)
	for k, v := range r.Header {
		proxyReq.Header[k] = v
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		audit.ResponseCode = 502
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	audit.ResponseCode = resp.StatusCode
}

func proxyHTTPSConnection(w http.ResponseWriter, r *http.Request, audit *AuditLog) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		audit.ResponseCode = 500
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		audit.ResponseCode = 503
		return
	}
	defer clientConn.Close()

	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		audit.ResponseCode = 502
		return
	}
	defer targetConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
	}()

	wg.Wait()
	audit.ResponseCode = 200
}

// 审计日志
func saveAuditLog(audit AuditLog) {
	if !config.Audit.Enabled {
		return
	}

	f, err := os.OpenFile(config.Audit.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data, _ := json.Marshal(audit)
	f.Write(data)
	f.WriteString("\n")
}

func colorizeRisk(level string) string {
	switch level {
	case "低":
		return color.GreenString("%s 🟢", level)
	case "中":
		return color.YellowString("%s 🟡", level)
	case "高":
		return color.RedString("%s 🔴", level)
	default:
		return level
	}
}
