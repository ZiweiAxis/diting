package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

const (
	appID     = "cli_a90d5a960cf89cd4"
	appSecret = "8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"
)

func main() {
	color.Cyan("╔════════════════════════════════════════════════════════╗")
	color.Cyan("║         Diting 治理网关 v0.9.0                        ║")
	color.Cyan("║    企业级智能体零信任治理平台 - 飞书长连接            ║")
	color.Cyan("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.Green("✓ 配置加载成功")
	color.White("  App ID: %s", appID)
	fmt.Println()

	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	color.Yellow("🔗 启动飞书长连接...")

	// 创建飞书客户端
	client := lark.NewClient(appID, appSecret)

	// 创建 WebSocket 客户端
	cli := larkws.NewClient(appID, appSecret,
		larkws.WithEventHandler(func(ctx context.Context, event *larkws.Event) error {
			color.Cyan("\n[%s] 📨 收到事件", time.Now().Format("15:04:05"))
			fmt.Printf("  事件类型: %s\n", event.Header.EventType)
			
			// 打印完整事件
			color.White("  事件内容: %+v\n", event)
			
			return nil
		}),
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

	time.Sleep(2 * time.Second)
	color.Green("  ✓ WebSocket 客户端已启动")
	color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 等待中断或错误
	select {
	case err := <-errChan:
		color.Red("✗ 长连接错误: %v", err)
	case <-interrupt:
		color.Yellow("\n收到中断信号，正在关闭...")
		cancel()
	}
	
	_ = client
}
