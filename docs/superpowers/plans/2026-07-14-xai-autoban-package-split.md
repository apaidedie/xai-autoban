# xai-autoban Package Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the flat `package main` CPA plugin into `internal/*` domain packages with an `App` wiring layer, without changing runtime behavior (v0.5.8).

**Architecture:** Leaf packages (`config`, `host`, `ban`, `audit`) have no internal deps. Mid-layer `action` owns disable HTTP client. Upper layers (`probe`, `schedule`, `usage`, `creds`, `ui`, `mgmt`) call down. Root `main` + `abi_cgo.go` hold CGO and `App` singleton. Migration is move+rename+export, not rewrite.

**Tech Stack:** Go 1.21+, CGO shared library, `gopkg.in/yaml.v3`, vendored `cpasdk/pluginabi|pluginapi`.

## Global Constraints

- Plugin version string remains `"0.5.8"`.
- No config field renames, no Management path changes, no default value changes.
- Behavior checklist in `docs/superpowers/specs/2026-07-14-xai-autoban-package-split-design.md` §6 is mandatory.
- `go test ./...` must pass after every task that compiles.
- Do **not** git commit unless the user explicitly asks (repo may not be a git worktree).
- `management_client.go` moves into `internal/action` (not `mgmt`) to avoid `action` ↔ `mgmt` cycles.
- Shared auth-id helpers (`authIDsEqual`, `authIDAliases`) live in `internal/ban` as exported funcs (ban already needs them).
- xAI detection helpers (`IsXAIAuth`, `AuthKey`, `IsXAICandidate`, `CandidateEmail`) live in `internal/xai` (tiny package) so `probe`/`schedule`/`creds` do not cycle.

---

## File structure (target)

| Path | Responsibility |
|------|----------------|
| `main.go` | `App`, envelopes, `handleMethod`, register/shutdown |
| `abi_cgo.go` | CGO export/import host callbacks |
| `internal/config/` | `PluginConfig`, parse/normalize, `ConfigFields`, `Store` |
| `internal/host/` | `Client` interface, `Real`, `Stub` |
| `internal/ban/` | `Entry`, `State`, id aliases/equality |
| `internal/audit/` | `Log`, `Event` |
| `internal/persist/` | `Persister` |
| `internal/action/` | `Engine`, management disabler client |
| `internal/xai/` | provider detection + auth key helpers |
| `internal/probe/` | `Service`, recheck 429/selected |
| `internal/schedule/` | `Pick` handler logic |
| `internal/usage/` | usage failure handler |
| `internal/creds/` | credential list/paging/counts |
| `internal/ui/` | status HTML page |
| `internal/mgmt/` | management routes + status assembly |
| `scripts/build.sh` | local build script |
| `cpasdk/*` | unchanged |

---

### Task 1: Leaf packages — config, host, ban, audit, xai

**Files:**
- Create: `internal/config/config.go` (from root `config.go`)
- Create: `internal/host/host.go` (from root `host.go`)
- Create: `internal/ban/state.go` (from root `ban_state.go`)
- Create: `internal/ban/id.go` (export `AuthIDsEqual`, `AuthIDAliases` from `scheduler.go` helpers)
- Create: `internal/audit/audit.go` (from root `audit.go`)
- Create: `internal/xai/xai.go` (from `isXAIAuth`, `authKey`, `isXAICandidate`, `candidateEmail` in probe/scheduler)
- Delete after main wiring works (Task 7): root `config.go`, `host.go`, `ban_state.go`, `audit.go` — for this task **copy** first; keep root files until Task 7 to avoid a long red build if preferred. **Preferred for agents:** move file content into packages and leave thin root wrappers only if needed; full cutover in Task 7.

