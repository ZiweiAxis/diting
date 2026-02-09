# 飞书 WebSocket 长连接问题修复报告

## 问题诊断

### 当前问题
- Diting 服务显示"获取 endpoint 失败: 响应格式错误"
- 飞书提示"应用未建立长连接"
- API 调用返回 `404 page not found`

### 根本原因
**飞书开放平台未启用「长连接」功能**

通过诊断工具测试发现：
```bash
POST https://open.feishu.cn/open-apis/im/v1/stream/get
HTTP 状态码: 404
响应: 404 page not found
```

这表明：
1. 应用在飞书开放平台未开启「事件订阅 - 长连接」功能
2. 或者该应用类型不支持长连接（某些应用类型只支持 HTTP 回调）

## 解决方案

### 方案 1: 启用飞书长连接（推荐）

#### 步骤：
1. 登录飞书开放平台: https://open.feishu.cn/app
2. 找到你的应用: `xxxx`
3. 进入「事件订阅」配置页面
4. 选择「长连接」模式
5. 启用长连接功能
6. 添加需要订阅的事件：
   - `im.message.receive_v1` (接收消息)
7. 保存配置

#### 验证：
启用后，运行诊断工具：
```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
./diagnose_feishu.sh
```

应该看到：
```
✅ API 调用成功
✅ WebSocket URL: wss://...
```

### 方案 2: 使用 HTTP 回调模式（备选）

如果应用不支持长连接，可以使用 HTTP 回调模式。

#### 优点：
- 更简单，无需维护 WebSocket 连接
- 更稳定，飞书主动推送事件

#### 缺点：
- 需要公网可访问的 URL
- 需要处理事件验证

#### 实现：
见 `main_http_callback.go`（已创建）

## 代码修复

### 主要改进

#### 1. 改进的 `getFeishuWSEndpoint()` 函数

```go
func getFeishuWSEndpoint() (string, error) {
    token, err := getFeishuToken()
    if err != nil {
        return "", fmt.Errorf("获取 token 失败: %v", err)
    }

    apiURL := "https://open.feishu.cn/open-apis/im/v1/stream/get"
    reqBody := []byte("{}")

    req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
    if err != nil {
        return "", fmt.Errorf("创建请求失败: %v", err)
    }

    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("请求失败: %v", err)
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("读取响应失败: %v", err)
    }

    // 检查 HTTP 状态码
    if resp.StatusCode == 404 {
        return "", fmt.Errorf("API 端点不存在 (404)，请在飞书开放平台启用事件订阅功能")
    }

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("HTTP 状态码错误: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
    }

    // 使用结构体解析
    var wsResp FeishuWSResponse
    if err := json.Unmarshal(bodyBytes, &wsResp); err != nil {
        return "", fmt.Errorf("解析响应失败: %v", err)
    }

    if wsResp.Code != 0 {
        return "", fmt.Errorf("飞书 API 错误 (code=%d): %s", wsResp.Code, wsResp.Msg)
    }

    if wsResp.Data.URL == "" {
        return "", fmt.Errorf("响应中未找到 WebSocket URL")
    }

    return wsResp.Data.URL, nil
}
```

#### 2. Token 缓存机制

```go
var (
    feishuToken      string
    feishuTokenMutex sync.RWMutex
    feishuTokenExpiry time.Time
)

func getFeishuToken() (string, error) {
    feishuTokenMutex.RLock()
    if feishuToken != "" && time.Now().Before(feishuTokenExpiry) {
        token := feishuToken
        feishuTokenMutex.RUnlock()
        return token, nil
    }
    feishuTokenMutex.RUnlock()

    // ... 获取新 token ...
    
    // 提前 5 分钟过期
    feishuTokenExpiry = time.Now().Add(time.Duration(expire-300) * time.Second)
    return token, nil
}
```

#### 3. 详细的调试日志

```go
log.Printf("  [DEBUG] API 响应状态码: %d", resp.StatusCode)
log.Printf("  [DEBUG] API 响应内容: %s", string(bodyBytes))
log.Printf("  [DEBUG] 发送心跳: %d", time.Now().Unix())
log.Printf("  [DEBUG] 收到消息: %s", string(message))
```

## 测试步骤

### 1. 备份原文件
```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
cp main_ws.go main_ws.backup.go
```

### 2. 使用修复版本
```bash
cp main_ws_fixed.go main_ws.go
```

### 3. 运行诊断工具
```bash
./diagnose_feishu.sh
```

### 4. 启动服务
```bash
go run main_ws.go
```

### 5. 观察日志
应该看到：
```
🔗 启动飞书长连接...
  [DEBUG] API 响应状态码: 200
  ✓ 获取 endpoint 成功
    wss://...
  ✓ WebSocket 连接已建立
```

## 常见问题

### Q1: 仍然返回 404
**A:** 需要在飞书开放平台启用长连接功能（见方案 1）

### Q2: 返回权限错误
**A:** 检查应用权限：
- `im:message` (读取消息)
- `im:message:send_as_bot` (发送消息)

### Q3: WebSocket 连接后立即断开
**A:** 检查心跳机制是否正常工作

### Q4: 收不到消息
**A:** 
1. 确认已订阅 `im.message.receive_v1` 事件
2. 先给机器人发送一条消息建立会话
3. 检查机器人是否在群组中

## 文件清单

- `main_ws_fixed.go` - 修复后的主程序
- `diagnose_feishu.sh` - 诊断工具
- `test_api.sh` / `test_api2.sh` - API 测试脚本
- `FEISHU_WEBSOCKET_FIX.md` - 本文档

## 下一步

1. 在飞书开放平台启用长连接
2. 运行诊断工具验证
3. 替换主程序文件
4. 重启服务测试

## 联系支持

如果问题仍未解决：
1. 检查飞书开放平台应用配置
2. 查看完整的 API 响应日志
3. 确认应用类型是否支持长连接
