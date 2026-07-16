# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持策略配置、定时/手动巡检、禁用/删除、refresh 重授权与运维台。

版本：**0.5.45**

## 工作原理（简要）

| 路径 | 作用 |
|------|------|
| **Usage 实时** | 请求失败 → 按语义隔离；**成功 → 释放隔离**（真实流量为准） |
| **Scheduler** | 选号时跳过隔离账本中的凭证 |
| **Probe / 复检** | 主动 `POST /responses` 探测；OAuth 走 `cli-chat-proxy.grok.com`（与 CPA 一致） |

### 状态处理

| 上游 | 默认 |
|------|------|
| 成功（真实调用） | 释放隔离；可选按策略启用 |
| `401` / token 失效 | 隔离（可 reauth） |
| `402` / free-usage | **仅真实 usage 失败时**隔离；probe 402 **不**隔离 |
| `403` 软权限拒绝 | 默认连续 3 次才隔离 |
| `429` 裸限流 | 仅隔离 |
| `5xx` 等 | 一般忽略 |

### Probe 上游（与 CPA 对齐）

| 凭证 | 探测地址 | 头 |
|------|----------|-----|
| OAuth（有 refresh_token） | `https://cli-chat-proxy.grok.com/v1` | Bearer + Grok CLI 头 |
| API Key | `https://api.x.ai/v1` | Bearer |

默认 `probe_mode=responses_mini`：真实 `POST /responses`；可选 `models` 轻量列表。

## 功能

- 运维台：筛选 / 全选当前筛选 / 批量释放·隔离·禁用·启用·删除 / 复检 / 巡检配置
- Management 真删除（失败则禁用/隔离 + `pending_delete`）
- reauth：`refresh_token` → `auth.x.ai`
- 配置持久化：`xai-autoban-state.json`（默认）
- 兼容 **CPA-Manager-Plus**（resource GET 写通道）

### 与 cpa-auth-inspect

| 插件 | 职责 |
|------|------|
| **xai-autoban** | 运行时熔断、隔离、巡检、refresh 重授权 |
| **[cpa-auth-inspect](https://github.com/YOUYCG/cpa-auth-inspect)** | 多厂商巡检、Chromium Device OAuth |

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

重启 CPA → 插件商店安装 → 运维台：

```text
/v0/resource/plugins/xai-autoban/status
```

### 手动构建

需要 Go 1.21+、CGO。

```bash
./scripts/build.sh          # Linux/macOS
powershell -File scripts/build.ps1   # Windows
```

## 配置入口

| 入口 | 用途 |
|------|------|
| **运维台 → 编辑配置** | 巡检、策略、动作（主用） |
| **插件管理** | 启用 + `management_key` / env / url / `disable_via` |

### CPA-Manager-Plus 密钥

| 密钥 | 用途 | 填本插件？ |
|------|------|------------|
| `cpamp_...` 面板密钥 | 登录 CPAMP | **否** |
| CPA `remote-management.secret-key` | 禁用/删除凭证 | **是** |

## License

MIT
