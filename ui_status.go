package main

import "html"

func statusPage() string {
	name := html.EscapeString(pluginName)
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + name + `</title>
<style>
:root{color-scheme:dark;--bg:#070b14;--panel:#101a2c;--line:rgba(148,163,184,.16);--text:#f8fafc;--muted:#93a4c3;--cyan:#22d3ee;--blue:#3b82f6;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--violet:#a78bfa;--mono:ui-monospace,Consolas,monospace;--sans:Inter,ui-sans-serif,system-ui,"Segoe UI",sans-serif}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:var(--sans);color:var(--text);background:radial-gradient(1000px 500px at 10% -10%,rgba(34,211,238,.1),transparent 50%),radial-gradient(800px 400px at 100% 0,rgba(59,130,246,.1),transparent 45%),linear-gradient(180deg,#070b14,#0a101c);font-size:14px}
.shell{max-width:1540px;margin:0 auto;padding:18px 20px 36px}
.top{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;margin-bottom:14px}
.kicker{display:inline-flex;align-items:center;gap:8px;color:var(--cyan);font-size:11px;font-weight:800;letter-spacing:.12em}
.kicker i{width:7px;height:7px;border-radius:50%;background:var(--cyan);box-shadow:0 0 0 4px rgba(34,211,238,.15)}
h1{margin:8px 0 0;font-size:26px;font-weight:800;letter-spacing:-.03em}
.sub{margin:6px 0 0;color:var(--muted);font-size:13px}
.live{padding:8px 12px;border-radius:999px;border:1px solid var(--line);background:rgba(15,23,42,.75);color:var(--green);font-size:12px;font-weight:800}
.banner{padding:11px 14px;border-radius:12px;margin-bottom:12px;border:1px solid rgba(52,211,153,.3);background:rgba(6,78,59,.35);color:#bbf7d0;font-weight:700}
.banner.warn{border-color:rgba(251,191,36,.35);background:rgba(120,53,15,.35);color:#fde68a}
.panel{background:linear-gradient(180deg,rgba(18,28,46,.96),rgba(12,20,34,.98));border:1px solid var(--line);border-radius:16px;margin-bottom:12px;overflow:hidden;box-shadow:0 16px 40px rgba(0,0,0,.35)}
.phd{display:flex;justify-content:space-between;align-items:center;gap:10px;padding:12px 14px;border-bottom:1px solid var(--line)}
.phd h2{margin:0;font-size:12px;font-weight:800;letter-spacing:.08em;color:#dbe4f3}
.hint{color:var(--muted);font-size:12px}
.cfg-pills{display:flex;flex-wrap:wrap;gap:8px;padding:12px 14px;border-bottom:1px solid rgba(148,163,184,.08)}
.pill-cfg{display:inline-flex;align-items:center;gap:6px;height:30px;padding:0 11px;border-radius:999px;border:1px solid var(--line);background:rgba(7,12,22,.55);font-size:12px;font-weight:750;color:#dbe4f3}
.pill-cfg em{font-style:normal;color:var(--muted);font-weight:700}
.pill-cfg .on{color:var(--green)}.pill-cfg .off{color:var(--amber)}
.hist-wrap{max-height:0;overflow:hidden;opacity:0;transition:max-height .18s ease,opacity .15s ease,padding .15s ease}
.hist-wrap.open{max-height:280px;opacity:1;padding-bottom:4px}
.hist-toggle{height:30px;padding:0 10px;font-size:12px}
@media (prefers-reduced-motion:reduce){.hist-wrap{transition:none}}
.chips{display:flex;flex-wrap:wrap;gap:8px;padding:0 14px 14px}
.schip{height:auto;min-height:52px;min-width:88px;flex:1 1 88px;display:flex;flex-direction:column;align-items:flex-start;justify-content:center;gap:6px;padding:10px 12px;border-radius:14px;border:1px solid var(--line);background:rgba(7,12,22,.55);color:var(--text);cursor:pointer;transition:border-color .12s ease,box-shadow .12s ease,background .12s ease,transform .12s ease}
.schip span{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.06em}
.schip b{font-size:22px;font-weight:850;font-variant-numeric:tabular-nums;line-height:1}
.schip:hover{background:rgba(15,23,42,.85);border-color:rgba(148,163,184,.28)}
.schip.active{border-color:rgba(34,211,238,.55);box-shadow:0 0 0 1px rgba(34,211,238,.22) inset,0 0 20px rgba(34,211,238,.08);background:rgba(8,47,73,.28)}
.schip.healthy.active{border-color:rgba(52,211,153,.5);box-shadow:0 0 0 1px rgba(52,211,153,.2) inset}
.schip.s401.active{border-color:rgba(59,130,246,.55)}
.schip.s402.active{border-color:rgba(251,191,36,.55)}
.schip.s403.active{border-color:rgba(251,113,133,.55)}
.schip.s429.active{border-color:rgba(167,139,250,.55)}
.schip.disabled.active{border-color:rgba(148,163,184,.45)}
.schip:focus-visible{outline:2px solid rgba(34,211,238,.65);outline-offset:2px}
@media (prefers-reduced-motion:reduce){.schip{transition:none}}
.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;padding:0 14px 14px}
.metric{position:relative;background:rgba(7,12,22,.45);border:1px solid var(--line);border-radius:14px;padding:14px}
.metric:before{content:"";position:absolute;left:0;top:0;bottom:0;width:3px;border-radius:14px 0 0 14px;background:linear-gradient(180deg,var(--cyan),var(--blue))}
.metric.m402:before{background:linear-gradient(180deg,var(--amber),#f59e0b)}
.metric.m403:before{background:linear-gradient(180deg,var(--red),#e11d48)}
.metric.m429:before{background:linear-gradient(180deg,var(--violet),#7c3aed)}
.metric .l{color:var(--muted);font-size:11px;font-weight:800}
.metric .n{margin-top:8px;font-size:30px;font-weight:850;font-variant-numeric:tabular-nums}
.row{display:flex;align-items:center;gap:10px;flex-wrap:wrap;padding:12px 14px;border-top:1px solid rgba(148,163,184,.08)}
input[type=search],input[type=password],input[type=text],input[type=number],select{height:38px;min-width:160px;flex:1;border:1px solid rgba(148,163,184,.28);border-radius:11px;background:rgba(7,12,22,.85);color:var(--text);padding:0 12px;font:inherit;outline:none}
input:focus,select:focus{border-color:rgba(96,165,250,.7);box-shadow:0 0 0 3px rgba(59,130,246,.16)}
label.chk{display:flex;align-items:center;gap:8px;color:#dbe4f3;font-weight:650;white-space:nowrap}
button{height:38px;border:1px solid rgba(148,163,184,.28);border-radius:11px;background:rgba(30,41,59,.9);color:var(--text);padding:0 13px;font:inherit;font-weight:750;cursor:pointer}
button:hover{background:rgba(51,65,85,.95)}button:disabled{opacity:.35;cursor:not-allowed}
.bp{background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8;color:#fff}
.bd{background:rgba(244,63,94,.18);border-color:rgba(251,113,133,.4);color:#fecdd3}
.bg{background:transparent}.bs{background:rgba(15,23,42,.7)}
.msg{min-height:18px;color:var(--muted);font-size:12.5px;font-weight:700}
.msg.err{color:#fda4af}
.table-wrap{overflow:auto;max-height:56vh}
.table-wrap.fade{opacity:.45}
table{width:100%;border-collapse:separate;border-spacing:0;min-width:1040px;transition:opacity .12s ease}
@media (prefers-reduced-motion:reduce){table{transition:none}}
th{position:sticky;top:0;z-index:1;background:rgba(15,23,42,.96);color:#c7d4ea;font-size:11px;font-weight:800;letter-spacing:.07em;padding:11px 12px;border-bottom:1px solid var(--line);text-align:left}
td{padding:12px;border-bottom:1px solid rgba(148,163,184,.08);color:#dbe4f3;vertical-align:middle}
tr:hover td{background:rgba(56,189,248,.05)}
td code{font-family:var(--mono);font-size:12px;color:#fff;background:rgba(2,6,23,.75);border:1px solid rgba(148,163,184,.22);border-radius:8px;padding:4px 7px;display:inline-block;max-width:340px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.badge{display:inline-flex;align-items:center;justify-content:center;min-width:48px;height:26px;border-radius:999px;font-weight:850;font-size:12px;border:1px solid transparent;padding:0 8px}
.b401{color:#93c5fd;background:rgba(59,130,246,.14);border-color:rgba(59,130,246,.28)}
.b402{color:#fcd34d;background:rgba(245,158,11,.14);border-color:rgba(245,158,11,.28)}
.b403{color:#fda4af;background:rgba(244,63,94,.14);border-color:rgba(244,63,94,.28)}
.b429{color:#ddd6fe;background:rgba(139,92,246,.16);border-color:rgba(167,139,250,.3)}
.bhealthy{color:#6ee7b7;background:rgba(16,185,129,.12);border-color:rgba(52,211,153,.28)}
.bdisabled{color:#cbd5e1;background:rgba(148,163,184,.12);border-color:rgba(148,163,184,.28)}
.bbanned{color:#fde68a;background:rgba(245,158,11,.12);border-color:rgba(245,158,11,.25)}
.pill{display:inline-flex;height:24px;align-items:center;padding:0 9px;border-radius:999px;background:rgba(148,163,184,.1);border:1px solid rgba(148,163,184,.16);font-size:12px;font-weight:750}
.remain{font-family:var(--mono);font-weight:800;color:#fff;font-size:12px}
.acts{display:flex;flex-wrap:wrap;gap:6px}
.row-action{height:30px;padding:0 10px;border-radius:9px;font-size:12px;background:#1e293b;border-color:#475569}
.row-action:hover{background:#2563eb;border-color:#1d4ed8}
.row-action.danger:hover{background:rgba(244,63,94,.25);border-color:rgba(251,113,133,.45);color:#fecdd3}
.sub2{display:block;color:var(--muted);font-size:11px;margin-top:2px}
.empty{padding:48px;text-align:center;color:var(--muted);font-weight:700}
.foot{color:var(--muted);font-size:12px;line-height:1.6;padding:0 2px}
.drawer-mask{position:fixed;inset:0;background:rgba(2,6,23,.55);backdrop-filter:blur(2px);opacity:0;pointer-events:none;transition:opacity .18s;z-index:40}
.drawer-mask.open{opacity:1;pointer-events:auto}
.drawer{position:fixed;top:0;right:0;height:100vh;width:min(440px,100vw);background:linear-gradient(180deg,#0f172a,#0b1220);border-left:1px solid var(--line);box-shadow:-20px 0 50px rgba(0,0,0,.45);transform:translateX(100%);transition:transform .2s ease;z-index:50;display:flex;flex-direction:column}
.drawer.open{transform:translateX(0)}
.dh{display:flex;justify-content:space-between;align-items:flex-start;gap:10px;padding:16px;border-bottom:1px solid var(--line)}
.dh h3{margin:0;font-size:16px;font-weight:800}.dh p{margin:6px 0 0;color:var(--muted);font-size:12px;line-height:1.5}
.db{flex:1;overflow:auto;padding:14px 16px 20px}
.sec{border:1px solid var(--line);border-radius:14px;padding:12px;margin-bottom:12px;background:rgba(15,23,42,.55)}
.sec h4{margin:0 0 10px;font-size:12px;letter-spacing:.08em;color:#cbd5e1}
.fg{display:grid;gap:8px;margin-bottom:10px}
.fg label{font-size:12px;color:var(--muted);font-weight:700}
.choice{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.choice button{height:auto;min-height:54px;padding:10px;text-align:left;border-radius:12px;background:rgba(7,12,22,.7)}
.choice button.active{border-color:rgba(52,211,153,.55);box-shadow:0 0 0 1px rgba(52,211,153,.25) inset;background:rgba(6,78,59,.25)}
.choice b{display:block;font-size:13px;margin-bottom:4px}
.choice span{display:block;color:var(--muted);font-size:11px;font-weight:600;line-height:1.35}
.df{display:flex;justify-content:flex-end;gap:8px;padding:12px 16px;border-top:1px solid var(--line);background:rgba(2,6,23,.4)}
.hist{display:flex;flex-wrap:wrap;gap:8px;padding:12px 14px}
.hist button{height:auto;min-width:150px;padding:10px;text-align:left}
.hist b{display:block}.hist small{display:block;color:#93a4c3;margin-top:2px}
@media(max-width:700px){h1{font-size:22px}}
</style>
</head>
<body>
<div class="shell">
  <div class="top">
    <div>
      <div class="kicker"><i></i>运维台</div>
      <h1>xAI Autoban</h1>
      <p class="sub">凭据隔离控制台 · v` + pluginVersion + ` · 仅处理 xAI</p>
    </div>
    <div style="display:flex;gap:8px;align-items:center">
      <div class="live" id="syncState">准备中</div>
      <button class="bp" id="openConfigBtn" type="button">编辑配置</button>
    </div>
  </div>

  <div id="authBanner" class="banner warn">正在检测管理密钥…</div>

  <section class="panel">
    <div class="phd"><h2>凭证健康度</h2><div class="hint">点击芯片筛选 · 右上角可编辑巡检配置</div></div>
    <div class="cfg-pills" id="cfgPills">
      <span class="pill-cfg"><em>定时</em> <span id="sumProbeEnabled">-</span></span>
      <span class="pill-cfg"><em>间隔</em> <span id="sumInterval">-</span></span>
      <span class="pill-cfg"><em>执行</em> <span id="sumAutoExec">-</span></span>
      <span class="pill-cfg"><em>问题</em> <span id="sumProbeAction">-</span></span>
      <span class="pill-cfg"><em>成功</em> <span id="sumOnSuccess">-</span></span>
      <span class="pill-cfg"><em>模式</em> <span id="sumMode">-</span></span>
    </div>
    <div class="chips" id="statusChips" role="toolbar" aria-label="凭证状态筛选">
      <button type="button" class="schip active" data-filter="all"><span>全部</span><b id="c_all">-</b></button>
      <button type="button" class="schip healthy" data-filter="healthy"><span>健康</span><b id="c_healthy">-</b></button>
      <button type="button" class="schip banned" data-filter="banned"><span>隔离</span><b id="c_banned">-</b></button>
      <button type="button" class="schip s401" data-filter="401"><span>401</span><b id="c_401">-</b></button>
      <button type="button" class="schip s402" data-filter="402"><span>402</span><b id="c_402">-</b></button>
      <button type="button" class="schip s403" data-filter="403"><span>403</span><b id="c_403">-</b></button>
      <button type="button" class="schip s429" data-filter="429"><span>429</span><b id="c_429">-</b></button>
      <button type="button" class="schip disabled" data-filter="disabled"><span>已禁用</span><b id="c_disabled">-</b></button>
    </div>
    <div class="metrics" hidden>
      <div class="metric"><div class="l">当前隔离</div><div class="n" id="total">-</div></div>
      <div class="metric m402"><div class="l">402</div><div class="n" id="count402">-</div></div>
      <div class="metric m403"><div class="l">403</div><div class="n" id="count403">-</div></div>
      <div class="metric m429"><div class="l">429</div><div class="n" id="count429">-</div></div>
    </div>
  </section>

  <section class="panel">
    <div class="phd">
      <h2>巡检历史</h2>
      <div style="display:flex;gap:8px;align-items:center">
        <div class="hint">内存记录，重启后清空</div>
        <button class="hist-toggle bg" id="toggleHistBtn" type="button" aria-expanded="false">展开</button>
      </div>
    </div>
    <div class="hist-wrap" id="histWrap">
      <div class="hist" id="probeHistory">暂无记录</div>
    </div>
  </section>

  <section class="panel">
    <div class="phd">
      <h2>控制面</h2>
      <div class="hint" id="authHint">写操作走 Management API</div>
    </div>
    <div class="row" id="keyRow">
      <input id="mgmtKeyInput" type="password" placeholder="粘贴 CPA 管理密钥" autocomplete="off">
      <button class="bp" id="saveKeyBtn" type="button">保存密钥</button>
      <button class="bg" id="clearKeyBtn" type="button">清除</button>
      <button class="bs" id="toggleKeyBtn" type="button" hidden>管理密钥</button>
    </div>
    <div class="row">
      <input id="search" type="search" placeholder="搜索 Auth ID / 名称 / 原因" autocomplete="off">
      <button class="bp" type="button" onclick="loadData()">刷新</button>
      <button id="btnProbe" class="bs" type="button" onclick="runProbe()" disabled>立即巡检</button>
      <button id="clearFilterBtn" class="bg" type="button">清除筛选</button>
      <label class="chk"><input id="autoRefresh" type="checkbox" checked> 30 秒自动刷新</label>
    </div>
    <div class="row">
      <button id="unbanSelected" class="bs" type="button" onclick="bulkAct('unban')" disabled>解禁所选 (0)</button>
      <button id="banSelected" class="bs" type="button" onclick="bulkAct('ban')" disabled>ban 所选</button>
      <button id="disableSelected" class="bs" type="button" onclick="bulkAct('disable')" disabled>disable 所选</button>
      <button id="reenableSelected" class="bs" type="button" onclick="bulkAct('reenable')" disabled>reenable 所选</button>
      <button id="unbanAll" class="bd" type="button" onclick="unbanAll()" disabled>全部解禁</button>
    </div>
    <div class="row"><div id="message" class="msg">系统待命</div></div>
  </section>

  <section class="panel">
    <div class="phd"><h2>全部凭证</h2><div class="hint" id="resultCount">0 条</div></div>
    <div class="table-wrap">
      <table>
        <thead><tr>
          <th style="width:40px"><input id="selectPage" type="checkbox"></th>
          <th>Auth ID</th><th>名称</th><th>状态</th><th>动作</th><th>原因</th><th>剩余</th><th>最近巡检</th><th>操作</th>
        </tr></thead>
        <tbody id="rows"></tbody>
      </table>
      <div id="empty" class="empty" hidden>没有匹配的凭证</div>
    </div>
  </section>
  <p class="foot">
    点状态芯片筛选 · 有密钥后可单行/批量操作。
    <b>ban</b>=内存隔离（调度跳过）；<b>disable</b>=写入凭证停用。
    配置改动立即生效，进程重启后回落 config.yaml。
  </p>
</div>

<div class="drawer-mask" id="drawerMask"></div>
<aside class="drawer" id="drawer" aria-hidden="true">
  <div class="dh">
    <div>
      <h3>服务端巡检配置</h3>
      <p>配置定时巡检、自动执行模式，以及失败/成功策略。保存后立即应用到当前插件进程。</p>
    </div>
    <button class="bg" id="closeConfigBtn" type="button">✕</button>
  </div>
  <div class="db">
    <div class="sec">
      <h4>调度</h4>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_enabled" type="checkbox"> 启用定时巡检</label>
      <div class="fg"><label>间隔（秒）</label><input id="f_probe_interval_seconds" type="number" min="30" step="1"></div>
      <div class="fg"><label>超时（秒）</label><input id="f_probe_timeout_seconds" type="number" min="5" step="1"></div>
      <div class="fg"><label>并发</label><input id="f_probe_concurrency" type="number" min="1" step="1"></div>
      <div class="fg"><label>QPS</label><input id="f_probe_qps" type="number" min="0.1" step="0.1"></div>
      <div class="fg"><label>探测模式</label>
        <select id="f_probe_mode"><option value="models">models（轻量）</option><option value="responses_mini">responses_mini（更准）</option></select>
      </div>
    </div>
    <div class="sec">
      <h4>自动执行（对齐 Codex 巡检）</h4>
      <div class="choice" id="autoExecChoices" style="margin-bottom:10px">
        <button type="button" data-v="0"><b>只输出结果</b><span>巡检只记录；失败最多 ban 展示，不 disable/delete</span></button>
        <button type="button" data-v="1"><b>自动执行</b><span>按下方策略处理问题账号与恢复</span></button>
      </div>
      <div class="fg"><label>成功策略 probe_on_success</label>
        <div class="choice" id="successChoices">
          <button type="button" data-v="none"><b>不处理</b><span>仅记录，不改 ban/disabled</span></button>
          <button type="button" data-v="unban"><b>自动解 ban</b><span>清除内存隔离（默认）</span></button>
          <button type="button" data-v="reenable"><b>重新启用</b><span>disabled=false，不碰 ban</span></button>
          <button type="button" data-v="unban_and_reenable"><b>解 ban + 启用</b><span>同时恢复调度与启用态</span></button>
        </div>
      </div>
      <div class="fg"><label>问题账号策略 probe_action</label>
        <div class="choice" id="failChoices">
          <button type="button" data-v="ban"><b>仅 ban</b><span>内存隔离，最安全</span></button>
          <button type="button" data-v="disable"><b>禁用账号</b><span>写入 disabled=true</span></button>
          <button type="button" data-v="delete"><b>删除（回退）</b><span>无正式删除 API 时回退 disable</span></button>
        </div>
      </div>
      <div class="fg"><label>delete 回退</label>
        <select id="f_delete_fallback"><option value="disable">disable</option><option value="ban">ban</option></select>
      </div>
    </div>
    <div class="sec">
      <h4>按状态默认动作</h4>
      <div class="fg"><label>401</label><select id="f_action_on_401"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>402</label><select id="f_action_on_402"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>403</label><select id="f_action_on_403"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>429（建议 ban）</label><select id="f_action_on_429"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>动作冷却（秒）</label><input id="f_action_cooldown_seconds" type="number" min="0" step="1"></div>
    </div>
  </div>
  <div class="df">
    <button class="bg" id="discardConfigBtn" type="button">丢弃更改</button>
    <button class="bp" id="saveConfigBtn" type="button">保存并生效</button>
  </div>
</aside>

<script>
const resourceBase='/v0/resource/plugins/xai-autoban';
const mgmtBase='/v0/management/plugins/xai-autoban';
const KEY_STORE='xai_autoban_management_key';
const state={bans:[],credentials:[],counts:{},filter:'all',query:'',selected:new Set(),timer:null,mgmtKey:'',settings:{},success:'unban',fail:'ban',autoExecute:true,history:[]};
const $=id=>document.getElementById(id);
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function readManagementKey(){
  try{const m=localStorage.getItem(KEY_STORE); if(m&&m.trim()) return m.trim();}catch(_){}
  const keys=['cliproxyapi_management_key','management_key','cpa_management_key','managementKey','management-password','apiKey','token'];
  for(const k of keys){try{const v=localStorage.getItem(k); if(v&&v.trim()&&v.length<512) return v.trim();}catch(_){}}
  try{
    for(let i=0;i<localStorage.length;i++){
      const k=localStorage.key(i); if(!k) continue;
      const raw=localStorage.getItem(k); if(!raw||raw.length>8000) continue;
      if(/management|mgmt|cpa|cliproxy/i.test(k)&&raw.trim()&&!raw.trim().startsWith('{')&&raw.length<512) return raw.trim();
      if(raw.trim().startsWith('{')){
        try{
          const obj=JSON.parse(raw); const st=[obj];
          while(st.length){const cur=st.pop(); if(!cur||typeof cur!=='object') continue;
            for(const [kk,vv] of Object.entries(cur)){
              if(typeof vv==='string'&&vv.trim()&&vv.length<512&&/management|mgmt|password|apiKey|token/i.test(kk)) return vv.trim();
              if(vv&&typeof vv==='object') st.push(vv);
            }}
        }catch(_){}
      }
    }
  }catch(_){}
  return '';
}
function setActionEnabled(ok){
  const ids=['btnProbe','unbanSelected','banSelected','disableSelected','reenableSelected','unbanAll','saveConfigBtn'];
  ids.forEach(id=>{const el=$(id); if(el) el.disabled=!ok;});
  const n=state.selected.size;
  if(ok){
    ['unbanSelected','banSelected','disableSelected','reenableSelected'].forEach(id=>{const el=$(id); if(el) el.disabled=n===0;});
  }
  if($('unbanSelected')) $('unbanSelected').textContent='解禁所选 ('+n+')';
}
function setAuthUI(){
  state.mgmtKey=readManagementKey();
  const ok=!!state.mgmtKey;
  const b=$('authBanner'); b.className='banner'+(ok?'':' warn');
  b.textContent=ok?'已授权：可写操作（解禁 / ban / disable / reenable / 巡检 / 配置）。':'只读模式：请先保存管理密钥再执行写操作。';
  const keyRow=$('keyRow');
  const input=$('mgmtKeyInput');
  const saveBtn=$('saveKeyBtn');
  const clearBtn=$('clearKeyBtn');
  const toggle=$('toggleKeyBtn');
  if(ok){
    if(input){ input.hidden=true; input.placeholder='已保存管理密钥'; }
    if(saveBtn) saveBtn.hidden=true;
    if(clearBtn) clearBtn.hidden=false;
    if(toggle){ toggle.hidden=false; toggle.textContent='更换密钥'; }
    if($('authHint')) $('authHint').textContent='已授权 · 密钥仅保存在本机浏览器';
  }else{
    if(input){ input.hidden=false; input.placeholder='粘贴 CPA 管理密钥'; }
    if(saveBtn) saveBtn.hidden=false;
    if(clearBtn) clearBtn.hidden=true;
    if(toggle) toggle.hidden=true;
    if($('authHint')) $('authHint').textContent='写操作需要管理密钥';
  }
  setActionEnabled(ok); return ok;
}
async function apiResource(path){
  const r=await fetch(resourceBase+path,{cache:'no-store'}); const t=await r.text();
  let d; try{d=JSON.parse(t)}catch(_){throw new Error(t||('HTTP '+r.status))}
  if(!r.ok) throw new Error(d.error||('HTTP '+r.status)); return d;
}
async function apiMgmt(method,path,body){
  if(!state.mgmtKey) throw new Error('缺少管理密钥');
  const r=await fetch(mgmtBase+path,{method,cache:'no-store',headers:{
    'Authorization':'Bearer '+state.mgmtKey,'Content-Type':'application/json',
    'X-Management-Key':state.mgmtKey,'X-Api-Key':state.mgmtKey
  },body:body?JSON.stringify(body):undefined});
  const t=await r.text(); let d; try{d=JSON.parse(t)}catch(_){throw new Error(t||('HTTP '+r.status))}
  if(!r.ok) throw new Error(d.error||d.message||('HTTP '+r.status)); return d;
}
function setMessage(text,err=false){$('message').textContent=text;$('message').className='msg'+(err?' err':'')}
function counts(){const o={401:0,402:0,403:0,429:0}; for(const b of state.bans) if(o[b.status_code]!==undefined) o[b.status_code]++; return o}
function paintChips(){
  const c=state.counts||{};
  const set=(id,v)=>{const el=$(id); if(el) el.textContent=String(v??0)};
  set('c_all',c.all); set('c_healthy',c.healthy); set('c_banned',c.banned);
  set('c_401',c['401']); set('c_402',c['402']); set('c_403',c['403']); set('c_429',c['429']); set('c_disabled',c.disabled);
  document.querySelectorAll('#statusChips [data-filter]').forEach(btn=>{
    btn.classList.toggle('active', btn.dataset.filter===state.filter);
  });
}
function setFilter(f){
  if(state.filter===f) state.filter='all'; else state.filter=f||'all';
  const wrap=document.querySelector('.table-wrap');
  if(wrap){ wrap.classList.add('fade'); setTimeout(()=>{paintChips(); render(); wrap.classList.remove('fade');},90); }
  else { paintChips(); render(); }
}
function matchFilter(c){
  const f=state.filter||'all';
  if(f==='all') return true;
  if(f==='healthy') return !c.disabled && !c.banned;
  if(f==='banned') return !!c.banned;
  if(f==='disabled') return !!c.disabled;
  if(f==='401'||f==='402'||f==='403'||f==='429') return !!c.banned && String(c.status_code)===f;
  return true;
}
function filtered(){
  const q=state.query.toLowerCase();
  const list=(state.credentials&&state.credentials.length)?state.credentials:state.bans.map(b=>({
    auth_id:b.auth_id, status_code:b.status_code, reason:b.reason, action:b.action,
    banned:true, disabled:false, status:String(b.status_code||'banned'),
    banned_at:b.banned_at, reset_at:b.reset_at, remaining_seconds:b.remaining_seconds, source:b.source
  }));
  return list.filter(c=>{
    if(!matchFilter(c)) return false;
    if(!q) return true;
    return [c.auth_id,c.name,c.label,c.reason,c.action,c.status].some(x=>String(x||'').toLowerCase().includes(q));
  });
}
function formatDate(v){const d=new Date(v); return Number.isNaN(d.getTime())?v:d.toLocaleString('zh-CN',{hour12:false})}
function formatRemaining(s){s=Math.max(0,Number(s||0)); const d=Math.floor(s/86400),h=Math.floor(s%86400/3600),m=Math.floor(s%3600/60); if(d)return d+'天 '+h+'小时'; if(h)return h+'小时 '+m+'分'; return m+'分钟'}
function reasonLabel(r){return ({payment_required:'额度不足',forbidden:'禁止访问',unauthorized:'未授权',rate_limited:'限流',rate_limited_fallback:'限流(默认等待)',probe_failed:'巡检失败',manual:'手动'})[r]||r||'-'}
function labelAction(a){return ({ban:'仅 ban',disable:'禁用',delete:'删除/回退',none:'不处理',unban:'自动解 ban',reenable:'重新启用',unban_and_reenable:'解 ban+启用'})[a]||a||'-'}

function renderSettingsSummary(s){
  state.settings=s||{};
  const pe=$('sumProbeEnabled');
  if(pe){ pe.textContent=s.probe_enabled?'已启用':'关闭'; pe.className=s.probe_enabled?'on':'off'; }
  if($('sumInterval')) $('sumInterval').textContent=(s.probe_interval_seconds||'-')+'s';
  const auto=s.auto_execute!==false;
  const ae=$('sumAutoExec');
  if(ae){ ae.textContent=auto?'自动执行':'只输出'; ae.className=auto?'on':'off'; }
  if($('sumProbeAction')) $('sumProbeAction').textContent=labelAction(s.probe_action);
  if($('sumOnSuccess')) $('sumOnSuccess').textContent=labelAction(s.probe_on_success);
  if($('sumMode')) $('sumMode').textContent=s.probe_mode||'-';
}
function renderHistory(list){
  state.history=list||[];
  const el=$('probeHistory'); if(!el) return;
  if(!state.history.length){ el.textContent='暂无记录'; return; }
  el.innerHTML=state.history.slice(0,12).map(run=>{
    const r=run.result||{};
    const mode=r.report_only?'只输出':'自动执行';
    const st=run.error?'失败':'完成';
    return '<button type="button" class="bs" title="#'+run.id+'">'+
      '<b>#'+run.id+' · '+st+'</b>'+
      '<small>'+esc(r.finished_at||r.started_at||'')+' · '+esc(r.trigger||'')+'</small>'+
      '<small style="color:#cbd5e1">'+mode+' · 检'+(r.checked||0)+' 成'+(r.ok||0)+' 败'+(r.failed||0)+'</small></button>';
  }).join('');
}
function fillDrawer(s){
  $('f_probe_enabled').checked=!!s.probe_enabled;
  $('f_probe_interval_seconds').value=s.probe_interval_seconds??600;
  $('f_probe_timeout_seconds').value=s.probe_timeout_seconds??20;
  $('f_probe_concurrency').value=s.probe_concurrency??3;
  $('f_probe_qps').value=s.probe_qps??2;
  $('f_probe_mode').value=s.probe_mode||'models';
  $('f_delete_fallback').value=s.delete_fallback||'disable';
  $('f_action_on_401').value=s.action_on_401||'ban';
  $('f_action_on_402').value=s.action_on_402||'ban';
  $('f_action_on_403').value=s.action_on_403||'ban';
  $('f_action_on_429').value=s.action_on_429||'ban';
  $('f_action_cooldown_seconds').value=s.action_cooldown_seconds??60;
  state.success=s.probe_on_success||'unban';
  state.fail=s.probe_action||'ban';
  state.autoExecute=s.auto_execute!==false;
  paintChoices();
}
function paintChoices(){
  document.querySelectorAll('#successChoices button').forEach(b=>b.classList.toggle('active',b.dataset.v===state.success));
  document.querySelectorAll('#failChoices button').forEach(b=>b.classList.toggle('active',b.dataset.v===state.fail));
  document.querySelectorAll('#autoExecChoices button').forEach(b=>b.classList.toggle('active',(b.dataset.v==='1')===!!state.autoExecute));
}
function openDrawer(){
  if(!state.mgmtKey){ setMessage('请先保存管理密钥再编辑配置',true); return; }
  fillDrawer(state.settings||{});
  $('drawer').classList.add('open'); $('drawerMask').classList.add('open'); $('drawer').setAttribute('aria-hidden','false');
}
function closeDrawer(){ $('drawer').classList.remove('open'); $('drawerMask').classList.remove('open'); $('drawer').setAttribute('aria-hidden','true'); }
function collectDraft(){
  return {
    probe_enabled: $('f_probe_enabled').checked,
    probe_interval_seconds: Number($('f_probe_interval_seconds').value||0),
    probe_timeout_seconds: Number($('f_probe_timeout_seconds').value||0),
    probe_concurrency: Number($('f_probe_concurrency').value||0),
    probe_qps: Number($('f_probe_qps').value||0),
    probe_mode: $('f_probe_mode').value,
    probe_on_success: state.success,
    probe_action: state.fail,
    auto_execute: !!state.autoExecute,
    delete_fallback: $('f_delete_fallback').value,
    action_on_401: $('f_action_on_401').value,
    action_on_402: $('f_action_on_402').value,
    action_on_403: $('f_action_on_403').value,
    action_on_429: $('f_action_on_429').value,
    action_cooldown_seconds: Number($('f_action_cooldown_seconds').value||0)
  };
}
async function saveSettings(){
  try{
    setMessage('正在保存配置…');
    const res=await apiMgmt('PUT','/settings',collectDraft());
    renderSettingsSummary(res.settings||{});
    setMessage('配置已生效'+(res.note?(' · '+res.note):''));
    closeDrawer();
    await loadData(true);
  }catch(e){ setMessage(e.message,true); }
}
async function loadData(silent=false){
  try{
    if(!silent){ $('syncState').textContent='同步中'; setMessage('正在加载…'); }
    const data=await apiResource('/data');
    state.bans=data.bans||[];
    state.credentials=data.credentials||[];
    state.counts=data.counts||{};
    if(data.settings) renderSettingsSummary(data.settings);
    if(data.probe&&data.probe.history) renderHistory(data.probe.history);
    for(const id of [...state.selected]) if(!state.credentials.some(x=>x.auth_id===id)&&!state.bans.some(x=>x.auth_id===id)) state.selected.delete(id);
    const c=counts();
    if($('total')) $('total').textContent=String(data.count||0);
    if($('count402')) $('count402').textContent=String(c[402]||0);
    if($('count403')) $('count403').textContent=String(c[403]||0);
    if($('count429')) $('count429').textContent=String(c[429]||0);
    paintChips();
    $('syncState').textContent='在线';
    setMessage('已更新 · '+new Date().toLocaleTimeString('zh-CN',{hour12:false}));
    render();
  }catch(e){ $('syncState').textContent='异常'; setMessage(e.message,true); }
}
function statusBadge(c){
  const st=c.status||(c.disabled?'disabled':(c.banned?String(c.status_code||'banned'):'healthy'));
  const map={healthy:['bhealthy','健康'],disabled:['bdisabled','已禁用'],'401':['b401','401'],'402':['b402','402'],'403':['b403','403'],'429':['b429','429'],banned:['bbanned','隔离']};
  const [cls,label]=map[st]||['bbanned',st];
  let html='<span class="badge '+cls+'">'+esc(label)+'</span>';
  if(c.disabled&&c.banned) html+=' <span class="pill">仍隔离</span>';
  return html;
}
function rowActions(c){
  const id=esc(c.auth_id);
  const dis=state.mgmtKey?'':'disabled';
  const btns=[];
  if(c.banned) btns.push('<button class="row-action" data-act="unban" data-id="'+id+'" '+dis+'>解禁</button>');
  if(!c.banned) btns.push('<button class="row-action" data-act="ban" data-id="'+id+'" '+dis+'>ban</button>');
  if(!c.disabled) btns.push('<button class="row-action danger" data-act="disable" data-id="'+id+'" '+dis+'>disable</button>');
  if(c.disabled) btns.push('<button class="row-action" data-act="reenable" data-id="'+id+'" '+dis+'>reenable</button>');
  return '<div class="acts">'+btns.join('')+'</div>';
}
function probeCell(c){
  if(!c.last_probe_at) return '<span class="pill">未巡检</span>';
  const ok=c.last_probe_ok===true;
  const mark=ok?'成功':'失败';
  const code=c.last_probe_status?(' · '+c.last_probe_status):'';
  return esc(mark+code)+'<span class="sub2">'+esc(formatDate(c.last_probe_at))+'</span>';
}
function render(){
  const list=filtered();
  const filterLabel={all:'全部',healthy:'健康',banned:'隔离',disabled:'已禁用','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
  $('resultCount').textContent=list.length+' 条 · 筛选 '+filterLabel;
  $('rows').innerHTML=list.map(c=>{
    const name=c.name||c.label||'-';
    const remain=c.banned?formatRemaining(c.remaining_seconds):'-';
    return '<tr>'+
      '<td><input type="checkbox" data-id="'+esc(c.auth_id)+'" '+(state.selected.has(c.auth_id)?'checked':'')+'></td>'+
      '<td><code title="'+esc(c.auth_id)+'">'+esc(c.auth_id)+'</code></td>'+
      '<td>'+esc(name)+'</td>'+
      '<td>'+statusBadge(c)+'</td>'+
      '<td><span class="pill">'+esc(c.action||(c.banned?'ban':'-'))+'</span></td>'+
      '<td>'+esc(reasonLabel(c.reason)||'-')+'</td>'+
      '<td class="remain">'+esc(remain)+'</td>'+
      '<td>'+probeCell(c)+'</td>'+
      '<td>'+rowActions(c)+'</td></tr>';
  }).join('');
  const empty=$('empty');
  empty.hidden=list.length>0;
  empty.textContent=state.filter==='all'&&!state.query?'暂无 xAI 凭证':'没有匹配的凭证 · 可清除筛选';
  document.querySelectorAll('#rows input[type=checkbox]').forEach(input=>input.addEventListener('change',()=>{
    input.checked?state.selected.add(input.dataset.id):state.selected.delete(input.dataset.id);
    setActionEnabled(!!state.mgmtKey);
  }));
  document.querySelectorAll('#rows [data-act]').forEach(btn=>btn.addEventListener('click',()=>runRowAction(btn.dataset.act,btn.dataset.id)));
  setActionEnabled(!!state.mgmtKey);
}
async function runRowAction(act,id){
  if(!id) return;
  const labels={unban:'解禁',ban:'ban 隔离',disable:'禁用',reenable:'重新启用'};
  if(!confirm('确认对凭证执行「'+(labels[act]||act)+'」？\n'+id)) return;
  try{
    setMessage('正在执行 '+ (labels[act]||act) +'…');
    if(act==='unban'){
      await apiMgmt('POST','/unban',{auth_id:id});
    }else{
      await apiMgmt('POST','/apply-action',{auth_id:id,action:act,force:true});
    }
    state.selected.delete(id);
    setMessage('已完成 · '+(labels[act]||act));
    await loadData(true);
  }catch(e){ setMessage(e.message,true); }
}
async function unbanOne(id){ return runRowAction('unban',id); }
async function bulkAct(act){
  const ids=[...state.selected];
  if(!ids.length){ setMessage('请先勾选凭证',true); return; }
  const labels={unban:'解禁',ban:'ban',disable:'disable',reenable:'reenable'};
  if(!confirm('确认对所选 '+ids.length+' 条执行「'+(labels[act]||act)+'」？')) return;
  try{
    let i=0;
    for(const id of ids){
      i++; setMessage('正在处理 '+i+'/'+ids.length+' …');
      if(act==='unban') await apiMgmt('POST','/unban',{auth_id:id});
      else await apiMgmt('POST','/apply-action',{auth_id:id,action:act,force:true});
    }
    state.selected.clear();
    setMessage('批量完成 · '+(labels[act]||act)+' × '+ids.length);
    await loadData(true);
  }catch(e){ setMessage(e.message,true); }
}
async function unbanSelected(){ return bulkAct('unban'); }
async function unbanAll(){ if(!confirm('确认解禁全部隔离？')) return; try{ await apiMgmt('POST','/unban-all',{}); state.selected.clear(); setMessage('已全部解禁'); await loadData(true);}catch(e){setMessage(e.message,true)} }
async function runProbe(){ if(!confirm('立即巡检全部 xAI 凭据？')) return; try{ setMessage('巡检中…'); const res=await apiMgmt('POST','/probe',{force:false}); setMessage('巡检完成 成功='+(res.result&&res.result.ok)+' 失败='+(res.result&&res.result.failed)+(res.result&&res.result.report_only?'（只输出结果）':'')); await loadData(true);}catch(e){setMessage(e.message,true)} }

$('saveKeyBtn').onclick=()=>{const v=$('mgmtKeyInput').value.trim(); if(!v){setMessage('请先粘贴管理密钥',true);return;} localStorage.setItem(KEY_STORE,v); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('密钥已保存');};
$('clearKeyBtn').onclick=()=>{localStorage.removeItem(KEY_STORE); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('已清除密钥');};
if($('toggleKeyBtn')) $('toggleKeyBtn').onclick=()=>{
  const input=$('mgmtKeyInput'); const saveBtn=$('saveKeyBtn');
  if(!input) return;
  input.hidden=!input.hidden;
  if(saveBtn) saveBtn.hidden=input.hidden;
  if(!input.hidden) input.focus();
};
if($('clearFilterBtn')) $('clearFilterBtn').onclick=()=>{state.filter='all'; state.query=''; if($('search')) $('search').value=''; paintChips(); render(); setMessage('已清除筛选');};
$('search').oninput=e=>{state.query=e.target.value.trim(); render();};
$('selectPage').onchange=e=>{for(const c of filtered()) e.target.checked?state.selected.add(c.auth_id):state.selected.delete(c.auth_id); render();};
$('autoRefresh').onchange=()=>{if(state.timer) clearInterval(state.timer); state.timer=$('autoRefresh').checked?setInterval(()=>loadData(true),30000):null;};
document.querySelectorAll('#statusChips [data-filter]').forEach(btn=>btn.onclick=()=>setFilter(btn.dataset.filter));
if($('toggleHistBtn')) $('toggleHistBtn').onclick=()=>{
  const wrap=$('histWrap'); const btn=$('toggleHistBtn'); if(!wrap||!btn) return;
  const open=wrap.classList.toggle('open');
  btn.textContent=open?'收起':'展开';
  btn.setAttribute('aria-expanded', open?'true':'false');
};
$('openConfigBtn').onclick=openDrawer; $('closeConfigBtn').onclick=closeDrawer; $('drawerMask').onclick=closeDrawer;
$('discardConfigBtn').onclick=()=>{fillDrawer(state.settings||{}); setMessage('已恢复为当前生效配置');};
$('saveConfigBtn').onclick=saveSettings;
document.querySelectorAll('#successChoices button').forEach(b=>b.onclick=()=>{state.success=b.dataset.v; paintChoices();});
document.querySelectorAll('#failChoices button').forEach(b=>b.onclick=()=>{state.fail=b.dataset.v; paintChoices();});
document.querySelectorAll('#autoExecChoices button').forEach(b=>b.onclick=()=>{state.autoExecute=b.dataset.v==='1'; paintChoices();});

setAuthUI();
if($('autoRefresh').checked) state.timer=setInterval(()=>loadData(true),30000);
loadData();
</script>
</body>
</html>`
}
