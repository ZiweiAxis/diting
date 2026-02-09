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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	configpkg "diting/internal/config"
)

// ============================================================================
// Configuration Structures
// ============================================================================

type Config struct {
	Proxy ProxyConfig `json:"proxy"`
	LLM   LLMConfig   `json:"llm"`
	Feishu FeishuConfig `json:"feishu"`
	Risk  RiskConfig  `json:"risk"`
	Audit AuditConfig `json:"audit"`
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
	ChatID                 string `json:"chat_id"` // 可选；当 approval_user_id 报 open_id cross app 时，改用 chat 发审批并轮询此 chat
	ApprovalTimeoutMinutes int    `json:"approval_timeout_minutes"`
	UseInteractiveCard     bool   `json:"use_interactive_card"`
	UseMessageReply        bool   `json:"use_message_reply"`
	PollIntervalSeconds    int    `json:"poll_interval_seconds"`
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

// ============================================================================
// Feishu API Structures
// ============================================================================

type FeishuTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

type FeishuMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

type FeishuMessageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MessageID string `json:"message_id"`
	} `json:"data"`
}

type FeishuMessagesResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		HasMore bool `json:"has_more"`
		Items   []struct {
			MessageID  string `json:"message_id"`
			ChatID     string `json:"chat_id"`
			Sender     struct {
				SenderID struct {
					UserID string `json:"user_id"`
				} `json:"sender_id"`
			} `json:"sender"`
			Body struct {
				Content string `json:"content"`
			} `json:"body"`
			CreateTime string `json:"create_time"`
		} `json:"items"`
	} `json:"data"`
}

// ============================================================================
// Claude API Structures
// ============================================================================

