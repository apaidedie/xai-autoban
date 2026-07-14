# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持可配置 ban 时长/动作、定时+手动巡检、disable/delete（best-effort）、管理面板。

版本：**0.4.0**

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

或使用 API 安装：

```bash
curl -X POST \
  -H "Authorization: Bearer YOUR_MANAGEMENT_KEY" \
  "http://YOUR_CPA_HOST:8317/v0/management/plugin-store/xai-autoban/install"
```

多来源同 ID 时可加 `?source=<sourceID>`。  
这里的管理密钥只用于执行安装操作。

### 4. 打开面板

```text
http://YOUR_CPA_HOST:8317/v0/resource/plugins/xai-autoban/status
```

也可在管理中心插件菜单点击 **xAI Autoban**。

> 解禁 / 巡检等写操作需要管理密钥；资源页会尝试读取同源管理中心已保存的密钥后调用 Management API。

### 运维台（0.4.0）

- 状态芯片：全部 / 健康 / 隔离 / 401 / 402 / 403 / 429 / 已禁用（点击筛选）
- 列表展示全部 xAI 凭证（不仅是隔离账本）
- 单行与批量：`unban` / `ban` / `disable` / `reenable`
- `/data` 返回 `credentials[]` 与 `counts`

---

## 方式 B：手动安装

### 构建

需要 Go 1.21+、CGO 和 C 编译器。推荐 Debian 12 构建以兼容官方镜像：

```bash
docker run --rm \
  -v "$PWD:/src" \
  -w /src \
  golang:1.24-bookworm \
  sh -c 'go test ./... && CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o dist/xai-autoban.so .'
```

### 放置动态库

```text
plugins/linux/amd64/xai-autoban.so
plugins/windows/amd64/xai-autoban.dll
plugins/darwin/arm64/xai-autoban.dylib
```

配置同上，重启 CPA。

---

## 功能

| 上游状态 | 默认时长 | 默认动作 |
| --- | --- | --- |
| `401` | 24h | ban |
| `402` | 7d | ban |
| `403` | 24h | ban |
| `429` | Retry-After / 限流头，否则 30m | ban |

- 仅处理 `xai` provider
- 调度阶段跳过 ban 凭据，并委托 CPA 内置调度
- 可配置 ban 时长与 `ban|disable|delete`
- 定时 + 手动巡检；成功动作 `none|unban|reenable|unban_and_reenable`
- 动作冷却、并发/QPS、脱敏审计、可选 `state_file`
- 敏感操作仅 Management API（公开 `/action` 已移除）
- 运维台支持按状态筛选凭证，并手动切换 ban / disable / reenable

### 常用配置

```yaml
plugins:
  configs:
    xai-autoban:
      enabled: true
      priority: 200
      ban_401_seconds: 86400
      ban_402_seconds: 604800
      ban_403_seconds: 86400
      ban_429_fallback_seconds: 1800
      action_on_401: ban
      action_on_402: ban
      action_on_403: ban
      action_on_429: ban
      probe_enabled: true
      probe_interval_seconds: 600
      probe_concurrency: 3
      probe_qps: 2
      probe_mode: models
      probe_action: ban
      probe_on_success: unban
      action_cooldown_seconds: 60
      delete_fallback: disable
      scheduler_delegate: round-robin
      state_file: ""
      audit_max_events: 200
```

## 管理 API

需要管理密钥：

```text
GET  /v0/management/plugins/xai-autoban/bans
GET  /v0/management/plugins/xai-autoban/audit
POST /v0/management/plugins/xai-autoban/unban
POST /v0/management/plugins/xai-autoban/unban-all
POST /v0/management/plugins/xai-autoban/import
POST /v0/management/plugins/xai-autoban/probe
POST /v0/management/plugins/xai-autoban/apply-action   # ban|disable|delete|reenable
PUT  /v0/management/plugins/xai-autoban/settings
```

只读资源：

```text
GET /v0/resource/plugins/xai-autoban/status
GET /v0/resource/plugins/xai-autoban/data
```

## 发布到插件商店

1. `registry.json` 放在仓库根目录（本仓库已提供）
2. 推送 tag `v0.4.0`，GitHub Actions 构建多平台 zip + `checksums.txt`
3. CPA 配置 `store-sources` 指向：

```text
https://raw.githubusercontent.com/apaidedie/xai-autoban/main/registry.json
```

Release 资产格式：

```text
xai-autoban_<version>_<goos>_<goarch>.zip
checksums.txt
```

zip 根目录直接包含 `xai-autoban.so|.dylib|.dll`。

## License

MIT（改编自 [akihitohyh/xai-autoban](https://github.com/akihitohyh/xai-autoban)）
