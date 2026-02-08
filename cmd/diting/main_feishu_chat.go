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
	Enabled                bool   `json:"enabled"`
	AppID                  string `json:"app_id"`
	AppSecret              string `json:"app_secret"`
	ApprovalUserID         string `json:"approval_user_id"`
	ApprovalTimeoutMinutes int    `json:"approval_timeout_minutes"`
	UseInteractiveCard     bool   `json:"use_interactive_card"`
	UseMessageReply        bool   `json:"use_message_reply"`
	PollIntervalSeconds    int    `json:"poll_interval_seconds"`
	EventPort              int    `json:"event_port"`
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
	Status         string    `json:"status"`
	MessageID      string    `json:"message_id"`
	ChatID         string    `json:"chat_id"`
}

// 全局变量
var (
	config           AppConfig
	approvalRequests = sync.Map{}
	feishuToken      string
	feishuTokenMutex sync.RWMutex
	userChatID       string
	userChatIDMutex  sync.RWMutex
)

func main() {
	// 加载配置
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置默认事件端口
	if config.Feishu.EventPort == 0 {
		config.Feishu.EventPort = 9000
	}

	printBanner()
	os.MkdirAll("logs", 0755)

	// 启动飞书事件监听服务
	if config.Feishu.Enabled {
		go startFeishuEventServer()
	}

	// 启动代理服务器
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
	color.Cyan("║         Diting 治理网关 v0.4.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书事件订阅          ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  LLM: %s", config.LLM.Model)
	if config.Feishu.Enabled {
		color.White("  飞书: 事件订阅模式")
		color.White("  事件端口: %d", config.Feishu.EventPort)
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

// 启动飞书事件服务器
func startFeishuEventServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/feishu/event", handleFeishuEvent)
	
	addr := fmt.Sprintf(":%d", config.Feishu.EventPort)
	
	color.Cyan("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书事件监听服务...")
	color.Green("✓ 飞书事件服务已启动")
	color.White("  监听地址: http://localhost%s/feishu/event", addr)
	color.Yellow("  请在飞书开放平台配置此地址为事件订阅 URL")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	if err := server.ListenAndServe(); err != nil {
		log.Printf("飞书事件服务器错误: %v", err)
	}
}

// 处理飞书事件
func handleFeishuEvent(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// URL 验证
	if challenge, ok := event["challenge"].(string); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"challenge": challenge,
		})
		color.Green("\n[%s] ✓ 飞书 URL 验证成功", time.Now().Format("15:04:05"))
		return
	}

	// 处理消息事件
	if header, ok := event["header"].(map[string]interface{}); ok {
		eventType, _ := header["event_type"].(string)
		
		if eventType == "im.message.receive_v1" {
			handleMessageReceive(event)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"code": "0"})
}

// 处理接收到的消息
func handleMessageReceive(event map[string]interface{}) {
	eventData, ok := event["event"].(map[string]interface{})
	if !ok {
		return
	}

	message, ok := eventData["message"].(map[string]interface{})
	if !ok {
		return
	}

	messageType, _ := message["message_type"].(string)
	if messageType != "text" {
		return
	}

	// 解析文本内容
	content, _ := message["content"].(string)
	var textContent map[string]string
	json.Unmarshal([]byte(content), &textContent)
	text := textContent["text"]

	// 获取 chat_id
	chatID, _ := message["chat_id"].(string)
	
	// 保存 chat_id（用于后续发送消息）
	if chatID != "" {
		userChatIDMutex.Lock()
		userChatID = chatID
		userChatIDMutex.Unlock()
	}

	// 获取发送者信息
	sender, ok := eventData["sender"].(map[string]interface{})
	if !ok {
		return
	}

	senderID, _ := sender["sender_id"].(map[string]interface{})
	openID, _ := senderID["open_id"].(string)
	userID, _ := senderID["user_id"].(string)

	color.Cyan("\n[%s] 📨 收到飞书消息", time.Now().Format("15:04:05"))
	fmt.Printf("  发送者 open_id: %s\n", openID)
	fmt.Printf("  发送者 user_id: %s（若改用默认 main 轮询模式，可填到 config approval_user_id）\n", userID)
	fmt.Printf("  Chat ID: %s（本会话，审批消息会发到此）\n", chatID)
	fmt.Printf("  内容: %s\n", text)

	// 检查审批回复
	checkApprovalReply(text, chatID)
}

