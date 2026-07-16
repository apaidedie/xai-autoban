# xai-autoban Backlog Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the approved backlog: compile-safe probe dual-endpoint/retry, true Management delete, async probe progress, disabled-target flags, UI classification, Windows build/docs, and explicit `action_on_402` for quota.

**Architecture:** Keep CPA CGO plugin shape. Probe owns HTTP probe + job progress; action.Engine owns isolate/delete via host or Management API (direct no-proxy HTTP); config/UI/mgmt routes expose new knobs and fields without ABI changes.

**Tech Stack:** Go, existing `internal/*` packages, `cpasdk/pluginapi`, CPA Management HTTP, embedded ops UI in `internal/ui/status.go`.

## Global Constraints

- Version label may stay **0.5.8** unless a release commit explicitly bumps it.
- Do not break existing management routes used by the ops UI.
- Bare temporary 429 remains **ban-only** isolation (not disable) on auto paths.
- Free-usage exhaustion uses **402 duration** and **`action_on_402`**.
- Management HTTP must use **direct no-proxy** transport (same as disable).
- Every task ends with `go test ./... -count=1` green (or package-scoped tests when noted).
- No git commits unless the user explicitly asks.

---

### Task 1: Fix probe compile + lock ProbeOne behavior with tests

**Files:**
- Modify: `internal/probe/probe.go` (add classify import; keep existing dual-endpoint logic)
- Create: `internal/probe/probe_one_test.go`
- Test: `go test ./internal/probe/ -count=1`

**Interfaces:**
- Consumes: `classify.IsFreeUsageExhausted`, `classify.ExtractError`
- Produces: `ProbeOne(cfg, host, f) (status int, body string, err error)` — already the public shape

- [ ] **Step 1: Write failing tests for ProbeOne**

Create `internal/probe/probe_one_test.go`:

```go
package probe

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"xai-autoban/cpasdk/pluginapi"
	"xai-autoban/internal/config"
	"xai-autoban/internal/host"
)

func TestProbeOneModelsSuccess(t *testing.T) {
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPDoFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			t.Fatalf("unexpected url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "models"
	st, body, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 || !strings.Contains(body, "grok-4.5") {
		t.Fatalf("st=%d body=%q err=%v", st, body, err)
	}
}

func TestProbeOneBare429RetriesOnce(t *testing.T) {
	var hits int32
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPDoFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			n := atomic.AddInt32(&hits, 1)
			if n == 1 {
				return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(`{"error":{"message":"rate limited"}}`)}, nil
			}
			return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"id":"ok"}`)}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 {
		t.Fatalf("st=%d err=%v hits=%d", st, err, hits)
	}
	if hits < 2 {
		t.Fatalf("expected retry, hits=%d", hits)
	}
}

func TestProbeOneFreeUsageNoRetry(t *testing.T) {
	var hits int32
	body429 := `{"error":{"code":"free-usage-exhausted","message":"used all the included free usage"}}`
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPDoFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			atomic.AddInt32(&hits, 1)
			return pluginapi.HTTPResponse{StatusCode: 429, Body: []byte(body429)}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, body, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if st != 429 || err == nil {
		t.Fatalf("st=%d err=%v body=%q", st, err, body)
	}
	if hits != 1 {
		t.Fatalf("free-usage must not retry, hits=%d", hits)
	}
}

