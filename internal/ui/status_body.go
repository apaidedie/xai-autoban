package ui

const statusBodyTemplate = `
</head>
<body>
<div class="shell">
  <div class="top">
    <div>
      <div class="kicker"><i></i>运维台 · xAI 账号巡检</div>
      <h1>xAI Autoban</h1>
      <p class="sub">隔离 · 禁用 · 启用 · 复检 · v__PLUGIN_VERSION__</p>
    </div>
    <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <div class="live" id="syncState">准备中</div>
      <button class="bs" id="btnRefresh" type="button" onclick="loadData()" title="刷新列表与统计">刷新</button>
      <button class="bp" id="btnProbe" type="button" onclick="runProbe()" disabled>立即巡检</button>
      <button class="bs" id="openConfigBtn" type="button">编辑配置</button>
    </div>
  </div>

  <section class="panel">
    <div class="phd">
      <h2>当前巡检配置</h2>
      <div class="hint">主配置入口 · 点右上角「编辑配置」修改（插件管理仅负责启用与服务端密钥）</div>
    </div>
    <div class="cfg-grid" id="cfgPills">
      <div class="cfg-card"><div class="l">定时巡检</div><div class="v" id="sumProbeEnabled">-</div></div>
      <div class="cfg-card"><div class="l">间隔</div><div class="v" id="sumInterval">-</div></div>
      <div class="cfg-card accent"><div class="l">自动执行</div><div class="v" id="sumAutoExec">-</div></div>
      <div class="cfg-card"><div class="l">问题策略</div><div class="v" id="sumProbeAction">-</div></div>
      <div class="cfg-card"><div class="l">成功策略</div><div class="v" id="sumOnSuccess">-</div></div>
      <div class="cfg-card"><div class="l">探测模式</div><div class="v" id="sumMode">-</div></div>
    </div>
  </section>

  <div class="qcards" id="overviewCards">
    <button type="button" class="qcard info" data-jump="all" data-filter="all" title="xAI 认证文件总数（AuthList）">
      <div class="ql">全部凭证</div><div class="qn" id="ov_all">0</div><div class="qs">认证文件</div>
    </button>
    <button type="button" class="qcard ok" data-jump="healthy" data-filter="healthy" title="未禁用且未在插件隔离账本 → 可参与调度">
      <div class="ql">健康</div><div class="qn" id="ov_healthy">0</div><div class="qs">可调度</div>
    </button>
    <button type="button" class="qcard warn" data-jump="banned" data-filter="banned" title="【隔离账本】插件内部记录，调度会跳过。与下方 401–429（按隔离条目的状态码计数）口径不同；可与「已禁用」重叠。">
      <div class="ql">当前隔离</div><div class="qn" id="ov_banned">0</div><div class="qs" id="ov_banned_sub">账本 · 跳过调度</div>
    </button>
    <button type="button" class="qcard disabled-card" data-jump="disabled" data-filter="disabled" title="【CPA 禁用】凭证开关关闭（Auth.Disabled）。与插件隔离是两件事：可同时存在。">
      <div class="ql">已禁用</div><div class="qn" id="c_disabled">0</div><div class="qs">CPA 关闭</div>
    </button>
    <button type="button" class="qcard info" data-jump="probe" id="ov_probe_card" title="点击立即全量巡检。定时巡检开启后约 45 秒内首次执行；进行中会显示进度。">
      <div class="ql">上次巡检</div><div class="qn" id="ov_probe">—</div><div class="qs" id="ov_probe_sub">点击立即巡检</div>
    </button>
  </div>
  <div class="code-strip" id="codeStrip" role="toolbar" aria-label="状态码筛选">
    <button type="button" class="code-chip s401" data-filter="401" title="【状态码计数】隔离账本里 status=401 的条数（需重授权/Token 失效），不是全部 401 探测结果。">
      <span class="cl">401 · 重授权</span><b id="ov_401">0</b>
    </button>
    <button type="button" class="code-chip s402" data-filter="402" title="【状态码计数】隔离账本里 status=402 的条数（额度/free-usage）。探测 402 默认不写入隔离。">
      <span class="cl">402 · 无额度</span><b id="ov_402">0</b>
    </button>
    <button type="button" class="code-chip s403" data-filter="403" title="【状态码计数】隔离账本里 status=403 的条数。软 403 需连续多次才隔离。">
      <span class="cl">403 · 禁止</span><b id="ov_403">0</b>
    </button>
    <button type="button" class="code-chip s429" data-filter="429" title="【状态码计数】隔离账本里 status=429 的条数（限流）。">
      <span class="cl">429 · 限流</span><b id="ov_429">0</b>
    </button>
    <button type="button" class="code-chip s-api" data-filter="using_api" id="usingApiFilterBtn" title="【API 模式】凭证 using_api=true。数字来自缓存/抽样；点击筛选，再点取消。">
      <span class="cl">API · 模式</span><b id="ov_using_api">0</b>
    </button>
  </div>
  <details class="legend" id="statusLegend">
    <summary><span>读懂状态口径（隔离 ≠ 禁用 ≠ 状态码卡）</span><span class="chev">展开</span></summary>
    <div class="legend-body">
      <div class="row2">
        <span class="k">健康</span><span>未禁用、未在隔离账本 → 调度可选</span>
        <span class="k">隔离</span><span><b>插件账本</b>：调度跳过；CPA 开关可能仍是启用。可点「释放」清账本</span>
        <span class="k">禁用</span><span><b>CPA 开关</b>：凭证关闭；与隔离独立，可同时「禁用 + 兼隔离」</span>
        <span class="k">401–429</span><span>只统计<strong>已写入隔离账本</strong>的状态码，不是实时探测全量分布</span>
        <span class="k">软 403</span><span>连续失败达到阈值才隔离；行上 1/3 表示连击进度</span>
        <span class="k">API 模式</span><span>CPA using_api；自动开启见配置「自动 API 模式」（默认关闭，更安全）</span>
        <span class="k">真实流量</span><span>usage 成功会释放隔离，并在 30 分钟内跳过探测（防误伤）</span>
      </div>
    </div>
  </details>
  <div id="statusChips" hidden aria-hidden="true">
    <button type="button" data-filter="all"><b id="c_all">-</b></button>
    <button type="button" data-filter="healthy"><b id="c_healthy">-</b></button>
    <button type="button" data-filter="banned"><b id="c_banned">-</b></button>
    <b id="c_401">-</b><b id="c_402">-</b><b id="c_403">-</b><b id="c_429">-</b>
    <span id="f_401">0</span><span id="f_402">0</span><span id="f_403">0</span><span id="f_429">0</span>
  </div>
  <span id="total" hidden>0</span>
  <span id="count402" hidden>0</span>
  <span id="count403" hidden>0</span>
  <span id="count429" hidden>0</span>

  <section class="panel">
    <div class="phd">
      <div>
        <h2>凭证列表</h2>
        <div class="hint" id="listHint">点击上方卡片筛选 · 勾选后复检或批量操作</div>
      </div>
      <div class="hint" id="resultCount">0 条</div>
    </div>

    <div class="toolbar">
      <div class="tools">
        <input id="search" type="search" placeholder="搜索账号 / Auth ID / 原因" autocomplete="off">
        <div class="tools-end">
          <button class="bp" id="recheckSelected" type="button" onclick="recheckSelected()" disabled title="对勾选凭证做上游复检">复检所选 (0)</button>
          <details class="more">
            <summary class="bs">操作</summary>
            <div class="more-menu">
              <button type="button" id="unbanSelected" onclick="bulkAct('unban')" disabled>释放所选</button>
              <button type="button" id="banSelected" onclick="bulkAct('ban')" disabled>隔离所选</button>
              <button type="button" id="disableSelected" onclick="bulkAct('disable')" disabled>禁用所选</button>
              <button type="button" id="reenableSelected" onclick="bulkAct('reenable')" disabled>启用所选</button>
              <button type="button" id="usingApiSelected" onclick="bulkAct('using_api')" disabled title="开启 CPA「使用 API 模式」(using_api)，OAuth 403 时可试">API 模式所选</button>
              <button type="button" class="danger" id="deleteSelected" onclick="bulkAct('delete')" disabled>删除所选</button>
              <div class="more-div"></div>
              <label class="chk"><input id="autoRefresh" type="checkbox" checked> 30 秒自动刷新</label>
            </div>
          </details>
        </div>
      </div>
      <div class="sel-bar">
        <label class="sel-inline"><input id="selectPage" type="checkbox"> 本页全选</label>
        <button type="button" class="sel-link" id="selectFilterBtn" title="勾选当前筛选下全部凭证（跨页，最多 800）">全选当前筛选</button>
        <button type="button" class="sel-link" id="clearSelectedBtn" title="清空勾选">清除</button>
        <span class="sel-count" id="selectedHint"></span>
      </div>
    </div>

    <div class="row msg-row"><div id="message" class="msg">系统待命</div></div>
    <div class="progress-panel" id="progressPanel">
      <div class="progress-meta">
        <span class="pl" id="progressLabel">处理中</span>
        <span class="pc" id="progressCount">0/0</span>
      </div>
      <div class="progress" id="progress"><i id="progressBar"></i></div>
      <div class="op-result" id="opResult" hidden></div>
    </div>

    <div class="card-list" id="rows"></div>
    <div id="empty" class="empty" hidden>没有匹配的凭证</div>
    <div class="pager" id="pager">
      <div class="pinfo" id="pageInfo">第 1 / 1 页</div>
      <div class="pbtns">
        <button class="bg" id="prevPageBtn" type="button">上一页</button>
        <button class="bg" id="nextPageBtn" type="button">下一页</button>
      </div>
    </div>
  </section>

  <section class="panel">
    <div class="phd">
      <h2>巡检历史</h2>
      <button class="hist-toggle bg" id="toggleHistBtn" type="button" aria-expanded="false">展开</button>
    </div>
    <div class="hist-wrap" id="histWrap">
      <div class="hist" id="probeHistory">暂无记录</div>
    </div>
  </section>

  <p class="foot">
    <b>隔离</b>=插件账本跳过调度 · <b>禁用</b>=CPA 关凭证 · <b>启用</b>=打开 CPA 开关 · <b>删除</b>=Management 删文件。
    口径见上方「读懂状态」。禁用/删除需插件管理里配置 CPA Management Key（不要填 cpamp_ 面板密钥）。
  </p>
  <input id="importFile" type="file" accept="application/json,.json" hidden>
</div>
<div class="toast" id="toast" role="status" aria-live="polite"></div>

<div class="drawer-mask" id="drawerMask"></div>
<aside class="drawer" id="drawer" aria-hidden="true">
  <div class="dh">
    <div>
      <h3>运维配置（主入口）</h3>
      <p>巡检、自动执行与失败/成功策略请在此修改。保存后立即生效。启用与服务端 Management 密钥仅在插件管理配置。</p>
    </div>
    <button class="bg" id="closeConfigBtn" type="button">✕</button>
  </div>
  <div class="db">
    <div class="sec">
      <h4>调度</h4>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_enabled" type="checkbox"> 打开定时巡检</label>
      <div class="fg"><label>间隔（秒）</label><input id="f_probe_interval_seconds" type="number" min="30" step="1"></div>
      <div class="fg"><label>超时（秒）</label><input id="f_probe_timeout_seconds" type="number" min="5" step="1"></div>
      <div class="fg"><label>并发</label><input id="f_probe_concurrency" type="number" min="1" step="1"></div>
      <div class="fg"><label>QPS</label><input id="f_probe_qps" type="number" min="0.1" step="0.1"></div>
      <div class="fg"><label>探测模式</label>
        <select id="f_probe_mode"><option value="responses_mini">responses · 真实请求（推荐）</option><option value="models">models（轻量列表）</option></select>
      </div>
      <label class="chk" style="margin-bottom:8px"><input id="f_probe_include_disabled" type="checkbox"> 巡检包含已禁用凭证</label>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_only_disabled" type="checkbox"> 仅巡检已禁用凭证</label>
      <div class="fg"><label>自动 API 模式</label>
        <select id="f_auto_using_api" title="探测/复检失败时是否自动写 using_api。会改变账号走 API 路径，可能影响额度/限流；默认关闭更安全。">
          <option value="off">关闭 · 仅手动（推荐，更安全）</option>
          <option value="on_403">仅 403 时自动开 API 模式</option>
          <option value="on_fail">401/402/403 都自动开</option>
        </select>
      </div>
      <p class="hint" style="margin:0 0 10px;line-height:1.45">自动开启会改 CPA 凭证字段；不确定时保持「关闭」，用列表「API 模式所选」手动开。</p>
    </div>
    <div class="sec">
      <h4>自动执行（对齐 Codex 巡检）</h4>
      <div class="choice" id="autoExecChoices" style="margin-bottom:10px">
        <button type="button" data-v="0"><b>只输出结果</b><span>巡检只记录；失败最多写入隔离展示，不禁用/删除</span></button>
        <button type="button" data-v="1"><b>自动执行</b><span>按下方策略处理问题账号与恢复</span></button>
      </div>
      <div class="fg"><label>成功策略</label>
        <div class="choice" id="successChoices">
          <button type="button" data-v="none"><b>不处理</b><span>仅记录，不改隔离/禁用状态</span></button>
          <button type="button" data-v="unban"><b>自动取消隔离</b><span>清除插件内隔离（默认）</span></button>
          <button type="button" data-v="reenable"><b>启用凭证</b><span>打开凭证，不改隔离</span></button>
          <button type="button" data-v="unban_and_reenable"><b>取消隔离 + 启用</b><span>同时恢复调度与打开凭证</span></button>
        </div>
      </div>
      <div class="fg"><label>问题账号策略</label>
        <div class="choice" id="failChoices">
          <button type="button" data-v="ban"><b>仅隔离</b><span>插件内跳过调度，最安全</span></button>
          <button type="button" data-v="disable"><b>禁用凭证</b><span>关闭 CPA 凭证</span></button>
          <button type="button" data-v="delete"><b>删除</b><span>Management 删除；失败则禁用/隔离并标记待删</span></button>
        </div>
      </div>
      <div class="fg"><label>删除失败时回退</label>
        <select id="f_delete_fallback">
          <option value="disable">禁用</option>
          <option value="ban">隔离</option>
        </select>
      </div>
    </div>
    <div class="sec">
      <h4>失败动作（按状态码）</h4>
      <div class="fg"><label>401</label><select id="f_action_on_401"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>402</label><select id="f_action_on_402"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>403</label><select id="f_action_on_403"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>429（建议仅隔离）</label><select id="f_action_on_429"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>动作冷却（秒）</label><input id="f_action_cooldown_seconds" type="number" min="0" step="1"></div>
    </div>
  </div>
  <div class="df">
    <button class="bg" id="discardConfigBtn" type="button">丢弃更改</button>
    <button class="bp" id="saveConfigBtn" type="button">保存并生效</button>
  </div>
</aside>

`
