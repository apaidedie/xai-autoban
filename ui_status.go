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
      --bg:#0b1220;
      --panel:#121a2b;
      --panel-2:#182235;
      --text:#eef3ff;
      --text-2:#d7e0f2;
      --muted:#a9b6cf;
      --line:#2a3750;
      --blue:#3b82f6;
      --blue-2:#2563eb;
      --red:#f87171;
      --red-bg:rgba(248,113,113,.14);
      --amber:#fbbf24;
      --amber-bg:rgba(251,191,36,.14);
      --green:#34d399;
      --green-bg:rgba(52,211,153,.14);
      --purple:#c4b5fd;
      --purple-bg:rgba(167,139,250,.16);
      --shadow:0 8px 24px rgba(0,0,0,.28);
    }
    *{box-sizing:border-box}
    body{
      margin:0;
      background:var(--bg);
      color:var(--text);
      font-family:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;
      font-size:14px;
      line-height:1.45;
    }
    header{
      background:linear-gradient(180deg,#111a2c 0%,#0d1524 100%);
      border-bottom:1px solid var(--line);
    }
    .header-inner{
      max-width:1480px;margin:auto;padding:18px 24px;
      display:flex;justify-content:space-between;align-items:center;gap:16px;
    }
    .brand h1{margin:0;font-size:22px;font-weight:750;letter-spacing:.2px}
    .brand p{margin:6px 0 0;color:var(--muted);font-size:13px}
    #syncState{color:var(--green);font-weight:650}
    main{max-width:1480px;margin:auto;padding:20px 24px 40px}
    .auth-banner{
      padding:12px 14px;border-radius:10px;margin-bottom:14px;
      border:1px solid #6b5a1f;background:#2a230f;color:#fde68a;font-weight:600;
    }
    .auth-banner.ok{border-color:#14532d;background:#0f241a;color:#86efac}
    .stats{display:grid;grid-template-columns:repeat(4,minmax(140px,1fr));gap:12px;margin-bottom:14px}
    .stat{
      background:var(--panel);border:1px solid var(--line);border-radius:12px;
      padding:16px;box-shadow:var(--shadow);
    }
    .stat-label{color:var(--muted);font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.04em}
    .stat-value{margin-top:8px;font-size:30px;font-weight:800;color:var(--text)}
    .toolbar{
      background:var(--panel);border:1px solid var(--line);border-radius:12px;
      box-shadow:var(--shadow);margin-bottom:14px;overflow:hidden;
    }
    .toolbar-row{
      display:flex;align-items:center;gap:10px;padding:12px 14px;flex-wrap:wrap;
    }
    .toolbar-row+.toolbar-row{border-top:1px solid var(--line)}
    input[type=search], input[type=password], input[type=text]{
      height:38px;min-width:220px;flex:1;
      border:1px solid #3b4b68;border-radius:9px;
      background:#0d1524;color:var(--text);padding:0 12px;font:inherit;
    }
    input::placeholder{color:#8090ad}
    label{color:var(--text-2);display:flex;align-items:center;gap:8px;white-space:nowrap}
    button{
      height:38px;border:1px solid #3b4b68;border-radius:9px;
      background:#1a2740;color:var(--text);padding:0 14px;
      font:inherit;font-weight:700;cursor:pointer;
    }
    button:hover{background:#223352}
    button:disabled{opacity:.38;cursor:not-allowed}
    .primary{background:var(--blue);border-color:var(--blue-2);color:#fff}
    .primary:hover{background:var(--blue-2)}
    .danger{background:var(--red-bg);border-color:#7f1d1d;color:#fecaca}
    .quiet{background:transparent}
    .message{min-height:20px;color:var(--muted);font-size:13px;font-weight:600}
    .message.error{color:#fca5a5}
    .table-shell{
      background:var(--panel);border:1px solid var(--line);border-radius:12px;
      box-shadow:var(--shadow);overflow:hidden;
    }
    .table-wrap{overflow:auto;max-height:66vh}
    table{border-collapse:collapse;width:100%;min-width:1040px}
    th,td{
      padding:12px 14px;text-align:left;border-bottom:1px solid #24324a;
      vertical-align:middle;color:var(--text);
    }
    th{
      position:sticky;top:0;z-index:1;
      background:#1a2438;color:#f3f7ff;font-size:12px;font-weight:800;
      letter-spacing:.03em;text-transform:uppercase;
    }
    tbody tr:hover{background:rgba(59,130,246,.08)}
    td code{
      font-family:ui-monospace,SFMono-Regular,Consolas,monospace;
      font-size:12.5px;color:#f8fafc;background:#0d1524;
      border:1px solid #334155;border-radius:6px;padding:3px 7px;
      word-break:break-all;
    }
    .badge{
      display:inline-flex;align-items:center;justify-content:center;
      min-width:48px;height:26px;border-radius:999px;font-weight:800;font-size:12px;
    }
    .b401{color:#93c5fd;background:rgba(59,130,246,.18)}
    .b402{color:var(--amber);background:var(--amber-bg)}
    .b403{color:var(--red);background:var(--red-bg)}
    .b429{color:var(--purple);background:var(--purple-bg)}
    .reason,.action,.time,.remaining{color:var(--text-2);font-weight:600}
    .remaining{font-variant-numeric:tabular-nums;color:#fff}
    .row-action{
      height:32px;padding:0 10px;font-size:12px;
      background:#1d4ed8;border-color:#1d4ed8;color:#fff;
    }
    .empty{padding:54px;text-align:center;color:var(--muted);font-weight:650}
    .footer-note{color:var(--muted);font-size:12px;margin:12px 2px 0;line-height:1.6}
    .key-row{width:100%}
    .key-row input{min-width:280px}
    @media(max-width:900px){
      .stats{grid-template-columns:repeat(2,minmax(120px,1fr))}
      .header-inner,main{padding-left:14px;padding-right:14px}
    }
  </style>
</head>
<body>
  <header>
    <div class="header-inner">
      <div class="brand">
        <h1>xAI Autoban</h1>
        <p>CPA 凭据隔离控制台 — v` + pluginVersion + `</p>
      </div>
      <div id="syncState">准备中</div>
    </div>
  </header>
  <main>
    <div id="authBanner" class="auth-banner">正在检测管理密钥…</div>

    <section class="stats" aria-label="统计">
      <div class="stat"><div class="stat-label">当前隔离</div><div class="stat-value" id="total">-</div></div>
      <div class="stat"><div class="stat-label">402</div><div class="stat-value" id="count402">-</div></div>
      <div class="stat"><div class="stat-label">403</div><div class="stat-value" id="count403">-</div></div>
      <div class="stat"><div class="stat-label">429</div><div class="stat-value" id="count429">-</div></div>
    </section>

    <section class="toolbar">
      <div class="toolbar-row key-row">
        <input id="mgmtKeyInput" type="password" placeholder="粘贴 CPA 管理密钥（仅保存在本浏览器 localStorage）" autocomplete="off">
        <button class="primary" id="saveKeyBtn" type="button">保存密钥</button>
        <button class="quiet" id="clearKeyBtn" type="button">清除</button>
      </div>
      <div class="toolbar-row">
        <input id="search" type="search" placeholder="搜索 Auth ID 或原因" autocomplete="off">
        <button class="primary" type="button" onclick="loadData()">刷新</button>
        <button id="btnProbe" type="button" onclick="runProbe()" disabled>立即巡检</button>
        <label><input id="autoRefresh" type="checkbox" checked> 30 秒自动刷新</label>
      </div>
      <div class="toolbar-row">
        <button id="unbanSelected" type="button" onclick="unbanSelected()" disabled>解禁所选</button>
        <button id="unbanAll" class="danger" type="button" onclick="unbanAll()" disabled>全部解禁</button>
        <button id="unban402" type="button" onclick="unbanStatus(402)" disabled>解禁全部 402</button>
        <button id="unban403" type="button" onclick="unbanStatus(403)" disabled>解禁全部 403</button>
        <button id="unban429" type="button" onclick="unbanStatus(429)" disabled>解禁全部 429</button>
      </div>
      <div class="toolbar-row"><div id="message" class="message">准备加载数据</div></div>
    </section>

    <section class="table-shell">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th style="width:42px"><input id="selectPage" type="checkbox"></th>
              <th>Auth ID</th>
              <th>状态</th>
              <th>动作</th>
              <th>原因</th>
              <th>隔离时间</th>
              <th>自动解禁</th>
              <th>剩余</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody id="rows"></tbody>
        </table>
        <div id="empty" class="empty" hidden>当前没有隔离凭据</div>
      </div>
    </section>
    <p class="footer-note">
      表格为高对比暗色主题。解禁 / 巡检需要管理密钥：可自动读取管理中心已登录密钥，或在上方手动粘贴保存。
      密钥只存在当前浏览器，不会回传到插件配置。
    </p>
  </main>
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

      const keys=[
        'cliproxyapi_management_key','management_key','cpa_management_key','managementKey',
        'management-password','managementPassword','apiKey','api_key','token','auth_token'
      ];
      for(const k of keys){
        try{
          const v=localStorage.getItem(k);
          if(v&&v.trim()&&v.length<512) return v.trim();
        }catch(_){}
      }
      try{
        for(let i=0;i<localStorage.length;i++){
          const k=localStorage.key(i);
          if(!k) continue;
          const raw=localStorage.getItem(k);
          if(!raw||raw.length>8000) continue;
          if(/management|mgmt|cpa|cliproxy/i.test(k) && raw.trim() && !raw.trim().startsWith('{') && raw.length<512){
            return raw.trim();
          }
          if(raw.trim().startsWith('{')||raw.trim().startsWith('[')){
            try{
              const obj=JSON.parse(raw);
              const stack=[obj];
              while(stack.length){
                const cur=stack.pop();
                if(!cur||typeof cur!=='object') continue;
                for(const [kk,vv] of Object.entries(cur)){
                  if(typeof vv==='string' && vv.trim() && vv.length<512 && /management|mgmt|password|apiKey|api_key|token/i.test(kk)){
                    return vv.trim();
                  }
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
      ['btnProbe','unbanSelected','unbanAll','unban402','unban403','unban429'].forEach(id=>{
        const el=$(id); if(el) el.disabled=!ok;
      });
      if(ok){
        $('unbanSelected').disabled=state.selected.size===0;
      }
    }

    function setAuthUI(){
      state.mgmtKey=readManagementKey();
      const ok=!!state.mgmtKey;
      const banner=$('authBanner');
      banner.className='auth-banner'+(ok?' ok':'');
      banner.textContent=ok
        ? '已就绪：可执行解禁 / 巡检（密钥来自本页保存或管理中心登录态）。'
        : '未检测到管理密钥：当前只读。请在上方粘贴 CPA 管理密钥并点「保存密钥」，或先在管理中心登录后再刷新。';
      if(ok && !$('mgmtKeyInput').value){
        $('mgmtKeyInput').placeholder='已保存管理密钥（输入框留空可继续使用）';
      }
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
      const response=await fetch(mgmtBase+path,{
        method, cache:'no-store', headers,
        body: body?JSON.stringify(body):undefined
      });
      const text=await response.text();
      let data; try{data=JSON.parse(text)}catch(_){throw new Error(text||('HTTP '+response.status))}
      if(!response.ok) throw new Error(data.error||data.message||('HTTP '+response.status));
      return data;
    }

    function setMessage(text,error=false){
      $('message').textContent=text;
      $('message').className='message'+(error?' error':'');
    }
    function counts(){
      const out={401:0,402:0,403:0,429:0};
      for(const ban of state.bans){ if(out[ban.status_code]!==undefined) out[ban.status_code]++; }
      return out;
    }
    function filtered(){
      const q=state.query.toLowerCase();
      return state.bans.filter(b=>!q||String(b.auth_id).toLowerCase().includes(q)||String(b.reason||'').toLowerCase().includes(q));
    }
    function formatDate(v){
      const d=new Date(v);
      return Number.isNaN(d.getTime())?v:d.toLocaleString('zh-CN',{hour12:false});
    }
    function formatRemaining(seconds){
      seconds=Math.max(0,Number(seconds||0));
      const d=Math.floor(seconds/86400),h=Math.floor(seconds%86400/3600),m=Math.floor(seconds%3600/60);
      if(d) return d+'天 '+h+'小时';
      if(h) return h+'小时 '+m+'分';
      return m+'分钟';
    }
    function reasonLabel(reason){
      return ({
        payment_required:'无额度/需付费',
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
        if(!silent){ $('syncState').textContent='同步中'; setMessage('正在加载状态...'); }
        const data=await apiResource('/data');
        state.bans=data.bans||[];
        for(const id of [...state.selected]) if(!state.bans.some(x=>x.auth_id===id)) state.selected.delete(id);
        const c=counts();
        $('total').textContent=String(data.count||0);
        $('count402').textContent=String(c[402]||0);
        $('count403').textContent=String(c[403]||0);
        $('count429').textContent=String(c[429]||0);
        $('syncState').textContent='已同步';
        setMessage('已更新：'+new Date().toLocaleTimeString('zh-CN',{hour12:false}));
        render();
      }catch(error){
        $('syncState').textContent='同步异常';
        setMessage(error.message,true);
      }
    }

    function render(){
      const list=filtered();
      $('rows').innerHTML=list.map(ban=>'<tr>'+
        '<td><input type="checkbox" data-id="'+esc(ban.auth_id)+'" '+(state.selected.has(ban.auth_id)?'checked':'')+'></td>'+
        '<td><code title="'+esc(ban.auth_id)+'">'+esc(ban.auth_id)+'</code>'+(ban.pending_delete?' <span title="待删除">⚠</span>':'')+'</td>'+
        '<td><span class="badge b'+ban.status_code+'">'+ban.status_code+'</span></td>'+
        '<td class="action">'+esc(ban.action||'ban')+'</td>'+
        '<td class="reason">'+esc(reasonLabel(ban.reason))+'</td>'+
        '<td class="time">'+esc(formatDate(ban.banned_at))+'</td>'+
        '<td class="time">'+esc(formatDate(ban.reset_at))+'</td>'+
        '<td class="remaining">'+esc(formatRemaining(ban.remaining_seconds))+'</td>'+
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
      if(!confirm('确认解禁凭据？\\n'+id)) return;
      try{
        setMessage('正在解禁...');
        await apiMgmt('POST','/unban',{auth_id:id});
        state.selected.delete(id);
        setMessage('已解禁 '+id);
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }
    async function unbanSelected(){
      const ids=[...state.selected];
      if(!ids.length||!confirm('确认解禁所选 '+ids.length+' 条？')) return;
      try{
        for(const id of ids){ await apiMgmt('POST','/unban',{auth_id:id}); }
        state.selected.clear();
        setMessage('已解禁所选');
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }
    async function unbanAll(){
      if(!confirm('确认解禁全部？')) return;
      try{
        await apiMgmt('POST','/unban-all',{});
        state.selected.clear();
        setMessage('已全部解禁');
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }
    async function unbanStatus(status){
      const ids=state.bans.filter(x=>x.status_code===status).map(x=>x.auth_id);
      if(!ids.length){ setMessage('没有状态 '+status+' 的隔离项'); return; }
      if(!confirm('确认解禁全部 '+ids.length+' 条 '+status+'？')) return;
      try{
        for(const id of ids){ await apiMgmt('POST','/unban',{auth_id:id}); }
        setMessage('已解禁状态 '+status);
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }
    async function runProbe(){
      if(!confirm('立即巡检全部 xAI 凭据？')) return;
      try{
        setMessage('巡检中...');
        const res=await apiMgmt('POST','/probe',{force:false});
        setMessage('巡检完成 ok='+(res.result&&res.result.ok)+' failed='+(res.result&&res.result.failed));
        await loadData(true);
      }catch(e){ setMessage(e.message,true); }
    }

    $('saveKeyBtn').addEventListener('click',()=>{
      const v=$('mgmtKeyInput').value.trim();
      if(!v){ setMessage('请先粘贴管理密钥',true); return; }
      localStorage.setItem(KEY_STORE,v);
      $('mgmtKeyInput').value='';
      setAuthUI();
      setMessage('管理密钥已保存到本浏览器');
    });
    $('clearKeyBtn').addEventListener('click',()=>{
      localStorage.removeItem(KEY_STORE);
      $('mgmtKeyInput').value='';
      setAuthUI();
      setMessage('已清除本页保存的管理密钥');
    });
    $('search').addEventListener('input',e=>{state.query=e.target.value.trim(); render();});
    $('selectPage').addEventListener('change',e=>{
      for(const ban of filtered()){ e.target.checked?state.selected.add(ban.auth_id):state.selected.delete(ban.auth_id); }
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
