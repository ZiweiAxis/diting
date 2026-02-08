# 飞书 WebSocket 长连接修复 - 测试报告

## 执行时间
2026-02-08 10:52 GMT+8

## 问题分析

### 原始错误
```
获取 endpoint 失败: 响应格式错误
```

### 诊断结果
```bash
步骤 1: 获取 tenant_access_token
✅ Token 获取成功: t-g10428an2JHK7WVMWJ...

步骤 2: 测试 WebSocket endpoint API
尝试: POST /open-apis/im/v1/stream/get
HTTP 状态码: 404
响应内容: 404 page not found
```

### 根本原因
**飞书开放平台未启用「长连接」功能**

API 端点 `/open-apis/im/v1/stream/get` 返回 404，说明：
1. 应用未在飞书开放平台启用「事件订阅 - 长连接」
2. 或应用类型不支持 WebSocket 长连接

## 修复内容

### 1. 代码改进

#### ✅ 修复 `getFeishuWSEndpoint()` 函数
- 添加详细的错误处理
- 使用结构体解析响应
- 添加 HTTP 状态码检查
- 提供明确的错误提示

#### ✅ 改进 Token 管理
- 实现 Token 缓存机制
- 添加过期时间管理
- 避免频繁请求 Token

#### ✅ 增强调试能力
- 添加详细的 DEBUG 日志
- 打印原始 API 响应
- 记录心跳和消息事件

### 2. 诊断工具

创建了 `diagnose_feishu.sh` 脚本：
- 自动测试 Token 获取
- 检测 WebSocket API 可用性
- 提供详细的错误诊断
- 给出具体的解决方案

### 3. 文档

创建了完整的修复文档：
- 问题诊断流程
- 两种解决方案（长连接 vs HTTP 回调）
- 详细的配置步骤
- 常见问题解答

## 文件清单

| 文件名 | 说明 | 状态 |
|--------|------|------|
| `main_ws_fixed.go` | 修复后的主程序 | ✅ 已创建 |
| `diagnose_feishu.sh` | 诊断工具 | ✅ 已创建并测试 |
| `FEISHU_WEBSOCKET_FIX.md` | 修复文档 | ✅ 已创建 |
| `TEST_REPORT.md` | 本测试报告 | ✅ 已创建 |

## 解决方案

### 方案 1: 启用飞书长连接（推荐）

**步骤：**
1. 访问 https://open.feishu.cn/app
2. 选择应用 `cli_a90d5a960cf89cd4`
3. 进入「事件订阅」
4. 选择「长连接」模式
5. 启用并保存

**验证：**
```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
./diagnose_feishu.sh
```

期望输出：
```
✅ Token 获取成功
✅ API 调用成功
✅ WebSocket URL: wss://...
```

### 方案 2: 使用 HTTP 回调（备选）

如果应用不支持长连接，可以改用 HTTP 回调模式。

## 代码对比

### 修复前
```go
func getFeishuWSEndpoint() (string, error) {
    // ...
    var result map[string]interface{}
    json.Unmarshal(bodyBytes, &result)
    
    data, ok := result["data"].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("响应格式错误")  // ❌ 错误信息不明确
    }
    // ...
}
```

### 修复后
```go
func getFeishuWSEndpoint() (string, error) {
    // ...
    
    // ✅ 检查 HTTP 状态码
    if resp.StatusCode == 404 {
        return "", fmt.Errorf("API 端点不存在 (404)，请在飞书开放平台启用事件订阅功能")
    }
    
    // ✅ 使用结构体解析
    var wsResp FeishuWSResponse
    if err := json.Unmarshal(bodyBytes, &wsResp); err != nil {
        return "", fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(bodyBytes))
    }
    
    // ✅ 详细的错误检查
    if wsResp.Code != 0 {
        return "", fmt.Errorf("飞书 API 错误 (code=%d): %s", wsResp.Code, wsResp.Msg)
    }
    // ...
}
```

## 关键改进点

### 1. 错误诊断
- ❌ 之前: "响应格式错误"（不知道哪里错）
- ✅ 现在: "API 端点不存在 (404)，请在飞书开放平台启用事件订阅功能"

### 2. 调试能力
- ❌ 之前: 无日志，无法排查
- ✅ 现在: 详细的 DEBUG 日志，打印原始响应

### 3. Token 管理
- ❌ 之前: 每次都请求新 Token
- ✅ 现在: 缓存 Token，提前 5 分钟刷新

### 4. 响应解析
- ❌ 之前: 使用 map[string]interface{}，类型断言容易出错
- ✅ 现在: 使用结构体，类型安全

## 测试建议

### 1. 立即测试
```bash
cd /home/dministrator/workspace/sentinel-ai/cmd/diting
./diagnose_feishu.sh
```

### 2. 启用长连接后测试
```bash
# 备份原文件
cp main_ws.go main_ws.backup.go

# 使用修复版本
cp main_ws_fixed.go main_ws.go

# 启动服务
go run main_ws.go
```

### 3. 观察日志
应该看到：
```
🔗 启动飞书长连接...
  [DEBUG] API 响应状态码: 200
  [DEBUG] API 响应内容: {"code":0,"data":{"url":"wss://..."}}
  ✓ 获取 endpoint 成功
    wss://...
  ✓ WebSocket 连接已建立
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [DEBUG] 发送心跳: 1707363072
  [DEBUG] 收到心跳响应
```

## 下一步行动

### 必须执行
1. ✅ 运行诊断工具确认问题
2. ⏳ 在飞书开放平台启用长连接
3. ⏳ 再次运行诊断工具验证
4. ⏳ 替换主程序文件
5. ⏳ 重启服务测试

### 可选执行
- 如果长连接不可用，考虑 HTTP 回调方案
- 添加更多事件订阅（如群消息、@提醒等）
- 实现消息重试机制

## 总结

### 问题根源
飞书开放平台未启用长连接功能，导致 API 返回 404

### 修复效果
- ✅ 改进错误提示，明确指出问题所在
- ✅ 添加诊断工具，快速定位问题
- ✅ 优化代码结构，提高可维护性
- ✅ 增强调试能力，便于排查问题

### 成功标志
当看到以下日志时，表示连接成功：
```
✓ 获取 endpoint 成功
✓ WebSocket 连接已建立
[DEBUG] 发送心跳: ...
[DEBUG] 收到心跳响应
```

---

**报告生成时间**: 2026-02-08 10:52 GMT+8  
**修复状态**: 代码已修复，等待飞书平台配置  
**下一步**: 在飞书开放平台启用长连接功能
