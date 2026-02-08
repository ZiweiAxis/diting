package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

const (
	appID     = "cli_a90d5a960cf89cd4"
	appSecret = "8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"
)

func main() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v1.0.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书长连接            ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  App ID: %s", appID)
	fmt.Println()

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
				if msg.MessageType != nil {
					fmt.Printf("  消息类型: %s\n", *msg.MessageType)
				}
				
				// 解析文本消息
				if msg.MessageType != nil && *msg.MessageType == "text" && msg.Content != nil {
					fmt.Printf("  内容: %s\n", *msg.Content)
					color.Green("  ✓ 消息接收成功")
				}
			}
			
			return nil
		})

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接...")

	// 创建 WebSocket 客户端
	cli := larkws.NewClient(appID, appSecret,
		larkws.WithEventHandler(handler),
	)

	// 启动长连接
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动
	errChan := make(chan error, 1)
	go func() {
		color.Green("  ✓ 正在连接...")
		err := cli.Start(ctx)
		if err != nil {
			errChan <- err
		}
	}()

	time.Sleep(3 * time.Second)
	color.Green("  ✓ WebSocket 客户端已启动")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	color.White("等待飞书消息...")

	// 等待中断或错误
	select {
	case err := <-errChan:
		color.Red("\n✗ 长连接错误: %v", err)
	case <-interrupt:
		color.Yellow("\n收到中断信号，正在关闭...")
		cancel()
		time.Sleep(1 * time.Second)
	}
}
