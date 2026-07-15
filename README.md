# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持可配置策略、定时/手动巡检、禁用/删除、refresh 重授权与运维台。

版本：**0.5.37**

## 功能

| 上游状态 | 默认时长 | 默认动作 |
| --- | --- | --- |
| `401` / reauth / token 过期 | 24h | 隔离 |
| `402` / 额度用尽 | 7d | 按 `action_on_402` |
| `403` | 24h | 隔离 |
| `429` 裸限流 | Retry-After 或 30m | **仅隔离** |

- 仅处理 `xai` provider
- Usage 失败钩子 + Scheduler 跳过隔离账本 + Probe 巡检
- 语义分类：裸 429 vs free-usage vs reauth
- Management **真删除**；失败则禁用/隔离 + `pending_delete`
- 异步巡检 + 进度；含已禁用 / 仅已禁用
- **reauth**：`refresh_token` → `auth.x.ai` + 实探 `/models`
- 运维台：按状态筛选、**全选当前筛选**、批量隔离/禁用/启用/删除/复检

### 与 cpa-auth-inspect

| 插件 | 职责 |
|------|------|
| **xai-autoban** | 运行时熔断、调度隔离、巡检、refresh 重授权 |
| **[cpa-auth-inspect](https://github.com/YOUYCG/cpa-auth-inspect)** | 多厂商巡检、Chromium Device OAuth 自动重登 |

## 安装

### 方式 A：插件商店（推荐）

`config.yaml`：

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
      disable_via: management_api
      management_key_env: CPA_MANAGEMENT_KEY
```

重启 CPA → **插件商店 → xAI Autoban → 安装**。

运维台：

```text
/v0/resource/plugins/xai-autoban/status
```

（CPA-Manager-Plus 下从「插件管理」打开同一页面即可。）

### 方式 B：手动构建

需要 Go 1.21+、CGO 与 C 编译器。

```bash
# Linux / macOS
./scripts/build.sh

# Windows PowerShell
powershell -File scripts/build.ps1
```

产物放到：

```text
plugins/linux/amd64/xai-autoban.so
plugins/windows/amd64/xai-autoban.dll
plugins/darwin/arm64/xai-autoban.dylib
```

## 配置入口

| 入口 | 用途 |
|------|------|
| **运维台 → 编辑配置** | 日常：巡检、策略、动作 |
| **插件管理** | 启用 + 服务端密钥（`management_key` / `management_key_env` / `management_url` / `disable_via`） |

### 密钥（CPA-Manager-Plus）

| 密钥 | 用途 | 填进本插件？ |
|------|------|----------------|
| CPAMP Admin Key（`cpamp_...`） | 登录 CPAMP 面板 | **否** |
| CPA Management Key（`remote-management.secret-key`） | 禁用/删除凭证 | **是** |

运维台写操作（取消隔离、保存配置、复检等）在 CPAMP 下走 **resource GET**（`/ops`），浏览器无需粘贴密钥。  
插件内 `management_key` 仅给**插件进程**调用 CPA 做禁用/删除用。

## License

MIT
