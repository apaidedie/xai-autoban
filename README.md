# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持策略配置、定时/手动巡检、禁用/删除、refresh 重授权与运维台。

版本：**0.5.48**

> **走向 1.0：** 稳定性契约见 [STABILITY.md](./STABILITY.md)（保证/不保证、配置冻结表、1.0 清单）。

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

| 配置 | 默认 | 说明 |
|------|------|------|
| `auto_using_api` | `on_403` | 探测/复检时是否自动开 CPA「使用 API 模式」：`off` / `on_403` / `on_fail`（401/402/403） |

## 功能

- 运维台：筛选 / 全选当前筛选 / 批量释放·隔离·禁用·启用·**API 模式**·删除 / 复检 / 巡检配置
- 列表字段：API 模式 / 软 403 进度 / 最近巡检
- `auto_using_api`：探测/复检 OAuth 失败时可选自动开 API 模式（默认仅 403）
- Management 真删除（失败则禁用/隔离 + `pending_delete`）
- reauth：`refresh_token` → `auth.x.ai`
- 配置持久化：`xai-autoban-state.json`（默认；本地运行产物，已 gitignore）
- 兼容 **CPA-Manager-Plus**（resource GET 写通道）

## 目录结构

```
main.go / abi_cgo.go     # 插件入口与 CGO ABI
internal/
  action/   隔离动作、禁用/删除/using_api、Management 客户端
  ban/      隔离账本
  classify/ 上游语义分类
  config/   配置默认值与归一化
  creds/    运维台凭证列表投影
  host/     CPA host 回调
  mgmt/     管理路由 + CPAMP resource ops
  probe/    巡检 / 复检 / auto using_api
  reauth/   refresh_token 刷新
  schedule/ 选号跳过隔离
  ui/       运维台 HTML/JS
  usage/    实时 usage 成功/失败
docs/superpowers/        # 现行设计/计划
docs/archive/            # 历史 plan/spec
scripts/                 # build.sh / build.ps1
registry.json            # 插件商店
```

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
