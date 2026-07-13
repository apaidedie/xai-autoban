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
    :root{
      color-scheme:dark;
      --bg0:#070b14;
      --bg1:#0c1220;
      --panel:rgba(16,24,40,.92);
      --panel-solid:#101a2c;
      --panel-2:#152038;
      --line:rgba(148,163,184,.16);
      --line-2:rgba(148,163,184,.28);
      --text:#f8fafc;
      --text-2:#dbe4f3;
      --muted:#93a4c3;
      --cyan:#22d3ee;
      --blue:#60a5fa;
      --blue-strong:#3b82f6;
      --violet:#a78bfa;
      --green:#34d399;
      --amber:#fbbf24;
      --red:#fb7185;
      --shadow:0 18px 50px rgba(0,0,0,.45);
      --radius:16px;
      --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
      --sans:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;
    }
    *{box-sizing:border-box}
    body{
      margin:0;min-height:100vh;color:var(--text);font-family:var(--sans);
      background:
        radial-gradient(1200px 600px at 10% -10%, rgba(34,211,238,.12), transparent 55%),
        radial-gradient(900px 500px at 90% 0%, rgba(96,165,250,.12), transparent 50%),
        radial-gradient(700px 400px at 70% 100%, rgba(167,139,250,.08), transparent 45%),
        linear-gradient(180deg,var(--bg0),var(--bg1) 40%, #0a101c);
      font-size:14px;line-height:1.45;
    }
    .shell{max-width:1520px;margin:0 auto;padding:22px 24px 40px}
    .topbar{
      display:flex;justify-content:space-between;align-items:flex-start;gap:16px;
      margin-bottom:18px;
    }
    .brand-kicker{
      display:inline-flex;align-items:center;gap:8px;
      color:var(--cyan);font-size:11px;font-weight:800;letter-spacing:.14em;text-transform:uppercase;
    }
    .brand-kicker .dot{
      width:7px;height:7px;border-radius:50%;background:var(--cyan);
      box-shadow:0 0 0 4px rgba(34,211,238,.15);animation:pulse 1.8s ease-in-out infinite;
    }
    @keyframes pulse{0%,100%{opacity:1}50%{opacity:.45}}
    h1{
      margin:8px 0 0;font-size:28px;font-weight:800;letter-spacing:-.03em;
      background:linear-gradient(90deg,#fff 0%,#c7d7ff 55%,#7dd3fc 100%);
      -webkit-background-clip:text;background-clip:text;color:transparent;
    }
    .subtitle{margin:8px 0 0;color:var(--muted);font-size:13px}
    .live{
      display:flex;align-items:center;gap:10px;padding:10px 14px;border-radius:999px;
      border:1px solid var(--line);background:rgba(15,23,42,.7);backdrop-filter:blur(10px);
      color:var(--text-2);font-size:12px;font-weight:700;
    }
    .live strong{color:var(--green)}
    .banner{
      display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;
      padding:12px 16px;border-radius:14px;margin-bottom:14px;
      border:1px solid rgba(52,211,153,.28);background:linear-gradient(90deg,rgba(6,78,59,.45),rgba(15,23,42,.7));
      color:#bbf7d0;font-weight:700;
    }
    .banner.warn{
      border-color:rgba(251,191,36,.35);
      background:linear-gradient(90deg,rgba(120,53,15,.45),rgba(15,23,42,.7));
      color:#fde68a;
    }
    .grid-metrics{
      display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:14px;
    }
    .metric{
      position:relative;overflow:hidden;
      background:linear-gradient(180deg,rgba(24,36,58,.95),rgba(12,20,36,.95));
      border:1px solid var(--line);border-radius:var(--radius);padding:16px 16px 14px;
      box-shadow:var(--shadow);
    }
    .metric::before{
      content:"";position:absolute;inset:0 auto 0 0;width:3px;
      background:linear-gradient(180deg,var(--cyan),var(--blue));
    }
    .metric.m402::before{background:linear-gradient(180deg,var(--amber),#f59e0b)}
    .metric.m403::before{background:linear-gradient(180deg,var(--red),#e11d48)}
    .metric.m429::before{background:linear-gradient(180deg,var(--violet),#7c3aed)}
    .metric-label{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.08em;text-transform:uppercase}
    .metric-value{margin-top:10px;font-size:34px;font-weight:850;letter-spacing:-.04em;font-variant-numeric:tabular-nums}
    .metric-foot{margin-top:6px;color:var(--muted);font-size:12px}
    .panel{
      background:linear-gradient(180deg,rgba(18,28,46,.95),rgba(12,20,34,.96));
      border:1px solid var(--line);border-radius:18px;box-shadow:var(--shadow);overflow:hidden;
      margin-bottom:14px;
    }
    .panel-hd{
      display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;
      padding:14px 16px;border-bottom:1px solid var(--line);
      background:linear-gradient(90deg,rgba(34,211,238,.06),transparent 40%);
    }
    .panel-hd h2{margin:0;font-size:13px;font-weight:800;letter-spacing:.08em;text-transform:uppercase;color:var(--text-2)}
    .panel-hd .hint{color:var(--muted);font-size:12px}
    .rows{display:flex;flex-direction:column}
    .row{
      display:flex;align-items:center;gap:10px;flex-wrap:wrap;
      padding:12px 16px;border-top:1px solid rgba(148,163,184,.08);
    }
    .row:first-child{border-top:0}
    input[type=search], input[type=password], input[type=text]{
      height:40px;min-width:220px;flex:1;
      border:1px solid var(--line-2);border-radius:12px;
      background:rgba(7,12,22,.85);color:var(--text);padding:0 14px;font:inherit;
      outline:none;transition:border-color .15s, box-shadow .15s;
    }
    input:focus{border-color:rgba(96,165,250,.7);box-shadow:0 0 0 3px rgba(59,130,246,.18)}
    input::placeholder{color:#7d8eab}
    label.chk{display:flex;align-items:center;gap:8px;color:var(--text-2);font-weight:650;white-space:nowrap}
    button{
      height:40px;border:1px solid var(--line-2);border-radius:12px;
      background:rgba(30,41,59,.9);color:var(--text);padding:0 14px;
      font:inherit;font-weight:750;cursor:pointer;transition:transform .08s, background .15s, border-color .15s;
    }
    button:hover{background:rgba(51,65,85,.95);border-color:rgba(148,163,184,.4)}
    button:active{transform:translateY(1px)}
    button:disabled{opacity:.35;cursor:not-allowed;transform:none}
    .btn-primary{
      background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8;color:#fff;
      box-shadow:0 8px 20px rgba(37,99,235,.28);
    }
    .btn-primary:hover{background:linear-gradient(180deg,#60a5fa,#3b82f6)}
    .btn-danger{
      background:linear-gradient(180deg,rgba(244,63,94,.2),rgba(127,29,29,.35));
      border-color:rgba(251,113,133,.4);color:#fecdd3;
    }
    .btn-ghost{background:transparent}
    .btn-soft{
      background:rgba(15,23,42,.7);border-color:var(--line);
    }
    .seg{display:flex;gap:8px;flex-wrap:wrap}
    .msg{
      min-height:20px;color:var(--muted);font-size:12.5px;font-weight:700;
      display:flex;align-items:center;gap:8px;
    }
    .msg.error{color:#fda4af}
    .msg .spinner{
      width:12px;height:12px;border-radius:50%;
      border:2px solid rgba(148,163,184,.25);border-top-color:var(--cyan);
      animation:spin .7s linear infinite;
    }
    @keyframes spin{to{transform:rotate(360deg)}}
    .table-wrap{overflow:auto;max-height:62vh}
    table{width:100%;border-collapse:separate;border-spacing:0;min-width:1080px}
    thead th{
      position:sticky;top:0;z-index:2;
      background:rgba(15,23,42,.96);backdrop-filter:blur(8px);
      color:#c7d4ea;font-size:11px;font-weight:800;letter-spacing:.08em;text-transform:uppercase;
      padding:12px 14px;border-bottom:1px solid var(--line);text-align:left;
    }
    tbody td{
      padding:13px 14px;border-bottom:1px solid rgba(148,163,184,.08);
      color:var(--text-2);vertical-align:middle;
    }
    tbody tr{transition:background .12s}
    tbody tr:hover{background:rgba(56,189,248,.05)}
    tbody tr:nth-child(even){background:rgba(255,255,255,.015)}
    tbody tr:nth-child(even):hover{background:rgba(56,189,248,.07)}
    td code{
      font-family:var(--mono);font-size:12px;color:#f8fafc;
      background:rgba(2,6,23,.75);border:1px solid rgba(148,163,184,.22);
      border-radius:8px;padding:5px 8px;display:inline-block;max-width:360px;
      overflow:hidden;text-overflow:ellipsis;white-space:nowrap;vertical-align:middle;
    }
    .badge{
      display:inline-flex;align-items:center;justify-content:center;gap:6px;
      min-width:52px;height:28px;border-radius:999px;font-weight:850;font-size:12px;
      border:1px solid transparent;
    }
    .b401{color:#93c5fd;background:rgba(59,130,246,.14);border-color:rgba(59,130,246,.28)}
    .b402{color:#fcd34d;background:rgba(245,158,11,.14);border-color:rgba(245,158,11,.28)}
    .b403{color:#fda4af;background:rgba(244,63,94,.14);border-color:rgba(244,63,94,.28)}
    .b429{color:#ddd6fe;background:rgba(139,92,246,.16);border-color:rgba(167,139,250,.3)}
    .pill{
      display:inline-flex;align-items:center;height:26px;padding:0 10px;border-radius:999px;
      background:rgba(148,163,184,.1);border:1px solid rgba(148,163,184,.16);
      color:var(--text);font-size:12px;font-weight:750;
    }
    .time{font-variant-numeric:tabular-nums;color:#cbd5e1;font-size:12.5px}
    .remain{
      font-variant-numeric:tabular-nums;font-weight:800;color:#fff;
      font-family:var(--mono);font-size:12px;
    }
    .row-action{
      height:32px;padding:0 12px;border-radius:10px;font-size:12px;
      background:linear-gradient(180deg,#334155,#1e293b);border-color:#475569;color:#fff;
    }
    .row-action:hover{background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8}
    .empty{padding:56px 20px;text-align:center;color:var(--muted);font-weight:700}
    .foot{color:var(--muted);font-size:12px;line-height:1.7;padding:0 4px}
    .foot code{font-family:var(--mono);color:#cbd5e1}
    @media(max-width:980px){
      .grid-metrics{grid-template-columns:repeat(2,minmax(0,1fr))}
      .shell{padding:16px}
      h1{font-size:24px}
      td code{max-width:220px}
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="topbar">
      <div>
        <div class="brand-kicker"><span class="dot"></span>OPS CONSOLE</div>
        <h1>xAI Autoban</h1>
        <p class="subtitle">凭据隔离运维台 · v` + pluginVersion + ` · 仅影响 xAI provider</p>
      </div>
      <div class="live"><span>链路</span><strong id="syncState">准备中</strong></div>
    </div>

    <div id="authBanner" class="banner warn">正在检测管理密钥…</div>

    <section class="grid-metrics" aria-label="指标">
      <article class="metric">
        <div class="metric-label">Active Bans</div>
        <div class="metric-value" id="total">-</div>
        <div class="metric-foot">当前隔离凭据</div>
      </article>
      <article class="metric m402">
        <div class="metric-label">402 Payment</div>
        <div class="metric-value" id="count402">-</div>
        <div class="metric-foot">额度 / 订阅问题</div>
      </article>
      <article class="metric m403">
        <div class="metric-label">403 Forbidden</div>
        <div class="metric-value" id="count403">-</div>
        <div class="metric-foot">拒绝访问</div>
      </article>
      <article class="metric m429">
        <div class="metric-label">429 Rate Limit</div>
        <div class="metric-value" id="count429">-</div>
        <div class="metric-foot">限流冷却</div>
      </article>
    </section>

    <section class="panel">
      <div class="panel-hd">
        <h2>Control Plane</h2>
        <div class="hint">密钥仅存本浏览器 · 写操作走 /v0/management</div>
      </div>
      <div class="rows">
        <div class="row">
          <input id="mgmtKeyInput" type="password" placeholder="粘贴 CPA 管理密钥（可留空继续使用已保存密钥）" autocomplete="off">
          <button class="btn-primary" id="saveKeyBtn" type="button">保存密钥</button>
          <button class="btn-ghost" id="clearKeyBtn" type="button">清除</button>
        </div>
        <div class="row">
          <input id="search" type="search" placeholder="过滤 Auth ID / 原因 / 动作" autocomplete="off">
          <button class="btn-primary" type="button" onclick="loadData()">刷新</button>
          <button id="btnProbe" class="btn-soft" type="button" onclick="runProbe()" disabled>立即巡检</button>
          <label class="chk"><input id="autoRefresh" type="checkbox" checked> 30s 自动刷新</label>
        </div>
        <div class="row seg">
          <button id="unbanSelected" class="btn-soft" type="button" onclick="unbanSelected()" disabled>解禁所选 (0)</button>
          <button id="unbanAll" class="btn-danger" type="button" onclick="unbanAll()" disabled>全部解禁</button>
          <button id="unban402" class="btn-soft" type="button" onclick="unbanStatus(402)" disabled>清 402</button>
          <button id="unban403" class="btn-soft" type="button" onclick="unbanStatus(403)" disabled>清 403</button>
          <button id="unban429" class="btn-soft" type="button" onclick="unbanStatus(429)" disabled>清 429</button>
          <button id="unban401" class="btn-soft" type="button" onclick="unbanStatus(401)" disabled>清 401</button>
        </div>
        <div class="row"><div id="message" class="msg">系统待命</div></div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-hd">
        <h2>Ban Ledger</h2>
        <div class="hint" id="resultCount">0 records</div>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th style="width:42px"><input id="selectPage" type="checkbox"></th>
              <th>Auth ID</th>
              <th>Status</th>
              <th>Action</th>
              <th>Reason</th>
              <th>Banned At</th>
              <th>Reset At</th>
              <th>TTL</th>
              <th>Ops</th>
            </tr>
          </thead>
          <tbody id="rows"></tbody>
        </table>
        <div id="empty" class="empty" hidden>No active bans — 号池当前健康</div>
      </div>
    </section>

    <p class="foot">
      运维台风格面板。敏感操作需管理密钥（自动读取登录态或手动保存）。
      公开 <code>/action</code> 已移除；解禁/巡检走 Management API。
    </p>
  </div>

  <script>
    const resourceBase='/v0/resource/plugins/xai-autoban';
    const mgmtBase='/v0/management/plugins/xai-autoban';
    const KEY_STORE='xai_autoban_management_key';
    const state={bans:[],query:'',selected:new Set(),timer:null,mgmtKey:''};
    const $=id=>document.getElementById(id);
    const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

    function readManagementKey(){
      try{
        const manual=localStorage.getItem(KEY_STORE);
        if(manual&&manual.trim()) return manual.trim();
      }catch(_){}
      const keys=['cliproxyapi_management_key','management_key','cpa_management_key','managementKey','management-password','managementPassword','apiKey','api_key','token','auth_token'];
      for(const k of keys){
        try{const v=localStorage.getItem(k); if(v&&v.trim()&&v.length<512) return v.trim();}catch(_){}
      }
      try{
        for(let i=0;i<localStorage.length;i++){
          const k=localStorage.key(i); if(!k) continue;
          const raw=localStorage.getItem(k); if(!raw||raw.length>8000) continue;
          if(/management|mgmt|cpa|cliproxy/i.test(k) && raw.trim() && !raw.trim().startsWith('{') && raw.length<512) return raw.trim();
          if(raw.trim().startsWith('{')||raw.trim().startsWith('[')){
            try{
              const obj=JSON.parse(raw); const stack=[obj];
              while(stack.length){
                const cur=stack.pop(); if(!cur||typeof cur!=='object') continue;
                for(const [kk,vv] of Object.entries(cur)){
                  if(typeof vv==='string' && vv.trim() && vv.length<512 && /management|mgmt|password|apiKey|api_key|token/i.test(kk)) return vv.trim();
                  if(vv&&typeof vv==='object') stack.push(vv);
                }
              }
            }catch(_){}
          }
        }
      }catch(_){}
      return '';
    }

    function setActionEnabled(ok){
      ['btnProbe','unbanSelected','unbanAll','unban401','unban402','unban403','unban429'].forEach(id=>{
        const el=$(id); if(el) el.disabled=!ok;
      });
      if(ok) $('unbanSelected').disabled=state.selected.size===0;
    }

    function setAuthUI(){
      state.mgmtKey=readManagementKey();
      const ok=!!state.mgmtKey;
      const banner=$('authBanner');
      banner.className='banner'+(ok?'':' warn');
      banner.textContent=ok
        ? '控制面已授权：可执行解禁 / 巡检。密钥来自本页保存或管理中心登录态。'
        : '只读模式：请在下方粘贴 CPA 管理密钥并保存，或先登录管理中心后再刷新。';
      if(ok && !$('mgmtKeyInput').value) $('mgmtKeyInput').placeholder='已保存管理密钥（输入框留空可继续使用）';
      setActionEnabled(ok);
      return ok;
    }

    async function apiResource(path){
      const response=await fetch(resourceBase+path,{cache:'no-store'});
      const text=await response.text();
      let data; try{data=JSON.parse(text)}catch(_){throw new Error(text||('HTTP '+response.status))}
      if(!response.ok) throw new Error(data.error||('HTTP '+response.status));
      return data;
    }

    async function apiMgmt(method, path, body){
      if(!state.mgmtKey) throw new Error('缺少管理密钥');
      const headers={
        'Authorization':'Bearer '+state.mgmtKey,
        'Content-Type':'application/json',
        'X-Management-Key':state.mgmtKey,
        'X-Api-Key':state.mgmtKey
      };
      const response=await fetch(mgmtBase+path,{method,cache:'no-store',headers,body:body?JSON.stringify(body):undefined});
      const text=await response.text();
      let data; try{data=JSON.parse(text)}catch(_){throw new Error(text||('HTTP '+response.status))}
      if(!response.ok) throw new Error(data.error||data.message||('HTTP '+response.status));
      return data;
    }

    function setMessage(text,error=false,busy=false){
      $('message').className='msg'+(error?' error':'');
      $('message').innerHTML=(busy?'<span class="spinner"></span>':'')+esc(text);
    }
    function counts(){
      const out={401:0,402:0,403:0,429:0};
      for(const ban of state.bans){ if(out[ban.status_code]!==undefined) out[ban.status_code]++; }
      return out;
    }
    function filtered(){
      const q=state.query.toLowerCase();
      return state.bans.filter(b=>!q||[b.auth_id,b.reason,b.action].some(x=>String(x||'').toLowerCase().includes(q)));
    }
    function formatDate(v){
      const d=new Date(v);
      return Number.isNaN(d.getTime())?v:d.toLocaleString('zh-CN',{hour12:false});
    }
    function formatRemaining(seconds){
      seconds=Math.max(0,Number(seconds||0));
      const d=Math.floor(seconds/86400),h=Math.floor(seconds%86400/3600),m=Math.floor(seconds%3600/60);
      if(d) return d+'d '+h+'h';
      if(h) return h+'h '+m+'m';
      return m+'m';
    }
    function reasonLabel(reason){
      return ({
        payment_required:'额度不足',
        forbidden:'禁止访问',
        unauthorized:'未授权',
        rate_limited:'限流',
        rate_limited_fallback:'限流(默认等待)',
        probe_failed:'巡检失败',
        manual:'手动'
      })[reason]||reason||'-';
    }

    async function loadData(silent=false){
      try{
        if(!silent){ $('syncState').textContent='同步中'; setMessage('拉取隔离账本...',false,true); }
        const data=await apiResource('/data');
        state.bans=data.bans||[];
        for(const id of [...state.selected]) if(!state.bans.some(x=>x.auth_id===id)) state.selected.delete(id);
        const c=counts();
        $('total').textContent=String(data.count||0);
        $('count402').textContent=String(c[402]||0);
        $('count403').textContent=String(c[403]||0);
        $('count429').textContent=String(c[429]||0);
        $('syncState').textContent='ONLINE';
        setMessage('账本已更新 · '+new Date().toLocaleTimeString('zh-CN',{hour12:false}));
        render();
      }catch(error){
        $('syncState').textContent='DEGRADED';
        setMessage(error.message,true);
      }
    }

    function render(){
      const list=filtered();
      $('resultCount').textContent=list.length+' records';
      $('rows').innerHTML=list.map(ban=>'<tr>'+
        '<td><input type="checkbox" data-id="'+esc(ban.auth_id)+'" '+(state.selected.has(ban.auth_id)?'checked':'')+'></td>'+
        '<td><code title="'+esc(ban.auth_id)+'">'+esc(ban.auth_id)+'</code>'+(ban.pending_delete?' <span title="pending delete">⚠</span>':'')+'</td>'+
        '<td><span class="badge b'+ban.status_code+'">'+ban.status_code+'</span></td>'+
        '<td><span class="pill">'+esc(ban.action||'ban')+'</span></td>'+
        '<td>'+esc(reasonLabel(ban.reason))+'</td>'+
        '<td class="time">'+esc(formatDate(ban.banned_at))+'</td>'+
        '<td class="time">'+esc(formatDate(ban.reset_at))+'</td>'+
        '<td class="remain">'+esc(formatRemaining(ban.remaining_seconds))+'</td>'+
        '<td><button class="row-action" data-unban="'+esc(ban.auth_id)+'" '+(state.mgmtKey?'':'disabled')+'>解禁</button></td>'+
      '</tr>').join('');
      $('empty').hidden=list.length>0;
      document.querySelectorAll('#rows input[type=checkbox]').forEach(input=>input.addEventListener('change',()=>{
        input.checked?state.selected.add(input.dataset.id):state.selected.delete(input.dataset.id);
        setActionEnabled(!!state.mgmtKey);
        $('unbanSelected').textContent='解禁所选 ('+state.selected.size+')';
      }));
      document.querySelectorAll('#rows [data-unban]').forEach(btn=>btn.addEventListener('click',()=>unbanOne(btn.dataset.unban)));
      $('unbanSelected').textContent='解禁所选 ('+state.selected.size+')';
      setActionEnabled(!!state.mgmtKey);
    }

    async function unbanOne(id){
      if(!confirm('确认解禁？\\n'+id)) return;
      try{ setMessage('解禁中...',false,true); await apiMgmt('POST','/unban',{auth_id:id}); state.selected.delete(id); setMessage('已解禁'); await loadData(true);}catch(e){setMessage(e.message,true)}
    }
    async function unbanSelected(){
      const ids=[...state.selected];
      if(!ids.length||!confirm('确认解禁所选 '+ids.length+' 条？')) return;
      try{ setMessage('批量解禁中...',false,true); for(const id of ids) await apiMgmt('POST','/unban',{auth_id:id}); state.selected.clear(); setMessage('所选已解禁'); await loadData(true);}catch(e){setMessage(e.message,true)}
    }
    async function unbanAll(){
      if(!confirm('确认解禁全部？')) return;
      try{ setMessage('全部解禁中...',false,true); await apiMgmt('POST','/unban-all',{}); state.selected.clear(); setMessage('已全部解禁'); await loadData(true);}catch(e){setMessage(e.message,true)}
    }
    async function unbanStatus(status){
      const ids=state.bans.filter(x=>x.status_code===status).map(x=>x.auth_id);
      if(!ids.length){ setMessage('没有 '+status+' 记录'); return; }
      if(!confirm('确认清理全部 '+ids.length+' 条 '+status+'？')) return;
      try{ setMessage('清理 '+status+' 中...',false,true); for(const id of ids) await apiMgmt('POST','/unban',{auth_id:id}); setMessage(status+' 已清理'); await loadData(true);}catch(e){setMessage(e.message,true)}
    }
    async function runProbe(){
      if(!confirm('立即巡检全部 xAI 凭据？')) return;
      try{
        setMessage('巡检执行中...',false,true);
        const res=await apiMgmt('POST','/probe',{force:false});
        setMessage('巡检完成 · ok='+(res.result&&res.result.ok)+' · failed='+(res.result&&res.result.failed));
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }

    $('saveKeyBtn').addEventListener('click',()=>{
      const v=$('mgmtKeyInput').value.trim();
      if(!v){ setMessage('请先粘贴管理密钥',true); return; }
      localStorage.setItem(KEY_STORE,v); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('管理密钥已保存');
    });
    $('clearKeyBtn').addEventListener('click',()=>{
      localStorage.removeItem(KEY_STORE); $('mgmtKeyInput').value=''; setAuthUI(); setMessage('已清除本页密钥');
    });
    $('search').addEventListener('input',e=>{state.query=e.target.value.trim(); render();});
    $('selectPage').addEventListener('change',e=>{
      for(const ban of filtered()) e.target.checked?state.selected.add(ban.auth_id):state.selected.delete(ban.auth_id);
      render();
    });
    $('autoRefresh').addEventListener('change',()=>{
      if(state.timer) clearInterval(state.timer);
      state.timer=$('autoRefresh').checked?setInterval(()=>loadData(true),30000):null;
    });

    setAuthUI();
    if($('autoRefresh').checked) state.timer=setInterval(()=>loadData(true),30000);
    loadData();
  </script>
</body>
</html>`
}
