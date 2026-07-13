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
:root{color-scheme:dark;--bg:#070b14;--panel:#101a2c;--line:rgba(148,163,184,.16);--text:#f8fafc;--muted:#93a4c3;--cyan:#22d3ee;--blue:#3b82f6;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--violet:#a78bfa;--mono:ui-monospace,Consolas,monospace;--sans:Inter,ui-sans-serif,system-ui,sans-serif}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:var(--sans);color:var(--text);background:radial-gradient(1000px 500px at 10% -10%,rgba(34,211,238,.1),transparent 50%),radial-gradient(800px 400px at 100% 0,rgba(59,130,246,.1),transparent 45%),linear-gradient(180deg,#070b14,#0a101c);font-size:14px}
.shell{max-width:1540px;margin:0 auto;padding:18px 20px 36px}
.top{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;margin-bottom:14px}
.kicker{display:inline-flex;align-items:center;gap:8px;color:var(--cyan);font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
.kicker i{width:7px;height:7px;border-radius:50%;background:var(--cyan);box-shadow:0 0 0 4px rgba(34,211,238,.15)}
h1{margin:8px 0 0;font-size:26px;font-weight:800;letter-spacing:-.03em}
.sub{margin:6px 0 0;color:var(--muted);font-size:13px}
.live{padding:8px 12px;border-radius:999px;border:1px solid var(--line);background:rgba(15,23,42,.75);color:var(--green);font-size:12px;font-weight:800}
.banner{padding:11px 14px;border-radius:12px;margin-bottom:12px;border:1px solid rgba(52,211,153,.3);background:rgba(6,78,59,.35);color:#bbf7d0;font-weight:700}
.banner.warn{border-color:rgba(251,191,36,.35);background:rgba(120,53,15,.35);color:#fde68a}
.panel{background:linear-gradient(180deg,rgba(18,28,46,.96),rgba(12,20,34,.98));border:1px solid var(--line);border-radius:16px;margin-bottom:12px;overflow:hidden;box-shadow:0 16px 40px rgba(0,0,0,.35)}
.phd{display:flex;justify-content:space-between;align-items:center;gap:10px;padding:12px 14px;border-bottom:1px solid var(--line)}
.phd h2{margin:0;font-size:12px;font-weight:800;letter-spacing:.08em;text-transform:uppercase;color:#dbe4f3}
.hint{color:var(--muted);font-size:12px}
.cfg-summary{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:10px;padding:12px 14px}
.chip{background:rgba(7,12,22,.55);border:1px solid var(--line);border-radius:12px;padding:12px}
.chip .l{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.06em;text-transform:uppercase}
.chip .v{margin-top:8px;font-size:15px;font-weight:800}
.chip .v.on{color:var(--green)}.chip .v.off{color:var(--amber)}
.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;padding:0 14px 14px}
.metric{position:relative;background:rgba(7,12,22,.45);border:1px solid var(--line);border-radius:14px;padding:14px}
.metric:before{content:"";position:absolute;left:0;top:0;bottom:0;width:3px;border-radius:14px 0 0 14px;background:linear-gradient(180deg,var(--cyan),var(--blue))}
.metric.m402:before{background:linear-gradient(180deg,var(--amber),#f59e0b)}
.metric.m403:before{background:linear-gradient(180deg,var(--red),#e11d48)}
.metric.m429:before{background:linear-gradient(180deg,var(--violet),#7c3aed)}
.metric .l{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.06em;text-transform:uppercase}
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
table{width:100%;border-collapse:separate;border-spacing:0;min-width:1040px}
th{position:sticky;top:0;z-index:1;background:rgba(15,23,42,.96);color:#c7d4ea;font-size:11px;font-weight:800;letter-spacing:.07em;text-transform:uppercase;padding:11px 12px;border-bottom:1px solid var(--line);text-align:left}
td{padding:12px;border-bottom:1px solid rgba(148,163,184,.08);color:#dbe4f3;vertical-align:middle}
tr:hover td{background:rgba(56,189,248,.05)}
td code{font-family:var(--mono);font-size:12px;color:#fff;background:rgba(2,6,23,.75);border:1px solid rgba(148,163,184,.22);border-radius:8px;padding:4px 7px;display:inline-block;max-width:340px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.badge{display:inline-flex;align-items:center;justify-content:center;min-width:48px;height:26px;border-radius:999px;font-weight:850;font-size:12px;border:1px solid transparent}
.b401{color:#93c5fd;background:rgba(59,130,246,.14);border-color:rgba(59,130,246,.28)}
.b402{color:#fcd34d;background:rgba(245,158,11,.14);border-color:rgba(245,158,11,.28)}
.b403{color:#fda4af;background:rgba(244,63,94,.14);border-color:rgba(244,63,94,.28)}
.b429{color:#ddd6fe;background:rgba(139,92,246,.16);border-color:rgba(167,139,250,.3)}
.pill{display:inline-flex;height:24px;align-items:center;padding:0 9px;border-radius:999px;background:rgba(148,163,184,.1);border:1px solid rgba(148,163,184,.16);font-size:12px;font-weight:750}
.remain{font-family:var(--mono);font-weight:800;color:#fff;font-size:12px}
.row-action{height:30px;padding:0 10px;border-radius:9px;font-size:12px;background:#1e293b;border-color:#475569}
.row-action:hover{background:#2563eb;border-color:#1d4ed8}
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
.sec h4{margin:0 0 10px;font-size:12px;letter-spacing:.08em;text-transform:uppercase;color:#cbd5e1}
.fg{display:grid;gap:8px;margin-bottom:10px}
.fg label{font-size:12px;color:var(--muted);font-weight:700}
.choice{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.choice.col1{grid-template-columns:1fr}
.choice button{height:auto;min-height:54px;padding:10px;text-align:left;border-radius:12px;background:rgba(7,12,22,.7)}
.choice button.active{border-color:rgba(52,211,153,.55);box-shadow:0 0 0 1px rgba(52,211,153,.25) inset;background:rgba(6,78,59,.25)}
.choice b{display:block;font-size:13px;margin-bottom:4px}
.choice span{display:block;color:var(--muted);font-size:11px;font-weight:600;line-height:1.35}
.df{display:flex;justify-content:flex-end;gap:8px;padding:12px 16px;border-top:1px solid var(--line);background:rgba(2,6,23,.4)}
.hist{display:flex;flex-wrap:wrap;gap:8px;padding:12px 14px}
.hist button{height:auto;min-width:150px;padding:10px;text-align:left}
.hist b{display:block}.hist small{display:block;color:#93a4c3;margin-top:2px}
@media(max-width:1100px){.cfg-summary{grid-template-columns:repeat(3,minmax(0,1fr))}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:700px){.cfg-summary,.metrics{grid-template-columns:1fr 1fr}h1{font-size:22px}}
</style>
</head>
<body>
<div class="shell">
  <div class="top">
    <div>
      <div class="kicker"><i></i>OPS CONSOLE</div>
      <h1>xAI Autoban</h1>
      <p class="sub">Credential isolation console · v` + pluginVersion + ` · xAI only</p>
    </div>
    <div style="display:flex;gap:8px;align-items:center">
      <div class="live" id="syncState">READY</div>
      <button class="bp" id="openConfigBtn" type="button">Edit Config</button>
    </div>
  </div>

  <div id="authBanner" class="banner warn">Checking management key...</div>

  <section class="panel">
    <div class="phd"><h2>Current Probe Config</h2><div class="hint">Top-right Edit Config changes post-probe actions</div></div>
    <div class="cfg-summary">
      <div class="chip"><div class="l">Schedule</div><div class="v" id="sumProbeEnabled">-</div></div>
      <div class="chip"><div class="l">Interval</div><div class="v" id="sumInterval">-</div></div>
      <div class="chip"><div class="l">Auto Exec</div><div class="v" id="sumAutoExec">-</div></div>
      <div class="chip"><div class="l">Problem Policy</div><div class="v" id="sumProbeAction">-</div></div>
      <div class="chip"><div class="l">Success Policy</div><div class="v" id="sumOnSuccess">-</div></div>
      <div class="chip"><div class="l">Probe Mode</div><div class="v" id="sumMode">-</div></div>
    </div>
    <div class="metrics">
      <div class="metric"><div class="l">Active Bans</div><div class="n" id="total">-</div></div>
      <div class="metric m402"><div class="l">402</div><div class="n" id="count402">-</div></div>
      <div class="metric m403"><div class="l">403</div><div class="n" id="count403">-</div></div>
      <div class="metric m429"><div class="l">429</div><div class="n" id="count429">-</div></div>
    </div>
  </section>

  <section class="panel">
    <div class="phd"><h2>Probe History</h2><div class="hint">In-memory run log (cleared on restart)</div></div>
    <div class="hist" id="probeHistory">No runs yet</div>
  </section>

  <section class="panel">
    <div class="phd"><h2>Control Plane</h2><div class="hint">Writes use Management API</div></div>
    <div class="row">
      <input id="mgmtKeyInput" type="password" placeholder="Paste CPA management key (leave empty to reuse saved key)" autocomplete="off">
      <button class="bp" id="saveKeyBtn" type="button">Save Key</button>
      <button class="bg" id="clearKeyBtn" type="button">Clear</button>
    </div>
    <div class="row">
      <input id="search" type="search" placeholder="Filter Auth ID / reason / action" autocomplete="off">
      <button class="bp" type="button" onclick="loadData()">Refresh</button>
      <button id="btnProbe" class="bs" type="button" onclick="runProbe()" disabled>Probe Now</button>
      <label class="chk"><input id="autoRefresh" type="checkbox" checked> 30s auto refresh</label>
    </div>
    <div class="row">
      <button id="unbanSelected" class="bs" type="button" onclick="unbanSelected()" disabled>Unban Selected (0)</button>
      <button id="unbanAll" class="bd" type="button" onclick="unbanAll()" disabled>Unban All</button>
      <button id="unban401" class="bs" type="button" onclick="unbanStatus(401)" disabled>Clear 401</button>
      <button id="unban402" class="bs" type="button" onclick="unbanStatus(402)" disabled>Clear 402</button>
      <button id="unban403" class="bs" type="button" onclick="unbanStatus(403)" disabled>Clear 403</button>
      <button id="unban429" class="bs" type="button" onclick="unbanStatus(429)" disabled>Clear 429</button>
    </div>
    <div class="row"><div id="message" class="msg">Standby</div></div>
  </section>

  <section class="panel">
    <div class="phd"><h2>Ban Ledger</h2><div class="hint" id="resultCount">0 records</div></div>
    <div class="table-wrap">
      <table>
        <thead><tr>
          <th style="width:40px"><input id="selectPage" type="checkbox"></th>
          <th>Auth ID</th><th>Status</th><th>Action</th><th>Reason</th><th>Banned At</th><th>Reset At</th><th>TTL</th><th>Ops</th>
        </tr></thead>
        <tbody id="rows"></tbody>
      </table>
      <div id="empty" class="empty" hidden>No active bans</div>
    </div>
  </section>
  <p class="foot">
    Aligned with CPA-Manager-Plus Codex inspection: report-only vs auto-execute + problem-account policy.
    <b>ban</b> = plugin memory isolation (scheduler skip), does NOT flip CPA enable switch.
    Use <b>disable</b> if credential list should show disabled. Runtime settings apply immediately; restart reloads config.yaml.
  </p>
</div>

<div class="drawer-mask" id="drawerMask"></div>
<aside class="drawer" id="drawer" aria-hidden="true">
  <div class="dh">
    <div>
      <h3>Server Probe Config</h3>
      <p>Configure schedule, auto-execute mode, and problem/success policies. Save applies to current plugin process.</p>
    </div>
    <button class="bg" id="closeConfigBtn" type="button">X</button>
  </div>
  <div class="db">
    <div class="sec">
      <h4>Schedule</h4>
      <label class="chk" style="margin-bottom:10px"><input id="f_probe_enabled" type="checkbox"> Enable scheduled probe</label>
      <div class="fg"><label>Interval (seconds)</label><input id="f_probe_interval_seconds" type="number" min="30" step="1"></div>
      <div class="fg"><label>Timeout (seconds)</label><input id="f_probe_timeout_seconds" type="number" min="5" step="1"></div>
      <div class="fg"><label>Concurrency</label><input id="f_probe_concurrency" type="number" min="1" step="1"></div>
      <div class="fg"><label>QPS</label><input id="f_probe_qps" type="number" min="0.1" step="0.1"></div>
      <div class="fg"><label>Probe mode</label>
        <select id="f_probe_mode"><option value="models">models (light)</option><option value="responses_mini">responses_mini (accurate)</option></select>
      </div>
    </div>
    <div class="sec">
      <h4>Auto Execute (Codex style)</h4>
      <div class="choice" id="autoExecChoices" style="margin-bottom:10px">
        <button type="button" data-v="0"><b>Report only</b><span>Probe records results; failures max ban for visibility, no disable/delete</span></button>
        <button type="button" data-v="1"><b>Auto execute</b><span>Apply problem/success policies below</span></button>
      </div>
      <div class="fg"><label>Success policy probe_on_success</label>
        <div class="choice" id="successChoices">
          <button type="button" data-v="none"><b>None</b><span>Log only, no ban/disabled change</span></button>
          <button type="button" data-v="unban"><b>Unban</b><span>Clear memory ban (default)</span></button>
          <button type="button" data-v="reenable"><b>Re-enable</b><span>disabled=false, keep ban map</span></button>
          <button type="button" data-v="unban_and_reenable"><b>Unban + enable</b><span>Restore schedule and enable flag</span></button>
        </div>
      </div>
      <div class="fg"><label>Problem policy probe_action</label>
        <div class="choice" id="failChoices">
          <button type="button" data-v="ban"><b>Ban only</b><span>Memory isolation, safest</span></button>
          <button type="button" data-v="disable"><b>Disable account</b><span>Write disabled=true</span></button>
          <button type="button" data-v="delete"><b>Delete (fallback)</b><span>No formal delete API: fallback disable</span></button>
        </div>
      </div>
      <div class="fg"><label>delete fallback</label>
        <select id="f_delete_fallback"><option value="disable">disable</option><option value="ban">ban</option></select>
      </div>
    </div>
    <div class="sec">
      <h4>Default action by status</h4>
      <div class="fg"><label>401</label><select id="f_action_on_401"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>402</label><select id="f_action_on_402"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>403</label><select id="f_action_on_403"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>429 (prefer ban)</label><select id="f_action_on_429"><option>ban</option><option>disable</option><option>delete</option></select></div>
      <div class="fg"><label>Action cooldown (seconds)</label><input id="f_action_cooldown_seconds" type="number" min="0" step="1"></div>
    </div>
  </div>
  <div class="df">
    <button class="bg" id="discardConfigBtn" type="button">Discard</button>
    <button class="bp" id="saveConfigBtn" type="button">Save & Apply</button>
  </div>
</aside>

<script>
const resourceBase='/v0/resource/plugins/xai-autoban';
const mgmtBase='/v0/management/plugins/xai-autoban';
const KEY_STORE='xai_autoban_management_key';
const state={bans:[],query:'',selected:new Set(),timer:null,mgmtKey:'',settings:{},success:'unban',fail:'ban',autoExecute:true,history:[]};
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
  ['btnProbe','unbanSelected','unbanAll','unban401','unban402','unban403','unban429','saveConfigBtn'].forEach(id=>{const el=$(id); if(el) el.disabled=!ok;});
  if(ok) $('unbanSelected').disabled=state.selected.size===0;
}
function setAuthUI(){
  state.mgmtKey=readManagementKey();
  const ok=!!state.mgmtKey;
  const b=$('authBanner'); b.className='banner'+(ok?'':' warn');
  b.textContent=ok?'Authorized: unban / probe / edit config enabled.':'Read-only: save management key before write actions.';
  if(ok&&!$('mgmtKeyInput').value) $('mgmtKeyInput').placeholder='Key saved (leave empty to keep using it)';
  setActionEnabled(ok); return ok;
}
async function apiResource(path){
  const r=await fetch(resourceBase+path,{cache:'no-store'}); const t=await r.text();
  let d; try{d=JSON.parse(t)}catch(_){throw new Error(t||('HTTP '+r.status))}
  if(!r.ok) throw new Error(d.error||('HTTP '+r.status)); return d;
}
async function apiMgmt(method,path,body){
  if(!state.mgmtKey) throw new Error('Missing management key');
  const r=await fetch(mgmtBase+path,{method,cache:'no-store',headers:{
    'Authorization':'Bearer '+state.mgmtKey,'Content-Type':'application/json',
    'X-Management-Key':state.mgmtKey,'X-Api-Key':state.mgmtKey
  },body:body?JSON.stringify(body):undefined});
  const t=await r.text(); let d; try{d=JSON.parse(t)}catch(_){throw new Error(t||('HTTP '+r.status))}
  if(!r.ok) throw new Error(d.error||d.message||('HTTP '+r.status)); return d;
}
function setMessage(text,err=false){$('message').textContent=text;$('message').className='msg'+(err?' err':'')}
function counts(){const o={401:0,402:0,403:0,429:0}; for(const b of state.bans) if(o[b.status_code]!==undefined) o[b.status_code]++; return o}
function filtered(){const q=state.query.toLowerCase(); return state.bans.filter(b=>!q||[b.auth_id,b.reason,b.action].some(x=>String(x||'').toLowerCase().includes(q)))}
function formatDate(v){const d=new Date(v); return Number.isNaN(d.getTime())?v:d.toLocaleString(undefined,{hour12:false})}
function formatRemaining(s){s=Math.max(0,Number(s||0)); const d=Math.floor(s/86400),h=Math.floor(s%86400/3600),m=Math.floor(s%3600/60); if(d)return d+'d '+h+'h'; if(h)return h+'h '+m+'m'; return m+'m'}
function reasonLabel(r){return ({payment_required:'payment required',forbidden:'forbidden',unauthorized:'unauthorized',rate_limited:'rate limited',rate_limited_fallback:'rate limited (fallback)',probe_failed:'probe failed',manual:'manual'})[r]||r||'-'}
function labelAction(a){return ({ban:'ban only',disable:'disable',delete:'delete/fallback',none:'none',unban:'unban',reenable:'re-enable',unban_and_reenable:'unban+enable'})[a]||a||'-'}

function renderSettingsSummary(s){
  state.settings=s||{};
  $('sumProbeEnabled').textContent=s.probe_enabled?'Enabled':'Off';
  $('sumProbeEnabled').className='v '+(s.probe_enabled?'on':'off');
  $('sumInterval').textContent='every '+(s.probe_interval_seconds||'-')+'s';
  const auto=s.auto_execute!==false;
  $('sumAutoExec').textContent=auto?'Auto execute':'Report only';
  $('sumAutoExec').className='v '+(auto?'on':'off');
  $('sumProbeAction').textContent=labelAction(s.probe_action);
  $('sumOnSuccess').textContent=labelAction(s.probe_on_success);
  $('sumMode').textContent=s.probe_mode||'-';
}
function renderHistory(list){
  state.history=list||[];
  const el=$('probeHistory'); if(!el) return;
  if(!state.history.length){ el.textContent='No runs yet'; return; }
  el.innerHTML=state.history.slice(0,12).map(run=>{
    const r=run.result||{};
    const mode=r.report_only?'report-only':'auto-exec';
    const st=run.error?'failed':'done';
    return '<button type="button" class="bs" title="#'+run.id+'">'+
      '<b>#'+run.id+' · '+st+'</b>'+
      '<small>'+esc(r.finished_at||r.started_at||'')+' · '+esc(r.trigger||'')+'</small>'+
      '<small style="color:#cbd5e1">'+mode+' · chk '+(r.checked||0)+' ok '+(r.ok||0)+' fail '+(r.failed||0)+'</small></button>';
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
  if(!state.mgmtKey){ setMessage('Save management key before editing config',true); return; }
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
    setMessage('Saving config...');
    const res=await apiMgmt('PUT','/settings',collectDraft());
    renderSettingsSummary(res.settings||{});
    setMessage('Config applied'+(res.note?(' · '+res.note):''));
    closeDrawer();
    await loadData(true);
  }catch(e){ setMessage(e.message,true); }
}
async function loadData(silent=false){
  try{
    if(!silent){ $('syncState').textContent='SYNC'; setMessage('Loading...'); }
    const data=await apiResource('/data');
    state.bans=data.bans||[];
    if(data.settings) renderSettingsSummary(data.settings);
    if(data.probe&&data.probe.history) renderHistory(data.probe.history);
    for(const id of [...state.selected]) if(!state.bans.some(x=>x.auth_id===id)) state.selected.delete(id);
    const c=counts();
    $('total').textContent=String(data.count||0);
    $('count402').textContent=String(c[402]||0);
    $('count403').textContent=String(c[403]||0);
    $('count429').textContent=String(c[429]||0);
    $('syncState').textContent='ONLINE';
    setMessage('Updated · '+new Date().toLocaleTimeString(undefined,{hour12:false}));
    render();
  }catch(e){ $('syncState').textContent='DEGRADED'; setMessage(e.message,true); }
}
function render(){
  const list=filtered(); $('resultCount').textContent=list.length+' records';
  $('rows').innerHTML=list.map(ban=>'<tr>'+
    '<td><input type="checkbox" data-id="'+esc(ban.auth_id)+'" '+(state.selected.has(ban.auth_id)?'checked':'')+'></td>'+
    '<td><code title="'+esc(ban.auth_id)+'">'+esc(ban.auth_id)+'</code></td>'+
    '<td><span class="badge b'+ban.status_code+'">'+ban.status_code+'</span></td>'+
    '<td><span class="pill">'+esc(ban.action||'ban')+'</span></td>'+
    '<td>'+esc(reasonLabel(ban.reason))+'</td>'+
    '<td>'+esc(formatDate(ban.banned_at))+'</td>'+
    '<td>'+esc(formatDate(ban.reset_at))+'</td>'+
    '<td class="remain">'+esc(formatRemaining(ban.remaining_seconds))+'</td>'+
    '<td><button class="row-action" data-unban="'+esc(ban.auth_id)+'" '+(state.mgmtKey?'':'disabled')+'>Unban</button></td></tr>').join('');
  $('empty').hidden=list.length>0;
  document.querySelectorAll('#rows input[type=checkbox]').forEach(input=>input.addEventListener('change',()=>{
    input.checked?state.selected.add(input.dataset.id):state.selected.delete(input.dataset.id);
    setActionEnabled(!!state.mgmtKey); $('unbanSelected').textContent='Unban Selected ('+state.selected.size+')';
  }));
  document.querySelectorAll('#rows [data-unban]').forEach(btn=>btn.addEventListener('click',()=>unbanOne(btn.dataset.unban)));
  $('unbanSelected').textContent='Unban Selected ('+state.selected.size+')'; setActionEnabled(!!state.mgmtKey);
}
async function unbanOne(id){ if(!confirm('Unban?\n'+id)) return; try{ setMessage('Unbanning...'); await apiMgmt('POST','/unban',{auth_id:id}); state.selected.delete(id); setMessage('Unbanned'); await loadData(true);}catch(e){setMessage(e.message,true)} }
async function unbanSelected(){ const ids=[...state.selected]; if(!ids.length||!confirm('Unban selected '+ids.length+'?')) return; try{ for(const id of ids) await apiMgmt('POST','/unban',{auth_id:id}); state.selected.clear(); setMessage('Selected unbanned'); await loadData(true);}catch(e){setMessage(e.message,true)} }
async function unbanAll(){ if(!confirm('Unban all?')) return; try{ await apiMgmt('POST','/unban-all',{}); state.selected.clear(); setMessage('All unbanned'); await loadData(true);}catch(e){setMessage(e.message,true)} }
async function unbanStatus(status){ const ids=state.bans.filter(x=>x.status_code===status).map(x=>x.auth_id); if(!ids.length){setMessage('No '+status);return;} if(!confirm('Clear '+ids.length+' x '+status+'?')) return; try{ for(const id of ids) await apiMgmt('POST','/unban',{auth_id:id}); setMessage(status+' cleared'); await loadData(true);}catch(e){setMessage(e.message,true)} }
async function runProbe(){ if(!confirm('Probe all xAI credentials now?')) return; try{ setMessage('Probing...'); const res=await apiMgmt('POST','/probe',{force:false}); setMessage('Probe done ok='+(res.result&&res.result.ok)+' fail='+(res.result&&res.result.failed)+(res.result&&res.result.report_only?' (report-only)':'')); await loadData(true);}catch(e){setMessage(e.message,true)} }

$('saveKeyBtn').onclick=()=>{const v=$('mgmtKeyInput').value.trim(); if(!v){setMessage('Paste management key first',true);return;} localStorage.setItem(KEY_STORE,v); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('Key saved');};
$('clearKeyBtn').onclick=()=>{localStorage.removeItem(KEY_STORE); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('Key cleared');};
$('search').oninput=e=>{state.query=e.target.value.trim(); render();};
$('selectPage').onchange=e=>{for(const ban of filtered()) e.target.checked?state.selected.add(ban.auth_id):state.selected.delete(ban.auth_id); render();};
$('autoRefresh').onchange=()=>{if(state.timer) clearInterval(state.timer); state.timer=$('autoRefresh').checked?setInterval(()=>loadData(true),30000):null;};
$('openConfigBtn').onclick=openDrawer; $('closeConfigBtn').onclick=closeDrawer; $('drawerMask').onclick=closeDrawer;
$('discardConfigBtn').onclick=()=>{fillDrawer(state.settings||{}); setMessage('Reverted to effective config');};
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