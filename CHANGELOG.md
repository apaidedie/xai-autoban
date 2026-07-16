# Changelog

## 1.0.4 - 2026-07-16

### UI
- **恢复大卡片布局**：配置六格 + 概览大卡 + 401–429 大条（用户偏好）
- 保留：行状态精简、复检结果摘要、不内嵌密钥、行操作「···」

## 1.0.3 - 2026-07-16

### UI
- （已回退顶栏压缩布局）曾尝试折叠配置与胶囊筛选

## 1.0.2 - 2026-07-16

### UI / UX
- 复检结果：摘要 + 最多 5 条短明细（邮箱 · 状态码 · 连击中），不再刷 100+ 行文件名
- 结果面板高度收紧；长输出自动折叠「…另 N 条」
- 页面不再内嵌 Management 密钥（防 view-source 泄露）
- 概览文案缩短；复检确认框精简

## 1.0.1 - 2026-07-16

### UI
- 列表状态精简：**一个主徽章**（健康/禁用/隔离/401–429）+ 最多 2 个辅标
- 去掉「已禁用+仍隔离+隔离+禁用」叠罗汉；禁用且隔离 →「兼隔离」
- 副行只保留一句中文原因 + 剩余时间 + 必要时探测码
- 复检结果文案中文化（连击/宽限）

## 1.0.0 - 2026-07-16

### Stable
- **Stable contract per [STABILITY.md](./STABILITY.md)** — guarantees, non-guarantees, frozen config keys, ops vocabulary
- Production-ready xAI autoban for CLIProxyAPI / CPA-Manager-Plus
- Config freeze from 0.9.0 continues under 1.x policy (no remove/rename of frozen keys without major)

### Includes (cumulative)
- Usage melt + heal; scheduler skip isolation ledger
- Probe/recheck aligned with CPA (OAuth cli-chat-proxy / API api.x.ai)
- Soft 403 streak; probe 402 no-isolate; real usage 402 isolates
- Manual + auto `using_api` (default `on_403`); write-back verify
- Ops console: filter, bulk actions, API mode, list fields, CPAMP resource ops
- Contract CI suite + frozen ops key guard

## 0.9.0 - 2026-07-16

### Freeze
- **Config freeze window** opened: no remove/rename of frozen ops/install keys (see [STABILITY.md](./STABILITY.md) §3)
- Policy: 0.9.x = bugfix only toward 1.0.0
- CI guard: `TestFrozenOpsKeysInPublicView` ensures every `OpsSettingsKeys` entry is in PublicView

### Note
- Jump from 0.5.49 → 0.9.0 marks freeze intent (semver minor for pre-1.0 readiness), not a breaking API wipe.

## 0.5.49 - 2026-07-16

### Fix
- HTTP **402** 不再误标为 `permission_denied` / 软 403 连击；归为 `quota_exhausted`，usage 可立即隔离
- 软 403 streak 仅作用于真正的 403 permission

### Test
- STABILITY 契约单测：`stability_contract_test.go`、`usage/handle_test.go`（软 403、usage 释放、probe 402 不隔离、using_api 校验、删除回退）

### Docs
- [STABILITY.md](./STABILITY.md) 清单勾选 CI 测试项

## 0.5.48 - 2026-07-16

### Feat / Chore
- 运维列表展示：**API 模式** / **软403 n/need** / 最近巡检（原有）；行内 API 快捷按钮
- 拆分 `internal/action/using_api.go`；`action`/`creds` 单测补强
- Release：去掉 `release.published` 双触发；注释防空资产竞态；默认版本 0.5.48
- 历史 plan/spec 移入 `docs/archive/`

## 0.5.47 - 2026-07-16

### Feat / Fix
- `auto_using_api` 配置：`off` | `on_403`（默认）| `on_fail`；运维台可改
- Probe/复检自动开 API 模式：默认仅 403；每 run 每账号最多 1 次
- `SetUsingAPI` 写后 `AuthGet` 校验；Management 未反映则回退 host save
- 手动「API 模式」仍不受 auto 限制

## 0.5.46 - 2026-07-16

### Feat
- 支持开启 CPA「使用 API 模式」：`apply-action` / 操作菜单 **API 模式所选**（`using_api=true`）
- Management `auth-files/fields` 优先，失败回退 `host.auth.save`
- Probe / 复检所选：OAuth 401/402/403 时自动尝试 `using_api` 并重探一次
- 开启 API 模式时清除该账号隔离记录

## 0.5.45 - 2026-07-15

### UI
- 凭证列表工具栏重构：搜索 + 复检/操作右对齐；勾选控制收进同一条选择栏
- 进度结果区改为深色主题柔和样式（去掉刺眼大红底）
- streak/grace 类复检结果用中性 warn 样式

## 0.5.44 - 2026-07-15

### UI
- 去掉「复检 429」；「更多」改为「操作」
- 操作菜单：释放/隔离/禁用/启用/删除所选合在一栏；去掉「全部释放」

## 0.5.43 - 2026-07-15

### UI
- 批量操作真实进度条 + **已完成/总数**
- 复检所选分批执行并随进度更新
- 结果固定显示在进度条下方（不再只依赖右下角短 toast）

## 0.5.42 - 2026-07-15

### Chore
- 文档与 registry 对齐当前行为（probe OAuth 路径、usage 成功释放、软 403、CPAMP）
- 清理过时版本提示文案；reauth User-Agent 去硬编码版本号
- `ExtractAccessToken` 复用 `parseAuthMaterial`

## 0.5.41 - 2026-07-15

### Fix
- Probe OAuth 走 `cli-chat-proxy.grok.com` + Grok CLI 头（对齐 CPA，消除假 402/403）
- API Key 仍走 `api.x.ai`；`/responses` 使用 string `input`

## 0.5.36 – 0.5.40 - 2026-07-15

### Fix
- 真实 usage 成功 → 释放隔离；成功后 30 分钟内 probe/复检不误封
- Probe/复检 402 不隔离；复检不再 ForceSet 一次 403 就封（软 403 默认连 3 次）
- 默认 `responses` 真实探测；配置持久化；CPAMP 写通道与 UI 整理

## 0.5.29 – 0.5.35 - 2026-07-15

### Feature / UI
- 全选当前筛选、删除所选、释放措辞
- 刷新位置、更多菜单重排；配置保存与 ops 写通道加固

## 0.5.8 – 0.5.28 - 2026-07

### Core
- 包拆分、语义分类、真 DELETE、异步巡检、reauth、运维台