// 检查审批回复
func checkApprovalReply(text, chatID string) {
	text = strings.ToLower(strings.TrimSpace(text))
	
	approveKeywords := []string{"批准", "approve", "y", "yes", "同意"}
	rejectKeywords := []string{"拒绝", "reject", "n", "no", "不同意"}

	var decision string
	for _, keyword := range approveKeywords {
		if text == keyword {
			decision = "approved"
			break
		}
	}
	
	if decision == "" {
		for _, keyword := range rejectKeywords {
			if text == keyword {
				decision = "rejected"
				break
			}
		}
	}

	if decision != "" {
		approvalRequests.Range(func(key, value interface{}) bool {
			req := value.(*ApprovalRequest)
			if req.Status == "pending" {
				req.Status = decision
				approvalRequests.Store(key, req)
				
				color.Green("  ✓ 审批决策: %s", decision)
				
				confirmMsg := "✅ 已批准操作"
				if decision == "rejected" {
					confirmMsg = "❌ 已拒绝操作"
				}
				sendFeishuMessageToChat(chatID, confirmMsg)
				
				return false
			}
			return true
		})
	}
}

// HTTP 代理处理
func handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	color.Cyan("\n[%s] 收到 HTTP 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  方法: %s\n", color.YellowString(r.Method))
	fmt.Printf("  URL: %s\n", color.WhiteString(r.URL.String()))

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	riskLevel := assessRisk(r.Method, r.URL.Path, string(bodyBytes))
	fmt.Printf("  风险等级: %s\n", colorizeRisk(riskLevel))

	audit := AuditLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Host:      r.URL.Host,
		Path:      r.URL.Path,
		Body:      string(bodyBytes),
		RiskLevel: riskLevel,
	}

	var decision string
	var intentAnalysis string

	if riskLevel == "低" {
		decision = "ALLOW"
		color.Green("  决策: 自动放行")
	} else {
		intentAnalysis = analyzeIntent(r.Method, r.URL.Path, string(bodyBytes))
		fmt.Printf("\n  🤖 意图分析:\n")
		color.Cyan("  %s", intentAnalysis)
		fmt.Println()

		if config.Feishu.Enabled {
			decision = requestFeishuApproval(r.Method, r.URL.String(), r.URL.Host, riskLevel, intentAnalysis)
		} else {
			decision = "DENY"
			color.Red("  决策: 自动拒绝（飞书未启用）")
		}
	}

	audit.IntentAnalysis = intentAnalysis
	audit.Decision = decision

	if decision == "ALLOW" {
		color.Green("\n  ✓ 请求已放行")
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
	for _, m := range config.Risk.AutoApproveMethods {
		if method == m {
			return "低"
		}
	}

	for _, m := range config.Risk.DangerousMethods {
		if method == m {
			return "高"
		}
	}

	for _, p := range config.Risk.DangerousPaths {
		if strings.Contains(strings.ToLower(path), p) {
			return "高"
		}
	}

	return "中"
}

func assessRiskHTTPS(host string) string {
	for _, domain := range config.Risk.SafeDomains {
		if strings.Contains(host, domain) {
			return "低"
		}
	}
	return "中"
}

// 意图分析
func analyzeIntent(method, path, body string) string {
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

	message := fmt.Sprintf(`🚨 Diting 高风险操作审批

操作: %s %s
风险等级: %s
意图分析: %s

请回复：
✅ "批准" 或 "approve" 或 "y" 来批准
❌ "拒绝" 或 "reject" 或 "n" 来拒绝

⏱️ %d分钟内未响应将自动拒绝
请求ID: %s`, method, path, riskLevel, intentAnalysis, config.Feishu.ApprovalTimeoutMinutes, requestID)

	// 获取 chat_id
	userChatIDMutex.RLock()
	chatID := userChatID
	userChatIDMutex.RUnlock()

	if chatID == "" {
		color.Red("  ✗ 未找到 chat_id，请先与机器人建立会话")
		return "DENY"
	}

	if err := sendFeishuMessageToChat(chatID, message); err != nil {
		color.Red("  ✗ 发送飞书消息失败: %v", err)
		return "DENY"
	}

	req.ChatID = chatID
	approvalRequests.Store(requestID, &req)

	color.Yellow("  ⏳ 等待飞书审批...")

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
	ticker := time.NewTicker(1 * time.Second)
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

			if time.Now().After(deadline) {
				if val, ok := approvalRequests.Load(requestID); ok {
					req := val.(*ApprovalRequest)
					req.Status = "timeout"
					approvalRequests.Store(requestID, req)
					
					if req.ChatID != "" {
						sendFeishuMessageToChat(req.ChatID, fmt.Sprintf("⏱️ 审批超时，请求 %s 已自动拒绝", requestID))
					}
				}
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

func sendFeishuMessageToChat(chatID, content string) error {
	token, err := getFeishuToken()
	if err != nil {
		return err
	}

	contentJSON, _ := json.Marshal(map[string]string{"text": content})
	
	reqBody, _ := json.Marshal(map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	})

	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id",
		bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		return fmt.Errorf("飞书 API 错误: %v", result["msg"])
	}

	return nil
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