type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Messages    []ClaudeMessage `json:"messages"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type IntentAnalysis struct {
	RiskLevel   string `json:"risk_level"`   // "safe", "medium", "high"
	Reason      string `json:"reason"`
	Recommended string `json:"recommended"`  // "approve", "deny", "review"
}

// ============================================================================
// Approval System
// ============================================================================

type ApprovalRequest struct {
	ID          string
	Method      string
	URL         string
	Host        string
	Path        string
	Body        string
	Intent      *IntentAnalysis
	Timestamp   time.Time
	ResponseCh  chan ApprovalResponse
	ReplyChatID string // 若审批消息发到了群/会话，轮询时用此 chat_id
}

type ApprovalResponse struct {
	Approved bool
	Reason   string
}

// ============================================================================
// Global State
// ============================================================================

var (
	config            Config
	feishuAccessToken string
	tokenExpiry       time.Time
	tokenMutex        sync.RWMutex
	approvalRequests  = make(map[string]*ApprovalRequest)
	approvalMutex     sync.RWMutex
	auditFile         *os.File
)

// ============================================================================
// Main Function
// ============================================================================

func main() {
	// 配置：.env + config.yaml（与 All-in-One 一致，已移除 config.json）
	_ = configpkg.LoadEnvFile(".env", true)
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); err != nil {
		configPath = "config.example.yaml"
	}
	cfg, err := configpkg.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v (ensure config.yaml or config.example.yaml exists)", err)
	}
	config = mainConfigFromUnified(cfg)
	log.Printf("Configuration loaded successfully")

	// Initialize audit logging
	if config.Audit.Enabled {
		if err := initAuditLog(); err != nil {
			log.Fatalf("Failed to initialize audit log: %v", err)
		}
		defer auditFile.Close()
	}

	// Start Feishu message polling if enabled
	if config.Feishu.Enabled {
		go pollFeishuMessagesLoop()
	}

	// Start proxy server
	log.Printf("Starting Diting proxy server on %s", config.Proxy.Listen)
	server := &http.Server{
		Addr:    config.Proxy.Listen,
		Handler: http.HandlerFunc(handleRequest),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Proxy server failed: %v", err)
	}
}

// ============================================================================
// Configuration Loading
// ============================================================================

// mainConfigFromUnified 从统一配置 config.yaml（含 env 覆盖）映射为 main 的 Config。
func mainConfigFromUnified(cfg *configpkg.Config) Config {
	out := Config{
		Proxy: ProxyConfig{Listen: cfg.Proxy.ListenAddr, TimeoutSeconds: 30},
		Feishu: FeishuConfig{
			Enabled:                cfg.Delivery.Feishu.Enabled,
			AppID:                  cfg.Delivery.Feishu.AppID,
			AppSecret:              cfg.Delivery.Feishu.AppSecret,
			ApprovalUserID:         cfg.Delivery.Feishu.ApprovalUserID,
			ChatID:                 cfg.Delivery.Feishu.ChatID,
			ApprovalTimeoutMinutes: cfg.Delivery.Feishu.ApprovalTimeoutMinutes,
			UseMessageReply:        cfg.Delivery.Feishu.UseMessageReply,
			PollIntervalSeconds:    cfg.Delivery.Feishu.PollIntervalSeconds,
		},
		Audit: AuditConfig{LogFile: cfg.Audit.Path, Enabled: cfg.Audit.Path != ""},
	}
	if cfg.LLM != nil {
		out.LLM = LLMConfig{
			Provider:    cfg.LLM.Provider,
			BaseURL:     cfg.LLM.BaseURL,
			APIKey:      cfg.LLM.APIKey,
			Model:       cfg.LLM.Model,
			MaxTokens:   cfg.LLM.MaxTokens,
			Temperature: cfg.LLM.Temperature,
		}
	}
	if cfg.Risk != nil {
		out.Risk = RiskConfig{
			DangerousMethods:   cfg.Risk.DangerousMethods,
			DangerousPaths:     cfg.Risk.DangerousPaths,
			AutoApproveMethods: cfg.Risk.AutoApproveMethods,
			SafeDomains:        cfg.Risk.SafeDomains,
		}
	}
	if out.LLM.MaxTokens == 0 {
		out.LLM.MaxTokens = 1024
	}
	return out
}

// ============================================================================
// Audit Logging
// ============================================================================

func initAuditLog() error {
	logDir := filepath.Dir(config.Audit.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	f, err := os.OpenFile(config.Audit.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}

	auditFile = f
	return nil
}

func logAudit(entry map[string]interface{}) {
	if !config.Audit.Enabled || auditFile == nil {
		return
	}

	entry["timestamp"] = time.Now().Format(time.RFC3339)
	data, _ := json.Marshal(entry)
	auditFile.Write(data)
	auditFile.Write([]byte("\n"))
}

// ============================================================================
// Feishu API Functions
// ============================================================================

func getTenantAccessToken() (string, error) {
	tokenMutex.RLock()
	if time.Now().Before(tokenExpiry) && feishuAccessToken != "" {
		token := feishuAccessToken
		tokenMutex.RUnlock()
		return token, nil
	}
	tokenMutex.RUnlock()

	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(tokenExpiry) && feishuAccessToken != "" {
		return feishuAccessToken, nil
	}

	reqBody := map[string]string{
		"app_id":     config.Feishu.AppID,
		"app_secret": config.Feishu.AppSecret,
	}
	reqData, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json",
		bytes.NewReader(reqData),
	)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp FeishuTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("feishu API error: %s", tokenResp.Msg)
	}

	feishuAccessToken = tokenResp.TenantAccessToken
	tokenExpiry = time.Now().Add(time.Duration(tokenResp.Expire-300) * time.Second)

	log.Printf("Feishu access token refreshed, expires in %d seconds", tokenResp.Expire)
	return feishuAccessToken, nil
}

func sendFeishuMessage(receiveID, content string) (string, error) {
	token, err := getTenantAccessToken()
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}

	// 配置里可能是 open_id(ou_xxx) 或 user_id，按前缀选择 receive_id_type
	receiveIDType := "user_id"
	if strings.HasPrefix(receiveID, "ou_") {
		receiveIDType = "open_id"
	}

	msgContent := map[string]string{"text": content}
	contentJSON, _ := json.Marshal(msgContent)

	msgReq := FeishuMessageRequest{
		ReceiveID: receiveID,
		MsgType:   "text",
		Content:   string(contentJSON),
	}
	reqData, _ := json.Marshal(msgReq)

	req, _ := http.NewRequest(
		"POST",
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+receiveIDType,
		bytes.NewReader(reqData),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var msgResp FeishuMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if msgResp.Code != 0 {
		msg := msgResp.Msg
		if strings.Contains(msg, "open_id cross app") {
			msg = msg + "（请将 config 中 approval_user_id 改为该用户在本应用下的 user_id，不要使用其他应用的 open_id。在飞书管理后台或通过「获取用户 user_id」接口查询）"
		}
		return "", fmt.Errorf("feishu API error: %s", msg)
	}

	return msgResp.Data.MessageID, nil
}

// 向群/会话发消息（receive_id_type=chat_id），用于 open_id cross app 时回退
func sendFeishuMessageToChat(chatID, content string) (string, error) {
	token, err := getTenantAccessToken()
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}
	msgContent := map[string]string{"text": content}
	contentJSON, _ := json.Marshal(msgContent)
	msgReq := FeishuMessageRequest{
		ReceiveID: chatID,
		MsgType:   "text",
		Content:   string(contentJSON),
	}
	reqData, _ := json.Marshal(msgReq)
	req, _ := http.NewRequest(
		"POST",
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id",
		bytes.NewReader(reqData),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send message to chat: %w", err)
	}
	defer resp.Body.Close()
	var msgResp FeishuMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if msgResp.Code != 0 {
		return "", fmt.Errorf("feishu API error: %s", msgResp.Msg)
	}
	return msgResp.Data.MessageID, nil
}

func pollFeishuMessages(containerID string, since time.Time, acceptAnySender bool) ([]string, error) {
	token, err := getTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	url := fmt.Sprintf(
		"https://open.feishu.cn/open-apis/im/v1/messages?container_id_type=chat&container_id=%s&start_time=%d",
		containerID,
		since.Unix(),
	)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll messages: %w", err)
	}
	defer resp.Body.Close()

	var msgResp FeishuMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if msgResp.Code != 0 {
		return nil, fmt.Errorf("feishu API error: %s", msgResp.Msg)
	}

	var messages []string
	for _, item := range msgResp.Data.Items {
		if acceptAnySender || item.Sender.SenderID.UserID == config.Feishu.ApprovalUserID {
			var content map[string]string
			json.Unmarshal([]byte(item.Body.Content), &content)
			if text, ok := content["text"]; ok {
				messages = append(messages, strings.TrimSpace(text))
			}
		}
	}

	return messages, nil
}

func pollFeishuMessagesLoop() {
	if !config.Feishu.UseMessageReply {
		return
	}

	interval := time.Duration(config.Feishu.PollIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Started Feishu message polling (interval: %v)", interval)

	for range ticker.C {
		approvalMutex.RLock()
		requests := make([]*ApprovalRequest, 0, len(approvalRequests))
		for _, req := range approvalRequests {
			requests = append(requests, req)
		}
		approvalMutex.RUnlock()

		for _, req := range requests {
			// 若审批发到了 chat，则轮询该 chat；否则轮询原 approval_user 对应会话（需 API 支持）
			pollContainer := config.Feishu.ApprovalUserID
			acceptAny := false
			if req.ReplyChatID != "" {
				pollContainer = req.ReplyChatID
				acceptAny = true
			}
			messages, err := pollFeishuMessages(pollContainer, req.Timestamp, acceptAny)
			if err != nil {
				log.Printf("Error polling messages for request %s: %v", req.ID, err)
				continue
			}

			// Check for approval/denial keywords
			for _, msg := range messages {
				msgLower := strings.ToLower(msg)
				if strings.Contains(msgLower, req.ID[:8]) {
					var approved bool
					var reason string

					if strings.Contains(msgLower, "approve") || strings.Contains(msgLower, "同意") || 
					   strings.Contains(msgLower, "允许") || strings.Contains(msgLower, "yes") {
						approved = true
						reason = "Approved by user via Feishu message"
					} else if strings.Contains(msgLower, "deny") || strings.Contains(msgLower, "拒绝") || 
					          strings.Contains(msgLower, "禁止") || strings.Contains(msgLower, "no") {
						approved = false
						reason = "Denied by user via Feishu message"
					} else {
						continue
					}

					// Send response
					select {
					case req.ResponseCh <- ApprovalResponse{Approved: approved, Reason: reason}:
						log.Printf("Approval response sent for request %s: approved=%v", req.ID, approved)
					default:
					}

					// Remove from pending requests
					approvalMutex.Lock()
					delete(approvalRequests, req.ID)
					approvalMutex.Unlock()
					break
				}
			}
		}
	}
}

// ============================================================================
// Claude API Functions
// ============================================================================

func analyzeIntentWithClaude(method, path, body string) (*IntentAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze this HTTP request and determine its risk level and intent.

Method: %s
Path: %s
Body: %s

Respond with JSON only:
{
  "risk_level": "safe|medium|high",
  "reason": "brief explanation",
  "recommended": "approve|deny|review"
}`, method, path, body)

	reqBody := ClaudeRequest{
		Model:       config.LLM.Model,
		MaxTokens:   config.LLM.MaxTokens,
		Temperature: config.LLM.Temperature,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	reqData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", config.LLM.BaseURL+"/v1/messages", bytes.NewReader(reqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.LLM.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude API error %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("decode claude response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("empty claude response")
	}

	// Extract JSON from response
	text := claudeResp.Content[0].Text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var analysis IntentAnalysis
	if err := json.Unmarshal([]byte(text[start:end+1]), &analysis); err != nil {
		return nil, fmt.Errorf("parse intent analysis: %w", err)
	}

	return &analysis, nil
}

