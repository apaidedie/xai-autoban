# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持可配置 ban 时长/动作、定时/手动巡检、disable/delete、refresh 重授权、管理面板。

版本：**0.5.26**

## 方式 A：插件商店安装（推荐）

### 1. 配置 CPA

在 CPA 的 `config.yaml` 中启用插件并添加本仓库为插件源：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  store-sources:
    - "https://raw.githubusercontent.com/apaidedie/xai-autoban/main/registry.json"
  configs:
    xai-autoban:
      enabled: true
      priority: 200
```

### 2. 重启 CPA

### 3. 安装插件

进入 **CPA 管理中心 → 插件商店 → 找到 xAI Autoban → 点击安装**

或使用 API：

```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  "http://YOUR_CPA_HOST:8317/v0/management/plugin-store/xai-autoban/install"
```

### 4. 打开面板

```text
http://YOUR_CPA_HOST:8317/v0/resource/plugins/xai-autoban/status
```

写操作需要管理密钥。

## 方式 B：手动安装

### 构建

需要 Go 1.21+、CGO 与 C 编译器。

```bash
# Linux / macOS
./scripts/build.sh

# Windows PowerShell
powershell -File scripts/build.ps1
```

或 Debian 容器构建（兼容官方镜像）：

```bash
docker run --rm \
  -v "$PWD:/src" \
  -w /src \
  golang:1.24-bookworm \
  sh -c 'go test ./... && CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o dist/xai-autoban.so .'
```

放置：

```text
plugins/linux/amd64/xai-autoban.so
plugins/windows/amd64/xai-autoban.dll
plugins/darwin/arm64/xai-autoban.dylib
```

## Docker 安装（摘要）

1. 构建产物（`scripts/build.sh` → `.so`）
2. 挂载到 CPA `plugins/`
3. 启用 `plugins.enabled` 与 `configs.xai-autoban`
4. 设置 `CPA_MANAGEMENT_KEY`（disable/delete/reauth）
5. 打开 `/v0/resource/plugins/xai-autoban/status`

## 功能概要

| 上游状态 | 默认时长 | 默认动作 |
| --- | --- | --- |
| `401` / reauth / token 过期 | 24h | ban |
| `402` / 额度用尽 | 7d | 按 `action_on_402` |
| `403` | 24h | ban |
| `429` 裸限流 | Retry-After 或 30m | **仅 ban** |

- 仅处理 `xai` provider
- Usage 失败钩子 + Scheduler 跳过 ban + Probe 巡检
- 语义分类（body）：裸 429 vs free-usage-exhausted vs reauth
- Management **真 DELETE**；失败则 disable/ban + `pending_delete`
- 异步巡检 job + 进度；`probe_include_disabled` / `probe_only_disabled`
- **reauth**：`refresh_token` 直连 OAuth 刷新 + 实探 `/models`（无浏览器 OAuth）

### 与 cpa-auth-inspect 的分工

- **xai-autoban**：运行时熔断、调度隔离、巡检、refresh 重授权
- **[cpa-auth-inspect](https://github.com/YOUYCG/cpa-auth-inspect)**：多厂商巡检、Chromium Device OAuth 自动重登

## 配置入口（推荐）

| 入口 | 用途 |
|------|------|
| **运维台 → 编辑配置** | **主用**：巡检、策略、动作等日常配置 |
| **插件管理** | 仅：**启用**开关 + **服务端管理密钥**（`management_key_env` / `management_key` / `management_url` / `disable_via`） |

插件管理不再展示 ban 时长、probe 等长表单，避免与运维台重复。

### CPA-Manager-Plus（CPAMP）双密钥说明

| 密钥 | 用途 | 是否填进本插件 |
|------|------|----------------|
| **CPAMP Admin Key**（`cpamp_...`） | 登录 CPAMP 面板 | **否** |
| **CPA Management Key**（`remote-management.secret-key`） | CPAMP 连 CPA；本插件禁用/删除凭证 | **是** → `management_key` / `CPA_MANAGEMENT_KEY` |

运维台写操作优先走 `GET /v0/resource/plugins/xai-autoban/data?op=...`（CPAMP 对 resource **GET** 会自动用已保存的 CPA 密钥代理，浏览器无需密钥）。  
不要把 `cpamp_...` 填进插件 `management_key`。

### 安装时最小 yaml

```yaml
plugins:
  configs:
    xai-autoban:
      enabled: true
      priority: 200
      disable_via: management_api
      management_url: http://127.0.0.1:8317
      management_key_env: CPA_MANAGEMENT_KEY
```

其余策略在运维台保存；也可继续在 yaml 写全量字段（仍会解析，只是插件管理 UI 不展示）。

## License

MIT
