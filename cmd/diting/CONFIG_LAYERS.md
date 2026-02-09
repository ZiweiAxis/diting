# 配置说明：config.yaml + .env（已收敛）

仅保留两层配置，其余已收敛。

---

## 1. 主配置：config.yaml

- 程序加载**一个** YAML 文件。
- 默认选择：存在 **config.yaml** 则用其，否则用 **config.example.yaml**。
- 也可通过 **-config 文件路径** 或环境变量 **CONFIG_PATH** 指定。
- 仓库中只提交 **config.example.yaml**（模板）；本地复制为 **config.yaml** 并已加入 .gitignore。

## 2. 覆盖：.env

- 主配置中的敏感项与常用项由**环境变量**覆盖（如 DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET 等）。
- **.env** 在启动时被读取，写入环境变量，相当于「本地的 env 覆盖源」；不提交。
- 复制 **.env.example** 为 **.env** 并填写 DITING_* 即可。

## 3. 加载顺序（main）

1. `LoadEnvFile(".env", true)` → 把 .env 写入环境变量  
2. 确定 YAML 路径：`-config` > `CONFIG_PATH` > config.yaml（若存在）> config.example.yaml  
3. `config.Load(yamlPath)` → 读 YAML，再 `applyEnvOverrides` 用环境变量覆盖  

**已移除**：不再读取 config.json；不再使用 config.local.yaml / config.acceptance.yaml 的默认链。

---

## 4. 为何仓库里看不到 .env？

**.env 不提交**（在 .gitignore 中），只保留 **.env.example** 作为模板。你要做的是：**复制 .env.example 为 .env**，在本地填写飞书等私密项。原 acceptance 里的私密配置（app_id、app_secret、approval_user_id、chat_id 等）都应放在 **.env** 中，不要写入 config.yaml。

## 5. 首次使用

```bash
cd cmd/diting
cp config.example.yaml config.yaml
cp .env.example .env
# 编辑 .env，至少填写：DITING_FEISHU_APP_ID、DITING_FEISHU_APP_SECRET、DITING_FEISHU_APPROVAL_USER_ID（或 DITING_FEISHU_CHAT_ID）
./bin/diting   # 或 make watch / make run
```
