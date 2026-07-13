# xai-autoban

CLIProxyAPI 原生插件：自动隔离异常 xAI 凭据，支持可配置 ban 时长/动作、定时+手动巡检、disable/delete（best-effort）、管理面板。

版本：**0.3.0**

## 功能

| 上游状态 | 默认时长 | 默认动作 |
| --- | --- | --- |
| `401` | 24h | ban |
| `402` | 7d | ban |
| `403` | 24h | ban |
| `429` | Retry-After / 限流头，否则 30m | ban |

增强：

- 可配置 ban 时长与 `ban|disable|delete` 动作
- 定时 + 手动巡检（`models` / `responses_mini`）
- 探测成功动作：`none|unban|reenable|unban_and_reenable`
- 动作冷却、并发/QPS 限速、脱敏审计、可选 `state_file` 持久化
- scheduler 过滤 ban 后委托 CPA 内置调度（默认 round-robin）
- **敏感操作仅 Management API**；资源页只读，同源读取管理密钥后调用管理接口

## 构建

需要 Go 1.21+、CGO、C 编译器。

```bash
bash build.sh
```

Debian 12（兼容官方镜像）：

```bash
docker run --rm \
  -v "$PWD:/src" \
  -w /src \
  golang:1.24-bookworm \
  sh -c 'go test ./... && CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o dist/xai-autoban.so .'
```

## 安装

```text
plugins/linux/amd64/xai-autoban.so
plugins/windows/amd64/xai-autoban.dll
```

```yaml
plugins:
  enabled: true
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

## 面板与 API

资源页（只读数据 + UI）：

```text
http://<CPA_HOST>:8317/v0/resource/plugins/xai-autoban/status
GET /v0/resource/plugins/xai-autoban/data
```

管理 API（需要管理密钥）：

```text
GET  /v0/management/plugins/xai-autoban/bans
GET  /v0/management/plugins/xai-autoban/audit
POST /v0/management/plugins/xai-autoban/unban
POST /v0/management/plugins/xai-autoban/unban-all
POST /v0/management/plugins/xai-autoban/import
POST /v0/management/plugins/xai-autoban/probe
POST /v0/management/plugins/xai-autoban/apply-action
```

公开 `/action` 解禁接口已移除。

## 说明

- 仅处理 `xai` provider。
- 内存 ban 默认重启清空；配置 `state_file` 可持久化。
- `delete` 无宿主正式删除回调时回退 `delete_fallback`（默认 disable）并标记 `pending_delete`。
- 探测成功默认只 `unban`，不自动 re-enable。

## License

MIT
