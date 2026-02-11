// 3af-exec 最小 Node Agent（Story 8.2）：exec 前调用 3AF ExecAuth，allow 则执行命令，deny 则退出。
// 用法: 3af-exec [选项] -- <命令...>  或  3af-exec <命令...>
// 环境: DITING_3AF_URL（默认 http://localhost:8080）, DITING_AGENT_TOKEN（L0 身份）, DITING_SUBJECT（默认 $USER）
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"net/http"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "用法: 3af-exec [--url URL] [--token TOKEN] [--subject SUBJECT] -- <命令...>\n")
		os.Exit(2)
	}
	baseURL := os.Getenv("DITING_3AF_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	token := os.Getenv("DITING_AGENT_TOKEN")
	subject := os.Getenv("DITING_SUBJECT")
	if subject == "" {
		subject = os.Getenv("USER")
	}
	if subject == "" {
		subject = "default"
	}
	// 解析可选参数
	for len(args) > 0 {
		switch args[0] {
		case "--":
			args = args[1:]
			goto run
		case "--url":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "缺少 --url 参数值\n")
				os.Exit(2)
			}
			baseURL = args[1]
			args = args[2:]
		case "--token":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "缺少 --token 参数值\n")
				os.Exit(2)
			}
			token = args[1]
			args = args[2:]
		case "--subject":
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "缺少 --subject 参数值\n")
				os.Exit(2)
			}
			subject = args[1]
			args = args[2:]
		default:
			goto run
		}
	}
run:
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "缺少要执行的命令\n")
		os.Exit(2)
	}
	commandLine := strings.Join(args, " ")
	action := "exec:run"
	if len(args) > 0 && (args[0] == "sudo" || strings.HasPrefix(args[0], "sudo ")) {
		action = "exec:sudo"
	}
	hostname, _ := os.Hostname()
	resource := "local://" + hostname

	reqBody := map[string]interface{}{
		"subject":      subject,
		"action":       action,
		"resource":     resource,
		"command_line": commandLine,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(baseURL, "/")+"/auth/exec", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "3af-exec: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Agent-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "3af-exec: 请求 3AF 失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result struct {
		Decision     string `json:"decision"`
		PolicyRuleID string `json:"policy_rule_id,omitempty"`
		Reason       string `json:"reason,omitempty"`
		CheqID       string `json:"cheq_id,omitempty"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result.Decision == "allow" {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "3af-exec: 执行失败: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "3af-exec: 拒绝执行 (%s) %s\n", result.PolicyRuleID, result.Reason)
	os.Exit(1)
}
