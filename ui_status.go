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
    :root{color-scheme:light;--bg:#f5f7f9;--surface:#fff;--text:#17202a;--muted:#66727f;--line:#dce2e8;--red:#b42318;--red-bg:#fff1f0;--amber:#9a6700;--amber-bg:#fff8db;--blue:#175cd3;--green:#067647;--shadow:0 1px 2px rgba(16,24,40,.06)}
    *{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;font-size:14px}
    header{background:#182230;color:#fff}.header-inner{max-width:1440px;margin:auto;padding:18px 24px;display:flex;justify-content:space-between;gap:20px}.brand h1{font-size:21px;margin:0}.brand p{margin:5px 0 0;color:#b9c3cf;font-size:13px}
    main{max-width:1440px;margin:auto;padding:20px 24px 36px}.stats{display:grid;grid-template-columns:repeat(4,minmax(150px,1fr));gap:12px;margin-bottom:16px}.stat{background:var(--surface);border:1px solid var(--line);border-radius:7px;padding:16px;box-shadow:var(--shadow)}.stat-label{font-size:12px;color:var(--muted);font-weight:600}.stat-value{font-size:28px;font-weight:750;margin-top:7px}
    .toolbar{background:var(--surface);border:1px solid var(--line);border-radius:7px;box-shadow:var(--shadow);margin-bottom:14px}.toolbar-row{display:flex;align-items:center;gap:10px;padding:12px;flex-wrap:wrap}.toolbar-row+.toolbar-row{border-top:1px solid var(--line)}
    input[type=search]{height:36px;min-width:260px;flex:1;border:1px solid #bfc8d2;border-radius:6px;padding:0 11px}
    button{height:36px;border:1px solid #bfc8d2;border-radius:6px;background:#fff;color:#273240;padding:0 12px;font:inherit;font-weight:600;cursor:pointer}button:disabled{opacity:.45;cursor:not-allowed}.primary{background:#175cd3;color:#fff;border-color:#175cd3}.danger{color:#b42318;border-color:#f1a39b}
    .auth-banner{padding:10px 12px;border-radius:6px;background:#fff8db;color:#7a5a00;border:1px solid #f5e3a2;margin-bottom:12px}.auth-banner.ok{background:#ecfdf3;color:#067647;border-color:#abefc6}
    .table-shell{background:var(--surface);border:1px solid var(--line);border-radius:7px;box-shadow:var(--shadow);overflow:hidden}.table-wrap{overflow:auto;max-height:64vh}table{border-collapse:collapse;width:100%;min-width:980px}th,td{padding:10px 12px;text-align:left;border-bottom:1px solid #edf0f3}th{position:sticky;top:0;background:#f8fafb;color:#475467;font-size:12px;z-index:1}td code{font-family:Consolas,monospace;font-size:12px}.badge{display:inline-flex;min-width:45px;height:24px;border-radius:12px;align-items:center;justify-content:center;font-weight:750;font-size:12px}.b402{color:var(--amber);background:var(--amber-bg)}.b403{color:var(--red);background:var(--red-bg)}.b429{color:#6941c6;background:#f4f3ff}.b401{color:#175cd3;background:#eff6ff}.empty{padding:52px;text-align:center;color:var(--muted)}.message{min-height:20px;color:var(--muted);font-size:13px}.message.error{color:var(--red)}.footer-note{color:#7b8794;font-size:12px;margin:12px 2px 0}
  </style>
</head>
<body>
  <header><div class="header-inner"><div class="brand"><h1>xAI Autoban</h1><p>CPA credential isolation console — v` + pluginVersion + `</p></div><div id="syncState">准备中</div></div></header>
  <main>
    <div id="authBanner" class="auth-banner">正在检测管理密钥…</div>
    <section class="stats">
      <div class="stat"><div class="stat-label">当前隔离</div><div class="stat-value" id="total">-</div></div>
      <div class="stat"><div class="stat-label">402</div><div class="stat-value" id="count402">-</div></div>
      <div class="stat"><div class="stat-label">403</div><div class="stat-value" id="count403">-</div></div>
      <div class="stat"><div class="stat-label">429</div><div class="stat-value" id="count429">-</div></div>
    </section>
    <section class="toolbar">
      <div class="toolbar-row">
        <input id="search" type="search" placeholder="搜索 Auth ID 或原因" autocomplete="off">
        <button class="primary" onclick="loadData()">刷新</button>
        <button id="btnProbe" onclick="runProbe()" disabled>立即巡检</button>
        <label><input id="autoRefresh" type="checkbox" checked> 30 秒自动刷新</label>
      </div>
      <div class="toolbar-row">
        <button id="unbanSelected" onclick="unbanSelected()" disabled>解禁所选</button>
        <button id="unbanAll" class="danger" onclick="unbanAll()" disabled>全部解禁</button>
        <button id="unban402" onclick="unbanStatus(402)" disabled>解禁全部 402</button>
        <button id="unban403" onclick="unbanStatus(403)" disabled>解禁全部 403</button>
        <button id="unban429" onclick="unbanStatus(429)" disabled>解禁全部 429</button>
      </div>
      <div class="toolbar-row"><div id="message" class="message">准备加载数据</div></div>
    </section>
    <section class="table-shell">
      <div class="table-wrap">
        <table>
          <thead><tr><th><input id="selectPage" type="checkbox"></th><th>Auth ID</th><th>状态</th><th>动作</th><th>原因</th><th>隔离时间</th><th>自动解禁</th><th>剩余</th><th>操作</th></tr></thead>
          <tbody id="rows"></tbody>
        </table>
        <div id="empty" class="empty" hidden>当前没有隔离凭据</div>
      </div>
    </section>
    <p class="footer-note">资源页只读展示状态。解禁 / 巡检 / 手动动作需要管理密钥，通过 /v0/management 调用。公开 /action 已移除。</p>
  </main>
  <script>
    const resourceBase='/v0/resource/plugins/xai-autoban';
    const mgmtBase='/v0/management/plugins/xai-autoban';
    const state={bans:[],query:'',selected:new Set(),timer:null,mgmtKey:''};
    const $=id=>document.getElementById(id);
    const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

    function readManagementKey(){
      const keys=['cliproxyapi_management_key','management_key','cpa_management_key','managementKey'];
      for(const k of keys){
        try{
          const v=localStorage.getItem(k);
          if(v&&v.trim()) return v.trim();
        }catch(_){}
      }
      // common CPA webui nested json
      for(const k of Object.keys(localStorage||{})){
        try{
          const raw=localStorage.getItem(k);
          if(!raw||raw.length>5000) continue;
          if(raw.includes('management')&&raw.includes('key')){
            const obj=JSON.parse(raw);
            const cand=obj.managementKey||obj.management_key||obj.key||(obj.auth&&obj.auth.managementKey);
            if(typeof cand==='string'&&cand.trim()) return cand.trim();
          }
        }catch(_){}
      }
      return '';
    }

    function setAuthUI(){
      state.mgmtKey=readManagementKey();
      const ok=!!state.mgmtKey;
      const banner=$('authBanner');
      banner.className='auth-banner'+(ok?' ok':'');
      banner.textContent=ok?'已检测到管理密钥，可执行解禁/巡检等操作。':'未检测到管理密钥：页面只读。请在 CPA 管理中心登录保存密钥后刷新本页。';
      ['btnProbe','unbanSelected','unbanAll','unban402','unban403','unban429'].forEach(id=>{const el=$(id); if(el) el.disabled=!ok;});
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
      if(!state.mgmtKey) throw new Error('missing management key');
      const response=await fetch(mgmtBase+path,{
        method,
        cache:'no-store',
        headers:{
          'Authorization':'Bearer '+state.mgmtKey,
          'Content-Type':'application/json'
        },
        body: body?JSON.stringify(body):undefined
      });
      const text=await response.text();
      let data; try{data=JSON.parse(text)}catch(_){throw new Error(text||('HTTP '+response.status))}
      if(!response.ok) throw new Error(data.error||data.message||('HTTP '+response.status));
      return data;
    }

    function setMessage(text,error=false){$('message').textContent=text;$('message').className='message'+(error?' error':'')}
    function counts(){const out={401:0,402:0,403:0,429:0}; for(const ban of state.bans){ if(out[ban.status_code]!==undefined) out[ban.status_code]++; } return out;}
    function filtered(){const q=state.query.toLowerCase(); return state.bans.filter(b=>!q||b.auth_id.toLowerCase().includes(q)||(b.reason||'').toLowerCase().includes(q));}
    function formatDate(v){const d=new Date(v); return Number.isNaN(d.getTime())?v:d.toLocaleString('zh-CN',{hour12:false});}
    function formatRemaining(seconds){seconds=Math.max(0,Number(seconds||0)); const d=Math.floor(seconds/86400),h=Math.floor(seconds%86400/3600),m=Math.floor(seconds%3600/60); if(d)return d+'天 '+h+'小时'; if(h)return h+'小时 '+m+'分'; return m+'分钟';}

    async function loadData(silent=false){
      try{
        if(!silent){ $('syncState').textContent='同步中'; setMessage('正在加载状态...'); }
        const data=await apiResource('/data');
        state.bans=data.bans||[];
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
        '<td><code>'+esc(ban.auth_id)+'</code>'+(ban.pending_delete?' <span title="待删除">⚠</span>':'')+'</td>'+
        '<td><span class="badge b'+ban.status_code+'">'+ban.status_code+'</span></td>'+
        '<td>'+esc(ban.action||'ban')+'</td>'+
        '<td>'+esc(ban.reason||'')+'</td>'+
        '<td>'+esc(formatDate(ban.banned_at))+'</td>'+
        '<td>'+esc(formatDate(ban.reset_at))+'</td>'+
        '<td>'+esc(formatRemaining(ban.remaining_seconds))+'</td>'+
        '<td><button class="row-action" data-unban="'+esc(ban.auth_id)+'" '+(state.mgmtKey?'':'disabled')+'>解禁</button></td>'+
      '</tr>').join('');
      $('empty').hidden=list.length>0;
      document.querySelectorAll('#rows input[type=checkbox]').forEach(input=>input.addEventListener('change',()=>{
        input.checked?state.selected.add(input.dataset.id):state.selected.delete(input.dataset.id);
        $('unbanSelected').disabled=!state.mgmtKey||state.selected.size===0;
        $('unbanSelected').textContent='解禁所选 ('+state.selected.size+')';
      }));
      document.querySelectorAll('#rows [data-unban]').forEach(btn=>btn.addEventListener('click',()=>unbanOne(btn.dataset.unban)));
      $('unbanSelected').disabled=!state.mgmtKey||state.selected.size===0;
      $('unbanSelected').textContent='解禁所选 ('+state.selected.size+')';
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
