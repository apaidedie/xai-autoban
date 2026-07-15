# xai-autoban 中等拆包设计

**Date:** 2026-07-14  
**Version in scope:** 0.5.8（行为与版本号不变）  
**Status:** Approved (user: ok)  
**Approach:** 方案 1 — 按领域细拆 `internal/*`

## 1. Goals and non-goals

### Goals

- Root keeps only plugin entry: `main` wiring + CGO ABI + module metadata.
- Business logic moves into `internal/*` with ~1:1 mapping from current files.
- External behavior unchanged:
  - YAML config fields and defaults
  - Management API paths and resource routes
  - Ban / disable / delete / reenable / probe / scheduler semantics
  - Plugin version string `0.5.8`
- Success criterion: `go test ./...` green after migration.

### Non-goals

- No CPA ABI / SDK semantic upgrades beyond existing vendored `cpasdk`.
- No ops UI interaction rewrite (layout/JS behavior stays equivalent).
- No new features, no default config changes.
- No forced git commit of product code unless user asks.

## 2. Target layout

```
xai-autoban/
├── main.go                 # App wiring + handleMethod
├── abi_cgo.go              # CGO exports (package main)
├── main_test.go            # optional end-to-end / cross-package tests
├── go.mod / go.sum
├── registry.json / LICENSE / README.md
├── .gitignore
├── scripts/
│   └── build.sh
├── docs/
│   └── superpowers/specs/  # this design
├── .github/workflows/release.yml
├── cpasdk/
│   ├── pluginabi/
│   └── pluginapi/
└── internal/
    ├── config/
    ├── host/
    ├── ban/
    ├── action/
    ├── audit/
    ├── persist/
    ├── probe/              # probe + recheck
    ├── schedule/
    ├── usage/
    ├── mgmt/               # routes + direct management client
    ├── creds/
    └── ui/                 # status HTML (optional go:embed)
```

## 3. Dependency rules (no cycles)

```
main / abi_cgo
  → config, host, ban, action, audit, persist, probe, schedule, usage, mgmt, creds, ui
  → cpasdk/*

Layers:
  leaf:     ban, audit, config, host
  mid:      action, persist
  upper:    probe, schedule, usage, creds, ui, mgmt
```

| Package   | May import                                      | Must not import        |
|-----------|--------------------------------------------------|------------------------|
| config    | cpasdk, stdlib                                   | other internal         |
| host      | cpasdk, stdlib                                   | other internal         |
| ban       | stdlib                                           | other internal         |
| audit     | stdlib                                           | other internal         |
| persist   | ban, stdlib                                      | action/probe/mgmt      |
| action    | ban, host, audit, config (or narrow interfaces)  | mgmt/probe/schedule    |
| probe     | action, ban?, host, config                       | mgmt                   |
| schedule  | ban, config?, cpasdk                             | action/probe/mgmt      |
| usage     | action, config, cpasdk                           | mgmt                   |
| creds     | ban, probe result types, cpasdk/host types       | mgmt                   |
| ui        | prefer zero internal deps (string/embed only)    | business packages      |
| mgmt      | action, ban, probe, persist, audit, config, creds, ui, host | main            |

`mgmt` must not depend on `main`.  
`action` talks to CPA Management API through its own client type (today: `managementDisabler`); that client moves with disable logic into `action` or a small `internal/mgmtclient` if needed to avoid `action` ↔ `mgmt` cycle. **Decision:** keep HTTP disable client next to disable logic:

- Prefer `internal/action` owning disable application + optional subfile `management_client.go` moved into `action`, **or**
- `internal/mgmtclient` used by both `action` and `mgmt` routes.

**Chosen:** `internal/mgmtclient` only if shared; otherwise move `management_client.go` into `action` (production disable path) and keep route handlers in `mgmt` calling `action.Engine`. Route-only code in `mgmt` must not reimplement disable HTTP.

Final choice for implementation:

1. `internal/action` — `Engine` + disable helpers that need host + management HTTP.
2. Move `management_client.go` → `internal/action/management_client.go` (or `internal/cpahttp` if name clearer).
3. `internal/mgmt` — HTTP route dispatch, settings, backup, status assembly; calls engine/probe/bans.

## 4. File migration map

