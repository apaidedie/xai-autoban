package ui

const statusBodyTemplate = `
</head>
<body>
<div class="shell">
  <header class="top">
    <div class="top-brand">
      <div class="kicker"><i></i>CLIProxyAPI · xAI</div>
      <h1>xAI Autoban</h1>
      <p class="sub">凭证隔离 · 巡检复检 · 批量运维 · <span class="ver">v__PLUGIN_VERSION__</span></p>
    </div>
    <div class="top-actions">
      <div class="live" id="syncState">准备中</div>
      <button class="bs" id="btnRefresh" type="button" onclick="loadData()" title="刷新列表与统计">刷新</button>
      <button class="bs" id="openConfigBtn" type="button" title="编辑巡检与策略">配置</button>
    </div>
  </header>

  <section class="panel">
    <div class="phd">
      <h2>巡检配置</h2>
      <div class="hint">点「配置」修改 · 插件管理只负责启用与 Management 密钥</div>
    </div>
    <div class="cfg-grid" id="cfgPills">
      <div class="cfg-card"><div class="l">定时巡检</div><div class="v" id="sumProbeEnabled">-</div></div>
      <div class="cfg-card"><div class="l">巡检间隔</div><div class="v" id="sumInterval">-</div></div>
      <div class="cfg-card accent"><div class="l">自动执行</div><div class="v" id="sumAutoExec">-</div></div>
      <div class="cfg-card"><div class="l">失败策略</div><div class="v" id="sumProbeAction">-</div></div>
      <div class="cfg-card"><div class="l">成功策略</div><div class="v" id="sumOnSuccess">-</div></div>
      <div class="cfg-card"><div class="l">探测模式</div><div class="v" id="sumMode">-</div></div>
    </div>
    <div class="cfg-path" id="statePathHint" title="运维台配置与隔离账本持久化路径；CPA 重建请挂载此目录">状态文件：加载中…</div>
  </section>

  <section class="metrics-block" aria-label="概览与筛选">
  <div class="qcards" id="overviewCards">
    <button type="button" class="qcard info" data-jump="all" data-filter="all" title="xAI 凭证总数">
      <div class="ql">全部</div><div class="qn" id="ov_all">0</div><div class="qs">凭证</div>
    </button>
    <button type="button" class="qcard ok" data-jump="healthy" data-filter="healthy" title="未禁用、未隔离 → 可参与调度">
      <div class="ql">健康</div><div class="qn" id="ov_healthy">0</div><div class="qs">可调度</div>
    </button>
    <button type="button" class="qcard warn" data-jump="banned" data-filter="banned" title="隔离：插件账本，调度跳过。与下方状态码卡口径不同，可与禁用重叠。">
      <div class="ql">隔离</div><div class="qn" id="ov_banned">0</div><div class="qs" id="ov_banned_sub">账本 · 跳过调度</div>
    </button>
    <button type="button" class="qcard disabled-card" data-jump="disabled" data-filter="disabled" title="禁用：CPA 凭证开关关闭，与隔离是两件事">
      <div class="ql">禁用</div><div class="qn" id="c_disabled">0</div><div class="qs">CPA 关闭</div>
    </button>
    <button type="button" class="qcard info" data-jump="probe" id="ov_probe_card" title="点击立即全量巡检；定时开启后约 45 秒内首次执行">
      <div class="ql">巡检</div><div class="qn" id="ov_probe">—</div><div class="qs" id="ov_probe_sub">点击开始</div>
    </button>
  </div>
  <div class="code-strip" id="codeStrip" role="toolbar" aria-label="按状态筛选">
    <button type="button" class="code-chip s401" data-filter="401" title="隔离账本中状态码 401 的条数（需重授权）">
      <span class="cl">401 · 需重授</span><b id="ov_401">0</b>
    </button>
    <button type="button" class="code-chip s402" data-filter="402" title="隔离账本中状态码 402 的条数（额度不足）。usage/巡检/复检 402 均按 action_on_402 处理">
      <span class="cl">402 · 额度</span><b id="ov_402">0</b>
    </button>
    <button type="button" class="code-chip s403" data-filter="403" title="隔离账本中状态码 403 的条数。默认一次即按 action_on_403 处理（fail_streak_403=1）">
      <span class="cl">403 · 拒绝</span><b id="ov_403">0</b>
    </button>
    <button type="button" class="code-chip s429" data-filter="429" title="隔离账本中状态码 429 的条数（限流）">
      <span class="cl">429 · 限流</span><b id="ov_429">0</b>
    </button>
  </div>
  <details class="legend" id="statusLegend">
    <summary><span>用语说明</span><span class="chev">展开</span></summary>
    <div class="legend-body">
      <div class="row2">
        <span class="k">健康</span><span>未禁用、未隔离 → 可调度</span>
        <span class="k">隔离</span><span>插件账本，调度跳过；用「释放」清除</span>
        <span class="k">禁用</span><span>CPA 凭证开关关闭；与隔离独立</span>
        <span class="k">释放 / 启用</span><span>释放=清隔离账本 · 启用=打开 CPA 开关</span>
        <span class="k">巡检 / 复检</span><span>巡检=全量 · 复检=勾选；失败均按状态码动作（需自动执行）</span>
        <span class="k">401–429</span><span>仅统计<strong>隔离账本</strong>内状态码（动作=隔离时）；禁用/删除不进隔离账本</span>
        <span class="k">隔离 vs 禁用</span><span>隔离=插件账本跳过调度 · 禁用=关 CPA 开关；403 设禁用时只禁用、不隔离</span>
        <span class="k">401/402/403</span><span>默认出现一次即按状态码动作；成功策略「启用」可在复检/巡检成功后打开开关</span>
        <span class="k">真实流量</span><span>调用成功会释放隔离，并在 30 分钟内跳过巡检</span>
      </div>
    </div>
  </details>
  </section>
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

  <section class="panel panel-list">
    <div class="phd">
      <div>
        <h2>凭证</h2>
        <div class="hint" id="listHint">点上方卡片筛选 · 勾选后复检或批量操作</div>
      </div>
      <div class="hint count" id="resultCount">0 条</div>
    </div>

    <div class="toolbar">
      <div class="tools">
        <input id="search" type="search" placeholder="搜索邮箱 / 凭证 ID / 原因" autocomplete="off">
        <div class="tools-end">
          <button class="bp" id="recheckSelected" type="button" onclick="recheckSelected()" disabled title="对勾选凭证复检">复检 (0)</button>
          <details class="more">
            <summary class="bs">批量</summary>
            <div class="more-menu">
              <button type="button" id="unbanSelected" onclick="bulkAct('unban')" disabled>释放</button>
              <button type="button" id="banSelected" onclick="bulkAct('ban')" disabled>隔离</button>
              <button type="button" id="disableSelected" onclick="bulkAct('disable')" disabled>禁用</button>
              <button type="button" id="reenableSelected" onclick="bulkAct('reenable')" disabled>启用</button>
              <button type="button" class="danger" id="deleteSelected" onclick="bulkAct('delete')" disabled>删除</button>
              <div class="more-div"></div>
              <button type="button" onclick="exportInspect('reauth')">导出需重授</button>
              <button type="button" onclick="exportInspect('pending_delete')">导出待删</button>
              <div class="more-div"></div>
              <label class="chk"><input id="autoRefresh" type="checkbox" checked> 30 秒自动刷新</label>
            </div>
          </details>
        </div>
      </div>
      <div class="sel-bar">
        <label class="sel-inline"><input id="selectPage" type="checkbox"> 本页全选</label>
        <button type="button" class="sel-link" id="selectFilterBtn" title="勾选当前筛选全部（跨页，最多 800）">全选筛选</button>
        <button type="button" class="sel-link" id="clearSelectedBtn" title="清空勾选">清空</button>
        <span class="sel-count" id="selectedHint"></span>
      </div>
    </div>

    <div class="row msg-row"><div id="message" class="msg">就绪</div></div>
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

  <section class="panel panel-hist">
    <div class="phd">
      <h2>巡检记录</h2>
      <button class="hist-toggle bg" id="toggleHistBtn" type="button" aria-expanded="false">展开</button>
    </div>
    <div class="hist-wrap" id="histWrap">
      <div class="hist" id="probeHistory">暂无记录</div>
    </div>
  </section>

  <footer class="foot">
    <span><b>隔离</b> 插件账本 · <b>禁用</b> CPA 开关 · <b>释放</b> 清账本 · <b>启用</b> 开开关 · <b>巡检</b> 全量 · <b>复检</b> 勾选</span>
    <span class="foot-sub">禁用/删除需在插件管理配置 CPA Management Key（勿用 cpamp_ 面板密钥）</span>
  </footer>
  <input id="importFile" type="file" accept="application/json,.json" hidden>
</div>
<div class="toast" id="toast" role="status" aria-live="polite"></div>

<div class="drawer-mask" id="drawerMask"></div>
<aside class="drawer" id="drawer" aria-hidden="true">
  <div class="dh">
    <div>
      <h3>配置</h3>
      <p>巡检、自动执行与失败/成功策略。保存后立即生效。Management 密钥在插件管理中配置。</p>
    </div>
    <button class="bg" id="closeConfigBtn" type="button" title="关闭">关闭</button>
  </div>
  <div class="db">
    <div class="sec">
      <h4>巡检</h4>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_enabled" type="checkbox"> 开启定时巡检</label>
      <div class="fg"><label>间隔（秒）</label><input id="f_probe_interval_seconds" type="number" min="30" step="1"></div>
      <div class="fg"><label>超时（秒）</label><input id="f_probe_timeout_seconds" type="number" min="5" step="1"></div>
      <div class="fg"><label>并发</label><input id="f_probe_concurrency" type="number" min="1" step="1"></div>
      <div class="fg"><label>QPS</label><input id="f_probe_qps" type="number" min="0.1" step="0.1"></div>
      <div class="fg"><label>探测模式</label>
        <select id="f_probe_mode"><option value="responses_mini">responses · 真实请求（推荐）</option><option value="models">models（轻量列表）</option></select>
      </div>
      <label class="chk" style="margin-bottom:8px"><input id="f_probe_include_disabled" type="checkbox"> 巡检包含已禁用凭证</label>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_only_disabled" type="checkbox"> 仅巡检已禁用凭证</label>
    </div>
    <div class="sec">
      <h4>自动执行</h4>
      <div class="choice" id="autoExecChoices" style="margin-bottom:10px">
        <button type="button" data-v="0"><b>只记录</b><span>不自动禁用/删除；失败最多写入隔离</span></button>
        <button type="button" data-v="1"><b>自动执行</b><span>按下方失败/成功策略处理</span></button>
      </div>
      <div class="fg"><label>成功策略</label>
        <div class="choice" id="successChoices">
          <button type="button" data-v="none"><b>不处理</b><span>只记录，不改状态</span></button>
          <button type="button" data-v="unban"><b>释放</b><span>清除隔离（默认）</span></button>
          <button type="button" data-v="reenable"><b>启用</b><span>打开 CPA 开关</span></button>
          <button type="button" data-v="unban_and_reenable"><b>释放并启用</b><span>清账本 + 开开关</span></button>
        </div>
      </div>
      <div class="fg"><label>失败策略</label>
        <div class="choice" id="failChoices">
          <button type="button" data-v="ban"><b>隔离</b><span>跳过调度，最稳妥</span></button>
          <button type="button" data-v="disable"><b>禁用</b><span>关闭 CPA 凭证</span></button>
          <button type="button" data-v="delete"><b>删除</b><span>Management 删除；失败则回退</span></button>
        </div>
      </div>
      <div class="fg"><label>删除失败回退</label>
        <select id="f_delete_fallback">
          <option value="disable">禁用</option>
          <option value="ban">隔离</option>
        </select>
      </div>
    </div>
    <div class="sec">
      <h4>按状态码（真实失败）</h4>
      <div class="fg"><label>401 需重授</label><select id="f_action_on_401"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>402 额度</label><select id="f_action_on_402"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>403 拒绝</label><select id="f_action_on_403"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>429 限流</label><select id="f_action_on_429"><option value="ban">隔离</option><option value="disable">禁用</option><option value="delete">删除</option></select></div>
      <div class="fg"><label>动作冷却（秒）</label><input id="f_action_cooldown_seconds" type="number" min="0" step="1"></div>
    </div>
  </div>
  <div class="df">
    <button class="bg" id="discardConfigBtn" type="button">取消</button>
    <button class="bp" id="saveConfigBtn" type="button">保存</button>
  </div>
</aside>

`