func TestProbeOneResponsesFallbackCompletions(t *testing.T) {
	var sawCompletions bool
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "a1", AuthIndex: "1", Name: "xai-1", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"1": json.RawMessage(`{"access_token":"tok"}`)},
		HTTPDoFn: func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
			if strings.Contains(req.URL, "/models") {
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"data":[{"id":"grok-4.5"}]}`)}, nil
			}
			if strings.Contains(req.URL, "/responses") {
				return pluginapi.HTTPResponse{StatusCode: 403, Body: []byte(`{"error":{"message":"denied"}}`)}, nil
			}
			if strings.Contains(req.URL, "/chat/completions") {
				sawCompletions = true
				return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"choices":[]}`)}, nil
			}
			t.Fatalf("url %s", req.URL)
			return pluginapi.HTTPResponse{}, nil
		},
	}
	p := NewService(config.Default(), stub, nil)
	cfg := config.Default()
	cfg.ProbeMode = "responses_mini"
	st, _, err := p.ProbeOne(cfg, stub, stub.Files[0])
	if err != nil || st != 200 || !sawCompletions {
		t.Fatalf("st=%d err=%v sawCompletions=%v", st, err, sawCompletions)
	}
}
```

If `host.Stub` lacks `HTTPDoFn`, extend `internal/host/host.go` stub with an optional `HTTPDoFn` field used by `HTTPDo`.

- [ ] **Step 2: Run tests — expect compile failure or fail**

Run: `go test ./internal/probe/ -count=1 -v`

Expected: fail on missing `classify` import and/or missing stub field.

- [ ] **Step 3: Minimal fix**

In `internal/probe/probe.go` imports add:

```go
"xai-autoban/internal/classify"
```

Ensure `host.Stub` supports injectable `HTTPDo`:

```go
// in internal/host (stub type)
HTTPDoFn func(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error)

func (s *Stub) HTTPDo(req pluginapi.HTTPRequest) (pluginapi.HTTPResponse, error) {
	if s.HTTPDoFn != nil {
		return s.HTTPDoFn(req)
	}
	// existing default behavior
}
```

Keep existing ProbeOne dual-endpoint logic; only fix compile and any test-driven edge cases.

- [ ] **Step 4: Re-run tests**

Run: `go test ./internal/probe/ -count=1`

Expected: PASS

- [ ] **Step 5: Full suite**

Run: `go test ./... -count=1`

Expected: PASS

---

### Task 2: True Management delete

**Files:**
- Modify: `internal/action/management_client.go` — add `deleteAuthFile`
- Modify: `internal/action/engine.go` — `applyDelete` tries real delete first
- Modify: `main_test.go` — update `TestDeleteFallsBackToDisable`; add success path test
- Test: `go test . -count=1 -run Delete`

**Interfaces:**
- Consumes: existing `managementDisabler` key resolution + `HTTPDoer`
- Produces: `deleteAuthFile(authID, authIndex string) error` (unexported on disabler); engine `applyDelete` success clears pending

- [ ] **Step 1: Write failing tests**

Update/add in `main_test.go`:

```go
func TestDeleteViaManagementAPI(t *testing.T) {
	defaultApp.bans.ClearAll()
	var deleted bool
	stub := &host.Stub{
		Files: []pluginapi.HostAuthFileEntry{{ID: "auth-del", AuthIndex: "9", Name: "xai-del.json", Provider: "xai"}},
		JSONBy: map[string]json.RawMessage{"9": json.RawMessage(`{"access_token":"tok"}`)},
	}
	cfg := config.Default()
	cfg.DisableVia = "management_api"
	cfg.ManagementKey = "test-key"
	cfg.DeleteFallback = action.Disable
	eng := action.NewEngine(cfg, defaultApp.bans, audit.New(20), stub, nil)
	// inject management HTTP if engine exposes SetManagementHTTP / test hook
	// Prefer: eng.SetManagementDoer(func(req pluginapi.HTTPRequest, timeout int) (pluginapi.HTTPResponse, error) {
	//   if req.Method == http.MethodDelete && strings.Contains(req.URL, "/auth-files") {
	//     deleted = true
	//     return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
	//   }
	//   if req.Method == http.MethodGet && strings.Contains(req.URL, "/auth-files") {
	//     return pluginapi.HTTPResponse{StatusCode: 200, Body: []byte(`{"files":[{"id":"auth-del","name":"xai-del.json","auth_index":"9","provider":"xai"}]}`)}, nil
	//   }
	//   return pluginapi.HTTPResponse{StatusCode: 404}, nil
	// })
	now := time.Now()
	entry := ban.Entry{StatusCode: 401, Reason: "unauthorized", BannedAt: now, ResetAt: now.Add(time.Hour)}
	if err := eng.ApplyAction("auth-del", action.Delete, "probe", entry, true); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected management DELETE")
	}
	if len(stub.Saves) != 0 {
		t.Fatalf("true delete must not AuthSave fallback, saves=%d", len(stub.Saves))
	}
	snap := defaultApp.bans.Snapshot(now)
	if e, ok := snap["auth-del"]; ok && e.PendingDelete {
		t.Fatalf("pending_delete should be cleared on real delete: %#v", e)
	}
}

