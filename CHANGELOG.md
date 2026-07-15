# Changelog

## 0.5.29 - 2026-07-15

### Feature
- **全选当前筛选**：跨页勾选当前筛选（401/403/隔离等）下全部凭证（最多 800）
- **删除所选**：更多菜单中批量删除（Management 删除；失败按回退策略）
- 清除选择；批量操作单条失败不中断整批

## 0.5.28 - 2026-07-15

### Fix
- 编辑配置保存：CPAMP 下 POST resource 404，改为 `GET /ops?op=settings&payload=<base64url(json)>`
- 导入备份同样走 payload GET（体积过大时仍可能失败）

## 0.5.27 - 2026-07-15

### Fix
- GET ops 参数：`auth_id` / `auth_ids` 同时走 Header（`X-XAI-Autoban-Auth-Id(s)`），避免 query 丢失
- `auth_ids` 经 query 变成 JSON 字符串时正确解析（修复 `missing_auth_ids`）
- 行按钮 `data-id` 用 encodeURIComponent，避免特殊字符凭证 id 损坏

## 0.5.26 - 2026-07-15

### Fix
- **彻底避免**浏览器走 `/v0/management/plugins/*`（CPAMP 下用 CPA 密钥必报 invalid admin key，与密钥对错无关）
- 新增 resource `/ops` 写通道；支持 Header `X-XAI-Autoban-Op`
- GET 若误返回列表 payload 会继续尝试其它通道
- 错误信息展示各通道真实失败原因

## 0.5.25 - 2026-07-15

### Fix
- **CPA-Manager-Plus 兼容**：运维台写操作优先 `GET /data?op=`（CPAMP 对 resource GET 用已保存 CPA 密钥代理，无需浏览器密钥）
- POST resource 时附带 Authorization（CPA secret-key 或 cpamp_ 均可按 CPAMP 规则转发）
- 不再把 `/v0/management/plugins/*` 当成主要写通道（CPAMP 中 `plugins` 为保留路径，填 CPA 密钥会报 invalid admin key）
- README 明确：插件 `management_key` = CPA `remote-management.secret-key`，不是 `cpamp_...`

## 0.5.24 - 2026-07-15

### Fix
- Build: remove unused `os` import (0.5.23 release failed CI)

## 0.5.23 - 2026-07-15

### Fix
- Ops writes: try resource POST /data, then **GET /data?op=...** (CPA often only routes GET)
- Stronger management key resolution (multiple env names)
- Clearer errors when key missing vs invalid

## 0.5.22 - 2026-07-15

### Fix
- Ops writes use Management API with **server-injected** key from plugin manage / `CPA_MANAGEMENT_KEY` (no paste UI, no localStorage)
- Fallback to `POST /data` when management call fails
- Clear error if neither key nor resource channel works

## 0.5.21 - 2026-07-15

### Fix
- Harden resource path matching for relative `/data`

## 0.5.20 - 2026-07-15

### Fix
- Ops writes use `POST /v0/resource/plugins/xai-autoban/data` (same path as list GET) to avoid HTTP 404 on unregistered `/api`

## 0.5.19 - 2026-07-15

### Fix
- Ops write actions via resource path (no browser admin key)

## 0.5.18 - 2026-07-15

### Copy / consistency
- Fix outdated plugin-manage field descriptions (no browser key paste)
- Management route descriptions in Chinese: 隔离/取消隔离/禁用/启用
- 定时巡检「启用」→「打开」，避免与「启用凭证」混淆

## 0.5.17 - 2026-07-15

### Config UX
- Remove Management / disable_via block from ops console drawer (plugin manage only)

## 0.5.16 - 2026-07-15

### Fix
- Fully remove browser key inheritance (stale localStorage keys caused `invalid admin key`)
- Remove auth banner about keys; write ops use session cookies only (`credentials: include`)
- Clean leftover copy about ops-console key paste

## 0.5.15 - 2026-07-15

### Config UX
- Remove ops-console browser management-key paste UI (no more 保存/清除/更换密钥)
- Plugin management remains the place for server-side management key / enable switch

## 0.5.14 - 2026-07-15

### Config UX
- CPA「插件管理」ConfigFields 仅保留管理密钥相关：`management_key_env` / `management_key` / `management_url` / `disable_via`
- 日常巡检策略统一到运维台「编辑配置」（主入口文案 + Management 可选区）
- README 标明两入口分工

## 0.5.13 - 2026-07-15

### Fix
- Reauth token endpoint: use `https://auth.x.ai/oauth/token` (not `accounts.x.ai`, which returns HTTP 403)
- Always send Grok CLI `client_id` + User-Agent; discover token URL via OIDC when possible
- Clearer reauth error messages (JSON error / Cloudflare HTML)

## 0.5.12 - 2026-07-15

### UI polish
- Status-code chips: nowrap labels (`401 · 重授权`), no crushed vertical text
- Isolation card subtitle fixed meaning (ledger only); hover explains 40x口径差异
- Tighter vertical rhythm (config/list/rows); taller list viewport
- List hint when filtering by status code vs isolation ledger

## 0.5.11 - 2026-07-15

### UI
- Status-code filters match primary metric card size (grid strip)
- Unified Chinese terms: 隔离 / 取消隔离 / 禁用 / 启用 (no mixed ban wording)

## 0.5.10 - 2026-07-15

### UI
- Ops console layout: 5 primary metric cards + 401–429 code chip strip
- Adaptive credential rows (no empty action/reason columns)
- Reauth as primary action for 401 / token-expired accounts
- Compact config summary strip

## 0.5.9 - 2026-07-15

### Fixes
- Exclusive probe flight lock: scheduled + manual/async cannot run concurrently
- UI delete copy: real Management DELETE with disable/ban fallback
- Ban list API includes `classification`
- Reauth uses direct no-proxy HTTP to token endpoint; post-refresh `/models` probe
- Probe path: single AuthGet for local expiry + upstream probe
- Recheck429 non-429 failures use body semantic classify
- Status list: sample AuthGet JSON for `token_expired` / `needs_refresh` flags
- `go vet` clean test helpers; scripts/build.sh ROOT fix

### Features (from 0.5.8 hardening)
- Semantic failure classifier (429 vs free-usage vs reauth)
- True Management DELETE
- Async probe job + progress polling
- `probe_include_disabled` / `probe_only_disabled`
- refresh_token reauth API + UI button
- Summary cards always show 0 + hover; 401–429 overview cards

## 0.5.8

- Package split (`internal/*`), usage/probe body classify baseline
