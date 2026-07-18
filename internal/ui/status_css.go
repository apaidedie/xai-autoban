package ui

const statusCSS = `
:root{color-scheme:dark;--bg:#070b14;--panel:#101a2c;--line:rgba(148,163,184,.16);--text:#f8fafc;--muted:#93a4c3;--cyan:#22d3ee;--blue:#3b82f6;--green:#34d399;--amber:#fbbf24;--red:#fb7185;--violet:#a78bfa;--mono:ui-monospace,Consolas,monospace;--sans:Inter,ui-sans-serif,system-ui,"Segoe UI",sans-serif}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:var(--sans);color:var(--text);background:radial-gradient(1000px 500px at 10% -10%,rgba(34,211,238,.1),transparent 50%),radial-gradient(800px 400px at 100% 0,rgba(59,130,246,.1),transparent 45%),linear-gradient(180deg,#070b14,#0a101c);font-size:14px}
.shell{max-width:1440px;margin:0 auto;padding:16px 20px 32px}
.top{display:flex;justify-content:space-between;align-items:flex-start;gap:16px;margin-bottom:14px}
.top-brand{min-width:0}
.top-actions{display:flex;gap:8px;align-items:center;flex-wrap:wrap;flex-shrink:0}
.kicker{display:inline-flex;align-items:center;gap:8px;color:var(--cyan);font-size:11px;font-weight:800;letter-spacing:.1em;text-transform:none}
.kicker i{width:7px;height:7px;border-radius:50%;background:var(--cyan);box-shadow:0 0 0 4px rgba(34,211,238,.15)}
h1{margin:6px 0 0;font-size:24px;font-weight:800;letter-spacing:-.03em;line-height:1.15}
.sub{margin:6px 0 0;color:var(--muted);font-size:12.5px;line-height:1.4}
.sub .ver{font-family:var(--mono);font-size:11.5px;opacity:.9}
.live{padding:7px 12px;border-radius:999px;border:1px solid var(--line);background:rgba(15,23,42,.75);color:var(--green);font-size:12px;font-weight:800;white-space:nowrap}
.banner{padding:11px 14px;border-radius:12px;margin-bottom:12px;border:1px solid rgba(52,211,153,.3);background:rgba(6,78,59,.35);color:#bbf7d0;font-weight:700}
.banner.warn{border-color:rgba(251,191,36,.35);background:rgba(120,53,15,.35);color:#fde68a}
.panel{background:linear-gradient(180deg,rgba(18,28,46,.96),rgba(12,20,34,.98));border:1px solid var(--line);border-radius:14px;margin-bottom:12px;overflow:hidden;box-shadow:0 12px 32px rgba(0,0,0,.28)}
.panel-list{margin-top:2px}
.panel-hist{margin-top:0}
.phd{display:flex;justify-content:space-between;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid var(--line)}
.phd h2{margin:0;font-size:13px;font-weight:800;letter-spacing:.04em;color:#e2e8f0}
.phd .count{font-variant-numeric:tabular-nums;font-weight:750;color:#cbd5e1}
.hint{color:var(--muted);font-size:12px;line-height:1.4}
.drawer-hint{margin:0 0 10px;line-height:1.45}
.cfg-grid{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:8px;padding:12px 14px}
.cfg-card{background:rgba(7,12,22,.55);border:1px solid var(--line);border-radius:10px;padding:10px 12px;min-height:52px}
.cfg-card.accent{border-color:rgba(59,130,246,.4);box-shadow:0 0 0 1px rgba(59,130,246,.1) inset}
.cfg-card .l{color:var(--muted);font-size:11px;font-weight:750;letter-spacing:.02em}
.cfg-card .v{margin-top:7px;font-size:14px;font-weight:800;color:#f8fafc;line-height:1.2}
.cfg-card .v.on{color:var(--green)}.cfg-card .v.off{color:var(--amber)}
.cfg-path{padding:0 14px 12px;font-size:11px;color:var(--muted);font-family:var(--mono);word-break:break-all;line-height:1.4}
@media(max-width:1100px){.cfg-grid{grid-template-columns:repeat(3,minmax(0,1fr))}}
@media(max-width:700px){.cfg-grid{grid-template-columns:1fr 1fr}}
.metrics-block{margin:0 0 12px}
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
.foot{display:flex;flex-direction:column;gap:4px;color:var(--muted);font-size:12px;line-height:1.55;padding:4px 2px 8px}
.foot-sub{opacity:.85;font-size:11.5px}
.legend{margin:10px 0 0;border:1px solid var(--line);border-radius:12px;background:rgba(8,14,26,.55);overflow:hidden}
.legend>summary{list-style:none;cursor:pointer;display:flex;justify-content:space-between;align-items:center;gap:10px;padding:9px 12px;user-select:none;font-size:12px;font-weight:750;color:#cbd5e1}
.legend>summary::-webkit-details-marker{display:none}
.legend>summary .chev{color:var(--muted);font-weight:700;font-size:11px}
.legend[open]>summary{border-bottom:1px solid var(--line);color:#e2e8f0}
.legend-body{padding:10px 12px 12px;display:grid;gap:6px;font-size:12px;line-height:1.45;color:var(--muted)}
.legend-body b{color:#e2e8f0}
.legend-body .row2{display:grid;grid-template-columns:92px 1fr;gap:5px 12px;align-items:start}
.legend-body .k{color:var(--cyan);font-weight:800;white-space:nowrap;font-size:11.5px}
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
.qcards{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:10px;margin:0 0 10px}
.code-strip{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin:0}
@media(max-width:900px){.code-strip{grid-template-columns:1fr 1fr}}
@media(max-width:700px){.code-strip{grid-template-columns:1fr 1fr}}
.code-chip{
  min-height:76px;padding:12px 14px;border-radius:14px;border:1px solid var(--line);
  background:rgba(12,20,34,.92);color:var(--muted);font-weight:750;
  display:flex;flex-direction:column;align-items:flex-start;justify-content:center;gap:8px;
  cursor:pointer;text-align:left;height:auto;overflow:hidden
}
.code-chip .cl{
  font-size:11px;font-weight:800;color:var(--muted);letter-spacing:.04em;
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:100%;line-height:1.2
}
.code-chip b{font-size:22px;font-weight:850;font-variant-numeric:tabular-nums;color:var(--text);line-height:1}
.code-chip:hover{border-color:rgba(148,163,184,.35);background:rgba(15,23,42,.95);color:var(--text)}
.code-chip.active,.code-chip.on{border-color:rgba(34,211,238,.5);background:rgba(8,47,73,.35);box-shadow:0 0 0 1px rgba(34,211,238,.18) inset}
.code-chip.s401 b{color:#93c5fd}.code-chip.s401.active{border-color:rgba(59,130,246,.55)}
.code-chip.s402 b{color:#fcd34d}.code-chip.s402.active{border-color:rgba(251,191,36,.55)}
.code-chip.s403 b{color:#fda4af}.code-chip.s403.active{border-color:rgba(251,113,133,.55)}
.code-chip.s429 b{color:#ddd6fe}.code-chip.s429.active{border-color:rgba(167,139,250,.55)}
.code-chip.ghost{align-items:center;justify-content:center;color:var(--muted)}
.code-chip.ghost b{display:none}
.code-chip.ghost .cl{font-size:13px;letter-spacing:0;font-weight:750;white-space:nowrap}
.qcard{
  text-align:left;height:auto;min-height:80px;padding:12px 14px;border-radius:14px;
  border:1px solid var(--line);background:rgba(12,20,34,.92);
  box-shadow:none;transition:border-color .12s ease,background .12s ease
}
.qcard:hover{border-color:rgba(34,211,238,.35);background:rgba(15,23,42,.95)}
.qcard:focus-visible{outline:2px solid rgba(34,211,238,.65);outline-offset:2px}
.qcard .ql{color:var(--muted);font-size:11px;font-weight:800;letter-spacing:.04em;white-space:nowrap}
.qcard .qn{margin-top:8px;font-size:24px;font-weight:850;font-variant-numeric:tabular-nums;line-height:1}
.qcard .qs{margin-top:6px;color:var(--muted);font-size:11px;font-weight:650;line-height:1.25;min-height:1.25em;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.qcard.ok .qn{color:var(--green)}.qcard.warn .qn{color:var(--amber)}.qcard.bad .qn{color:var(--red)}.qcard.info .qn{color:var(--cyan)}
.qcard.disabled-card .qn{color:#cbd5e1}
.row-action{height:28px;padding:0 9px;border-radius:8px;font-size:12px}
.row-action.primary{background:linear-gradient(180deg,#3b82f6,#2563eb);border-color:#1d4ed8;color:#fff}
.row-action.primary:hover{background:linear-gradient(180deg,#60a5fa,#3b82f6);border-color:#2563eb;color:#fff}
@media(max-width:1100px){.qcards{grid-template-columns:repeat(3,minmax(0,1fr))}}
@media(max-width:700px){h1{font-size:22px}.qcards{grid-template-columns:1fr 1fr}}
@media (prefers-reduced-motion:reduce){.qcard{transition:none}}
`
