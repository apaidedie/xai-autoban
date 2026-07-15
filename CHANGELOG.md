# Changelog

## 0.5.40 - 2026-07-15

### Fix
- **Probe/复检 402 不隔离**：探测路径 free-usage/402 常误报；仅真实 usage 失败才按额度隔离
- 真实调用成功后刷新「巡检失败」为成功，避免「健康 + 巡检失败 402」矛盾展示
- UI：未隔离时显示「上次巡检异常（当前可用）」

## 0.5.39 - 2026-07-15

### Change
- **探测默认走 `/responses` 真实请求**（grok-4.5，`Reply with exactly: OK`）
- `responses_mini` / `responses`：POST responses 优先，失败再 fallback chat/completions
- `models` 仍为轻量 GET 列表模式
- 默认 `probe_mode=responses_mini`

## 0.5.38 - 2026-07-15

### Fix
- **复检误隔离**：不再一次 probe 403 就 ForceSet；走软 403 连续失败（默认 3 次）
- **真实调用成功后 30 分钟内**，probe/复检 403 不会再次隔离
- Probe `responses_mini` 优先 chat/completions（对齐真实 grok 流量）

## 0.5.37 - 2026-07-15

### Fix
- 软 403 连续失败才隔离；永久类（suspended/token expired）仍立即隔离
- 可配 `fail_streak_403` / `fail_streak_window_seconds`

## 0.5.36 - 2026-07-15

### Fix
- **真实调用成功即释放隔离**：usage 成功流量按 ground truth 清除 ban（修复 probe 误伤后后台仍可用却显示不健康）
- 自动执行 + 成功策略含「启用」时，成功流量也会尝试重新启用凭证
- Probe：models 失败时增加 chat/completions 回退（更接近真实 grok 流量）

## 0.5.35 - 2026-07-15

### UI
- 刷新移到右上角「立即巡检」左侧
- 更多菜单重排：所选批量 / 危险 / 全局；去掉导入导出备份
- 「取消隔离所选」→「释放所选」；行按钮「释放」；全部释放

## 0.5.34 - 2026-07-15

### Fix
- 巡检「already running」：接入已有任务进度，不再当失败
- 卡住任务：3 小时自动清锁；UI 无进度约 90s 可 force 重开

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
