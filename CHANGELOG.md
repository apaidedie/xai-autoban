# Changelog

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
