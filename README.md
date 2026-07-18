# xai-autoban

CLIProxyAPI 原生插件：按状态码自动**隔离 / 禁用 / 删除**异常 xAI 凭证，支持定时巡检、复检、成功策略与运维台。

版本：**1.1.8**（Stable · maintenance）

> **稳定性契约：** [STABILITY.md](./STABILITY.md)  
> **1.x 政策：** 不删/不改名冻结配置键（破坏性变更需 major）；默认策略变更写 CHANGELOG。

## 用语（运维台统一）

| 用语 | 含义 |
|------|------|
| **隔离** | 写入插件**隔离账本**，调度跳过；可「释放」或**到期自动释放** |
| **释放** | 清除隔离账本（不打开 CPA 开关） |
| **禁用** | 关闭 CPA 凭证开关；**不**因到期自动打开 |
| **启用** | 打开 CPA 凭证开关 |
| **释放并启用** | 清账本 + 开 CPA（成功策略） |
| **巡检** | 按配置对候选凭证全量探测 |
| **复检** | 对勾选凭证探测 |
| **状态码动作** | 401/402/403/429 各自执行隔离\|禁用\|删除 |
| **失败策略** | 其它失败的兜底动作（401–429 优先状态码） |
| **真实流量** | CPA usage 成功/失败回调 |

## 工作原理

| 路径 | 作用 |
|------|------|
| **真实流量** | 失败 → 按状态码动作；成功 → 释放隔离（可选启用） |
| **Scheduler** | 跳过隔离账本中的凭证 |
| **巡检 / 复检** | `POST /responses` 探测；OAuth 走 `cli-chat-proxy.grok.com` |

### 默认状态处理

| 上游 | 默认 |
|------|------|
| 成功（真实调用） | 释放隔离；可选按成功策略启用 |
| `401` | 按 `action_on_401`（默认隔离；可配删除） |
| `402` | 按 `action_on_402`（默认隔离，到期自动释放） |
| `403` | 按 `action_on_403`（默认隔离；可配禁用，禁用不写账本） |
| `429` | 按 `action_on_429`（默认隔离，优先响应头窗口） |
| `5xx` 等 | 一般忽略 |

### 巡检上游（与 CPA 对齐）

| 凭证 | 地址 |
|------|------|
| OAuth（有 refresh_token） | `https://cli-chat-proxy.grok.com/v1` |
| API Key | `https://api.x.ai/v1` |

默认 `probe_mode=responses_mini`。

## 功能

- 运维台：筛选 / 全选 / 批量释放·隔离·禁用·启用·删除 / 复检 / 巡检
- 大号池：跳过近期成功、每轮候选全量
- 导出需重授 / 待删（给 cpa-auth-inspect）
- Management 真删除（失败则禁用/隔离 + `pending_delete`）
- reauth：`refresh_token` → `auth.x.ai`
- 状态持久化：`state_file`（运维台配置 + 隔离账本）
- 兼容 **CPA-Manager-Plus**（resource 写通道）

## 安装

### 插件商店

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

运维台：

```text
/v0/resource/plugins/xai-autoban/status
```

### 手动构建

需要 Go 1.21+、CGO。

```bash
./scripts/build.sh
powershell -File scripts/build.ps1
```

## 配置入口

| 入口 | 用途 |
|------|------|
| **运维台 → 配置** | 巡检、成功/失败策略、状态码动作 |
| **插件管理** | 启用 + `management_key` / env / url / `disable_via` |

### 状态文件

相对路径会解析为绝对路径。Docker / 重建请挂载该目录，或设 `XAI_AUTOBAN_DATA_DIR` / 绝对 `state_file`。

### CPA-Manager-Plus 密钥

| 密钥 | 用途 | 填本插件？ |
|------|------|------------|
| `cpamp_...` 面板密钥 | 登录 CPAMP | **否** |
| CPA `remote-management.secret-key` | 禁用/删除/启用 | **是** |

## License

MIT
