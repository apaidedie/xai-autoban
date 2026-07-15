# Changelog

## 0.5.33 - 2026-07-15

### Fix
- 写操作优先 `GET /data?op=`（无自定义 Header），再试 /ops；适配反代子路径 `resourceBase`
- 全 404 时提示完整重启 CPA 并打印 base/version 便于排查

## 0.5.32 - 2026-07-15

### Fix
- **配置假成功**：禁止用列表里的 `settings` 当保存成功；settings 改走扁平 query（不再 pack payload）
- 空 patch 直接 400；query 布尔/数字类型强制转换
- 客户端校验 auto_execute / probe_action 等多字段；服务端返回 `applied` 数量

## 0.5.31 - 2026-07-15

### Fix
- **运维配置持久化**：保存写入 `xai-autoban-state.json`（默认），重载/重开抽屉仍保留
- 修复误把列表响应当「保存成功」（列表也带 `settings` 字段）
- 保存后校验关键字段是否生效

## 0.5.30 - 2026-07-15

### Chore
- 文档整理：README / registry 描述与当前运维台能力对齐
- 去掉无用 `mgmtBase`；统一 `ResolveManagementKey`
- 页脚文案：CPAMP 下写操作说明更准确

## 0.5.29 - 2026-07-15

### Feature
- **全选当前筛选**（跨页，最多 800）
- **删除所选**（二次确认；单条失败不中断整批）
- **清除选择**

## 0.5.26 – 0.5.28 - 2026-07-15

### Fix（CPA-Manager-Plus）
- 运维台写操作只走 resource，不走 `/v0/management/plugins/*`（避免误报 invalid admin key）
- 优先 `GET /ops` + Header 传 `auth_id`；配置保存用 `GET ?payload=base64url(json)`
- 解析 query 中的 `auth_ids` JSON 字符串

## 0.5.16 – 0.5.25 - 2026-07-15

### Ops / 配置
- 去掉浏览器粘贴管理密钥；插件管理仅保留启用与服务端密钥字段
- resource `/data` / `/ops` 写通道；GET `?op=` 兼容 CPA/CPAMP
- 术语统一：隔离 / 取消隔离 / 禁用 / 启用

## 0.5.13 – 0.5.15 - 2026-07-15

### Fix / Config
- Reauth：`https://auth.x.ai/oauth/token` + Grok CLI client_id
- 运维台与插件管理配置入口拆分

## 0.5.9 – 0.5.12 - 2026-07-15

### UI
- 指标卡 + 401–429 状态码筛选条
- 凭证行自适应；重授权主按钮；术语中文化

## 0.5.8 – 0.5.9 - 2026-07-15

### Core
- 包拆分 `internal/*`；语义分类器
- 真 Management DELETE；异步巡检 job
- `probe_include_disabled` / `probe_only_disabled`；refresh reauth
