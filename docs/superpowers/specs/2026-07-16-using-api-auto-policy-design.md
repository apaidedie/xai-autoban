# using_api Auto Policy + Write Verification Design

**Date:** 2026-07-16  
**Status:** Approved / implemented  
**Target version:** 0.5.47  
**Scope:** P0 only

## Goal

Make automatic “使用 API 模式” (`using_api`) **controllable, safe, and verifiable**:

1. Default: auto-enable only on OAuth **403** during probe/recheck.
2. Config: `auto_using_api` = `off` | `on_403` | `on_fail`.
3. Per probe/recheck **run**: each auth key auto-tried at most **once**.
4. Manual ops action `using_api` always available (not gated by auto policy).
5. `SetUsingAPI` **write-then-read** verification; failure surfaces as error + audit.

## Non-goals

- Lifetime / persistent “already tried” state across restarts
- Real usage-path auto flip of `using_api` (only probe + recheck-selected)
- Ops UI list columns for `using_api` / last probe (P1)
- Bulk server-side API, WebSockets/headers editing
- Changing soft-403 streak or usage success-unban behavior

## Decisions (confirmed)

| Topic | Choice |
|-------|--------|
| Default auto policy | `on_403` only |
| Scope this round | P0 only |
| Attempt limit | Once per auth key **per probe/recheck run** (in-memory for that job) |
| Approach | Config gate + shared heal helper + SetUsingAPI verify |

## Current baseline (pre-change)

- `ApplyAction("using_api")` / ops bulk “API 模式所选” → `SetUsingAPI(true)` + clear ban.
- `SetUsingAPI`: Management `PATCH auth-files/fields` first; host `auth.save` fallback; **no read-back verify**.
- Probe + recheck-selected: on **401/402/403**, always call `tryEnableUsingAPIAndReprobe` (hard-coded).

## Design

### 1. Config

```yaml
# off     — never auto-enable using_api
# on_403  — probe/recheck auto only when status == 403 (default)
# on_fail — auto on 401, 402, or 403
auto_using_api: on_403
```

| Field | Type | Default | Normalize |
|-------|------|---------|-----------|
| `auto_using_api` | string | `on_403` | empty/invalid → `on_403`; accept aliases: `false`/`0`→`off`, `true`/`1`/`403`→`on_403`, `all`/`fail`→`on_fail` |

Wire into:

- `PluginConfig` + `Default()` + `Normalize` / `MergePatch`
- `PublicView` / `OpsSettingsView` so ops console can read/write with other settings
- Ops config UI: one select or short text field under probe-related settings (minimal; no redesign)

### 2. Auto gate (single function)

```text
shouldAutoUsingAPI(cfg, status, mat, alreadyTriedThisRun) bool
```

Rules (all must pass):

1. `cfg.AutoUsingAPI != off`
2. `!alreadyTriedThisRun` for this auth key in current job
3. Material is OAuth/web (not `api_key`; not already `using_api == true`)
4. Status match:
   - `on_403`: `status == 403`
   - `on_fail`: `status ∈ {401,402,403}`

Call sites:

- `internal/probe/probe.go` (scheduled/manual full probe failure path)
- `internal/probe/recheck.go` (recheck-selected failure path)

**Not** called from usage failure handlers.

### 3. Per-run attempt map

Inside each probe/recheck job body (same lifetime as the job result):

```go
triedUsingAPI := map[string]struct{}{}
// before heal:
if _, ok := triedUsingAPI[key]; ok { skip auto }
// after deciding to attempt:
triedUsingAPI[key] = struct{}{}
```

- Key = same isolation key used for probe (`xai.AuthKey` / email preference already in path).
- Concurrent workers: protect with the job’s existing mutex or a dedicated `sync.Mutex` for the map.
- No persist to `xai-autoban-state.json`.

### 4. `tryEnableUsingAPIAndReprobe` behavior

After gate passes:

