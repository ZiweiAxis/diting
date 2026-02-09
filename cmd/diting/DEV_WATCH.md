# Watch 模式（本地开发）

修改 `.go` / `.yaml` / `.json` 后自动重新编译并重启 Diting All-in-One，无需每次手敲 `go build` 和启动命令。

## 使用

1. **安装开发依赖（含 air）**（一次性，需可访问外网）：
   ```bash
   cd cmd/diting && ./scripts/install-dev-deps.sh
   ```
   或仅安装 air：`go install github.com/air-verse/air@latest`  
   确保 `$GOPATH/bin` 或 `$HOME/go/bin` 在 `PATH` 中。

2. **启动 watch**（任选其一）：
   - 仓库根目录：`make watch`（需已安装 air）
   - 本目录（cmd/diting）：`air`
   - 无 air 时可用：`make watch-entr`（需系统有 entr，如 `apt install entr`）

3. 修改 `internal/` 或 `cmd/diting_allinone/` 下代码或 `config.example.yaml` 等，保存后应看到自动重新编译并重启进程。

**本地 watch/测试 默认配置：**  
- 仅 **config.yaml + .env**。不传 `-config` 时程序用 config.yaml（若有）否则 config.example.yaml；敏感项由 **.env** 覆盖。  
- 首次使用：`cp config.example.yaml config.yaml`，`cp .env.example .env` 并填写 DITING_FEISHU_*。

**飞书投递收不到消息时：**  
- 看启动日志：`.env 已加载`、`飞书: app_id/app_secret=?, approval_user_id或chat_id=?`。  
- 若为 `false`：在 `cmd/diting` 下复制 `.env.example` 为 `.env`，填写 `DITING_FEISHU_APP_ID`、`DITING_FEISHU_APP_SECRET`、`DITING_FEISHU_APPROVAL_USER_ID` 或 `DITING_FEISHU_CHAT_ID`。

## 配置

- 配置文件：本目录下 `.air.toml`
- 监听扩展：`go`, `yaml`, `yml`, `json`
- 排除：`bin/`, `data/`, `tmp/`, `*_test.go`
- 默认启动参数：`-config config.example.yaml`（可在 `.air.toml` 的 `full_bin` 中修改）

## 验证 watch 是否有效

1. 执行 `make watch` 或 `air`，等待首次编译完成并看到服务启动日志。
2. 在 `internal/` 下任意 `.go` 文件末尾加一空行或注释，保存。
3. 终端应出现“重新构建”并重启进程；若无，检查 `.air.toml` 的 `include_ext` / `exclude_dir` 是否覆盖该文件所在目录。

## Go 国内镜像（安装依赖时用）

若默认 `proxy.golang.org` 超时，安装脚本已内置国内代理，无需改本机配置：

- **goproxy.cn**（七牛云，主）
- **mirrors.aliyun.com/goproxy/**（阿里云，备）

本机想长期使用国内源可执行：  
`go env -w GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct`

## 更多

- 详见仓库根目录 `docs/DEVELOPMENT.md` 的 Watch 模式小节。