| Current file              | Target                                      |
|---------------------------|---------------------------------------------|
| `config.go`               | `internal/config/`                          |
| `host.go`                 | `internal/host/`                            |
| `ban_state.go`            | `internal/ban/`                             |
| `actions.go`              | `internal/action/`                          |
| `management_client.go`    | `internal/action/` (disable client)         |
| `audit.go`                | `internal/audit/`                           |
| `persist.go`              | `internal/persist/`                         |
| `probe.go` + `recheck.go` | `internal/probe/`                           |
| `scheduler.go`            | `internal/schedule/`                        |
| `usage.go`                | `internal/usage/`                           |
| `management.go`           | `internal/mgmt/`                            |
| `credentials.go`          | `internal/creds/`                           |
| `ui_status.go`            | `internal/ui/` (+ optional `status.html`)   |
| `abi_cgo.go`, `main.go`   | repo root                                   |
| `build.sh`                | `scripts/build.sh`                          |
| `main_test.go`            | root e2e and/or split per package           |
| `credentials_test.go`     | `internal/creds/`                           |

Exported names: clear package-qualified types (`ban.State`, `action.Engine`, `config.PluginConfig`, etc.). Constants preserve string values (`"ban"`, `"disable"`, …).

## 5. Runtime wiring

Replace scattered package-level mutables with an `App` owned by `main`:

```go
type App struct {
    cfg     *config.Store      // RWMutex + PluginConfig
    bans    *ban.State
    audit   *audit.Log
    host    host.Client
    engine  *action.Engine
    probe   *probe.Service
    persist *persist.Persister
}

func NewApp(h host.Client) *App
func (a *App) HandleMethod(method string, req []byte) ([]byte, error)
func (a *App) SetConfig(cfg config.PluginConfig)
func (a *App) Shutdown()
```

- Process singleton `defaultApp` for CGO entrypoints.
- Lifecycle:
  - `plugin.register` / `plugin.reconfigure`: parse config → `SetConfig` → `persist.Load` → start/stop probe
  - `plugin.shutdown` / CGO shutdown: `probe.Stop` + `persist.Flush`
- Version constant: single place in `main` (or `config`) `const pluginVersion = "0.5.8"`.

Envelope helpers (`okEnvelope` / `errorEnvelope`) stay in `main` or a tiny `internal/rpc` if shared; prefer `main` unless tests need them elsewhere.

## 6. Behavior preservation checklist

| Area | Must keep |
|------|-----------|
| Usage hook | Only `provider=xai` + `Failed`; classify 401/402/403/429 |
| 429 reset | Retry-After, x-ratelimit-reset headers, then fallback seconds |
| Scheduler | Filter banned xAI candidates; pick RR / fill-first on remainder; `Handled=false` if none filtered or none left |
| Ban storage key | Prefer email; alias index for auth id / `.json` basename |
| Disable path | Prefer Management API when key present; no `AuthSave` after successful management disable; note via fields PATCH |
| Management HTTP | Direct no-proxy transport to localhost; auth failure cooldown; proxy 403 not auth-cooldown |
| Probe | Skip disabled; concurrency + QPS; `auto_execute` report-only vs apply |
| Delete | Best-effort fallback to disable/ban + `pending_delete` |
| Audit | Truncate auth id; ring buffer max |
| Persist | Debounced write; atomic rename; corrupt → `.bad` |
| Public settings | No plaintext management key; only `management_key_configured` |

## 7. Scripts, CI, docs touch-ups

- Move `build.sh` → `scripts/build.sh`; keep behavior (`go test` + c-shared build).
- Update `.github/workflows/release.yml` if it references `./build.sh`.
- README build paths: point to `scripts/build.sh`.
- `.gitignore`: keep `dist/`, `*.so|dll|dylib|h`; add common editor/OS noise if missing.

## 8. Testing strategy

1. After each migration wave: `go test ./...`.
2. Unit tests live next to packages (`ban`, `config`, `action` with stub host, `creds`).
3. Existing `main_test.go` scenarios: reconstruct via `NewApp(stubHost)` or keep root integration tests importing internal packages.
4. No new product behavior assertions; preserve existing expectations.

## 9. Implementation order

1. Create `internal/*` packages; migrate leaves: `config` → `host` → `ban` → `audit` → `persist`.
2. Migrate `action` (+ management client) → `probe` → `schedule` → `usage`.
3. Migrate `creds` → `ui` → `mgmt`.
4. Rewrite `main` / `abi_cgo` wiring to `App`.
5. Move tests; update `scripts/build.sh`, workflow, README, `.gitignore`.
6. Final `go test ./...`; fix compile errors and any accidental import cycles.

## 10. Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Import cycles (action ↔ mgmt) | Disable HTTP client lives with `action`; `mgmt` only routes |
| Global state in tests | `NewApp` + inject stub host; avoid init-time side effects |
| CGO build breaks | Keep `abi_cgo.go` in `package main`; thin wrappers only |
| Behavior drift | Checklist §6; run full existing tests |
| Large `ui_status.go` | Move first as-is; optional embed split if compile OK |

## 11. Approval

- Approach B / package split option 1: approved by user.
- This design document: written after user confirmation (`ok`).
