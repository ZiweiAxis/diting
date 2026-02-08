package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func main() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v0.6.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书 SDK 集成         ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	appID := "cli_a90d5a960cf89cd4"
	appSecret := "8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"

	color.Green("✓ 配置加载成功")
	color.White("  App ID: %s", appID)
	fmt.Println()

	// 创建飞书客户端
	client := lark.NewClient(appID, appSecret)

	// 创建事件处理器
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			color.Cyan("\n[%s] 📨 收到飞书消息", time.Now().Format("15:04:05"))
			
			// 获取消息内容
			if event.Event.Message != nil {
				fmt.Printf("  消息 ID: %s\n", *event.Event.Message.MessageId)
				fmt.Printf("  Chat ID: %s\n", *event.Event.Message.ChatId)
				fmt.Printf("  消息类型: %s\n", *event.Event.Message.MessageType)
				
				// 解析文本消息
				if *event.Event.Message.MessageType == "text" {
					content := *event.Event.Message.Content
					fmt.Printf("  内容: %s\n", content)
				}
			}
			
			return nil
		})

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接...")

	// 创建 WebSocket 客户端
	cli := lark.NewEventDispatcherHandler("", "", eventHandler)

	// 启动长连接
	err := cli.Run(context.Background())
	if err != nil {
		color.Red("  ✗ 长连接失败: %v", err)
		os.Exit(1)
	}

	color.Green("  ✓ WebSocket 连接已建立")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 保持运行
	select {}
}
