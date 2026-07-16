package ui

import (
	"encoding/json"
	"html"
)

// StatusPage renders the ops console.
// serverMgmtKey is accepted for ABI compatibility but never embedded in HTML
// (resource ops use GET /data|/ops; secrets stay server-side).
func StatusPage(pluginName, pluginVersion, serverMgmtKey string) string {
	_ = serverMgmtKey
	name := html.EscapeString(pluginName)
	verJS, err := json.Marshal(pluginVersion)
	if err != nil {
		verJS = []byte(`""`)
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + name + `</title>
<style>
:root{color-scheme:dark;--bg:#070b14;--panel:#101a2c;--line:rgba(148,163,184,.16);--text:#f8fafc;--muted:#93a4c3;--cyan:#22d3ee;--blue:#3b82f6;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--violet:#a78bfa;--mono:ui-monospace,Consolas,monospace;--sans:Inter,ui-sans-serif,system-ui,"Segoe UI",sans-serif}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:var(--sans);color:var(--text);background:radial-gradient(1000px 500px at 10% -10%,rgba(34,211,238,.1),transparent 50%),radial-gradient(800px 400px at 100% 0,rgba(59,130,246,.1),transparent 45%),linear-gradient(180deg,#070b14,#0a101c);font-size:14px}
.shell{max-width:1540px;margin:0 auto;padding:14px 18px 28px}
.top{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;margin-bottom:12px}
.kicker{display:inline-flex;align-items:center;gap:8px;color:var(--cyan);font-size:11px;font-weight:800;letter-spacing:.12em}
.kicker i{width:7px;height:7px;border-radius:50%;background:var(--cyan);box-shadow:0 0 0 4px rgba(34,211,238,.15)}
h1{margin:8px 0 0;font-size:26px;font-weight:800;letter-spacing:-.03em}
.sub{margin:6px 0 0;color:var(--muted);font-size:13px}
.live{padding:8px 12px;border-radius:999px;border:1px solid var(--line);background:rgba(15,23,42,.75);color:var(--green);font-size:12px;font-weight:800}
.banner{padding:11px 14px;border-radius:12px;margin-bottom:12px;border:1px solid rgba(52,211,153,.3);background:rgba(6,78,59,.35);color:#bbf7d0;font-weight:700}
.banner.warn{border-color:rgba(251,191,36,.35);background:rgba(120,53,15,.35);color:#fde68a}
.panel{background:linear-gradient(180deg,rgba(18,28,46,.96),rgba(12,20,34,.98));border:1px solid var(--line);border-radius:16px;margin-bottom:10px;overflow:hidden;box-shadow:0 16px 40px rgba(0,0,0,.35)}
.phd{display:flex;justify-content:space-between;align-items:center;gap:10px;padding:10px 14px;border-bottom:1px solid var(--line)}
.phd h2{margin:0;font-size:12px;font-weight:800;letter-spacing:.08em;color:#dbe4f3}
.hint{color:var(--muted);font-size:12px}
.cfg-sum{display:flex;flex-wrap:wrap;gap:6px 14px;padding:8px 12px;align-items:center;color:var(--muted);font-size:12px;font-weight:650}
.cfg-sum b{color:#e2e8f0;font-weight:800}
.cfg-sum .dot{opacity:.35}
.cfg-details>summary{list-style:none;cursor:pointer;display:flex;justify-content:space-between;align-items:center;gap:10px;padding:10px 14px;border-bottom:1px solid transparent;user-select:none}
.cfg-details>summary::-webkit-details-marker{display:none}
.cfg-details[open]>summary{border-bottom-color:var(--line)}
.cfg-details .cfg-chev{color:var(--muted);font-size:12px;font-weight:750}
.cfg-grid{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:6px;padding:8px 12px 12px}
.cfg-card{background:rgba(7,12,22,.55);border:1px solid var(--line);border-radius:10px;padding:8px 10px;min-height:44px}
.cfg-card.accent{border-color:rgba(59,130,246,.45);box-shadow:0 0 0 1px rgba(59,130,246,.12) inset}
.cfg-card .l{color:var(--muted);font-size:10px;font-weight:800;letter-spacing:.04em}
.cfg-card .v{margin-top:5px;font-size:13px;font-weight:800;color:#f8fafc}
.cfg-card .v.on{color:var(--green)}.cfg-card .v.off{color:var(--amber)}
@media(max-width:1100px){.cfg-grid{grid-template-columns:repeat(3,minmax(0,1fr))}}
@media(max-width:700px){.cfg-grid{grid-template-columns:1fr 1fr}}
.toolbar{padding:12px 14px 10px;border-bottom:1px solid rgba(148,163,184,.08)}
.tools{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.tools input[type=search]{flex:1 1 220px;min-width:180px}
.tools-end{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin-left:auto}
.sel-bar{display:flex;flex-wrap:wrap;align-items:center;gap:6px 10px;margin-top:8px;padding:6px 10px;border-radius:10px;background:rgba(7,12,22,.4);border:1px solid rgba(148,163,184,.08)}
.sel-bar .sel-inline{display:inline-flex;align-items:center;gap:6px;color:#cbd5e1;font-size:12px;font-weight:700;cursor:pointer;user-select:none}
.sel-bar .sel-inline input{accent-color:var(--cyan)}
.sel-bar .sel-link{height:28px;padding:0 10px;border-radius:8px;border:0;background:transparent;color:var(--muted);font-size:12px;font-weight:750}
.sel-bar .sel-link:hover{color:var(--text);background:rgba(51,65,85,.45)}
.sel-bar .sel-link:disabled{opacity:.35;cursor:not-allowed}
.sel-bar .sel-count{margin-left:auto;font-size:12px;font-weight:800;color:var(--cyan);font-variant-numeric:tabular-nums;min-height:1em}
.sel-bar .sel-hint{color:var(--muted);font-size:11px;font-weight:650}
.more{position:relative}
.more>summary{list-style:none;cursor:pointer;display:inline-flex;align-items:center;height:38px;padding:0 12px;border-radius:11px}
.more>summary::-webkit-details-marker{display:none}
.more-menu{position:absolute;right:0;top:42px;z-index:20;min-width:188px;padding:6px;border-radius:12px;border:1px solid var(--line);background:rgba(15,23,42,.98);box-shadow:0 16px 40px rgba(0,0,0,.45);display:flex;flex-direction:column;gap:2px}
.more-menu button,.more-menu label{height:34px;justify-content:flex-start;text-align:left;background:transparent;border:0;width:100%;border-radius:8px;padding:0 10px;display:inline-flex;align-items:center;color:var(--text);font-weight:700;font-size:13px;cursor:pointer}
.more-menu button:hover{background:rgba(51,65,85,.8)}
.more-menu button.danger{color:#fda4af}
.more-menu button.danger:hover{background:rgba(244,63,94,.14)}
.more-menu button:disabled{opacity:.4;cursor:not-allowed}
.more-menu .more-div{height:1px;margin:4px 6px;background:rgba(148,163,184,.1)}
.auth-row{border-top:0!important;padding:6px 14px!important}
.auth-row.auth-ok{padding:6px 14px!important;opacity:.9}
.auth-row.auth-ok input{display:none}

.qcard.s401{border-color:rgba(59,130,246,.22)}.qcard.s402{border-color:rgba(251,191,36,.22)}.qcard.s403{border-color:rgba(251,113,133,.22)}.qcard.s429{border-color:rgba(167,139,250,.22)}
.qcard.active,.qcard.on{border-color:rgba(34,211,238,.5);box-shadow:0 0 0 1px rgba(34,211,238,.18) inset}
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
[hidden]{display:none!important}
.metrics{display:none!important}
.row{display:flex;align-items:center;gap:10px;flex-wrap:wrap;padding:12px 14px;border-top:1px solid rgba(148,163,184,.08)}
input[type=search],input[type=password],input[type=text],input[type=number],select{height:38px;min-width:160px;flex:1;border:1px solid rgba(148,163,184,.28);border-radius:11px;background:rgba(7,12,22,.85);color:var(--text);padding:0 12px;font:inherit;outline:none}
input:focus,select:focus{border-color:rgba(96,165,250,.7);box-shadow:0 0 0 3px rgba(59,130,246,.16)}
label.chk{display:flex;align-items:center;gap:8px;color:#dbe4f3;font-weight:650;white-space:nowrap}
button{height:38px;border:1px solid rgba(148,163,184,.28);border-radius:11px;background:rgba(30,41,59,.9);color:var(--text);padding:0 13px;font:inherit;font-weight:750;cursor:pointer}
button:hover{background:rgba(51,65,85,.95)}button:disabled{opacity:.35;cursor:not-allowed}
.bp{background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8;color:#fff}
.bd{background:rgba(244,63,94,.18);border-color:rgba(251,113,133,.4);color:#fecdd3}
.bg{background:transparent}.bs{background:rgba(15,23,42,.7)}
.msg-row{padding:2px 14px 6px!important;border-top:0!important}
.msg{min-height:16px;color:var(--muted);font-size:12px;font-weight:650}
.msg.err{color:#fda4af}
.progress-panel{display:none;margin:0 14px 12px;padding:12px 14px;border-radius:14px;border:1px solid rgba(148,163,184,.1);background:linear-gradient(180deg,rgba(15,23,42,.7),rgba(8,14,26,.75))}
.progress-panel.on{display:block}
.progress-meta{display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:8px;font-size:12px;font-weight:800}
.progress-meta .pl{color:#cbd5e1}
.progress-meta .pc{color:var(--cyan);font-family:var(--mono);font-variant-numeric:tabular-nums}
.progress{height:6px;border-radius:999px;background:rgba(148,163,184,.1);overflow:hidden}
.progress>i{display:block;height:100%;width:0;border-radius:999px;background:linear-gradient(90deg,var(--cyan),var(--blue));transition:width .15s ease}
/* Soft result panel — compact summary (not 100+ auth lines) */
.op-result{margin-top:10px;padding:10px 12px;border-radius:10px;font-size:12px;font-weight:650;line-height:1.5;border:1px solid rgba(148,163,184,.12);background:rgba(2,6,23,.35);color:#cbd5e1;white-space:pre-wrap;max-height:120px;overflow:auto}
.op-result.ok{border-color:rgba(34,211,238,.22);background:rgba(8,47,73,.28);color:#a5f3fc}
.op-result.warn{border-color:rgba(148,163,184,.16);background:rgba(30,41,59,.4);color:#e2e8f0}
.op-result.err{border-color:rgba(251,113,133,.22);background:rgba(69,10,10,.22);color:#fecdd3}
.toast{
  position:fixed;right:18px;bottom:18px;z-index:80;max-width:min(420px,92vw);
  padding:12px 14px;border-radius:12px;border:1px solid var(--line);
  background:rgba(15,23,42,.96);color:var(--text);font-size:13px;font-weight:700;
  box-shadow:0 16px 40px rgba(0,0,0,.4);transform:translateY(12px);opacity:0;
  transition:transform .16s ease,opacity .16s ease;pointer-events:none
}
.toast.show{transform:none;opacity:1}
.toast.ok{border-color:rgba(52,211,153,.4);background:rgba(6,78,59,.92);color:#bbf7d0}
.toast.err{border-color:rgba(251,113,133,.45);background:rgba(127,29,29,.92);color:#fecdd3}
.live.busy{color:var(--amber)}
.live.err{color:var(--red)}
@media (prefers-reduced-motion:reduce){.progress>i,.toast{transition:none}}
.table-wrap{overflow:auto;max-height:56vh}
.table-wrap.fade{opacity:.45}
table{width:100%;border-collapse:separate;border-spacing:0;min-width:1040px;transition:opacity .12s ease}
@media (prefers-reduced-motion:reduce){table{transition:none}}
.pager{display:flex;justify-content:space-between;align-items:center;gap:10px;padding:10px 14px;border-top:1px solid rgba(148,163,184,.08)}
.pager .pinfo{color:var(--muted);font-size:12px;font-weight:700}
.pager .pbtns{display:flex;gap:8px}
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
.pill{display:inline-flex;height:22px;align-items:center;padding:0 8px;border-radius:999px;background:rgba(148,163,184,.1);border:1px solid rgba(148,163,184,.16);font-size:11px;font-weight:700}
.pill.dim{opacity:.85;font-weight:650;color:#94a3b8}
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
.fchip.ghost,.code-chip.ghost{opacity:.8}
.card-list{display:flex;flex-direction:column;gap:6px;padding:8px 10px 10px;max-height:62vh;overflow:auto}
.rcard{
  display:grid;grid-template-columns:28px minmax(160px,1.3fr) minmax(160px,1fr) auto;
  gap:6px 10px;align-items:center;padding:8px 10px;border-radius:11px;border:1px solid var(--line);
  background:rgba(8,14,26,.72)
}
.rcard:hover{border-color:rgba(96,165,250,.28);background:rgba(15,23,42,.88)}
.rcard .acc .t{font-weight:750;color:#fff;font-size:13px;line-height:1.25;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.rcard .acc .id{margin-top:2px;font-family:var(--mono);font-size:10.5px;color:#93a4c3;max-width:100%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.rcard .mid{display:flex;flex-direction:column;gap:3px;min-width:0}
.rcard .mid-top{display:flex;flex-wrap:wrap;align-items:center;gap:5px}
.rcard .mid-sub{display:flex;flex-wrap:wrap;align-items:center;gap:5px;color:var(--muted);font-size:11px;line-height:1.3}
.rcard .mid-sub .sep{opacity:.35}
.rcard .ops{justify-self:end}
.rcard .ops .acts{display:flex;flex-wrap:nowrap;gap:5px;align-items:center}
.rcard .ops .row-more{position:relative}
.rcard .ops .row-more>summary{list-style:none;cursor:pointer;height:28px;padding:0 8px;border-radius:8px;border:1px solid rgba(148,163,184,.22);background:rgba(30,41,59,.75);font-size:12px;font-weight:750;display:inline-flex;align-items:center}
.rcard .ops .row-more>summary::-webkit-details-marker{display:none}
.rcard .ops .row-more-menu{position:absolute;right:0;top:32px;z-index:15;min-width:120px;padding:4px;border-radius:10px;border:1px solid var(--line);background:rgba(15,23,42,.98);box-shadow:0 12px 28px rgba(0,0,0,.4);display:flex;flex-direction:column;gap:2px}
.rcard .ops .row-more-menu button{height:30px;border:0;background:transparent;text-align:left;padding:0 10px;border-radius:7px;font-size:12px;font-weight:700;color:var(--text);cursor:pointer}
.rcard .ops .row-more-menu button:hover{background:rgba(51,65,85,.85)}
.rcard .ops .row-more-menu button.danger{color:#fda4af}
@media(max-width:900px){
  .rcard{grid-template-columns:28px 1fr auto;grid-template-areas:"ck acc ops" "mid mid mid";row-gap:6px}
  .rcard .ck{grid-area:ck}.rcard .acc{grid-area:acc}.rcard .mid{grid-area:mid}.rcard .ops{grid-area:ops;justify-self:end}
}
/* Single stats strip: all filters in one compact row */
.stats-strip{display:flex;flex-wrap:wrap;gap:6px;margin:0 0 10px;padding:2px 0}
.stat{
  height:36px;padding:0 11px;border-radius:999px;border:1px solid var(--line);
  background:rgba(12,20,34,.9);color:var(--muted);font-weight:750;font-size:12px;
  display:inline-flex;align-items:center;gap:7px;cursor:pointer;white-space:nowrap
}
.stat b{font-size:13px;font-weight:850;font-variant-numeric:tabular-nums;color:var(--text)}
.stat:hover{border-color:rgba(148,163,184,.35);color:var(--text);background:rgba(15,23,42,.95)}
.stat.active,.stat.on{border-color:rgba(34,211,238,.55);background:rgba(8,47,73,.4);color:#e0f2fe;box-shadow:0 0 0 1px rgba(34,211,238,.15) inset}
.stat.ok b{color:var(--green)}.stat.warn b{color:var(--amber)}.stat.bad b{color:var(--red)}.stat.info b{color:var(--cyan)}
.stat.s401 b{color:#93c5fd}.stat.s402 b{color:#fcd34d}.stat.s403 b{color:#fda4af}.stat.s429 b{color:#ddd6fe}
.stat.ghost{color:var(--muted);opacity:.85}
.stat.probe{margin-left:auto}
@media(max-width:700px){.stat.probe{margin-left:0}}
.row-action{height:28px;padding:0 9px;border-radius:8px;font-size:12px}
.row-action.primary{background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8;color:#fff}
.row-action.primary:hover{background:linear-gradient(180deg,#60a5fa,#3b82f6);border-color:#2563eb;color:#fff}
h1{font-size:22px}
@media(max-width:700px){h1{font-size:20px}}
/* keep legacy class hooks for JS active toggles */
.qcards,.code-strip{display:contents}
.qcard,.code-chip{/* superseded by .stat */}
</style>
</head>
<body>
<div class="shell">
  <div class="top">
    <div>
      <div class="kicker"><i></i>xAI 运维台 · v` + pluginVersion + `</div>
      <h1>凭证巡检</h1>
      <p class="sub" id="cfgOneLine">加载配置…</p>
    </div>
    <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <div class="live" id="syncState">准备中</div>
      <button class="bs" id="btnRefresh" type="button" onclick="loadData()" title="刷新">刷新</button>
      <button class="bp" id="btnProbe" type="button" onclick="runProbe()" disabled>巡检</button>
      <button class="bs" id="openConfigBtn" type="button">配置</button>
    </div>
  </div>

  <section class="panel">
    <details class="cfg-details" id="cfgDetails">
      <summary>
        <div class="cfg-sum" id="cfgSumLine"><span>巡检配置</span><span class="dot">·</span><b id="sumProbeEnabled">-</b><span class="dot">·</span><span id="sumInterval">-</span><span class="dot">·</span><b id="sumAutoExec">-</b><span class="dot">·</span><span id="sumProbeAction">-</span></div>
        <span class="cfg-chev">展开详情</span>
      </summary>
      <div class="cfg-grid" id="cfgPills">
        <div class="cfg-card"><div class="l">定时巡检</div><div class="v" id="sumProbeEnabled2">-</div></div>
        <div class="cfg-card"><div class="l">间隔</div><div class="v" id="sumInterval2">-</div></div>
        <div class="cfg-card accent"><div class="l">自动执行</div><div class="v" id="sumAutoExec2">-</div></div>
        <div class="cfg-card"><div class="l">问题策略</div><div class="v" id="sumProbeAction2">-</div></div>
        <div class="cfg-card"><div class="l">成功策略</div><div class="v" id="sumOnSuccess">-</div></div>
        <div class="cfg-card"><div class="l">探测模式</div><div class="v" id="sumMode">-</div></div>
      </div>
    </details>
  </section>

  <div class="stats-strip" id="overviewCards" role="toolbar" aria-label="筛选">
    <button type="button" class="stat info" data-jump="all" data-filter="all" title="全部 xAI 凭证">全部 <b id="ov_all">0</b></button>
    <button type="button" class="stat ok" data-jump="healthy" data-filter="healthy" title="可调度">健康 <b id="ov_healthy">0</b></button>
    <button type="button" class="stat warn" data-jump="banned" data-filter="banned" title="隔离账本">隔离 <b id="ov_banned">0</b></button>
    <button type="button" class="stat" data-jump="disabled" data-filter="disabled" title="CPA 已关闭">禁用 <b id="c_disabled">0</b></button>
    <button type="button" class="stat s401" data-filter="401" title="状态码 401">401 <b id="ov_401">0</b></button>
    <button type="button" class="stat s402" data-filter="402" title="状态码 402">402 <b id="ov_402">0</b></button>
    <button type="button" class="stat s403" data-filter="403" title="状态码 403">403 <b id="ov_403">0</b></button>
    <button type="button" class="stat s429" data-filter="429" title="状态码 429">429 <b id="ov_429">0</b></button>
    <button type="button" class="stat ghost" id="clearFilterBtn" data-filter="all" title="清除筛选">清除</button>
    <button type="button" class="stat probe info" data-jump="probe" id="ov_probe_card" title="立即巡检">巡检 <b id="ov_probe">—</b></button>
  </div>
  <div id="codeStrip" hidden aria-hidden="true"></div>
  <span id="ov_banned_sub" hidden></span>
  <span id="ov_probe_sub" hidden></span>
  <span id="sumInterval_legacy" hidden></span>
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
    <b>隔离</b>=插件内跳过调度；<b>禁用</b>=关闭凭证；<b>启用</b>=打开凭证；<b>删除</b>=Management 删除。
    日常策略在「编辑配置」；禁用/删除需插件管理中配置 CPA Management Key（非 cpamp_ 面板密钥）。
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
        <select id="f_auto_using_api" title="OAuth 探测失败时是否自动开启 CPA 使用 API 模式">
          <option value="on_403">仅 403 时自动（推荐）</option>
          <option value="on_fail">401/402/403 自动</option>
          <option value="off">关闭（仅手动）</option>
        </select>
      </div>
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

<script>
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
const PLUGIN_VERSION=` + string(verJS) + `;
const state={bans:[],credentials:[],counts:{},page:{page:1,page_size:50,total:0,pages:1,filter:'all',q:''},filter:'all',query:'',selected:new Set(),timer:null,searchTimer:null,toastTimer:null,busy:false,settings:{},success:'unban',fail:'ban',autoExecute:true,history:[]};
const $=id=>document.getElementById(id);
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function setActionEnabled(ok){
  const can=!!ok && !state.busy;
  const ids=['btnProbe','btnRefresh','unbanSelected','banSelected','disableSelected','reenableSelected','usingApiSelected','deleteSelected','recheckSelected','saveConfigBtn','selectFilterBtn','clearSelectedBtn'];
  ids.forEach(id=>{const el=$(id); if(el) el.disabled=!can;});
  const n=state.selected.size;
  if(can){
    ['unbanSelected','banSelected','disableSelected','reenableSelected','usingApiSelected','deleteSelected','recheckSelected'].forEach(id=>{const el=$(id); if(el) el.disabled=n===0;});
    if($('clearSelectedBtn')) $('clearSelectedBtn').disabled=n===0;
  }
  if($('unbanSelected')) $('unbanSelected').textContent=n?('释放所选 ('+n+')'):'释放所选';
  if($('deleteSelected')) $('deleteSelected').textContent=n?('删除所选 ('+n+')'):'删除所选';
  if($('banSelected')) $('banSelected').textContent=n?('隔离所选 ('+n+')'):'隔离所选';
  if($('disableSelected')) $('disableSelected').textContent=n?('禁用所选 ('+n+')'):'禁用所选';
  if($('reenableSelected')) $('reenableSelected').textContent=n?('启用所选 ('+n+')'):'启用所选';
  if($('usingApiSelected')) $('usingApiSelected').textContent=n?('API 模式所选 ('+n+')'):'API 模式所选';
  if($('recheckSelected')) $('recheckSelected').textContent=n?('复检所选 ('+n+')'):'复检所选';
  const sh=$('selectedHint');
  if(sh) sh.textContent=n?('已选 '+n):'';
  const sf=$('selectFilterBtn');
  if(sf){
    const fl={all:'全部',healthy:'健康',banned:'隔离',disabled:'已禁用','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
    sf.textContent=state.filter&&state.filter!=='all'?('全选 · '+fl):'全选当前筛选';
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
  const sub=$('ov_banned_sub');
  if(sub){
    // Keep isolation ledger meaning; do not paste 40x counts here (different口径).
    sub.textContent='隔离账本 · 调度跳过';
  }
  document.querySelectorAll('#overviewCards [data-filter], #statusChips [data-filter]').forEach(btn=>{
    const on=btn.dataset.filter===state.filter;
    btn.classList.toggle('active', on);
    btn.classList.toggle('on', on);
  });
}
function paintOverviewProbe(probe){
  const n=$('ov_probe'), card=$('ov_probe_card');
  if(!n) return;
  probe=probe||{};
  const ok=probe.last_ok, fail=probe.last_fail;
  if(probe.last_run && probe.last_run.indexOf('0001')!==0){
    n.textContent=String((ok||0))+'/'+String((ok||0)+(fail||0));
    if(card){
      card.className='stat probe'+(fail>0?' bad':(ok>0?' ok':' info'));
      card.title='上次巡检 '+(probe.last_run?formatDate(probe.last_run):'')+(fail>0?(' · 失败 '+fail):'');
    }
  }else{
    n.textContent='—';
    if(card){ card.className='stat probe info'; card.title=probe.enabled?'定时已开 · 点此立即巡检':'点此立即巡检'; }
  }
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
  if(toggle && state.filter===f) state.filter='all'; else state.filter=f||'all';
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

function setTxt(id, v, cls){
  const el=$(id); if(!el) return;
  el.textContent=v;
  if(cls!==undefined) el.className=cls;
}
function renderSettingsSummary(s){
  state.settings=s||{};
  const on=!!s.probe_enabled;
  const auto=s.auto_execute!==false;
  const pe=on?'已打开':'关闭';
  const iv=(s.probe_interval_seconds||'-')+'s';
  const ae=auto?'自动执行':'只输出';
  const act=labelAction(s.probe_action);
  const suc=labelAction(s.probe_on_success);
  const mode=s.probe_mode||'-';
  setTxt('sumProbeEnabled', pe, on?'on':'off');
  setTxt('sumProbeEnabled2', pe, on?'v on':'v off');
  setTxt('sumInterval', iv);
  setTxt('sumInterval2', iv);
  setTxt('sumAutoExec', ae, auto?'on':'off');
  setTxt('sumAutoExec2', ae, auto?'v on':'v off');
  setTxt('sumProbeAction', act);
  setTxt('sumProbeAction2', act);
  setTxt('sumOnSuccess', suc);
  setTxt('sumMode', mode);
  const one=$('cfgOneLine');
  if(one) one.textContent=pe+' · 间隔 '+iv+' · '+ae+' · 失败'+act+' · 成功'+suc+' · '+mode;
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
  if($('f_auto_using_api')) $('f_auto_using_api').value=s.auto_using_api||'on_403';
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
    auto_using_api: ($('f_auto_using_api')&&$('f_auto_using_api').value)||'on_403',
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
    ['auto_using_api', String(draft.auto_using_api||'on_403'), String(got.auto_using_api||'on_403')],
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
    setMessage('已更新 · '+new Date().toLocaleTimeString('zh-CN',{hour12:false}));
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
  if(c.using_api!==true && !c.disabled) more.push(['using_api','API 模式']);
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
  const p=primaryStatus(c);
  const tags=['<span class="badge '+p.cls+'" title="主状态">'+esc(p.label)+'</span>'];
  // At most 2 secondary pills — no 已禁用+禁用+隔离 triple stack
  if(c.disabled&&c.banned) tags.push('<span class="pill dim" title="CPA 已关，且插件仍在隔离账本">兼隔离</span>');
  if(c.using_api===true) tags.push('<span class="pill dim" title="使用 API 模式">API</span>');
  if(!c.banned && c.soft_403_streak>0){
    tags.push('<span class="pill dim" title="软 403 连击，满额才隔离">'+c.soft_403_streak+'/'+(c.soft_403_need||3)+'</span>');
  } else if(!c.banned && c.token_expired){
    tags.push('<span class="pill dim">过期</span>');
  } else if(!c.banned && c.needs_refresh && p.label!=='401'){
    tags.push('<span class="pill dim">待刷新</span>');
  }
  // One subtitle line only
  const sub=[];
  const why=shortWhy(c);
  if(why) sub.push(esc(why));
  if(c.banned&&c.remaining_seconds!=null&&c.remaining_seconds>=0){
    sub.push('<span class="remain">'+esc(formatRemaining(c.remaining_seconds))+'</span>');
  }
  if(c.last_probe_at){
    if(c.last_probe_ok===true && !c.banned && !c.disabled) sub.push('探测OK');
    else if(c.last_probe_ok===false && c.last_probe_status){
      // Skip if same as primary code already shown
      if(!(c.banned && Number(c.status_code)===Number(c.last_probe_status)))
        sub.push('探测'+c.last_probe_status);
    }
  }
  return '<div class="mid"><div class="mid-top">'+tags.join('')+'</div>'+
    (sub.length?'<div class="mid-sub">'+sub.join('<span class="sep">·</span>')+'</div>':'')+
    '</div>';
}
function render(){
  const list=filtered();
  const filterLabel={all:'全部',healthy:'健康',banned:'隔离',disabled:'已禁用','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
  const p=state.page||{};
  $('resultCount').textContent=(p.total!=null?p.total:list.length)+' 条'+(state.filter&&state.filter!=='all'?(' · '+filterLabel):'')+(p.pages>1?(' · '+ (p.page||1)+'/'+p.pages):'');
  const lh=$('listHint');
  if(lh){
    if(state.filter==='banned') lh.textContent='筛选：隔离账本';
    else if(state.filter==='disabled') lh.textContent='筛选：已禁用';
    else if(['401','402','403','429'].includes(state.filter)) lh.textContent='筛选：'+filterLabel+'（状态码口径，可与隔离账本不同）';
    else lh.textContent='点击上方卡片筛选 · 勾选后复检或批量操作';
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
  empty.textContent=state.filter==='all'&&!state.query?'暂无 xAI 凭证':'没有匹配的凭证 · 可清除筛选';
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
  const labels={unban:'释放',ban:'隔离',disable:'禁用',reenable:'启用',reauth:'重授权',using_api:'API 模式'};
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
  const labels={unban:'释放',ban:'隔离',disable:'禁用',reenable:'启用',reauth:'重授权',delete:'删除',using_api:'API 模式'};
  const danger=act==='delete'?'\n\n删除将调用 Management 删除凭证；失败则按删除回退策略禁用/隔离。不可轻易撤销。':(act==='using_api'?'\n\n将开启 CPA「使用 API 模式」(using_api=true)，并清除隔离记录。':'');
  if(!confirm('确认对所选 '+ids.length+' 条执行「'+(labels[act]||act)+'」？'+danger)) return;
  if(act==='delete' && !confirm('再次确认：删除所选 '+ids.length+' 条凭证？')) return;
  try{
    setBusy(true,'批量'+(labels[act]||act));
    setProgress(0, ids.length, '批量'+(labels[act]||act));
    let i=0, okN=0, failN=0;
    const fails=[];
    for(const id of ids){
      setMessage('正在'+(labels[act]||act)+' '+ (i+1)+'/'+ids.length+' …');
      try{
        if(act==='unban') await apiMgmt('POST','/unban',{auth_id:id});
        else if(act==='reauth') await apiMgmt('POST','/reauth',{auth_id:id,force:true});
        else await apiMgmt('POST','/apply-action',{auth_id:id,action:act,force:true});
        okN++;
        state.selected.delete(id);
      }catch(one){
        failN++;
        if(fails.length<8) fails.push((id||'')+': '+(one.message||one));
      }
      i++;
      setProgress(i, ids.length, '批量'+(labels[act]||act));
    }
    const msg='批量'+(labels[act]||act)+'完成 · 成功 '+okN+' / 共 '+ids.length+(failN?(' · 失败 '+failN):'');
    const detail=msg+(fails.length?('\n'+fails.join('\n')):'');
    setMessage(msg, failN>0);
    finishProgress(ids.length, ids.length, '批量完成');
    setOpResult(detail, failN>0?(okN>0?'warn':'err'):'ok');
    await loadData(true);
  }catch(e){ setMessage(e.message,true); setOpResult(e.message,'err'); }
  finally{ setBusy(false); }
}
async function selectCurrentFilter(){
  if(state.busy) return;
  const fl={all:'全部',healthy:'健康',banned:'隔离',disabled:'已禁用','401':'401','402':'402','403':'403','429':'429'}[state.filter]||state.filter;
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
  if(!confirm('复检所选 '+ids.length+' 条？\n成功将释放隔离并启用；失败按策略处理。')) return;
  const chunkSize=5;
  try{
    setBusy(true,'复检所选');
    setProgress(0, ids.length, '复检所选');
    let done=0, checked=0, okN=0, failed=0, unbanned=0, reenabled=0, skipped=0;
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
    const msg='复检完成 · 检 '+checked+' · 成功 '+okN+' · 失败 '+failed+' · 释放 '+unbanned+' · 启用 '+reenabled+' · 跳过 '+skipped;
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
if($('clearFilterBtn')) $('clearFilterBtn').onclick=()=>{state.filter='all'; state.query=''; state.page.page=1; if($('search')) $('search').value=''; paintChips(); loadData(true); setMessage('已清除筛选');};
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
  if(btn.id==='clearFilterBtn') return;
  btn.onclick=()=>setFilter(btn.dataset.filter,false);
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
</script>
</body>
</html>`
}
