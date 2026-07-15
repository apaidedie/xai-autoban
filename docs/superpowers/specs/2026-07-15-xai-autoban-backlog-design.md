# xai-autoban Backlog Hardening Design

**Date:** 2026-07-15  
**Status:** Approved  
**Version stays:** 0.5.8 (behavior upgrades only; no store version bump required unless release process needs it)

## Goal

Finish the post-classifier hardening backlog in priority order:

1. Probe reliability (429 short retry + dual endpoint) — **mostly implemented; must compile and be tested**
2. True Management API delete (with safe fallback)
3. Async probe job + real progress (`done/total`)
4. `probe_include_disabled` / `probe_only_disabled`
5. Ops UI shows `classification`
6. `build.ps1` + README Docker install section
7. Quota path explicitly respects `action_on_402`

## Non-goals

- Full UI redesign
- Removing Usage/Scheduler isolation
- Hard-coding only `cli-chat-proxy.grok.com`
- Changing CPA host ABI

## Current state (baseline)

| Area | State |
|------|--------|
| Package split | Done (`internal/*`) |
| Semantic classifier | Done (`internal/classify`) |
| Usage body classify | Done |
| Probe body classify | Done (wiring) |
| Probe 429 retry + dual endpoint | Code present in `ProbeOne`, but **build broken** (`classify` used without import) |
| Delete | Fake: disable/ban + `pending_delete` only |
| Async probe | Sync `POST /probe`; UI fakes progress 30→100 |
| Disabled in scheduled probe | Always skipped |
| Classification in UI | Field on ban entry; not on creds/UI |
| Windows build script | Missing (`scripts/build.sh` only; also has a broken ROOT line) |
| action_on_402 | Duration remaps to 402; action uses `ActionForStatus(402)` but should be explicit and tested |

## Design

### 1. Probe enhancement (high)

**Behavior (keep / lock in):**

- Return `(status, body, error)` from `ProbeOne`.
- Bare HTTP 429 (not free-usage exhaustion): one short retry (~350ms), then return second result.
- Free-usage exhaustion: **no** 429 retry; classify as `quota_exhausted`.
- Modes:
  - `models` (default): GET `/models`; on success return OK; on 401/402/403/429 optionally dual-check via POST `/responses` for a better body.
  - `responses_mini`: POST `/responses` with model ping; on transport error or 401/402/403/429 fallback POST `/chat/completions`.
- Model id: prefer `grok-4.5` / list pick from `/models` when available.

**Fix required now:**

- Import `xai-autoban/internal/classify` in `internal/probe/probe.go`.
- Add unit tests with host HTTP stubs covering: bare 429 retry, free-usage no-retry, responses→completions fallback, models success short-circuit.

### 2. True delete (high)

**API:** Management `DELETE /v0/management/auth-files` (or name-based variant used by CPA). Implementation must:

1. Resolve management key the same way as disable (`management_key` / env).
2. Resolve auth file name candidates (id, id.json, listed name) like note/status patch.
3. Call DELETE with direct no-proxy HTTP (same transport as disable).
4. On success: clear ban ledger for that id (or mark deleted + clear), audit `delete/ok`, drop `pending_delete`.
5. On failure (404/401/missing key/network): keep existing fallback path (`delete_fallback` = disable|ban), set `pending_delete=true`, audit `delete/fallback`.

**Config:** no new required fields. Reuse management URL/key. Optional later: `delete_via: management_api|fallback_only` — **out of scope**; always try management first when key present.

**Batch:** `apply-action` and probe auto path already call `ApplyAction(..., delete, ...)`. Engine single-delete is enough; bulk UI can loop existing apply-action.

**Tests:**

- Stub management HTTP: DELETE 200 → no disable save, ledger cleared or not pending.
- DELETE 404 / no key → fallback disable + pending_delete (existing test updated).

### 3. Async probe + progress (medium)

**Model:**

```text
POST /probe  → { ok, job_id, accepted: true }   // starts background run if idle
GET  /probe/status → { running, job_id, done, total, result?, error? }
```

- Only one probe job at a time; second POST returns 409 or `{ok:false, error:"probe already running"}` with current progress.
- `RunOnceTrigger` updates atomic/mutex progress: `done`, `total` after each credential finishes.
- Scheduled loop uses same progress fields.
- UI `runProbe()`: POST, then poll `/probe/status` every 400–800ms until `running=false`, drive real progress bar.

**Compatibility:** Keep response shape extensible. If clients expect sync `result`, support query `?wait=1` or body `{wait:true}` for old sync behavior (default async for UI).

**Decision (approved):**  
- Default `POST /probe` = **async accept** when called from management UI path.  
- Body `{ "wait": true }` keeps **sync** for scripts/tests.

### 4. probe_include_disabled / probe_only_disabled (medium)

**Config fields:**

| Field | Default | Meaning |
|-------|---------|---------|
| `probe_include_disabled` | `false` | Scheduled/manual full probe also probes disabled creds |
| `probe_only_disabled` | `false` | If true, only disabled creds (implies include) |

Selection logic in `RunOnceTrigger`:

```text
if only_disabled → target disabled xAI only
else if include_disabled → all xAI
else → enabled xAI only (current)
```

`recheck-selected` already includes disabled; unchanged.

### 5. UI classification (medium)

- Add `Classification string` to `creds.Info` from `ban.Entry.Classification`.
- `/data` and bans list expose it.
- Ops table: show classification pill next to reason (or replace raw reason when classification set).
- Labels (CN): `rate_limited`→限流, `quota_exhausted`→额度用尽, `reauth`→需重新授权, `permission_denied`→权限拒绝, `model_unavailable`→模型不可用, etc.

### 6. build.ps1 + README Docker (low)

- Add `scripts/build.ps1` (Windows): `go test ./...`, `CGO_ENABLED=1 go build -buildmode=c-shared`, output `dist/xai-autoban.dll`.
- Fix `scripts/build.sh` broken `ROOT=` line.
- README: short Docker section — mount plugin `.so`/`.dll`, env `CPA_MANAGEMENT_KEY`, enable plugin in `config.yaml`, open status URL.

### 7. action_on_402 tighten (low)

In `ClassifyFailureWithBody` for `quota_exhausted`:

1. Duration always uses 402 window (`Ban402Seconds`).
2. Action = `cfg.ActionOn402` (via `ActionForStatus(402)`), **not** hard-coded disable.
3. Bare `rate_limited` still forces `ban` regardless of `action_on_429` for auto-isolation safety (document this).
4. Unit/integration test: free-usage body + `action_on_402=disable` → disable; `=ban` → ban.

## Error handling

- Probe transport errors: no isolate (`ClassifyFailureWithBody` returns ok=false).
- Delete partial failure: fallback, never leave credential fully active without ledger if auto_execute intended isolation.
- Async job panic: recover, set job error, running=false.

## Testing strategy

- `go test ./... -count=1` green.
- New probe tests with stub host HTTP.
- New/updated delete management tests.
- Config normalize tests for new probe flags.
- Optional main_test for async probe accept + progress fields.

## Rollout

1. Fix compile + probe tests  
2. True delete  
3. Async progress  
4. Disabled flags  
5. UI classification  
6. Scripts/docs  
7. 402 action tests  

Each step independently shippable.
