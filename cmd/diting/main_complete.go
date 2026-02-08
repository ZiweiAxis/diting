package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// ============================================================================
// 配置结构
// ============================================================================

type Config struct {
	Proxy  ProxyConfig  `json:"proxy"`
	Feishu FeishuConfig `json:"feishu"`
	Risk   RiskConfig   `json:"risk"`
	Audit  AuditConfig  `json:"audit"`
}

type ProxyConfig struct {
	Listen         string `json:"listen"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type FeishuConfig struct {
	Enabled                bool   `json:"enabled"`
	AppID                  string `json:"app_id"`
	AppSecret              string `json:"app_secret"`
	ApprovalTimeoutMinutes int    `json:"approval_timeout_minutes"`
	ChatID                 string `json:"chat_id"` // 添加 Chat ID 配置
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
// 审批请求结构
// ============================================================================

type ApprovalRequest struct {
	ID        string    `json:"id"`
	Method    string    `json:"method"`
	URL       string    `json:"url"`
	Host      string    `json:"host"`
	RiskLevel string    `json:"risk_level"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // pending, approved, rejected, timeout
	Response  chan bool `json:"-"`      // 用于通知审批结果
}

// ============================================================================
// 审计日志结构
// ============================================================================

type AuditLog struct {
	Timestamp   time.Time `json:"timestamp"`
	RequestID   string    `json:"request_id"`
	Method      string    `json:"method"`
	URL         string    `json:"url"`
	Host        string    `json:"host"`
	RiskLevel   string    `json:"risk_level"`
	Status      string    `json:"status"`
	ApprovalID  string    `json:"approval_id,omitempty"`
	Duration    int64     `json:"duration_ms"`
	ClientAddr  string    `json:"client_addr"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

// ============================================================================
// Diting 主服务
// ============================================================================

type DitingService struct {
	config          *Config
	larkClient      *lark.Client
	wsClient        *larkws.Client
	approvalManager *ApprovalManager
	auditLogger     *AuditLogger
	ctx             context.Context
	cancel          context.CancelFunc
}

// ============================================================================
// 审批管理器
// ============================================================================

type ApprovalManager struct {
	mu              sync.RWMutex
	pendingRequests map[string]*ApprovalRequest // key: approval ID
	larkClient      *lark.Client
	config          *Config
}

func NewApprovalManager(larkClient *lark.Client, config *Config) *ApprovalManager {
	return &ApprovalManager{
		pendingRequests: make(map[string]*ApprovalRequest),
		larkClient:      larkClient,
		config:          config,
	}
}

// 创建审批请求
func (am *ApprovalManager) CreateApproval(method, url, host, riskLevel string) (*ApprovalRequest, error) {
	req := &ApprovalRequest{
		ID:        uuid.New().String(),
		Method:    method,
		URL:       url,
		Host:      host,
		RiskLevel: riskLevel,
		Timestamp: time.Now(),
		Status:    "pending",
		Response:  make(chan bool, 1),
	}

	am.mu.Lock()
	am.pendingRequests[req.ID] = req
	am.mu.Unlock()

	// 发送飞书消息
	if err := am.sendFeishuApproval(req); err != nil {
		return nil, fmt.Errorf("发送飞书审批失败: %w", err)
	}

	// 启动超时计时器
	go am.handleTimeout(req)

	return req, nil
}

// 发送飞书审批消息
func (am *ApprovalManager) sendFeishuApproval(req *ApprovalRequest) error {
	chatID := am.config.Feishu.ChatID
	if chatID == "" {
		return fmt.Errorf("未配置 Chat ID")
	}

	// 构建消息内容
	content := fmt.Sprintf(`🚨 高风险操作审批

📋 审批 ID: %s
🔗 请求方法: %s
🌐 目标 URL: %s
🏠 主机: %s
⚠️  风险等级: %s
⏰ 时间: %s

请回复：
✅ 批准 / approve / y
❌ 拒绝 / reject / n

⏱️  5分钟后自动拒绝`,
		req.ID[:8], // 只显示前8位
		req.Method,
		req.URL,
		req.Host,
		req.RiskLevel,
		req.Timestamp.Format("2006-01-02 15:04:05"),
	)

	// 发送消息
	resp, err := am.larkClient.Im.Message.Create(context.Background(), larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(fmt.Sprintf(`{"text":"%s"}`, strings.ReplaceAll(content, "\n", "\\n"))).
			Build()).
		Build())

	if err != nil {
		return err
	}

	if !resp.Success() {
		return fmt.Errorf("发送消息失败: %s", resp.Msg)
	}

	color.Green("  ✓ 审批消息已发送到飞书 (ID: %s)", req.ID[:8])
	return nil
}

// 处理超时
func (am *ApprovalManager) handleTimeout(req *ApprovalRequest) {
	timeout := time.Duration(am.config.Feishu.ApprovalTimeoutMinutes) * time.Minute
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		am.mu.Lock()
		if req.Status == "pending" {
			req.Status = "timeout"
			req.Response <- false
			delete(am.pendingRequests, req.ID)
			color.Yellow("  ⏱️  审批超时，自动拒绝 (ID: %s)", req.ID[:8])
		}
		am.mu.Unlock()
	case <-req.Response:
		// 已经有结果了，不需要超时处理
		return
	}
}

// 处理审批回复
func (am *ApprovalManager) HandleApprovalReply(content string) {
	content = strings.TrimSpace(strings.ToLower(content))

	// 判断是批准还是拒绝
	var approved bool
	var isApprovalKeyword bool

	if strings.Contains(content, "批准") || strings.Contains(content, "approve") ||
		content == "y" || content == "yes" || content == "同意" {
		approved = true
		isApprovalKeyword = true
	} else if strings.Contains(content, "拒绝") || strings.Contains(content, "reject") ||
		content == "n" || content == "no" || content == "不同意" {
		approved = false
		isApprovalKeyword = true
	}

	if !isApprovalKeyword {
		return // 不是审批关键词，忽略
	}

	// 查找最近的待审批请求
	am.mu.Lock()
	defer am.mu.Unlock()

	var latestReq *ApprovalRequest
	var latestTime time.Time

	for _, req := range am.pendingRequests {
		if req.Status == "pending" {
			if latestReq == nil || req.Timestamp.After(latestTime) {
				latestReq = req
				latestTime = req.Timestamp
			}
		}
	}

	if latestReq == nil {
		color.Yellow("  ⚠️  没有待审批的请求")
		return
	}

	// 更新状态
	if approved {
		latestReq.Status = "approved"
		color.Green("  ✅ 审批通过 (ID: %s)", latestReq.ID[:8])
	} else {
		latestReq.Status = "rejected"
		color.Red("  ❌ 审批拒绝 (ID: %s)", latestReq.ID[:8])
	}

	latestReq.Response <- approved
	delete(am.pendingRequests, latestReq.ID)
}

// ============================================================================
// 审计日志记录器
// ============================================================================

type AuditLogger struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
	enabled bool
}

func NewAuditLogger(config *AuditConfig) (*AuditLogger, error) {
	if !config.Enabled {
		return &AuditLogger{enabled: false}, nil
	}

	// 创建日志目录
	logDir := filepath.Dir(config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	return &AuditLogger{
		file:    file,
		encoder: json.NewEncoder(file),
		enabled: true,
	}, nil
}

func (al *AuditLogger) Log(log *AuditLog) error {
	if !al.enabled {
		return nil
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	return al.encoder.Encode(log)
}

func (al *AuditLogger) Close() error {
	if al.file != nil {
		return al.file.Close()
	}
	return nil
}

// ============================================================================
// 风险评估
// ============================================================================

func (ds *DitingService) assessRisk(method, url, host string) string {
	// 检查是否是自动批准的方法
	for _, m := range ds.config.Risk.AutoApproveMethods {
		if method == m {
			return "low"
		}
	}

	// 检查是否是危险方法
	isDangerousMethod := false
	for _, m := range ds.config.Risk.DangerousMethods {
		if method == m {
			isDangerousMethod = true
			break
		}
	}

	// 检查是否是危险路径
	isDangerousPath := false
	for _, p := range ds.config.Risk.DangerousPaths {
		if strings.Contains(strings.ToLower(url), strings.ToLower(p)) {
			isDangerousPath = true
			break
		}
	}

	// 检查是否是安全域名
	isSafeDomain := false
	for _, d := range ds.config.Risk.SafeDomains {
		if strings.Contains(host, d) {
			isSafeDomain = true
			break
		}
	}

	// 风险评估逻辑
	if isDangerousMethod && isDangerousPath {
		return "high"
	}
	if isDangerousMethod || isDangerousPath {
		return "medium"
	}
	if isSafeDomain {
		return "low"
	}

	return "medium"
}

// ============================================================================
// HTTP 代理处理
// ============================================================================

func (ds *DitingService) handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()

	color.Cyan("\n[%s] 📨 HTTP 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  请求 ID: %s\n", requestID[:8])
	fmt.Printf("  方法: %s\n", r.Method)
	fmt.Printf("  URL: %s\n", r.URL.String())
	fmt.Printf("  主机: %s\n", r.Host)

	// 风险评估
	riskLevel := ds.assessRisk(r.Method, r.URL.String(), r.Host)
	fmt.Printf("  风险等级: %s\n", riskLevel)

	// 审计日志
	auditLog := &AuditLog{
		Timestamp:  startTime,
		RequestID:  requestID,
		Method:     r.Method,
		URL:        r.URL.String(),
		Host:       r.Host,
		RiskLevel:  riskLevel,
		ClientAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	}

	// 低风险自动放行
	if riskLevel == "low" {
		color.Green("  ✓ 低风险，自动放行")
		auditLog.Status = "approved"
		ds.proxyHTTPRequest(w, r)
	} else {
		// 高风险需要审批
		color.Yellow("  ⚠️  高风险，需要审批")
		approval, err := ds.approvalManager.CreateApproval(r.Method, r.URL.String(), r.Host, riskLevel)
		if err != nil {
			color.Red("  ✗ 创建审批失败: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			auditLog.Status = "error"
			ds.auditLogger.Log(auditLog)
			return
		}

		auditLog.ApprovalID = approval.ID

		// 等待审批结果
		approved := <-approval.Response
		if approved {
			color.Green("  ✓ 审批通过，执行请求")
			auditLog.Status = "approved"
			ds.proxyHTTPRequest(w, r)
		} else {
			color.Red("  ✗ 审批拒绝，阻止请求")
			auditLog.Status = "rejected"
			http.Error(w, "Request Rejected", http.StatusForbidden)
		}
	}

	// 记录审计日志
	auditLog.Duration = time.Since(startTime).Milliseconds()
	ds.auditLogger.Log(auditLog)
}

// 实际执行 HTTP 代理
func (ds *DitingService) proxyHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// 创建新的请求
	outReq := r.Clone(context.Background())
	outReq.RequestURI = ""

	// 发送请求
	client := &http.Client{
		Timeout: time.Duration(ds.config.Proxy.TimeoutSeconds) * time.Second,
	}
	resp, err := client.Do(outReq)
	if err != nil {
		color.Red("  ✗ 代理请求失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// 复制状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	io.Copy(w, resp.Body)

	color.Green("  ✓ 请求完成 (状态码: %d)", resp.StatusCode)
}

// ============================================================================
// HTTPS 代理处理 (CONNECT)
// ============================================================================

func (ds *DitingService) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()

	color.Cyan("\n[%s] 🔒 HTTPS 请求", time.Now().Format("15:04:05"))
	fmt.Printf("  请求 ID: %s\n", requestID[:8])
	fmt.Printf("  方法: %s\n", r.Method)
	fmt.Printf("  主机: %s\n", r.Host)

	// 风险评估
	riskLevel := ds.assessRisk(r.Method, r.Host, r.Host)
	fmt.Printf("  风险等级: %s\n", riskLevel)

	// 审计日志
	auditLog := &AuditLog{
		Timestamp:  startTime,
		RequestID:  requestID,
		Method:     r.Method,
		URL:        r.Host,
		Host:       r.Host,
		RiskLevel:  riskLevel,
		ClientAddr: r.RemoteAddr,
	}

	// 低风险自动放行
	if riskLevel == "low" {
		color.Green("  ✓ 低风险，自动放行")
		auditLog.Status = "approved"
		ds.proxyHTTPSConnect(w, r)
	} else {
		// 高风险需要审批
		color.Yellow("  ⚠️  高风险，需要审批")
		approval, err := ds.approvalManager.CreateApproval(r.Method, r.Host, r.Host, riskLevel)
		if err != nil {
			color.Red("  ✗ 创建审批失败: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			auditLog.Status = "error"
			ds.auditLogger.Log(auditLog)
			return
		}

		auditLog.ApprovalID = approval.ID

		// 等待审批结果
		approved := <-approval.Response
		if approved {
			color.Green("  ✓ 审批通过，建立连接")
			auditLog.Status = "approved"
			ds.proxyHTTPSConnect(w, r)
		} else {
			color.Red("  ✗ 审批拒绝，阻止连接")
			auditLog.Status = "rejected"
			http.Error(w, "Request Rejected", http.StatusForbidden)
		}
	}

	// 记录审计日志
	auditLog.Duration = time.Since(startTime).Milliseconds()
	ds.auditLogger.Log(auditLog)
}

// 实际执行 HTTPS CONNECT
func (ds *DitingService) proxyHTTPSConnect(w http.ResponseWriter, r *http.Request) {
	// 连接到目标服务器
	destConn, err := net.DialTimeout("tcp", r.Host, time.Duration(ds.config.Proxy.TimeoutSeconds)*time.Second)
	if err != nil {
		color.Red("  ✗ 连接目标服务器失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer destConn.Close()

	// 劫持客户端连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		color.Red("  ✗ 不支持 Hijacking")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		color.Red("  ✗ Hijack 失败: %v", err)
		return
	}
	defer clientConn.Close()

	// 发送 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 双向转发数据
	go io.Copy(destConn, clientConn)
	io.Copy(clientConn, destConn)

	color.Green("  ✓ HTTPS 连接完成")
}

// ============================================================================
// 启动代理服务器
// ============================================================================

func (ds *DitingService) startProxyServer() error {
	server := &http.Server{
		Addr: ds.config.Proxy.Listen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				ds.handleHTTPS(w, r)
			} else {
				ds.handleHTTP(w, r)
			}
		}),
	}

	color.Green("✓ 代理服务器启动成功")
	color.White("  监听地址: %s", ds.config.Proxy.Listen)
	fmt.Println()

	return server.ListenAndServe()
}

// ============================================================================
// 启动飞书长连接
// ============================================================================

func (ds *DitingService) startFeishuWebSocket() error {
	// 创建事件处理器
	handler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			color.Cyan("\n[%s] 📨 收到飞书消息", time.Now().Format("15:04:05"))

			if event.Event.Message != nil {
				msg := event.Event.Message

				if msg.MessageId != nil {
					fmt.Printf("  消息 ID: %s\n", *msg.MessageId)
				}
				if msg.ChatId != nil {
					fmt.Printf("  Chat ID: %s\n", *msg.ChatId)
				}

				// 解析文本消息
				if msg.MessageType != nil && *msg.MessageType == "text" && msg.Content != nil {
					// 解析 JSON 内容
					var content map[string]interface{}
					if err := json.Unmarshal([]byte(*msg.Content), &content); err == nil {
						if text, ok := content["text"].(string); ok {
							fmt.Printf("  内容: %s\n", text)
							// 处理审批回复
							ds.approvalManager.HandleApprovalReply(text)
						}
					}
				}
			}

			return nil
		})

	// 创建 WebSocket 客户端
	ds.wsClient = larkws.NewClient(
		ds.config.Feishu.AppID,
		ds.config.Feishu.AppSecret,
		larkws.WithEventHandler(handler),
	)

	color.Green("✓ 飞书长连接启动成功")
	fmt.Println()

	// 启动长连接
	go func() {
		if err := ds.wsClient.Start(ds.ctx); err != nil {
			color.Red("✗ 飞书长连接错误: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)
	return nil
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	// 打印欢迎信息
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v2.0.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 完整集成版            ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 加载配置
	configFile := "config.json"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		color.Red("✗ 读取配置文件失败: %v", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		color.Red("✗ 解析配置文件失败: %v", err)
		os.Exit(1)
	}

	// 添加默认 Chat ID（如果配置中没有）
	if config.Feishu.ChatID == "" {
		config.Feishu.ChatID = "oc_2ffdc43f1b0b8fbde82e1548f2ae6ed4"
	}

	color.Green("✓ 配置加载成功")
	color.White("  App ID: %s", config.Feishu.AppID)
	color.White("  Chat ID: %s", config.Feishu.ChatID)
	color.White("  代理端口: %s", config.Proxy.Listen)
	fmt.Println()

	// 创建 Lark 客户端
	larkClient := lark.NewClient(config.Feishu.AppID, config.Feishu.AppSecret)

	// 创建审计日志记录器
	auditLogger, err := NewAuditLogger(&config.Audit)
	if err != nil {
		color.Red("✗ 创建审计日志记录器失败: %v", err)
		os.Exit(1)
	}
	defer auditLogger.Close()

	color.Green("✓ 审计日志记录器初始化成功")
	if config.Audit.Enabled {
		color.White("  日志文件: %s", config.Audit.LogFile)
	}
	fmt.Println()

	// 创建服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := &DitingService{
		config:          &config,
		larkClient:      larkClient,
		approvalManager: NewApprovalManager(larkClient, &config),
		auditLogger:     auditLogger,
		ctx:             ctx,
		cancel:          cancel,
	}

	// 启动飞书长连接
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接...")
	if err := service.startFeishuWebSocket(); err != nil {
		color.Red("✗ 启动飞书长连接失败: %v", err)
		os.Exit(1)
	}

	// 启动代理服务器
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🚀 启动代理服务器...")
	go func() {
		if err := service.startProxyServer(); err != nil {
			color.Red("✗ 代理服务器错误: %v", err)
		}
	}()

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Green("✓ Diting 治理网关已启动")
	color.White("  等待请求和审批消息...")
	fmt.Println()

	// 等待中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt
	color.Yellow("\n收到中断信号，正在关闭...")
	cancel()
	time.Sleep(1 * time.Second)
	color.Green("✓ 服务已停止")
}