// ============================================================================
// Approval Logic
// ============================================================================

func sendApprovalRequest(req *ApprovalRequest) error {
	if !config.Feishu.Enabled {
		return fmt.Errorf("feishu not enabled")
	}

	// Build approval message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("🔔 代理请求审批 [%s]\n\n", req.ID[:8]))
	msg.WriteString(fmt.Sprintf("方法: %s\n", req.Method))
	msg.WriteString(fmt.Sprintf("主机: %s\n", req.Host))
	msg.WriteString(fmt.Sprintf("路径: %s\n", req.Path))
	
	if req.Body != "" && len(req.Body) < 200 {
		msg.WriteString(fmt.Sprintf("请求体: %s\n", req.Body))
	}
	
	if req.Intent != nil {
		msg.WriteString(fmt.Sprintf("\n🤖 AI 分析:\n"))
		msg.WriteString(fmt.Sprintf("风险等级: %s\n", req.Intent.RiskLevel))
		msg.WriteString(fmt.Sprintf("原因: %s\n", req.Intent.Reason))
		msg.WriteString(fmt.Sprintf("建议: %s\n", req.Intent.Recommended))
	}
	
	msg.WriteString(fmt.Sprintf("\n请回复: approve %s 或 deny %s", req.ID[:8], req.ID[:8]))

	msgStr := msg.String()
	// 先尝试发到 approval_user_id
	messageID, err := sendFeishuMessage(config.Feishu.ApprovalUserID, msgStr)
	if err != nil && strings.Contains(err.Error(), "open_id cross app") && config.Feishu.ChatID != "" {
		// 回退：发到 chat_id（群/会话），并标记该请求用 chat 轮询
		messageID, err = sendFeishuMessageToChat(config.Feishu.ChatID, msgStr)
		if err != nil {
			return fmt.Errorf("send feishu message (chat fallback): %w", err)
		}
		req.ReplyChatID = config.Feishu.ChatID
		log.Printf("Approval request sent to Feishu chat (open_id cross app fallback), message_id=%s", messageID)
	} else if err != nil {
		return fmt.Errorf("send feishu message: %w", err)
	} else {
		log.Printf("Approval request sent to Feishu, message_id=%s", messageID)
	}

	// Store request for polling
	approvalMutex.Lock()
	approvalRequests[req.ID] = req
	approvalMutex.Unlock()

	return nil
}

