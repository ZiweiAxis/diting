package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// 飞书长连接配置
type FeishuWSConfig struct {
	AppID     string
	AppSecret string
}

// 飞书事件
type FeishuEvent struct {
	Schema string                 `json:"schema"`
	Header map[string]interface{} `json:"header"`
	Event  map[string]interface{} `json:"event"`
}

// 飞书长连接客户端
type FeishuWSClient struct {
	config FeishuWSConfig
	conn   *websocket.Conn
}

// 创建长连接客户端
func NewFeishuWSClient(appID, appSecret string) *FeishuWSClient {
	return &FeishuWSClient{
		config: FeishuWSConfig{
			AppID:     appID,
			AppSecret: appSecret,
		},
	}
}

// 连接飞书长连接
func (c *FeishuWSClient) Connect() error {
	// 获取长连接地址
	endpoint, err := c.getWSEndpoint()
	if err != nil {
		return fmt.Errorf("获取长连接地址失败: %v", err)
	}

	// 建立 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return fmt.Errorf("建立 WebSocket 连接失败: %v", err)
	}

	c.conn = conn
	log.Printf("✓ 飞书长连接已建立: %s", endpoint)
	return nil
}

// 获取长连接地址
func (c *FeishuWSClient) getWSEndpoint() (string, error) {
	// 获取 tenant_access_token
	token, err := getFeishuToken()
	if err != nil {
		return "", err
	}

	// 飞书长连接地址
	// 文档: https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/events/receive
	endpoint := fmt.Sprintf("wss://open.feishu.cn/connect?token=%s", token)
	return endpoint, nil
}

// 监听消息
func (c *FeishuWSClient) Listen(handler func(FeishuEvent)) error {
	defer c.conn.Close()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("读取消息失败: %v", err)
		}

		var event FeishuEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 处理事件
		handler(event)
	}
}

// 发送心跳
func (c *FeishuWSClient) SendHeartbeat() error {
	heartbeat := map[string]interface{}{
		"type": "HEARTBEAT",
	}
	return c.conn.WriteJSON(heartbeat)
}

// 启动心跳
func (c *FeishuWSClient) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := c.SendHeartbeat(); err != nil {
			log.Printf("发送心跳失败: %v", err)
		}
	}
}