**Interfaces:**
- Produces:
  - `config.PluginConfig` (same fields/yaml tags as today)
  - `config.Default() PluginConfig`
  - `config.ParseYAML(raw string) (PluginConfig, []string)`
  - `config.Normalize(cfg PluginConfig) (PluginConfig, []string)`
  - `config.MergePatch(base PluginConfig, patch map[string]any) (PluginConfig, []string)`
  - `config.Fields() []pluginapi.ConfigField`
  - `(PluginConfig).DurationForStatus(status int) time.Duration`
  - `(PluginConfig).ActionForStatus(status int) string`
  - `(PluginConfig).PublicView() map[string]any`
  - `host.Client` interface (methods AuthList/AuthGet/AuthSave/HTTPDo/Log)
  - `host.Real` (uses injectable `host.CallFn`)
  - `host.Stub` for tests
  - `ban.Entry`, `ban.State` with methods: `Set`, `ForceSet`, `Active`, `IsBannedCandidate`, `Clear`, `ClearAll`, `ClearStatus`, `ClearMany`, `Snapshot`, `ReplaceAll`
  - `ban.StorageKey(email, authID string) string`
  - `ban.AuthIDsEqual(a, b string) bool`
  - `ban.AuthIDAliases(id string) []string`
  - `audit.New(max int) *Log`, `(*Log).Add/List/SetMax`
  - `xai.Provider = "xai"`
  - `xai.IsAuth(f pluginapi.HostAuthFileEntry) bool`
  - `xai.AuthKey(f pluginapi.HostAuthFileEntry) string`
  - `xai.IsCandidate(c pluginapi.SchedulerAuthCandidate) bool`
  - `xai.CandidateEmail(c pluginapi.SchedulerAuthCandidate) string`

**Exported action string constants** stay in `action` package (Task 3). Config continues to use string values `"ban"|"disable"|"delete"` etc. after import.

- [ ] **Step 1: Create directories**

```powershell
New-Item -ItemType Directory -Force -Path internal\config,internal\host,internal\ban,internal\audit,internal\xai | Out-Null
```

- [ ] **Step 2: Port `config.go` → `internal/config/config.go`**

- Change `package main` → `package config`.
- Rename:
  - `defaultConfig` → `Default`
  - `parseConfigYAML` → `ParseYAML`
  - `normalizeConfig` → `Normalize`
  - `mergeConfigPatch` → `MergePatch`
  - `configFields` → `Fields`
  - methods `durationForStatus` / `actionForStatus` / `publicView` → exported `DurationForStatus` / `ActionForStatus` / `PublicView`
- Keep all yaml tags and default numeric values identical.
- Import `xai-autoban/cpasdk/pluginapi` only.
- Action string defaults: use literal `"ban"`, `"disable"`, `"unban"`, etc. (same strings as today) to avoid importing `action` (cycle risk). Do **not** import `internal/action` from config.

- [ ] **Step 3: Port `host.go` → `internal/host/host.go`**

- `package host`
- `HostClient` → `Client`
- `realHostClient` → `Real` with package var:

```go
// CallFn is set by CGO main to invoke host methods.
var CallFn func(method string, request []byte) ([]byte, error)
```

- Move `envelope` types used by host into `host` package (duplicate of main envelope OK) **or** decode with local private types matching `{ok, result, error}`.
- `stubHost` → `Stub` (exported fields for tests).

- [ ] **Step 4: Port ban state + id helpers**

- `ban_state.go` → `internal/ban/state.go`, types `banEntry`→`Entry`, `banState`→`State`, methods exported.
- Move `authIDsEqual` / `authIDAliases` from `scheduler.go` into `internal/ban/id.go` as `AuthIDsEqual` / `AuthIDAliases`.
- Inside `state.go`, call local `AuthIDAliases` / `AuthIDsEqual`.

- [ ] **Step 5: Port audit**

- `audit.go` → `internal/audit/audit.go`: `auditLog`→`Log`, `auditEvent`→`Event`, `newAuditLog`→`New`, `truncateID` stays private.

- [ ] **Step 6: Create `internal/xai/xai.go`**

Move logic from `isXAIAuth`, `authKey`, `isXAICandidate`, `candidateEmail` (probe.go / scheduler.go). Package const:

```go
const Provider = "xai"
```

- [ ] **Step 7: Compile packages only**

Run: `go test ./internal/config ./internal/host ./internal/ban ./internal/audit ./internal/xai`  
Expected: PASS (or no tests, exit 0). Packages must compile.

---

### Task 2: persist

**Files:**
- Create: `internal/persist/persist.go` (from `persist.go`)

**Interfaces:**
- Consumes: `*ban.State`
- Produces:
  - `persist.New(path string, bans *ban.State) *Persister`
  - `(*Persister).SetPath`, `Load`, `ScheduleSave`, `Flush`, `SaveNow`

- [ ] **Step 1: Port file**