func waitForApproval(requestID string, timeout time.Duration) ApprovalResponse {
	approvalMutex.RLock()
	req, exists := approvalRequests[requestID]
	approvalMutex.RUnlock()

	if !exists {
		return ApprovalResponse{Approved: false, Reason: "Request not found"}
	}

	select {
	case resp := <-req.ResponseCh:
		return resp
	case <-time.After(timeout):
		// Cleanup
		approvalMutex.Lock()
		delete(approvalRequests, requestID)
		approvalMutex.Unlock()
		
		return ApprovalResponse{Approved: false, Reason: "Approval timeout"}
	}
}

func needsApproval(method, host, path string) bool {
	// Check auto-approve methods
	for _, m := range config.Risk.AutoApproveMethods {
		if strings.EqualFold(method, m) {
			return false
		}
	}

	// Check safe domains
	for _, domain := range config.Risk.SafeDomains {
		if strings.Contains(host, domain) {
			return false
		}
	}

	// Check dangerous methods
	for _, m := range config.Risk.DangerousMethods {
		if strings.EqualFold(method, m) {
			return true
		}
	}

	// Check dangerous paths
	pathLower := strings.ToLower(path)
	for _, p := range config.Risk.DangerousPaths {
		if strings.Contains(pathLower, strings.ToLower(p)) {
			return true
		}
	}

	return false
}

