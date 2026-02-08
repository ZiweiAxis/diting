package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	appID := "cli_a90d5a960cf89cd4"
	appSecret := "8M3oj4XsRD7JLX0aIgNYedzqdQgaQeUo"

	// 1. 获取 tenant_access_token
	fmt.Println("1. 获取 tenant_access_token...")
	tokenReqBody, _ := json.Marshal(map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	})

	tokenResp, err := http.Post("https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json", bytes.NewBuffer(tokenReqBody))
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	defer tokenResp.Body.Close()

	tokenBody, _ := io.ReadAll(tokenResp.Body)
	fmt.Printf("Token 响应: %s\n\n", string(tokenBody))

	var tokenResult map[string]interface{}
	json.Unmarshal(tokenBody, &tokenResult)

	token, ok := tokenResult["tenant_access_token"].(string)
	if !ok {
		fmt.Println("获取 token 失败")
		return
	}
	fmt.Printf("✓ Token: %s\n\n", token)

	// 2. 获取 WebSocket endpoint
	fmt.Println("2. 获取 WebSocket endpoint...")
	
	// 尝试不同的请求体格式
	endpointReqBody, _ := json.Marshal(map[string]interface{}{})

	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/stream/get", bytes.NewBuffer(endpointReqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("原始响应: %s\n\n", string(bodyBytes))

	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)

	// 打印完整的响应结构
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("格式化响应:\n%s\n\n", string(prettyJSON))

	// 检查响应码
	if code, ok := result["code"].(float64); ok {
		fmt.Printf("响应码: %.0f\n", code)
		if code != 0 {
			fmt.Printf("错误信息: %v\n", result["msg"])
			return
		}
	}

	// 尝试获取 endpoint
	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("✓ Data 字段存在\n")
		if url, ok := data["url"].(string); ok {
			fmt.Printf("✓ WebSocket URL: %s\n", url)
		} else {
			fmt.Printf("✗ 未找到 url 字段\n")
			fmt.Printf("Data 内容: %+v\n", data)
		}
	} else {
		fmt.Printf("✗ 未找到 data 字段\n")
	}
}