func TestDeleteFallsBackToDisable(t *testing.T) {
	// keep existing; ensure DELETE failure or missing key still disables + pending_delete
}
```

If Engine has no test hook for management HTTP, add:

```go
func (e *Engine) SetManagementHTTPForTest(do HTTPDoer) {
	// set on internal managementDisabler.httpDo
}
```

- [ ] **Step 2: Run test — expect fail**

Run: `go test . -count=1 -run 'TestDelete' -v`

Expected: FAIL (no real DELETE)

- [ ] **Step 3: Implement delete on management client**

In `management_client.go`:

```go
func (m *managementDisabler) deleteAuthFile(authID, authIndex string) error {
	// resolve key; if missing return errManagementKeyMissing
	// list/find name candidates same as patchAuthNoteWithKey
	// DELETE baseURL+"/auth-files" with body {"name": name} OR query ?name=
	// Prefer body JSON {"name": name} if CPA accepts it; if unknown, try:
	//   DELETE .../auth-files/{name}
	// On 2xx return nil; else return HTTPError
}
```

CPA shape (implement against most common):

```http
DELETE /v0/management/auth-files
Authorization: Bearer <key>
Content-Type: application/json

{"name":"xai-del.json"}
```

If that fails in real CPA later, keep candidate loop for `name` / `id` / `id.json`.

- [ ] **Step 4: Wire applyDelete**

```go
func (e *Engine) applyDelete(authID, source string, entry ban.Entry, force bool) error {
	// 1) try management delete when key present (always try if management configured)
	// 2) on success: entry.PendingDelete=false; bans.Clear(authID) OR set short tombstone without pending
	//    Recommended: Clear ban + audit delete/ok (credential gone)
	// 3) on failure: existing fallback disable/ban + PendingDelete=true
}
```

Recommended success semantics:

- Real delete succeeded → `e.bans.Clear(authID)`, audit `delete/ok`, `notifyChanged`.
- Fallback → current behavior (`PendingDelete=true`).

- [ ] **Step 5: Run tests**

Run: `go test . -count=1 -run 'TestDelete' -v` then `go test ./... -count=1`

Expected: PASS

---

### Task 3: Async probe job + progress

**Files:**
- Modify: `internal/probe/probe.go` — job state, progress, StartJob / JobStatus
- Modify: `internal/mgmt/routes.go` — async POST /probe, GET /probe/status
- Modify: `internal/ui/status.go` — poll progress
- Modify: `main_test.go` — async accept + wait path
- Test: `go test . -count=1 -run Probe`

**Interfaces:**
- Produces:

```go
type JobStatus struct {
	Running bool   `json:"running"`
	JobID   int64  `json:"job_id"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Result  *Result `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (p *Service) StartJob(force bool, trigger string) (jobID int64, err error) // err if already running
func (p *Service) JobStatus() JobStatus
func (p *Service) RunOnceTrigger(...) // updates Done/Total under mutex
```

- [ ] **Step 1: Write failing API test**

```go
func TestProbeAsyncAccept(t *testing.T) {
	// POST /probe without wait → ok + job_id + running true or accepted
	// GET /probe/status → done/total fields present
}

func TestProbeWaitStillSync(t *testing.T) {
	// POST body {"wait":true} returns result when finished
}
```

- [ ] **Step 2: Implement job state on Service**

```go
// fields on Service:
jobRunning bool
jobID      int64
jobDone    int
jobTotal   int
jobResult  *Result
jobErr     string

func (p *Service) StartJob(force bool, trigger string) (int64, error) {
	p.mu.Lock()
	if p.jobRunning {
		p.mu.Unlock()
		return p.jobID, fmt.Errorf("probe already running")
	}
	p.jobRunning = true
	p.runSeq++
	id := p.runSeq
	p.jobID = id
	p.jobDone, p.jobTotal = 0, 0
	p.jobResult, p.jobErr = nil, ""
	p.mu.Unlock()
	go func() {
		res, err := p.RunOnceTrigger(force, trigger)
		p.mu.Lock()
		defer p.mu.Unlock()
		p.jobRunning = false
		p.jobResult = &res
		if err != nil {
			p.jobErr = err.Error()
		}
	}()
	return id, nil
}
```

In the per-credential loop of `RunOnceTrigger`, after each target finishes:

```go
p.mu.Lock()
p.jobDone++
p.mu.Unlock()
```

Set `jobTotal = len(targets)` before spawning workers.

- [ ] **Step 3: Routes**

Registration add:

```go
{Method: http.MethodGet, Path: ("/plugins/"+h.Name)+"/probe/status", Description: "Probe job progress."},
```

POST `/probe`:

```go
var body struct {
	Force bool `json:"force"`
	Wait  bool `json:"wait"`
}
if body.Wait {
	res, err := h.Probe.RunOnce(body.Force)
	// existing sync response
} else {
	id, err := h.Probe.StartJob(body.Force, "manual")
	// 200 {ok, job_id, accepted:true} or 409 if already running
}
```

GET `/probe/status` → `h.Probe.JobStatus()`.

- [ ] **Step 4: UI poll**

```javascript
async function runProbe(){
  if(state.busy||!confirm('立即巡检全部 xAI 凭据？')) return;
  try{
    setBusy(true,'巡检中'); setProgress(0,100);
    const acc=await apiMgmt('POST','/probe',{force:false,wait:false});
    // poll
    for(;;){
      const st=await apiMgmt('GET','/probe/status');
      const done=st.done||0, total=st.total||0;
      setProgress(done, total>0?total:100);
      if(!st.running){
        const r=st.result||{};
        const msg='巡检完成 成功='+(r.ok||0)+' 失败='+(r.failed||0);
        setMessage(msg); toast(msg, st.error?'err':'ok');
        break;
      }
      await new Promise(r=>setTimeout(r,500));
    }
    await loadData(true);
  }catch(e){ ... }
  finally{ setBusy(false); setProgress(0,0); }
}
```

- [ ] **Step 5: Tests green**

Run: `go test ./... -count=1`

---

### Task 4: probe_include_disabled / probe_only_disabled

**Files:**
- Modify: `internal/config/config.go` — fields, Default, Normalize, PublicView, ConfigFields
- Modify: `internal/probe/probe.go` — target selection
- Modify: `internal/ui/status.go` — settings checkboxes bind
- Test: config normalize + probe selection unit test

**Interfaces:**
- Config:

```go
ProbeIncludeDisabled bool `yaml:"probe_include_disabled"`
ProbeOnlyDisabled    bool `yaml:"probe_only_disabled"`
```

- [ ] **Step 1: Failing selection test**

```go
func TestRunOnceOnlyDisabled(t *testing.T) {
	// files: one enabled, one disabled; only_disabled true → Checked==1
}
```

- [ ] **Step 2: Config + selection**

```go
// RunOnceTrigger target filter:
for _, f := range files {
	if !xai.IsAuth(f) { continue }
	if cfg.ProbeOnlyDisabled {
		if !f.Disabled { continue }
	} else if !cfg.ProbeIncludeDisabled && f.Disabled {
		continue
	}
	targets = append(targets, f)
}
```

If `ProbeOnlyDisabled` true, treat as include disabled only (ignore include flag).

- [ ] **Step 3: UI settings**

Add two checkboxes under probe settings; save/load via existing settings map keys.

- [ ] **Step 4: Tests**

Run: `go test ./... -count=1`

---

### Task 5: Ops UI + API classification field

**Files:**
- Modify: `internal/creds/credentials.go` — `Classification` on `Info`; copy from ban entry
- Modify: `internal/mgmt/routes.go` — if any DTO duplicates Info fields, add classification
- Modify: `internal/ui/status.go` — label + render
- Test: `internal/creds` unit test for field propagation

- [ ] **Step 1: Failing creds test**

```go
func TestBuildIncludesClassification(t *testing.T) {
	// ban entry Classification=quota_exhausted → Info.Classification same
}
```

- [ ] **Step 2: Wire field**

```go
type Info struct {
	// ...
	Classification string `json:"classification,omitempty"`
}
// in Build when entry found:
item.Classification = entry.Classification
```

- [ ] **Step 3: UI**

```javascript
function classLabel(c){
  return ({
    rate_limited:'限流',
    quota_exhausted:'额度用尽',
    reauth:'需重新授权',
    permission_denied:'权限拒绝',
    model_unavailable:'模型不可用',
    probe_error:'巡检错误',
    healthy:'健康'
  })[c]||c||'';
}
// in render meta: show classLabel(c.classification) before/with reason
```

- [ ] **Step 4: Tests**

Run: `go test ./internal/creds/ ./... -count=1`

---

### Task 6: build.ps1 + fix build.sh + README Docker

**Files:**
- Create: `scripts/build.ps1`
- Modify: `scripts/build.sh` (fix ROOT)
- Modify: `README.md` (Docker section; keep encoding clean UTF-8)

- [ ] **Step 1: Fix build.sh ROOT**

```bash
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
```

- [ ] **Step 2: Add build.ps1**

```powershell
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
if (-not $Root) { $Root = Resolve-Path (Join-Path $PSScriptRoot "..") }
# Prefer:
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root
New-Item -ItemType Directory -Force -Path dist | Out-Null
go test ./...
$env:CGO_ENABLED = "1"
go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o "dist/xai-autoban.dll" .
Write-Host "built dist/xai-autoban.dll"
```

- [ ] **Step 3: README Docker section**

```markdown
## Docker 安装（摘要）

1. 构建插件产物（Linux: `scripts/build.sh` → `dist/xai-autoban.so`）
2. 挂载到 CPA `plugins/` 目录
3. `config.yaml` 启用 `plugins.enabled` 与 `configs.xai-autoban`
4. 设置 `CPA_MANAGEMENT_KEY`（或 `management_key`）以便 disable/delete
5. 打开 `/v0/resource/plugins/xai-autoban/status`
```

- [ ] **Step 4: Verify scripts parse**

Run: `go test ./... -count=1`  
Optional: `powershell -File scripts/build.ps1` only if CGO toolchain present (may skip on agent if no gcc).

---

### Task 7: Quota respects action_on_402

**Files:**
- Modify: `internal/action/engine.go` — quota branch uses `cfg.ActionOn402` explicitly
- Create or modify tests in `main_test.go` / action tests

- [ ] **Step 1: Failing test**

```go
func TestQuotaUsesActionOn402(t *testing.T) {
	cfg := config.Default()
	cfg.ActionOn402 = action.Disable
	eng := action.NewEngine(cfg, bans, audit.New(10), stub, nil)
	body := `{"error":{"code":"free-usage-exhausted","message":"used all the included free usage"}}`
	entry, ok := eng.ClassifyFailureWithBody(429, nil, body, time.Now())
	if !ok || entry.Classification != "quota_exhausted" || entry.Action != action.Disable {
		t.Fatalf("%#v ok=%v", entry, ok)
	}
	cfg.ActionOn402 = action.Ban
	eng.UpdateConfig(cfg) // or new engine
	entry, ok = eng.ClassifyFailureWithBody(429, nil, body, time.Now())
	if !ok || entry.Action != action.Ban {
		t.Fatalf("expected ban, got %#v", entry)
	}
}
```

- [ ] **Step 2: Implementation**

In quota branch after building entry:

```go
case judged.Classification == classify.QuotaExhausted || sc == http.StatusPaymentRequired:
	entry.Action = cfg.ActionOn402
	if entry.Action == "" {
		entry.Action = cfg.ActionForStatus(http.StatusPaymentRequired)
	}
	// duration already 402 window
```

Remove any path that forces disable for quota when config says ban.

Keep rate_limited → force Ban.

- [ ] **Step 3: Full suite**

Run: `go test ./... -count=1`

Expected: PASS

---

## Self-review

| Spec requirement | Task |
|------------------|------|
| Probe 429 retry + dual endpoint | Task 1 |
| True delete | Task 2 |
| Async probe + progress | Task 3 |
| include/only disabled | Task 4 |
| UI classification | Task 5 |
| build.ps1 + README Docker | Task 6 |
| action_on_402 tighten | Task 7 |
| Compile fix for classify import | Task 1 |

No TBD placeholders. Types aligned: `ProbeOne (int,string,error)`, `JobStatus`, config yaml names exact.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-xai-autoban-backlog.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — this session, executing-plans with checkpoints  

Which approach?
