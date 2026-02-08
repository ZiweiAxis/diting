package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

const (
	appID     = "cli_a90d5a960cf89cd4"
	appSecret = "8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"
)

func main() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v0.7.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书长连接            ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  App ID: %s", appID)
	fmt.Println()

	// 获取 tenant_access_token
	token, err := getTenantAccessToken()
	if err != nil {
		color.Red("✗ 获取 token 失败: %v", err)
		os.Exit(1)
	}
	color.Green("✓ Token 获取成功: %s...", token[:20])

	// 获取 WebSocket endpoint
	wsURL, err := getWebSocketEndpoint(token)
	if err != nil {
		color.Red("✗ 获取 WebSocket endpoint 失败: %v", err)
		os.Exit(1)
	}
	color.Green("✓ WebSocket endpoint: %s", wsURL)

	color.Cyan("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 建立 WebSocket 连接...")

	// 连接 WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		color.Red("✗ WebSocket 连接失败: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	color.Green("✓ WebSocket 连接已建立")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 启动心跳
	go sendHeartbeat(conn)

	// 处理中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// 接收消息
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				color.Red("\n✗ 读取消息失败: %v", err)
				return
			}

			handleMessage(message)
		}
	}()

	// 等待中断或连接关闭
	select {
	case <-done:
		color.Yellow("\n连接已关闭")
	case <-interrupt:
		color.Yellow("\n收到中断信号，正在关闭...")
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}

// 获取 tenant_access_token
func getTenantAccessToken() (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	})

	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, ok := result["tenant_access_token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}

	return token, nil
}

// 获取 WebSocket endpoint
func getWebSocketEndpoint(token string) (string, error) {
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/stream/get", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	
	// 打印原始响应用于调试
	fmt.Printf("  API 响应: %s\n", string(bodyBytes))

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查错误码
	if code, ok := result["code"].(float64); ok && code != 0 {
		return "", fmt.Errorf("API 错误: code=%v, msg=%v", code, result["msg"])
	}

	// 获取 URL
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data 字段不存在")
	}

	url, ok := data["url"].(string)
	if !ok {
		return "", fmt.Errorf("url 字段不存在")
	}

	return url, nil
}

// 发送心跳
func sendHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		heartbeat := map[string]interface{}{
			"type": "PING",
		}

		if err := conn.WriteJSON(heartbeat); err != nil {
			color.Red("发送心跳失败: %v", err)
			return
		}
		color.White("[%s] ❤️  心跳", time.Now().Format("15:04:05"))
	}
}

// 处理接收到的消息
func handleMessage(message []byte) {
	var event map[string]interface{}
	if err := json.Unmarshal(message, &event); err != nil {
		color.Red("解析消息失败: %v", err)
		return
	}

	// 打印消息类型
	eventType, _ := event["type"].(string)
	
	switch eventType {
	case "PONG":
		// 心跳响应
		color.White("[%s] 💓 PONG", time.Now().Format("15:04:05"))
	case "EVENT_CALLBACK":
		// 事件回调
		handleEventCallback(event)
	default:
		// 其他消息
		color.Cyan("\n[%s] 📩 收到消息", time.Now().Format("15:04:05"))
		prettyJSON, _ := json.MarshalIndent(event, "  ", "  ")
		fmt.Printf("  %s\n", string(prettyJSON))
	}
}

// 处理事件回调
func handleEventCallback(event map[string]interface{}) {
	header, ok := event["header"].(map[string]interface{})
	if !ok {
		return
	}

	eventType, _ := header["event_type"].(string)
	
	color.Cyan("\n[%s] 📨 收到事件: %s", time.Now().Format("15:04:05"), eventType)

	if eventType == "im.message.receive_v1" {
		handleMessageReceive(event)
	}
}

// 处理接收到的消息事件
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
	chatID, _ := message["chat_id"].(string)
	
	fmt.Printf("  Chat ID: %s\n", chatID)
	fmt.Printf("  消息类型: %s\n", messageType)

	if messageType == "text" {
		content, _ := message["content"].(string)
		var textContent map[string]string
		json.Unmarshal([]byte(content), &textContent)
		text := textContent["text"]
		
		fmt.Printf("  内容: %s\n", text)
		
		color.Green("  ✓ 消息接收成功")
	}
}