1. Mark key tried for this run (even if SetUsingAPI later fails — avoid hammering).
2. `engine.SetUsingAPI(key, true)`.
3. If SetUsingAPI fails: audit `using_api`/`error`; return `healed=false` so caller keeps **original** status/body/error (no outcome swap).
4. If SetUsingAPI succeeds: re-AuthGet + re-ProbeOneWithJSON; return `healed=true` with re-probe status/body/error.
5. Caller on re-probe success uses existing success path (`probe_on_success` etc.).
6. Caller on re-probe failure classifies using **new** status/body (not the pre-heal 403).

### 5. Write verification in `SetUsingAPI`

After successful management patch **or** host save:

1. `host.AuthGet(index)` (or re-list + get).
2. Parse JSON: accept `using_api` bool **or** string `"true"`/`"false"` (same as probe material parser).
3. If value != wanted → return error:  
   `using_api write not reflected (want=%v got=%v)`.
4. Management success but verify fail: do **not** silently accept; try host save once more only if management path was used and host save not yet tried; if still fail, return error.
5. Manual and auto both use this path (one implementation).

**Note:** Auth list lag is possible; verify uses AuthGet on the same index just written, not list.Disabled.

### 6. Manual path unchanged

| Path | Behavior |
|------|----------|
| `POST apply-action` `using_api` / `enable_api` / `api_mode` | Always `SetUsingAPI(true)` + clear bans; ignore `auto_using_api` |
| Ops bulk “API 模式所选” | Same |
| Disable auto | Users set `auto_using_api: off` |

### 7. Audit / logging

| Event | Source | Action | Result |
|-------|--------|--------|--------|
| Auto gate skip (policy off / not 403 / already tried) | probe/recheck | (optional debug log only; no audit spam) | — |
| Auto SetUsingAPI fail | probe/recheck | `using_api` | `error` |
| Auto heal re-probe OK | probe/recheck | `using_api` | `ok` msg `auto on_403 heal` |
| Manual using_api | manual | `using_api` | existing |
| Verify fail | any | `using_api` | `error` with verify message |

### 8. Testing

| Test | Expect |
|------|--------|
| Gate `off` + 403 | no SetUsingAPI |
| Gate `on_403` + 401 | no SetUsingAPI |
| Gate `on_403` + 403 | SetUsingAPI once + re-probe |
| Gate `on_fail` + 401 | SetUsingAPI attempted |
| Same key second 403 in same run | no second SetUsingAPI |
| SetUsingAPI host save sets field | AuthGet reflects true |
| SetUsingAPI verify fail | returns error; ApplyAction surfaces error |
| Manual using_api with `auto_using_api=off` | still works |

Prefer unit tests in `internal/probe` (gate + heal with stub host/engine) and `main_test` or action-level for verify.

### 9. Docs / version

- `CHANGELOG` 0.5.47, `main.go` / `registry.json` / `README` version bump
- README: one row for `auto_using_api` under probe config
- No registry schema_version change

## File touch list

| File | Change |
|------|--------|
| `internal/config/config.go` | field, default, normalize, views, merge |
| `internal/probe/probe.go` | gate + per-run map; tighten status filter |
| `internal/probe/recheck.go` | same gate + map |
| `internal/action/engine.go` | verify after SetUsingAPI |
| `internal/ui/status.go` | minimal ops config control for `auto_using_api` |
| `*_test.go` | gate + verify tests |
| `CHANGELOG.md`, `main.go`, `registry.json`, `README.md` | 0.5.47 |

## Success criteria

1. Fresh install default: only 403 triggers auto using_api on probe/recheck.
2. `auto_using_api: off` never auto-writes credentials.
3. One run cannot auto-flip the same key twice.
4. Failed write or non-reflected write does not claim success.
5. Manual API 模式 still works with auto off.
6. `go test ./...` green.

## Out of scope follow-ups (P1+)

- List columns: using_api, last probe, streak, lastOK grace
- Lifetime try ledger in state file
- Usage-path auto using_api
- Server-side bulk-action API
