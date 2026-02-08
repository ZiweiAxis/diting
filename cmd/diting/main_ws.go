package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// 配置结构 (保持不变)
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
	wsConn           *websocket.Conn
	wsConnMutex      sync.RWMutex
)

func main() {
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	printBanner()
	os.MkdirAll("logs", 0755)

	// 启动飞书长连接
	if config.Feishu.Enabled {
		go startFeishuWebSocket()
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
	color.Cyan("║         Diting 治理网关 v0.5.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书长连接            ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  LLM: %s", config.LLM.Model)
	if config.Feishu.Enabled {
		color.White("  飞书: 长连接模式 (WebSocket)")
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

// 启动飞书 WebSocket 长连接
func startFeishuWebSocket() {
	color.Cyan("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接...")

	for {
		if err := connectFeishuWebSocket(); err != nil {
			color.Red("  ✗ 长连接失败: %v", err)
			color.Yellow("  ⏳ 10秒后重试...")
			time.Sleep(10 * time.Second)
			continue
		}
		
		// 连接断开后重连
		color.Yellow("  ⏳ 连接断开，5秒后重连...")
		time.Sleep(5 * time.Second)
	}
}

// 连接飞书 WebSocket
func connectFeishuWebSocket() error {
	// 1. 获取 endpoint
	endpoint, err := getFeishuWSEndpoint()
	if err != nil {
		return fmt.Errorf("获取 endpoint 失败: %v", err)
	}

	color.Green("  ✓ 获取 endpoint 成功")
	color.White("    %s", endpoint)

	// 2. 建立 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return fmt.Errorf("建立 WebSocket 连接失败: %v", err)
	}

	wsConnMutex.Lock()
	wsConn = conn
	wsConnMutex.Unlock()

	color.Green("  ✓ WebSocket 连接已建立")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 3. 启动心跳
	go sendHeartbeat(conn)

	// 4. 接收消息
	return receiveMessages(conn)
}

// 获取 WebSocket endpoint
func getFeishuWSEndpoint() (string, error) {
	token, err := getFeishuToken()
	if err != nil {
		return "", err
	}

	// 调用飞书 API 获取 endpoint
	// 文档: https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/events/receive
	reqBody, _ := json.Marshal(map[string]interface{}{
		"app_id": config.Feishu.AppID,
	})

	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/stream/get", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		return "", fmt.Errorf("获取 endpoint 失败: %v", result["msg"])
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应格式错误")
	}

	endpoint, ok := data["url"].(string)
	if !ok {
		return "", fmt.Errorf("未找到 endpoint")
	}

	return endpoint, nil
}

// 发送心跳
func sendHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		wsConnMutex.RLock()
		if wsConn == nil {
			wsConnMutex.RUnlock()
			return
		}
		wsConnMutex.RUnlock()

		heartbeat := map[string]interface{}{
			"type": "PING",
			"data": map[string]interface{}{
				"ping": time.Now().Unix(),
			},
		}

		if err := conn.WriteJSON(heartbeat); err != nil {
			log.Printf("发送心跳失败: %v", err)
			return
		}
	}
}

// 接收消息
func receiveMessages(conn *websocket.Conn) error {
	defer func() {
		wsConnMutex.Lock()
		wsConn = nil
		wsConnMutex.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("读取消息失败: %v", err)
		}

		var event map[string]interface{}
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 处理不同类型的事件
		eventType, _ := event["type"].(string)
		
		switch eventType {
		case "PONG":
			// 心跳响应
			continue
		case "EVENT_CALLBACK":
			// 事件回调
			handleFeishuEvent(event)
		}
	}
}

// 处理飞书事件
func handleFeishuEvent(event map[string]interface{}) {
	header, ok := event["header"].(map[string]interface{})
	if !ok {
		return
	}

	eventType, _ := header["event_type"].(string)
	
	if eventType == "im.message.receive_v1" {
		handleMessageReceive(event)
	}
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

	content, _ := message["content"].(string)
	var textContent map[string]string
	json.Unmarshal([]byte(content), &textContent)
	text := textContent["text"]

	chatID, _ := message["chat_id"].(string)
	
	if chatID != "" {
		userChatIDMutex.Lock()
		userChatID = chatID
		userChatIDMutex.Unlock()
	}

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

// HTTP 代理处理 (保持不变，省略...)
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
	}

	audit.Duration = time.Since(startTime).Milliseconds()
	fmt.Printf("  耗时: %dms\n", audit.Duration)
	saveAuditLog(audit)
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	// 省略，与之前相同
}

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

func analyzeIntent(method, path, body string) string {
	return fmt.Sprintf("规则引擎: %s %s 操作需要审批", method, path)
}

func requestFeishuApproval(method, path, host, riskLevel, intentAnalysis string) string {
	requestID := uuid.New().String()

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

	userChatIDMutex.RLock()
	chatID := userChatID
	userChatIDMutex.RUnlock()

	if chatID == "" {
		color.Red("  ✗ 未找到 chat_id，请先与机器人发送消息建立会话")
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
	} else {
		color.Red("  ✗ 审批拒绝或超时")
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
				return "DENY"
			}
		}
	}
}

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
	// 省略，与之前相同
}

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