- `package persist`
- `statePersister` → `Persister`, `newStatePersister` → `New`
- Import `xai-autoban/internal/ban`
- JSON fields unchanged (`version`, `bans`)

- [ ] **Step 2: Compile**

Run: `go test ./internal/persist`  
Expected: PASS / compile OK

---

### Task 3: action engine + management client

**Files:**
- Create: `internal/action/engine.go` (from `actions.go`)
- Create: `internal/action/management_client.go` (from `management_client.go`)
- Create: `internal/action/const.go` (action/success/disableVia constants)

**Interfaces:**
- Consumes: `config.PluginConfig`, `*ban.State`, `*audit.Log`, `host.Client`
- Produces:
  - Constants: `Ban`, `Disable`, `Delete`, `SuccessNone`, `SuccessUnban`, `SuccessReenable`, `SuccessUnbanAndReenable`, `DisableViaHostAuth`, `DisableViaManagementAPI`
  - `action.NewEngine(cfg config.PluginConfig, bans *ban.State, audit *audit.Log, h host.Client, onChanged func()) *Engine`
  - `(*Engine).UpdateConfig`, `ClassifyFailure`, `ApplyFailure`, `ApplyAction`, `ApplySuccess`, `SetRequestManagementKey`, `ClearRequestManagementKey`, `RequestManagementKey`
  - `(*Engine).ManagementStatus() map[string]any` (wrap disabler status for UI)

- [ ] **Step 1: Port constants + engine**

- Replace `actionBan` string const with exported `Ban = "ban"` etc. String values must match exactly.
- Use `ban.Entry`, `ban.StorageKey`, `ban.AuthIDsEqual`, `xai` not required here.
- `lookupEmail` uses `host.Client.AuthList` and `ban.AuthIDsEqual` + local key match.

- [ ] **Step 2: Port management_client.go into action package**

- Types can stay unexported (`managementDisabler`) if only engine uses them.
- Export nothing unless tests need stubs: keep `httpDo` inject field for tests.
- Direct no-proxy transport behavior must be byte-for-byte same logic.

- [ ] **Step 3: Compile**

Run: `go test ./internal/action`  
Expected: compile OK

---

### Task 4: probe + recheck

**Files:**
- Create: `internal/probe/probe.go`
- Create: `internal/probe/recheck.go`

**Interfaces:**
- Consumes: `config.PluginConfig`, `host.Client`, `*action.Engine`, `*ban.State` (recheck uses global-like deps — inject via `Service` fields)
- Produces:
  - `probe.NewService(cfg, host, engine) *Service`
  - `(*Service).UpdateConfig`, `Start`, `Stop`, `RunOnce`, `RunOnceTrigger`, `Status`, `HistorySnapshot`, `LastResults`, `ProbeOne`
  - `(*Service).Recheck429(force bool) (Recheck429Result, error)`
  - `(*Service).RecheckSelected(authIDs []string, reenableOnOK bool) (RecheckSelectedResult, error)`
  - Types: `Result`, `Run`, `CredentialResult`, recheck result structs (exported)

**Important:** Today `recheck.go` uses package globals `bans`, `hostImpl`, `engine`, `probeSvc`, `audit`, `persister`. After split, recheck methods must be methods on `*Service` that hold references:

```go
type Service struct {
    // existing fields...
    bans    *ban.State
    audit   *audit.Log
    persist *persist.Persister // optional; or onChanged via engine
}
```

Wire these in `NewService` / a later `Service.Attach(bans, audit, persist)` from `App`.

- [ ] **Step 1: Port probe.go**

- Use `xai.IsAuth`, `xai.AuthKey`
- Use `action` constants for ban/report-only path
- `extractAccessToken` stays private in probe package

- [ ] **Step 2: Port recheck.go as methods**

- Remove dependency on package-level globals
- `indexAuthFiles` / `resolveAuthFile` private helpers in probe package

- [ ] **Step 3: Compile**

Run: `go test ./internal/probe`  
Expected: compile OK

---

### Task 5: schedule + usage

**Files:**
- Create: `internal/schedule/pick.go`
- Create: `internal/usage/handle.go`

**Interfaces:**
- Produces:
  - `schedule.Pick(raw []byte, bans *ban.State, delegate string) ([]byte, error)`  
    OR `schedule.HandlePick(req pluginapi.SchedulerPickRequest, bans *ban.State, delegate string) (pluginapi.SchedulerPickResponse, error)` plus envelope in main
  - Prefer returning response struct; let main wrap envelope:

```go
// schedule/pick.go
func Pick(req pluginapi.SchedulerPickRequest, bans *ban.State, delegate string) (pluginapi.SchedulerPickResponse, error)
```

  - `usage.Handle(raw []byte, engine *action.Engine) error` — side-effect only; main wraps empty OK envelope

- [ ] **Step 1: Port scheduler**

- RR counter: package-level `var rr uint64` in `schedule` (same as today)
- Use `xai.IsCandidate`, `xai.CandidateEmail`, `bans.IsBannedCandidate`

- [ ] **Step 2: Port usage**

- Provider check: `xai.Provider`
- Call `engine.ClassifyFailure` + `engine.ApplyFailure`

- [ ] **Step 3: Compile**

Run: `go test ./internal/schedule ./internal/usage`  
Expected: compile OK

---

### Task 6: creds + ui + mgmt

**Files:**
- Create: `internal/creds/credentials.go` (+ move `credentials_test.go`)
- Create: `internal/ui/status.go` (from `ui_status.go`; optional later embed)
- Create: `internal/mgmt/routes.go` (from `management.go`)
- Create: `internal/mgmt/status.go` if splitting helpers helps

**Interfaces:**
- Produces:
  - `creds.Build(...)`, `creds.Page(...)`, filter helpers — export what `mgmt` needs
  - `ui.StatusPage(pluginName string) string`
  - `mgmt.API` or handler struct:

```go
type Handler struct {
    Name    string // "xai-autoban"
    Version string
    App     // avoid cycle: use explicit deps instead
    Cfg     func() config.PluginConfig
    SetCfg  func(config.PluginConfig)
    Bans    *ban.State
    Audit   *audit.Log
    Engine  *action.Engine
    Probe   *probe.Service
    Persist *persist.Persister
    Host    host.Client
}

func (h *Handler) Registration() pluginapi.ManagementRegistrationResponse
func (h *Handler) Handle(req pluginapi.ManagementRequest) pluginapi.ManagementResponse
```

- [ ] **Step 1: Port credentials + tests**

- Move `credentials_test.go` → `internal/creds/credentials_test.go`
- Fix imports; run:

```
go test ./internal/creds -count=1
```

Expected: PASS (same 4 tests)

- [ ] **Step 2: Port UI**

- `statusPage()` → `ui.StatusPage(name string) string`
- Keep HTML/JS identical; only package + function name change
- Ensure page still calls same relative Management/resource paths (those are in JS strings — do not rewrite)

- [ ] **Step 3: Port management routes**

- Replace globals with `Handler` fields
- `pluginName` → `h.Name`
- `currentConfig` → `h.Cfg()`
- `setConfig` → `h.SetCfg`
- `engine` / `bans` / `audit` / `probeSvc` / `persister` / `hostImpl` → fields
- Bearer key still: `engine.SetRequestManagementKey` / defer clear
- `statusPage()` → `ui.StatusPage(h.Name)`

- [ ] **Step 4: Compile**

Run: `go test ./internal/creds ./internal/ui ./internal/mgmt`  
Expected: creds tests PASS; others compile

---

### Task 7: main App wiring + CGO + delete root duplicates

**Files:**
- Modify: `main.go` (rewrite as App)
- Modify: `abi_cgo.go` (point to App / host.CallFn)
- Modify: `main_test.go` (use `NewApp` + stubs)
- Delete root: `config.go`, `host.go`, `ban_state.go`, `actions.go`, `audit.go`, `persist.go`, `probe.go`, `recheck.go`, `scheduler.go`, `usage.go`, `management.go`, `management_client.go`, `credentials.go`, `credentials_test.go`, `ui_status.go`

**Interfaces:**
- Produces:

```go
const (
    pluginName    = "xai-autoban"
    pluginVersion = "0.5.8"
)

type App struct {
    mu      sync.Mutex
    cfg     config.PluginConfig
    bans    *ban.State
    audit   *audit.Log
    host    host.Client
    engine  *action.Engine
    probe   *probe.Service
    persist *persist.Persister
    mgmt    *mgmt.Handler
}

func NewApp(h host.Client) *App
func (a *App) HandleMethod(method string, request []byte) ([]byte, error)
func (a *App) SetConfig(cfg config.PluginConfig)
func (a *App) Config() config.PluginConfig
func (a *App) Shutdown()
```