// ============================================================================
// Proxy Handlers
// ============================================================================

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		handleHTTPS(w, r)
	} else {
		handleHTTP(w, r)
	}
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("HTTP: %s %s", r.Method, r.URL.String())

	// Read body for analysis
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Check if approval needed
	if needsApproval(r.Method, r.Host, r.URL.Path) {
		if !handleApprovalFlow(w, r, string(bodyBytes)) {
			return
		}
	}

	// Log audit
	logAudit(map[string]interface{}{
		"type":   "http_request",
		"method": r.Method,
		"host":   r.Host,
		"path":   r.URL.Path,
		"approved": true,
	})

	// Forward request
	client := &http.Client{
		Timeout: time.Duration(config.Proxy.TimeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	proxyReq, err := http.NewRequest(r.Method, r.URL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	log.Printf("HTTPS: CONNECT %s", r.Host)

	// Check if approval needed (for CONNECT we only check host)
	if needsApproval("CONNECT", r.Host, "") {
		if !handleApprovalFlow(w, r, "") {
			return
		}
	}

	// Log audit
	logAudit(map[string]interface{}{
		"type":     "https_connect",
		"host":     r.Host,
		"approved": true,
	})

	// Establish connection to target
	targetConn, err := net.DialTimeout("tcp", r.Host, time.Duration(config.Proxy.TimeoutSeconds)*time.Second)
	if err != nil {
		http.Error(w, "Failed to connect to target", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional copy
	go io.Copy(targetConn, clientConn)
	io.Copy(clientConn, targetConn)
}

func handleApprovalFlow(w http.ResponseWriter, r *http.Request, body string) bool {
	if !config.Feishu.Enabled {
		log.Printf("Approval needed but Feishu disabled, denying request")
		http.Error(w, "Request requires approval but approval system is disabled", http.StatusForbidden)
		return false
	}

	// Analyze intent with Claude
	var intent *IntentAnalysis
	if config.LLM.Provider == "anthropic" {
		var err error
		intent, err = analyzeIntentWithClaude(r.Method, r.URL.Path, body)
		if err != nil {
			log.Printf("Intent analysis failed: %v", err)
		}
	}

	// Create approval request
	approvalReq := &ApprovalRequest{
		ID:         uuid.New().String(),
		Method:     r.Method,
		URL:        r.URL.String(),
		Host:       r.Host,
		Path:       r.URL.Path,
		Body:       body,
		Intent:     intent,
		Timestamp:  time.Now(),
		ResponseCh: make(chan ApprovalResponse, 1),
	}

	// Send approval request
	if err := sendApprovalRequest(approvalReq); err != nil {
		log.Printf("Failed to send approval request: %v", err)
		http.Error(w, "Failed to request approval", http.StatusInternalServerError)
		return false
	}

	// Wait for approval
	timeout := time.Duration(config.Feishu.ApprovalTimeoutMinutes) * time.Minute
	response := waitForApproval(approvalReq.ID, timeout)

	// Log audit
	logAudit(map[string]interface{}{
		"type":     "approval_decision",
		"request_id": approvalReq.ID,
		"method":   r.Method,
		"host":     r.Host,
		"path":     r.URL.Path,
		"approved": response.Approved,
		"reason":   response.Reason,
	})

	if !response.Approved {
		log.Printf("Request denied: %s", response.Reason)
		http.Error(w, fmt.Sprintf("Request denied: %s", response.Reason), http.StatusForbidden)
		return false
	}

	log.Printf("Request approved: %s", response.Reason)
	return true
}
