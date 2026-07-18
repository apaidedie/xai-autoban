package ui

const statusJSTemplate = `
// Derive resource base from current page path so subpath reverse-proxy still works.
const resourceBase=(function(){
  try{
    const p=String(location.pathname||'');
    const marker='/plugins/xai-autoban';
    const i=p.indexOf(marker);
    if(i>=0){
      // .../v0/resource/plugins/xai-autoban/status → .../v0/resource/plugins/xai-autoban
      return p.slice(0, i+marker.length);
    }
  }catch(_){}
  return '/v0/resource/plugins/xai-autoban';
})();
// Never embed management secret in HTML (XSS / view-source risk). GET /ops is primary.
const SERVER_MGMT_KEY='';
const PLUGIN_VERSION="__PLUGIN_VERSION__";
const state={bans:[],credentials:[],counts:{},page:{page:1,page_size:50,total:0,pages:1,filter:'all',q:''},filter:'all',query:'',selected:new Set(),timer:null,searchTimer:null,toastTimer:null,busy:false,settings:{},success:'unban',fail:'ban',autoExecute:true,history:[]};
const $=id=>document.getElementById(id);
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function setActionEnabled(ok){
  const can=!!ok && !state.busy;
  const ids=['btnRefresh','unbanSelected','banSelected','disableSelected','reenableSelected','usingApiSelected','usingApiOffSelected','deleteSelected','recheckSelected','saveConfigBtn','selectFilterBtn','clearSelectedBtn'];
  ids.forEach(id=>{const el=$(id); if(el) el.disabled=!can;});
  const n=state.selected.size;
  if(can){
    ['unbanSelected','banSelected','disableSelected','reenableSelected','usingApiSelected','usingApiOffSelected','deleteSelected','recheckSelected'].forEach(id=>{const el=$(id); if(el) el.disabled=n===0;});
    if($('clearSelectedBtn')) $('clearSelectedBtn').disabled=n===0;
  }
  if($('unbanSelected')) $('unbanSelected').textContent=n?('释放 ('+n+')'):'释放';
  if($('deleteSelected')) $('deleteSelected').textContent=n?('删除 ('+n+')'):'删除';
  if($('banSelected')) $('banSelected').textContent=n?('隔离 ('+n+')'):'隔离';
  if($('disableSelected')) $('disableSelected').textContent=n?('禁用 ('+n+')'):'禁用';
  if($('reenableSelected')) $('reenableSelected').textContent=n?('启用 ('+n+')'):'启用';
  if($('usingApiSelected')) $('usingApiSelected').textContent=n?('开 API ('+n+')'):'开 API';
  if($('usingApiOffSelected')) $('usingApiOffSelected').textContent=n?('关 API ('+n+')'):'关 API';
  if($('recheckSelected')) $('recheckSelected').textContent=n?('复检 ('+n+')'):'复检';
  const sh=$('selectedHint');
  if(sh) sh.textContent=n?('已选 '+n):'';
  const sf=$('selectFilterBtn');
  if(sf){
    const fl={all:'全部',healthy:'健康',banned:'隔离',disabled:'禁用',using_api:'API 模式','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
    sf.textContent=state.filter&&state.filter!=='all'?('全选 · '+fl):'全选筛选';
  }
}
function setAuthUI(){
  setActionEnabled(true);
  return true;
}
// Writes use resource only (GET /ops preferred under CPAMP; never /v0/management/plugins/*).
function buildOpsQuery(op, payload){
  const q=new URLSearchParams();
  q.set('op', op);
  Object.keys(payload||{}).forEach(k=>{
    if(k==='op') return;
    const v=payload[k];
    if(v===undefined||v===null) return;
    if(typeof v==='object') q.set(k, JSON.stringify(v));
    else q.set(k, String(v));
  });
  return q.toString();
}
function opsMeta(op, payload){
  const p=payload||{};
  const meta={op:op};
  if(p.auth_id) meta.authId=String(p.auth_id);
  if(Array.isArray(p.auth_ids)) meta.authIds=p.auth_ids.map(String);
  else if(typeof p.auth_ids==='string' && p.auth_ids) meta.authIds=p.auth_ids;
  if(p.action) meta.action=String(p.action);
  return meta;
}
function isListPayload(d){
  // GET /data 列表：有 bans/counts，没有 ok/removed/accepted
  return !!(d && (Array.isArray(d.bans)||Array.isArray(d.credentials)) && d.counts && d.ok!==true && d.removed===undefined && d.accepted===undefined && !d.error);
}
function isOpsResult(d){
  // Must require ok/accepted — list /data also has settings and must not count as save success.
  if(!d || typeof d!=='object') return false;
  if(d.ok===true || d.accepted===true) return true;
  if(d.format==='xai-autoban-backup') return true;
  return false;
}
async function apiResource(path, opts){
  const method=(opts&&opts.method)||'GET';
  const body=opts&&opts.body;
  const withKey=!!(opts&&opts.withKey);
  const useHdr=opts&&opts.headers!==false;
  const opHdr=(opts&&opts.op)||'';
  const authId=(opts&&opts.authId)||'';
  const authIds=opts&&opts.authIds;
  const action=(opts&&opts.action)||'';
  const headers={};
  if(body!==undefined) headers['Content-Type']='application/json';
  // Custom headers first-try optional: some proxies mishandle unknown X-* on resource GET.
  if(useHdr){
    if(opHdr){ headers['X-XAI-Autoban-Op']=String(opHdr); headers['X-Plugin-Op']=String(opHdr); }
    if(authId){ headers['X-XAI-Autoban-Auth-Id']=String(authId); headers['X-Plugin-Auth-Id']=String(authId); }
    if(authIds){
      const s=Array.isArray(authIds)?JSON.stringify(authIds):String(authIds);
      headers['X-XAI-Autoban-Auth-Ids']=s; headers['X-Plugin-Auth-Ids']=s;
    }
    if(action){ headers['X-XAI-Autoban-Action']=String(action); headers['X-Plugin-Action']=String(action); }
  }
  if((withKey || (method!=='GET' && method!=='HEAD')) && SERVER_MGMT_KEY){
    headers['Authorization']='Bearer '+SERVER_MGMT_KEY;
    headers['X-Management-Key']=SERVER_MGMT_KEY;
  }
  const r=await fetch(resourceBase+path,{
    method,cache:'no-store',credentials:'same-origin',
    headers:Object.keys(headers).length?headers:undefined,
    body:body!==undefined?JSON.stringify(body):undefined
  });
  const t=await r.text(); let d; try{d=JSON.parse(t)}catch(_){throw new Error((t&&String(t).slice(0,120))||('HTTP '+r.status))}
  if(!r.ok) throw new Error(d.error||d.message||('HTTP '+r.status)); return d;
}
function b64url(str){
  const bytes=unescape(encodeURIComponent(str));
  let bin='';
  for(let i=0;i<bytes.length;i++) bin+=String.fromCharCode(bytes.charCodeAt(i)&0xff);
  return btoa(bin).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
}
function buildGetOpsURL(base, op, payload){
  // Prefer flat query. Only pack import / oversized.
  const flat=buildOpsQuery(op, payload);
  const needPack=op==='import'||flat.length>1800;
  if(!needPack) return base+'?'+flat;
  const rest=Object.assign({}, payload||{});
  delete rest.op;
  const pack=b64url(JSON.stringify(rest));
  return base+'?op='+encodeURIComponent(op)+'&payload='+encodeURIComponent(pack);
}
// Resource-only writes. Prefer GET /data (always registered) before /ops.
async function apiOps(op, extra){
  const payload=Object.assign({}, extra||{}, {op:op});
  // Drop noisy false bools from query (defaults server-side)
  Object.keys(payload).forEach(k=>{
    if(payload[k]===false && (k==='force'||k==='wait')) delete payload[k];
  });
  const meta=opsMeta(op, payload);
  const errs=[];
  async function tryOne(label, fn){
    try{
      const d=await fn();
      if(isListPayload(d)){ errs.push(label+': got_list_not_op'); return null; }
      if(d && d.error && d.ok!==true){ errs.push(label+': '+(d.message||d.error)); return null; }
      if(isOpsResult(d)) return d;
      errs.push(label+': unexpected_payload');
      return null;
    }catch(e){ errs.push(label+': '+(e.message||e)); return null; }
  }
  let d=null;
  // 1) GET /data query only (no custom headers) — most compatible with CPAMP
  d=await tryOne('GET /data', ()=>apiResource(buildGetOpsURL('/data', op, payload), {headers:false}));
  if(d) return d;
  // 2) GET /data + headers
  d=await tryOne('GET /data+hdr', ()=>apiResource(buildGetOpsURL('/data', op, payload), meta));
  if(d) return d;
  // 3) GET /ops query only
  d=await tryOne('GET /ops', ()=>apiResource(buildGetOpsURL('/ops', op, payload), {headers:false}));
  if(d) return d;
  // 4) GET /ops + headers
  d=await tryOne('GET /ops+hdr', ()=>apiResource(buildGetOpsURL('/ops', op, payload), meta));
  if(d) return d;
  // 5) POST body (needs CPA key or CPAMP admin on mutating resource)
  d=await tryOne('POST /data', ()=>apiResource('/data',Object.assign({method:'POST',body:payload,withKey:!!SERVER_MGMT_KEY}, meta)));
  if(d) return d;
  d=await tryOne('POST /ops', ()=>apiResource('/ops',Object.assign({method:'POST',body:payload,withKey:!!SERVER_MGMT_KEY}, meta)));
  if(d) return d;
  const all404=errs.every(e=>/404|not_found|not found/i.test(e));
  let hint='请升级插件并强刷；若仍 404：完整重启 CPA 以重新注册 resource。';
  if(all404) hint+=' base='+resourceBase+' ver='+PLUGIN_VERSION;
  throw new Error('写操作失败：'+errs.join(' | ')+'。'+hint);
}
function mapPathToOp(method,path,body){
  const p=String(path||'');
  if(method==='GET'&&p.indexOf('/probe/status')>=0) return {op:'probe_status'};
  if(method==='GET'&&p.indexOf('/backup')>=0) return {op:'backup'};
  if((method==='PUT'||method==='POST')&&p.indexOf('/settings')>=0) return Object.assign({op:'settings'}, body||{});
  if(method==='POST'&&p.indexOf('/unban-all')>=0) return Object.assign({op:'unban_all'}, body||{});
  if(method==='POST'&&p.indexOf('/unban')>=0) return Object.assign({op:'unban'}, body||{});
  if(method==='POST'&&p.indexOf('/probe')>=0) return Object.assign({op:'probe'}, body||{});
  if(method==='POST'&&p.indexOf('/apply-action')>=0) return Object.assign({op:'apply'}, body||{});
  if(method==='POST'&&p.indexOf('/reauth')>=0) return Object.assign({op:'reauth'}, body||{});
  if(method==='POST'&&p.indexOf('/bans-recheck-429')>=0) return Object.assign({op:'recheck429'}, body||{});
  if(method==='POST'&&p.indexOf('/recheck-selected')>=0) return Object.assign({op:'recheck_selected'}, body||{});
  if(method==='POST'&&p.indexOf('/list-ids')>=0) return Object.assign({op:'list_ids'}, body||{});
  if(method==='GET'&&p.indexOf('/list-ids')>=0) return Object.assign({op:'list_ids'}, body||{});
  if(method==='POST'&&p.indexOf('/import')>=0) return Object.assign({op:'import'}, body||{});
  return null;
}
async function apiMgmt(method,path,body){
  const mapped=mapPathToOp(method,path,body);
  if(!mapped){
    throw new Error('不支持的操作 '+method+' '+path+'（CPAMP 下不走 /v0/management/plugins）');
  }
  if(mapped.op==='probe_status'){
    try{ return await apiResource('/probe/status'); }catch(_){ /* fall */ }
  }
  return apiOps(mapped.op, mapped);
}
function setMessage(text,err=false){
  const m=$('message'); if(m){ m.textContent=text; m.className='msg'+(err?' err':''); }
}
function toast(text, kind=''){
  const el=$('toast'); if(!el) return;
  el.textContent=text||'';
  el.className='toast show'+(kind?' '+kind:'');
  if(state.toastTimer) clearTimeout(state.toastTimer);
  state.toastTimer=setTimeout(()=>{ el.className='toast'; }, 2800);
}
// Collapse long multi-line results: keep summary + first few detail lines.
function compactResultText(text){
  const raw=String(text||'').trim();
  if(!raw) return '';
  const lines=raw.split(/\r?\n/).map(s=>s.trim()).filter(Boolean);
  if(lines.length<=7) return lines.join('\n');
  const head=lines[0];
  const rest=lines.slice(1);
  const show=rest.slice(0,5);
  const more=rest.length-show.length;
  return [head, ...show, more>0?('…另 '+more+' 条，详见列表筛选'):''].filter(Boolean).join('\n');
}
function setOpResult(text, kind=''){
  const el=$('opResult'); if(!el) return;
  if(!text){ el.hidden=true; el.textContent=''; el.className='op-result'; return; }
  el.hidden=false;
  el.textContent=compactResultText(text);
  el.className='op-result'+(kind?' '+kind:'');
  const panel=$('progressPanel'); if(panel) panel.classList.add('on');
}
function clearOpResult(){ setOpResult(''); }
function setBusy(on, label){
  state.busy=!!on;
  const live=$('syncState');
  if(live){
    if(on){ live.textContent=label||'处理中'; live.className='live busy'; }
    else if(live.classList.contains('busy')){ live.textContent='在线'; live.className='live'; }
  }
  setActionEnabled(!on);
  if(on){
    clearOpResult();
    const panel=$('progressPanel'); if(panel) panel.classList.add('on');
  }
}
function setProgress(cur, total, label){
  const panel=$('progressPanel'), bar=$('progressBar');
  const pl=$('progressLabel'), pc=$('progressCount');
  if(!panel||!bar) return;
  if(total==null || total<0){
    // hide progress UI only when explicitly reset
    panel.classList.remove('on');
    bar.style.width='0%';
    if(pl) pl.textContent='';
    if(pc) pc.textContent='';
    return;
  }
  panel.classList.add('on');
  const t=Math.max(1, Number(total)||1);
  const c=Math.max(0, Math.min(t, Number(cur)||0));
  const pct=Math.max(0, Math.min(100, Math.round(c/t*100)));
  bar.style.width=(c>0?Math.max(2,pct):0)+'%';
  if(pl) pl.textContent=label||(c>=t?'已完成':'处理中');
  if(pc) pc.textContent=c+'/'+t+(c>=t?'（完成）':'');
}
function finishProgress(cur, total, label){
  setProgress(cur, total, label||'已完成');
}
function counts(){const o={401:0,402:0,403:0,429:0}; for(const b of state.bans) if(o[b.status_code]!==undefined) o[b.status_code]++; return o}
function paintChips(){
  const c=state.counts||{};
  const set=(id,v)=>{const el=$(id); if(el) el.textContent=String(v??0)};
  set('c_all',c.all??0); set('c_healthy',c.healthy??0); set('c_banned',c.banned??0);
  set('c_401',c['401']??0); set('c_402',c['402']??0); set('c_403',c['403']??0); set('c_429',c['429']??0); set('c_disabled',c.disabled??0);
  set('f_401',c['401']??0); set('f_402',c['402']??0); set('f_403',c['403']??0); set('f_429',c['429']??0);
  set('ov_all',c.all??0); set('ov_healthy',c.healthy??0); set('ov_banned',c.banned??0);
  set('ov_401',c['401']??0); set('ov_402',c['402']??0); set('ov_403',c['403']??0); set('ov_429',c['429']??0);
  set('ov_using_api', c.using_api??0);
  const sub=$('ov_banned_sub');
  if(sub) sub.textContent='账本 · 跳过调度';
  const disSub=document.querySelector('#overviewCards [data-filter="disabled"] .qs');
  if(disSub) disSub.textContent='CPA 关闭';
  const healthySub=document.querySelector('#overviewCards [data-filter="healthy"] .qs');
  if(healthySub) healthySub.textContent='可调度';
  document.querySelectorAll('#overviewCards [data-filter], #codeStrip [data-filter], #statusChips [data-filter]').forEach(btn=>{
    const on=btn.dataset.filter===state.filter;
    btn.classList.toggle('active', on);
    btn.classList.toggle('on', on);
  });
}
function paintOverviewProbe(probe){
  const n=$('ov_probe'), sub=$('ov_probe_sub'), card=$('ov_probe_card');
  if(!n) return;
  probe=probe||{};
  const ok=probe.last_ok, fail=probe.last_fail, err=probe.last_err, skip=probe.last_skip||0;
  const hasLast=probe.last_run && String(probe.last_run).indexOf('0001')!==0 && String(probe.last_run).length>4;
  if(probe.job_running){
    const done=probe.job_done||0, total=probe.job_total||0;
    n.textContent=(total>0?(done+'/'+total):'…');
    if(sub) sub.textContent='进行中'+(probe.job_age_sec?(' · '+Math.floor(probe.job_age_sec/60)+'分'):'');
    if(card) card.className='qcard warn';
    return;
  }
  if(hasLast){
    n.textContent=String((ok||0))+'/'+String((ok||0)+(fail||0));
    const bits=[];
    if(probe.next_run) bits.push('下次 '+formatDate(probe.next_run).replace(/^\d{4}\//,''));
    else if(probe.last_run) bits.push(formatDate(probe.last_run).replace(/^\d{4}\//,''));
    if(skip>0) bits.push('跳过'+skip);
    if(err) bits.push(String(err).slice(0,24));
    if(probe.auto_execute===false) bits.push('只记录');
    if(probe.running===false && probe.enabled) bits.push('调度停');
    if(sub) sub.textContent=bits.join(' · ')||'点击开始';
    if(card) card.className='qcard'+(fail>0?' bad':(ok>0?' ok':' info'));
    return;
  }
  n.textContent='—';
  if(probe.enabled){
    if(probe.running){
      const nr=probe.next_run?('下次 '+formatDate(probe.next_run).replace(/^\d{4}\//,'')):'约45秒内首次';
      if(sub) sub.textContent='调度中 · '+nr;
    }else{
      if(sub) sub.textContent='已开 · 调度未启动';
    }
  }else{
    if(sub) sub.textContent='已关 · 点击开始';
  }
  if(card) card.className='qcard info';
}
function jumpOverview(kind){
  if(kind==='probe'){
    runProbe();
    return;
  }
  setFilter(kind||'all', false);
  const list=document.querySelector('.card-list')||document.querySelector('.panel');
  if(list) list.scrollIntoView({behavior:'smooth',block:'start'});
}
function setFilter(f, toggle){
  // Toggle off when clicking the same filter again (incl. API 模式).
  if(toggle && state.filter===f) state.filter='all';
  else if(!toggle && state.filter===f) state.filter='all';
  else state.filter=f||'all';
  state.page.page=1;
  paintChips();
  loadData(true);
}
function filtered(){ return state.credentials||[]; }
function pageQueryString(){
  const p=new URLSearchParams();
  p.set('filter', state.filter||'all');
  p.set('page', String(state.page.page||1));
  p.set('page_size', String(state.page.page_size||50));
  if(state.query) p.set('q', state.query);
  return p.toString();
}
function paintPager(){
  const p=state.page||{page:1,pages:1,total:0,page_size:50};
  const info=$('pageInfo');
  if(info) info.textContent='第 '+(p.page||1)+' / '+(p.pages||1)+' 页 · 共 '+(p.total||0)+' 条 · 每页 '+(p.page_size||50);
  const prev=$('prevPageBtn'), next=$('nextPageBtn');
  if(prev) prev.disabled=!(p.page>1);
  if(next) next.disabled=!(p.page<p.pages);
}
function formatDate(v){const d=new Date(v); return Number.isNaN(d.getTime())?v:d.toLocaleString('zh-CN',{hour12:false})}
function formatRemaining(s){s=Math.max(0,Number(s||0)); const d=Math.floor(s/86400),h=Math.floor(s%86400/3600),m=Math.floor(s%3600/60); if(d)return d+'天 '+h+'小时'; if(h)return h+'小时 '+m+'分'; return m+'分钟'}
function reasonLabel(r){return ({payment_required:'额度不足',forbidden:'禁止访问',unauthorized:'未授权',rate_limited:'限流',rate_limited_fallback:'限流(默认等待)',probe_failed:'巡检失败',manual:'手动',token_expired:'Token 过期',needs_refresh:'待刷新'})[r]||r||'-'}
function classLabel(c){return ({rate_limited:'限流',quota_exhausted:'额度用尽',reauth:'需重新授权',permission_denied:'权限拒绝',model_unavailable:'模型不可用',probe_error:'巡检错误',healthy:'健康',token_expired:'Token 过期',needs_refresh:'待刷新'})[c]||c||''}
function labelAction(a){return ({ban:'隔离',disable:'禁用',delete:'删除',none:'不处理',unban:'释放',reenable:'启用',unban_and_reenable:'释放并启用',reauth:'重授权'})[a]||a||'-'}

function renderSettingsSummary(s){
  state.settings=s||{};
  const pe=$('sumProbeEnabled');
  if(pe){ pe.textContent=s.probe_enabled?'已开启':'已关闭'; pe.className=s.probe_enabled?'v on':'v off'; }
  if($('sumInterval')) $('sumInterval').textContent=(s.probe_interval_seconds||'-')+'s';
  const auto=s.auto_execute!==false;
  const ae=$('sumAutoExec');
  if(ae){ ae.textContent=auto?'自动执行':'只记录'; ae.className=auto?'v on':'v off'; }
  if($('sumProbeAction')) $('sumProbeAction').textContent=labelAction(s.probe_action);
  if($('sumOnSuccess')) $('sumOnSuccess').textContent=labelAction(s.probe_on_success);
  if($('sumMode')) $('sumMode').textContent=s.probe_mode||'-';
  const sp=$('statePathHint');
  if(sp){
    const path=s.state_file_resolved||s.state_file||'';
    sp.textContent=path?('状态文件：'+path):'状态文件：未配置（重启会丢失运维台配置）';
    sp.title=path?('运维台配置与隔离账本保存在此；Docker/重建请挂载该目录'):sp.title;
  }
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
  $('f_probe_mode').value=s.probe_mode||'responses_mini';
  if($('f_probe_include_disabled')) $('f_probe_include_disabled').checked=!!s.probe_include_disabled;
  if($('f_probe_only_disabled')) $('f_probe_only_disabled').checked=!!s.probe_only_disabled;
  if($('f_auto_using_api')) $('f_auto_using_api').value=s.auto_using_api||'off';
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
    probe_include_disabled: !!($('f_probe_include_disabled')&&$('f_probe_include_disabled').checked),
    probe_only_disabled: !!($('f_probe_only_disabled')&&$('f_probe_only_disabled').checked),
    auto_using_api: ($('f_auto_using_api')&&$('f_auto_using_api').value)||'off',
    probe_on_success: state.success,
    probe_action: state.fail,
    auto_execute: !!state.autoExecute,
    delete_fallback: $('f_delete_fallback').value,
    action_on_401: $('f_action_on_401').value,
    action_on_402: $('f_action_on_402').value,
    action_on_403: $('f_action_on_403').value,
    action_on_429: $('f_action_on_429').value,
    action_cooldown_seconds: Number($('f_action_cooldown_seconds').value||0),
    fail_streak_403: Number((state.settings&&state.settings.fail_streak_403)||1)
  };
}
function settingsMismatch(draft, got){
  if(!got) return '无 settings';
  const checks=[
    ['probe_interval_seconds', Number(draft.probe_interval_seconds), Number(got.probe_interval_seconds)],
    ['probe_timeout_seconds', Number(draft.probe_timeout_seconds), Number(got.probe_timeout_seconds)],
    ['probe_concurrency', Number(draft.probe_concurrency), Number(got.probe_concurrency)],
    ['auto_execute', !!draft.auto_execute, got.auto_execute!==false],
    ['probe_on_success', String(draft.probe_on_success||''), String(got.probe_on_success||'')],
    ['probe_action', String(draft.probe_action||''), String(got.probe_action||'')],
    ['probe_mode', String(draft.probe_mode||''), String(got.probe_mode||'')],
    ['probe_enabled', !!draft.probe_enabled, !!got.probe_enabled],
    ['auto_using_api', String(draft.auto_using_api||'off'), String(got.auto_using_api||'off')],
  ];
  for(const [k, want, have] of checks){
    if(want!==have) return k+' 期望 '+want+' 实际 '+have;
  }
  return '';
}
async function saveSettings(){
  try{
    setMessage('正在保存配置…');
    const draft=collectDraft();
    const res=await apiMgmt('PUT','/settings',draft);
    if(!res || res.ok!==true || !res.settings){
      throw new Error('保存未确认成功（未返回 ok/settings）。请升级插件并强刷。');
    }
    if(res.applied!=null && Number(res.applied)<1){
      throw new Error('服务端未应用任何字段（applied=0）。请检查代理是否丢弃 query。');
    }
    const bad=settingsMismatch(draft, res.settings);
    if(bad) throw new Error('保存后校验失败：'+bad);
    renderSettingsSummary(res.settings);
    setMessage('配置已保存'+(res.note?(' · '+res.note):'')+(res.applied!=null?(' · '+res.applied+' 项'):''));
    toast('配置已保存','ok');
    closeDrawer();
    await loadData(true);
  }catch(e){ setMessage(e.message,true); toast(e.message,'err'); }
}
async function loadData(silent=false){
  try{
    if(!silent){ $('syncState').textContent='同步中'; $('syncState').className='live busy'; setMessage('正在加载…'); }
    const data=await apiResource('/data?'+pageQueryString());
    state.bans=data.bans||[];
    state.credentials=data.credentials||[];
    state.counts=data.counts||{};
    if(data.page) state.page=Object.assign({page:1,page_size:50,total:0,pages:1}, data.page);
    if(data.settings) renderSettingsSummary(data.settings);
    if(data.probe){ paintOverviewProbe(data.probe); if(data.probe.history) renderHistory(data.probe.history); }
    for(const id of [...state.selected]) if(!state.credentials.some(x=>x.auth_id===id)&&!state.bans.some(x=>x.auth_id===id)) state.selected.delete(id);
    const c=counts();
    if($('total')) $('total').textContent=String(data.count||0);
    if($('count402')) $('count402').textContent=String(c[402]||0);
    if($('count403')) $('count403').textContent=String(c[403]||0);
    if($('count429')) $('count429').textContent=String(c[429]||0);
    paintChips(); paintPager();
    if(!state.busy){ $('syncState').textContent='在线'; $('syncState').className='live'; }
    setMessage('已刷新 · '+new Date().toLocaleTimeString('zh-CN',{hour12:false}));
    render();
  }catch(e){ $('syncState').textContent='异常'; $('syncState').className='live err'; setMessage(e.message,true); toast(e.message,'err'); }
}
// One primary badge: 健康 | 禁用 | 隔离 | 401/402/403/429
function primaryStatus(c){
  if(c.disabled) return {cls:'bdisabled', label:'禁用'};
  if(c.banned){
    const code=Number(c.status_code||0);
    if([401,402,403,429].includes(code)) return {cls:'b'+code, label:String(code)};
    return {cls:'bbanned', label:'隔离'};
  }
  if(c.token_expired||c.status==='401'||c.status_code===401) return {cls:'b401', label:'401'};
  return {cls:'bhealthy', label:'健康'};
}
function needsReauth(c){
  return !!(c.needs_refresh||c.token_expired||c.classification==='reauth'||c.status_code===401||c.status==='401');
}
function rowActions(c){
  const id=encodeURIComponent(c.auth_id||'');
  const dis=state.busy?'disabled':'';
  const primary=[];
  const more=[];
  // At most 2 primary buttons; rest under 更多
  if(needsReauth(c)) primary.push('<button class="row-action primary" data-act="reauth" data-id="'+id+'" '+dis+'>重授权</button>');
  if(c.banned) primary.push('<button class="row-action" data-act="unban" data-id="'+id+'" '+dis+'>释放</button>');
  else if(primary.length<2) primary.push('<button class="row-action" data-act="ban" data-id="'+id+'" '+dis+'>隔离</button>');
  else more.push(['ban','隔离']);
  if(c.disabled){
    if(primary.length<2) primary.push('<button class="row-action" data-act="reenable" data-id="'+id+'" '+dis+'>启用</button>');
    else more.push(['reenable','启用']);
  }else{
    more.push(['disable','禁用']);
  }
  if(c.using_api===true) more.push(['using_api_off','关 API']);
  else if(!c.disabled) more.push(['using_api','开 API']);
  if(c.banned && needsReauth(c) && !primary.some(x=>x.indexOf('data-act="unban"')>=0)) more.push(['unban','释放']);
  let html=primary.join('');
  if(more.length){
    html+='<details class="row-more"><summary>···</summary><div class="row-more-menu">';
    for(const [act,lab] of more){
      const danger=act==='disable'?' danger':'';
      html+='<button type="button" class="'+danger.trim()+'" data-act="'+act+'" data-id="'+id+'" '+dis+'>'+esc(lab)+'</button>';
    }
    html+='</div></details>';
  }
  return '<div class="acts">'+html+'</div>';
}
// Strip noise from reason for one-line subtitle.
function shortWhy(c){
  let cls=classLabel(c.classification);
  let r=String(c.reason||'');
  r=r.replace(/\s*[·•|]\s*streak\s*\d+/ig,'').replace(/\s*\(HTTP\s*\d+\)/ig,'').replace(/\s*·\s*/g,' ').trim();
  // Drop English duplicates of Chinese class
  const rl=reasonLabel(r);
  if(cls && (rl==='-'||!rl||rl===r && /permission|denied|forbidden|exhausted|rate/i.test(r))) return cls;
  if(cls && rl && rl!==cls && rl!=='-') return cls;
  if(cls) return cls;
  if(rl&&rl!=='-') return rl;
  return '';
}
function midCell(c){
  // One human-readable headline; soft403/probe stay secondary only when not banned.
  const parts=[];
  if(c.disabled) parts.push('禁用');
  if(c.banned){
    const code=Number(c.status_code||0);
    if([401,402,403,429].includes(code)) parts.push(String(code));
    else parts.push('隔离');
    if(c.disabled){ /* already 禁用 */ }
    else if(c.disabled===false){ /* no-op */ }
  }else if(!c.disabled){
    if(c.token_expired||c.status==='401') parts.push('401');
    else parts.push('健康');
  }
  if(c.disabled&&c.banned){
    // "禁用 · 403" already covers; add 兼隔离 only if no code
    if(![401,402,403,429].includes(Number(c.status_code||0))) parts.push('兼隔离');
  }
  if(c.using_api===true) parts.push('API');
  let head=parts.join(' · ');
  const p=primaryStatus(c);
  const tags=['<span class="badge '+p.cls+'" title="'+esc(head)+'">'+esc(p.label)+'</span>'];
  if(c.disabled&&c.banned) tags.push('<span class="pill dim">兼隔离</span>');
  if(c.using_api===true) tags.push('<span class="pill dim">API</span>');
  // soft 403 / probe: only when healthy-ish (not main drama)
  if(!c.banned && !c.disabled && c.soft_403_streak>0){
    tags.push('<span class="pill dim" title="软403连击">'+c.soft_403_streak+'/'+(c.soft_403_need||1)+'</span>');
  }
  const sub=[];
  if(c.banned&&c.remaining_seconds!=null&&c.remaining_seconds>=0){
    sub.push('<span class="remain">剩 '+esc(formatRemaining(c.remaining_seconds))+'</span>');
  }
  const why=shortWhy(c);
  if(why && !(c.banned && Number(c.status_code)>0)) sub.push(esc(why));
  if(!c.banned && c.last_probe_ok===false && c.last_probe_status){
    sub.push('探测'+c.last_probe_status);
  }
  return '<div class="mid"><div class="mid-top">'+tags.join('')+'</div>'+
    (sub.length?'<div class="mid-sub">'+sub.join('<span class="sep">·</span>')+'</div>':'')+
    '</div>';
}
function render(){
  const list=filtered();
  const filterLabel={all:'全部',healthy:'健康',banned:'隔离',disabled:'禁用',using_api:'API 模式','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
  const p=state.page||{};
  $('resultCount').textContent=(p.total!=null?p.total:list.length)+' 条'+(state.filter&&state.filter!=='all'?(' · '+filterLabel):'')+(p.pages>1?(' · '+ (p.page||1)+'/'+p.pages):'');
  const lh=$('listHint');
  if(lh){
    if(state.filter==='banned') lh.textContent='筛选：隔离';
    else if(state.filter==='disabled') lh.textContent='筛选：禁用';
    else if(state.filter==='using_api') lh.textContent='筛选：已开启 API 模式 · 再点卡片可取消';
    else if(['401','402','403','429'].includes(state.filter)) lh.textContent='筛选：'+filterLabel+'（状态码口径，可与隔离账本不同）';
    else lh.textContent='点上方卡片筛选 · 勾选后复检或批量操作';
  }
  paintPager();
  $('rows').innerHTML=list.map(c=>{
    const name=c.name||c.label||'-';
    const email=c.email||'';
    const title=email||name;
    return '<div class="rcard">'+
      '<div class="ck"><input type="checkbox" data-id="'+encodeURIComponent(c.auth_id||'')+'" '+(state.selected.has(c.auth_id)?'checked':'')+'></div>'+
      '<div class="acc"><div class="t" title="'+esc(title)+'">'+esc(title)+'</div><div class="id" title="'+esc(c.auth_id)+'">'+esc(c.auth_id)+'</div></div>'+
      midCell(c)+
      '<div class="ops">'+rowActions(c)+'</div>'+
    '</div>';
  }).join('');
  const empty=$('empty');
  empty.hidden=list.length>0;
  empty.textContent=state.filter==='all'&&!state.query?'暂无 xAI 凭证':'无匹配凭证 · 可改筛选';
  document.querySelectorAll('#rows input[type=checkbox]').forEach(input=>input.addEventListener('change',()=>{
    let id=input.dataset.id||'';
    try{ id=decodeURIComponent(id); }catch(_){}
    input.checked?state.selected.add(id):state.selected.delete(id);
    setActionEnabled(!state.busy);
  }));
  document.querySelectorAll('#rows [data-act]').forEach(btn=>btn.addEventListener('click',()=>{
    let id=btn.dataset.id||'';
    try{ id=decodeURIComponent(id); }catch(_){}
    runRowAction(btn.dataset.act,id);
  }));
  setActionEnabled(!state.busy);
}
async function runRowAction(act,id){
  if(!id||state.busy) return;
  const labels={unban:'释放',ban:'隔离',disable:'禁用',reenable:'启用',reauth:'重授权',using_api:'开 API',using_api_off:'关 API',delete:'删除'};
  if(!confirm('确认对凭证执行「'+(labels[act]||act)+'」？\n'+id)) return;
  try{
    setBusy(true, labels[act]||act);
    setProgress(0,1,labels[act]||act);
    setMessage('正在执行 '+(labels[act]||act)+'…');
    if(act==='unban') await apiMgmt('POST','/unban',{auth_id:id});
    else if(act==='reauth') await apiMgmt('POST','/reauth',{auth_id:id,force:true});
    else await apiMgmt('POST','/apply-action',{auth_id:id,action:act,force:true});
    finishProgress(1,1,labels[act]||'完成');
    state.selected.delete(id);
    const msg='已完成 · '+(labels[act]||act)+' · 1/1';
    setMessage(msg);
    setOpResult(msg,'ok');
    await loadData(true);
  }catch(e){ setMessage(e.message,true); setOpResult(e.message,'err'); }
  finally{ setBusy(false); }
}
async function unbanOne(id){ return runRowAction('unban',id); }
async function bulkAct(act){
  if(state.busy) return;
  const ids=[...state.selected];
  if(!ids.length){ setMessage('请先勾选凭证',true); setOpResult('请先勾选凭证','err'); return; }
  const labels={unban:'释放',ban:'隔离',disable:'禁用',reenable:'启用',reauth:'重授权',delete:'删除',using_api:'开 API',using_api_off:'关 API'};
  const danger=act==='delete'?'\n\n删除不可轻易撤销。':(act==='using_api'?'\n\n将开启 API 模式并清除隔离。':(act==='using_api_off'?'\n\n将关闭 API 模式，恢复 OAuth/代理路径。':''));
  if(!confirm('对所选 '+ids.length+' 条执行「'+(labels[act]||act)+'」？'+danger)) return;
  if(act==='delete' && !confirm('再次确认删除 '+ids.length+' 条？')) return;
  try{
    setBusy(true,'批量'+(labels[act]||act));
    setProgress(0, ids.length, '批量'+(labels[act]||act));
    // Server-side bulk when possible (unban goes single path still via apply unban)
    let usedBulk=false;
    if(act!=='unban' && act!=='reauth'){
      try{
        await apiMgmt('POST','/apply-action',{op:'bulk',action:act,auth_ids:ids,force:true});
        usedBulk=true;
      }catch(_){
        try{
          await apiOps('bulk',{action:act,auth_ids:ids,force:true});
          usedBulk=true;
        }catch(__){ usedBulk=false; }
      }
    }
    if(usedBulk){
      // poll bulk progress
      for(let i=0;i<600;i++){
        let st={};
        try{
          const r=await apiOps('bulk_status',{});
          st=(r&&r.bulk)||r||{};
        }catch(_){
          try{ const r=await apiMgmt('GET','/probe/status'); st=(r&&r.bulk)||{}; }catch(__){}
        }
        const done=st.done||0, total=st.total||ids.length;
        setProgress(done, total, '批量'+(labels[act]||act));
        setMessage('批量'+(labels[act]||act)+' '+done+'/'+total);
        if(st.running===false || (total>0 && done>=total && st.running!==true)){
          const okN=st.ok||0, failN=st.fail||0;
          const errs=Array.isArray(st.errors)?st.errors:[];
          const msg='批量'+(labels[act]||act)+'完成 · 成功 '+okN+' / 共 '+total+(failN?(' · 失败 '+failN):'');
          setMessage(msg, failN>0);
          finishProgress(total, total, '批量完成');
          setOpResult(msg+(errs.length?('\n'+errs.slice(0,6).join('\n')):''), failN>0?(okN>0?'warn':'err'):'ok');
          state.selected.clear();
          break;
        }
        await new Promise(r=>setTimeout(r,300));
      }
      await loadData(true);
      return;
    }
    // Fallback: sequential
    let i=0, okN=0, failN=0;
    const fails=[];
    for(const id of ids){
      setMessage('正在'+(labels[act]||act)+' '+(i+1)+'/'+ids.length+' …');
      try{
        if(act==='unban') await apiMgmt('POST','/unban',{auth_id:id});
        else if(act==='reauth') await apiMgmt('POST','/reauth',{auth_id:id,force:true});
        else await apiMgmt('POST','/apply-action',{auth_id:id,action:act,force:true});
        okN++; state.selected.delete(id);
      }catch(one){ failN++; if(fails.length<8) fails.push((id||'')+': '+(one.message||one)); }
      i++; setProgress(i, ids.length, '批量'+(labels[act]||act));
    }
    const msg='批量'+(labels[act]||act)+'完成 · 成功 '+okN+' / 共 '+ids.length+(failN?(' · 失败 '+failN):'');
    setMessage(msg, failN>0);
    finishProgress(ids.length, ids.length, '批量完成');
    setOpResult(msg+(fails.length?('\n'+fails.join('\n')):''), failN>0?(okN>0?'warn':'err'):'ok');
    await loadData(true);
  }catch(e){ setMessage(e.message,true); setOpResult(e.message,'err'); }
  finally{ setBusy(false); }
}
async function exportInspect(kind){
  try{
    setBusy(true,'导出中');
    const res=await apiOps('export',{kind:kind||'reauth'});
    const items=res.items||[];
    const blob=new Blob([JSON.stringify({kind:res.kind,count:res.count,items:items,note:res.note},null,2)],{type:'application/json'});
    const a=document.createElement('a');
    a.href=URL.createObjectURL(blob);
    a.download='xai-autoban-export-'+(kind||'reauth')+'-'+Date.now()+'.json';
    document.body.appendChild(a); a.click(); a.remove();
    setMessage('已导出 '+(res.count||items.length)+' 条 · '+(kind||'reauth'));
    toast('导出完成','ok');
  }catch(e){ setMessage(e.message,true); toast(e.message,'err'); }
  finally{ setBusy(false); }
}
async function selectCurrentFilter(){
  if(state.busy) return;
  const fl={all:'全部',healthy:'健康',banned:'隔离',disabled:'禁用',using_api:'API 模式','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
  try{
    setBusy(true,'拉取筛选 ID');
    setMessage('正在获取「'+fl+'」全部凭证 ID…');
    const res=await apiMgmt('POST','/list-ids',{filter:state.filter||'all',q:state.query||'',limit:800});
    const ids=res.auth_ids||[];
    if(!ids.length){
      setMessage('当前筛选下没有可勾选的凭证',true);
      toast('无匹配凭证','err');
      return;
    }
    // replace selection with filter set
    state.selected=new Set(ids);
    render();
    const note=res.truncated?('（共 '+res.total+'，已截断至 '+ids.length+'）'):'';
    setMessage('已全选「'+fl+'」'+ids.length+' 条'+note+' · 可在「更多」中批量操作');
    toast('已选 '+ids.length+' 条 · '+fl,'ok');
  }catch(e){ setMessage(e.message,true); toast(e.message,'err'); }
  finally{ setBusy(false); }
}
function clearSelection(){
  state.selected.clear();
  if($('selectPage')) $('selectPage').checked=false;
  render();
  setMessage('已清除选择');
}
async function unbanSelected(){ return bulkAct('unban'); }
async function pollProbeUntilDone(){
  let idle=0, lastDone=-1;
  for(;;){
    const st=await apiMgmt('GET','/probe/status');
    const done=st.done||0, total=st.total||0;
    const t=total>0?total:Math.max(done,1);
    setProgress(done, t, '巡检中');
    setMessage('巡检中… '+done+'/'+(total||'?'));
    if(done===lastDone) idle++; else { idle=0; lastDone=done; }
    if(!st.running){
      const r=st.result||{};
      const msg='巡检完成 · 成功 '+(r.ok||0)+' · 失败 '+(r.failed||0)+' · 检 '+(r.checked||done||0)+(r.report_only?'（只输出结果）':'');
      finishProgress(total>0?total:done||1, total>0?total:done||1, '巡检完成');
      setMessage(msg);
      setOpResult(msg+(st.error?('\n'+st.error):''), st.error?'err':((r.failed||0)>0?'warn':'ok'));
      if(st.error) throw new Error(st.error);
      return st;
    }
    if(idle>180 && done===0 && total===0){
      setMessage('巡检似乎卡住，强制重新开始…');
      await apiMgmt('POST','/probe',{force:true,wait:false});
      idle=0;
    }
    await new Promise(r=>setTimeout(r,500));
  }
}
async function runProbe(){
  if(state.busy||!confirm('立即巡检全部 xAI 凭据？')) return;
  try{
    setBusy(true,'巡检中'); setProgress(0,1,'巡检中');
    setMessage('巡检中…');
    let acc;
    try{
      acc=await apiMgmt('POST','/probe',{force:false,wait:false});
    }catch(e){
      const m=String(e.message||e);
      if(/already running/i.test(m)){
        setMessage('已有巡检在进行，接入进度…');
        acc={ok:true,accepted:true,already_running:true};
      }else throw e;
    }
    if(acc && acc.already_running) setMessage('已有巡检在进行，接入进度…');
    if(acc && acc.accepted===false && acc.error) throw new Error(acc.error);
    await pollProbeUntilDone();
    await loadData(true);
  }catch(e){ setMessage(e.message,true); setOpResult(e.message,'err'); }
  finally{ setBusy(false); }
}
async function recheckSelected(){
  if(state.busy) return;
  const ids=[...state.selected];
  if(!ids.length){ setMessage('请先勾选凭证',true); setOpResult('请先勾选凭证','err'); return; }
  if(!confirm('复检所选 '+ids.length+' 条？\n成功：释放隔离并可启用\n失败：按状态码动作（401/402/403/429）')) return;
  const chunkSize=5;
  try{
    setBusy(true,'复检所选');
    setProgress(0, ids.length, '复检所选');
    let done=0, checked=0, okN=0, failed=0, unbanned=0, reenabled=0, skipped=0, banned=0, disabled=0, deleted=0;
    const errs=[];
    for(let i=0;i<ids.length;i+=chunkSize){
      const part=ids.slice(i, i+chunkSize);
      setMessage('复检中… '+Math.min(i+part.length, ids.length)+'/'+ids.length);
      try{
        const res=await apiMgmt('POST','/recheck-selected',{auth_ids:part,reenable_on_ok:true});
        const r=res.result||{};
        checked+=(r.checked||0);
        okN+=(r.ok||0);
        failed+=(r.failed||0);
        unbanned+=(r.unbanned||0);
        reenabled+=(r.reenabled||0);
        skipped+=(r.skipped||0);
        banned+=(r.banned||0);
        disabled+=(r.disabled||0);
        deleted+=(r.deleted||0);
        if(Array.isArray(r.errors)){
          for(const e of r.errors){ if(errs.length<12) errs.push(String(e)); }
        }
      }catch(one){
        failed+=part.length;
        if(errs.length<12) errs.push((one.message||one)+' · batch@'+i);
      }
      done=Math.min(i+part.length, ids.length);
      setProgress(done, ids.length, '复检所选');
    }
    const msg='复检完成 · 检 '+checked+' · 成功 '+okN+' · 失败 '+failed+' · 释放 '+unbanned+' · 启用 '+reenabled+' · 隔离 '+banned+' · 禁用 '+disabled+' · 删除 '+deleted+' · 跳过 '+skipped;
    const detail=msg+(errs.length?('\n'+errs.join('\n')):'');
    setMessage(msg);
    finishProgress(ids.length, ids.length, '复检完成');
    // streak/grace 等探测噪声用中性样式，避免大块刺眼红底
    const soft=failed>0 && (okN>0 || /连击|宽限|不隔离|streak|grace|not isolated|skipped_/i.test(detail));
    // Prefer short summary in panel; detail lines already capped server-side.
    setOpResult(detail, failed>0?(soft?'warn':'err'):'ok');
    state.selected.clear();
    await loadData(true);
  }catch(e){ setMessage(e.message,true); setOpResult(e.message,'err'); }
  finally{ setBusy(false); }
}
async function exportBackup(){
  if(state.busy) return;
  try{
    setBusy(true,'导出中'); setProgress(40,100);
    setMessage('正在导出备份…');
    const data=await apiMgmt('GET','/backup');
    setProgress(100,100);
    const blob=new Blob([JSON.stringify(data,null,2)],{type:'application/json'});
    const url=URL.createObjectURL(blob);
    const a=document.createElement('a');
    const ts=new Date().toISOString().replace(/[:.]/g,'-');
    a.href=url; a.download='xai-autoban-backup-'+ts+'.json';
    document.body.appendChild(a); a.click(); a.remove();
    URL.revokeObjectURL(url);
    const n=(data.bans&&data.bans.length)||data.count||0;
    setMessage('备份已下载 · bans='+n);
    toast('备份已下载 · bans='+n,'ok');
  }catch(e){ setMessage(e.message,true); toast(e.message,'err'); }
  finally{ setBusy(false); setProgress(0,0); }
}
function importBackup(){
  if(state.busy) return;
  const f=$('importFile'); if(!f) return;
  f.value=''; f.click();
}
async function handleImportFile(file){
  if(!file||state.busy) return;
  try{
    setBusy(true,'导入中'); setProgress(20,100);
    setMessage('正在读取备份…');
    const text=await file.text();
    let obj; try{ obj=JSON.parse(text); }catch(_){ throw new Error('JSON 解析失败'); }
    const bansN=(obj.bans&&obj.bans.length)||(obj.status&&obj.status.bans&&obj.status.bans.length)||0;
    const hasSettings=!!(obj.settings||(obj.status&&obj.status.settings));
    if(!confirm('确认导入备份？\n隔离项约 '+bansN+' 条'+(hasSettings?'\n将同时应用 settings（运行时）':'')+'\n仅导入尚未过期的隔离记录。')) {
      setBusy(false); setProgress(0,0); return;
    }
    setProgress(60,100); setMessage('正在导入…');
    const res=await apiMgmt('POST','/import', obj);
    setProgress(100,100);
    const msg='导入完成 · bans='+(res.imported||0)+(res.settings_applied?' · 已应用 settings':'');
    setMessage(msg); toast(msg,'ok');
    await loadData(true);
  }catch(e){ setMessage(e.message,true); toast(e.message,'err'); }
  finally{ setBusy(false); setProgress(0,0); }
}

if($('importFile')) $('importFile').onchange=e=>{ const f=e.target.files&&e.target.files[0]; if(f) handleImportFile(f); };
// API 模式 chip: setFilter toggles off when clicked again.
if($('usingApiFilterBtn')) $('usingApiFilterBtn').onclick=()=>setFilter('using_api', true);
$('search').oninput=e=>{
  state.query=e.target.value.trim();
  state.page.page=1;
  if(state.searchTimer) clearTimeout(state.searchTimer);
  state.searchTimer=setTimeout(()=>loadData(true),280);
};
$('selectPage').onchange=e=>{for(const c of filtered()) e.target.checked?state.selected.add(c.auth_id):state.selected.delete(c.auth_id); render();};
if($('selectFilterBtn')) $('selectFilterBtn').onclick=()=>selectCurrentFilter();
if($('clearSelectedBtn')) $('clearSelectedBtn').onclick=()=>clearSelection();
if($('prevPageBtn')) $('prevPageBtn').onclick=()=>{ if((state.page.page||1)>1){ state.page.page--; loadData(true);} };
if($('nextPageBtn')) $('nextPageBtn').onclick=()=>{ if((state.page.page||1)<(state.page.pages||1)){ state.page.page++; loadData(true);} };
$('autoRefresh').onchange=()=>{if(state.timer) clearInterval(state.timer); state.timer=$('autoRefresh').checked?setInterval(()=>loadData(true),30000):null;};
document.querySelectorAll('#statusChips [data-filter]').forEach(btn=>btn.onclick=()=>setFilter(btn.dataset.filter,true));
document.querySelectorAll('#codeStrip [data-filter]').forEach(btn=>{
  if(btn.id==='usingApiFilterBtn') return;
  btn.onclick=()=>setFilter(btn.dataset.filter,true);
});
document.querySelectorAll('#overviewCards [data-jump]').forEach(btn=>btn.onclick=()=>jumpOverview(btn.dataset.jump));
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
`