- [ ] **Step 1: Implement `NewApp`**

Construction order:

1. `bans := &ban.State{}` (zero value OK if maps lazy-init)
2. `audit := audit.New(200)`
3. `persist := persist.New("", bans)`
4. `engine := action.NewEngine(config.Default(), bans, audit, h, persist.ScheduleSave)`
5. `probe := probe.NewService(...)` with bans/audit/persist attached
6. `mgmt := &mgmt.Handler{ Name: pluginName, Version: pluginVersion, ...callbacks... }`
7. return App

- [ ] **Step 2: Wire `handleMethod` to `defaultApp`**

```go
var defaultApp = NewApp(host.Real{})

func handleMethod(method string, request []byte) ([]byte, error) {
    return defaultApp.HandleMethod(method, request)
}
```

Map methods:
- register/reconfigure → parse YAML, `SetConfig`, `persist.Load`, probe start
- shutdown → `Shutdown`
- usage → `usage.Handle` + ok envelope
- scheduler → `schedule.Pick` + ok envelope
- management.register → `mgmt.Registration`
- management.handle → `mgmt.Handle`

- [ ] **Step 3: CGO host bridge**

In `abi_cgo.go` after init:

```go
host.CallFn = invokeHost
```

Remove old `hostCallFn` from main/host if fully moved.

- [ ] **Step 4: Delete obsolete root `.go` business files**

Keep only: `main.go`, `abi_cgo.go`, `main_test.go` (+ go.mod etc.)

- [ ] **Step 5: Fix `main_test.go`**

- Build apps via `NewApp(&host.Stub{...})`
- Replace direct global `bans`/`engine` access with `app.bans` / exported test helpers if needed
- Prefer exporting test hooks only if required: e.g. `func (a *App) Bans() *ban.State` for tests — or keep tests in internal packages

- [ ] **Step 6: Full test**

Run: `go test ./... -count=1`  
Expected: all PASS

---

### Task 8: scripts, gitignore, README paths

**Files:**
- Create: `scripts/build.sh` (move from `build.sh`)
- Delete: root `build.sh` (or leave thin wrapper that calls scripts/build.sh)
- Modify: `README.md` build section path
- Modify: `.gitignore` if needed
- Modify: `.github/workflows/release.yml` only if it references `build.sh` (currently uses inline `go build` — **no change required** unless you want consistency)

- [ ] **Step 1: Move build script**

```powershell
New-Item -ItemType Directory -Force -Path scripts | Out-Null
Move-Item -Force build.sh scripts\build.sh
```

Ensure script still `cd` to repo root (`ROOT` from script dir parent).

- [ ] **Step 2: README**

Update manual build examples to `scripts/build.sh` or keep docker one-liner using `go build` at root (still valid).

- [ ] **Step 3: `.gitignore`**

Ensure:

```
/dist/
*.so
*.h
*.dll
*.dylib
.DS_Store
*.exe
```

- [ ] **Step 4: Final verification**

```
go test ./... -count=1
```

Expected: PASS

Optional CGO build (if toolchain present):

```
# Linux/macOS or WSL
CGO_ENABLED=1 go build -buildmode=c-shared -o dist/xai-autoban.so .
```

On Windows without CGO, skip shared build; tests alone suffice for this task.

---

## Spec coverage checklist

| Spec section | Task |
|--------------|------|
| Target layout | Tasks 1–8 |
| Dependency rules / no cycles | Tasks 1, 3, 6 (client in action) |
| management_client in action | Task 3 |
| File migration map | Tasks 1–7 |
| App wiring | Task 7 |
| Behavior preservation | Tasks 3–7 (no logic rewrite) |
| scripts/CI/docs | Task 8 |
| Testing strategy | Steps in each task + Task 7 full suite |
| Implementation order | Task order 1→8 |

## Placeholder / consistency self-review

- No TBD steps.
- Exported names consistent: `ban.State`, `action.Engine`, `config.PluginConfig`, `probe.Service`, `mgmt.Handler`.
- Action string values remain `"ban"|"disable"|"delete"|...`.
- Commits optional (user must request).

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-14-xai-autoban-package-split.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — same session, batch with checkpoints  

**Which approach?**
