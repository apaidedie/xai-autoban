# using_api Auto Policy Implementation Plan

> **For agentic workers:** Execute task-by-task with TDD. Checkboxes track progress.

**Goal:** Controllable auto `using_api` (default `on_403`), per-run once, write-verify on `SetUsingAPI`.

**Architecture:** Config field gates a pure `ShouldAutoUsingAPI`; probe/recheck pass a per-run map; `SetUsingAPI` AuthGet-verifies.

**Tech Stack:** Go, existing host.Stub tests, ops UI JS in `status.go`.

**Spec:** `docs/superpowers/specs/2026-07-16-using-api-auto-policy-design.md`

## Global Constraints

- Version **0.5.47**
- Default `auto_using_api: on_403`
- Manual `using_api` never gated
- No state-file lifetime ledger
- No usage-path auto flip

---

### Task 1: Config `auto_using_api`

**Files:** `internal/config/config.go`, new `internal/config/config_test.go` (or extend if exists)

- [x] Test normalize aliases + default
- [x] Add field, Default, Normalize, PublicView, OpsSettingsKeys
- [x] `go test ./internal/config/...`

### Task 2: Gate `ShouldAutoUsingAPI`

**Files:** `internal/probe/using_api.go`, `internal/probe/using_api_test.go`

- [x] Table tests: off/403/401/on_fail/api_key/already using_api/tried
- [x] Implement pure function
- [x] `go test ./internal/probe/ -run ShouldAuto`

### Task 3: Wire probe + recheck + per-run map

**Files:** `internal/probe/probe.go`, `internal/probe/recheck.go`

- [x] Replace hard-coded 401/402/403 with gate + map
- [x] Mark tried before SetUsingAPI; write fail keeps original outcome

### Task 4: `SetUsingAPI` write verify

**Files:** `internal/action/engine.go`, `main_test.go`

- [x] After write, AuthGet; fail if not reflected
- [x] Mgmt OK but verify fail → host save fallback then re-verify
- [x] Host stub reflects AuthSave into JSONBy for verify tests

### Task 5: Ops UI + version docs

**Files:** `internal/ui/status.go`, CHANGELOG, main.go, registry, README

- [x] Select for `auto_using_api` in 编辑配置
- [x] Bump 0.5.47

### Task 6: Full test

- [x] `go test ./internal/config ./internal/probe .` green
