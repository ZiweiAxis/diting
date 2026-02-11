# 3AF gRPC 契约 (v1.0)

本目录为 3AF（AI Agent Audit & Firewall）与 Node Agent 之间的**统一通信契约**，采用 gRPC + Protobuf。方案定位：**Docker/混合环境安全内核**。核心场景：**Docker 容器内执行** 与 **本机直接执行**。

## 传输模式（双模）

| 场景     | 底层传输           | 说明 |
|----------|--------------------|------|
| **同机** | gRPC over Unix Domain Socket | Node Agent 与 3AF 同机时使用；身份由服务端通过 **SO_PEERCRED** 从连接提取，**不得信任客户端请求中的 PeerCred**。 |
| **跨机** | gRPC over TCP + mTLS | 身份使用 Profile 下发的 SessionToken 或 mTLS 证书。 |

## 主要 RPC

- **GetSandboxProfile**：冷启动拉取全量配置（边界、Hot Cache、Sudo 预授权、version、逃生策略、SessionToken）。Request 不含 PeerCred，同机时由服务端从传输层注入。
- **ExecAuth**：单次执行鉴权；支持 `env` 快照与 `audit_metadata`。
- **AuthStream**：长连接双向流。**首包为握手**（AuthStreamInit：client_id、resource、agent_version）；支持鉴权请求、异步审批 Push、**服务端主动推送 ProfileUpdate**（Hot Cache 失效/更新）、ping/pong 心跳。

## v1.0 要点

- **PeerCred**：仅由服务端从 UDS 提取；Request 中不用于信任决策。
- **SandboxProfile.version**：用于配置漂移检测与 Hot Cache 失效。
- **SudoHotCacheEntry.run_as_user**：明确以何用户执行（默认 root）。
- **Sudo 匹配**：`command_pattern` 建议 Glob；Node 侧应对 command_line 做 Tokenization 后再匹配。
- **AuthStreamResponse.profile_update**：服务端可推送配置更新，Node 应刷新本地 Hot Cache 或 TTL。

## 生成代码

需安装 `protoc` 与对应语言的 gRPC 插件。Go 示例：

```bash
# 安装 buf / protoc-gen-go / protoc-gen-go-grpc 后
cd cmd/diting
buf generate proto/
# 或
protoc -I proto --go_out=proto --go-grpc_out=proto proto/3af_exec.proto
```

具体 `go_package` 与生成路径请按仓库 go.mod 的 module 名自行调整。
