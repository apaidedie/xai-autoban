# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持策略配置、定时/手动巡检、禁用/删除、refresh 重授权与运维台。

版本：**1.1.2**（Stable · maintenance）

> **稳定性契约：** [STABILITY.md](./STABILITY.md) — 保证/不保证、配置冻结表、运维入口。  
> **1.x 政策：** 不删/不改名冻结配置键（破坏性变更需 major）；默认策略变更写 CHANGELOG。

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
| `auto_using_api` | `off` | 探测/复检失败是否自动写 `using_api`：`off`（默认，更安全）/ `on_403` / `on_fail` |

## 功能

- 运维台：筛选 / 全选 / 批量释放·隔离·禁用·启用·API 模式·删除 / 复检 / 巡检
- 大号池：增量巡检（跳过近期成功、每轮限批）、using_api 缓存与后台刷新
- 策略预设（保守/标准/激进）；导出需重授/待删（给 cpa-auth-inspect）
- `auto_using_api` 默认 **off**（更安全）
- Management 真删除（失败则禁用/隔离 + `pending_delete`）
- reauth：`refresh_token` → `auth.x.ai`
- 状态持久化：`state_file`（默认 `xai-autoban-state.json`，运行时解析为绝对路径；运维台配置 + 隔离账本）
- 兼容 **CPA-Manager-Plus**（resource 写通道）

## 目录结构

```
main.go / abi_cgo.go     # 插件入口与 CGO ABI
internal/
  action/   隔离动作、禁用/删除/using_api、Management 客户端
  ban/      隔离账本
  classify/ 上游语义分类
  config/   配置默认值与归一化
  creds/    凭证列表投影 + using_api 缓存
  host/     CPA host 回调
  mgmt/     管理路由 / 状态组装 / bulk / export
  probe/    巡检 / 复检 / 增量调度 / bulk 进度
  reauth/   refresh_token 刷新
  schedule/ 选号跳过隔离
  ui/       运维台（status + css/body/script）
  usage/    实时 usage 成功/失败
docs/superpowers/        # 现行设计/计划
docs/archive/            # 历史 plan/spec
scripts/                 # build.sh / build.ps1
.github/workflows/       # Release CI
registry.json            # 插件商店
STABILITY.md             # 1.x 契约
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

### 状态文件（运维台配置 + 隔离账本）

相对路径会解析为**绝对路径**，优先已有文件与可写数据目录：

1. 环境变量：`XAI_AUTOBAN_DATA_DIR` → `CPA_DATA_DIR` → `CLIPROXYAPI_DATA_DIR` → `DATA_DIR` → `CPA_HOME`
2. 可执行文件旁 `data/`、用户 config、工作目录 `data/`
3. 也可在插件配置写绝对 `state_file`

Docker / CPA 重建时请**挂载**该目录，否则运维台设置与隔离账本会丢失。运维台配置区会显示当前路径。

### CPA-Manager-Plus 密钥

| 密钥 | 用途 | 填本插件？ |
|------|------|------------|
| `cpamp_...` 面板密钥 | 登录 CPAMP | **否** |
| CPA `remote-management.secret-key` | 禁用/删除凭证 | **是** |

## License

MIT
