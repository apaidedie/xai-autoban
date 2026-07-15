# Changelog

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
