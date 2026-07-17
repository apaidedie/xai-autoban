# xai-autoban Stability Contract

**Status:** **Stable 1.0.0**  
**Audience:** Operators running CLIProxyAPI (CPA) / CPA-Manager-Plus (CPAMP)

### 1.x policy

- **Do not remove or rename** frozen keys in §3 without a **major** version bump.
- **New keys** allowed in minor/patch; document in CHANGELOG.
- **Default value changes** for safety-critical paths (isolate / delete / `auto_using_api`) require CHANGELOG callout.
- Behavior guarantees in §1 are the operator contract for 1.0+.

---

## 1. Guarantees (what we promise)

### 1.1 Isolation model

| Concept | Meaning |
|---------|---------|
| **隔离 (ban ledger)** | Plugin-internal; scheduler skips the credential. Does **not** by itself flip CPA UI「禁用」. |
| **禁用 (disabled)** | CPA credential toggle (`Auth.Disabled`), via Management API and/or host save. |
| **API 模式 (`using_api`)** | CPA xAI flag to prefer official API path for OAuth files. |
| **删除** | Management DELETE auth file; on failure → `delete_fallback` + `pending_delete` marker. |

### 1.2 Traffic paths (priority)

1. **Real usage** is ground truth: successful request **clears isolation** for that account (when applicable).
2. **Probe / recheck** are active checks; they must not override a *recent* real success window (false-positive guard).
3. **Scheduler** only consults the ban ledger (plus host disabled state as provided by CPA).

### 1.3 Default failure semantics (unless ops config changes actions)

| Signal | Default behavior |
|--------|------------------|
| Real usage **success** | Clear isolation |
| **401** / token invalid | Isolate (reauth possible) |
| **402** / free-usage | Isolate on **real usage** failure only; **probe 402 does not isolate** |
| Soft **403** (permission-style) | Need `fail_streak_403` consecutive fails (default **3**) before isolate; hard account bans may isolate immediately |
| Bare **429** | Isolate only (short window / header reset when present) |
| **5xx** / model unavailable | Generally **no** isolate |

### 1.4 Probe routing (aligned with CPA)

| Credential | Default upstream |
|------------|------------------|
| OAuth (refresh_token, not using_api) | `https://cli-chat-proxy.grok.com/v1` + Grok CLI headers |
| API key / `using_api=true` | `https://api.x.ai/v1` (or explicit `base_url`) |

Default mode: `probe_mode=responses_mini` → real `POST /responses`.

### 1.5 Auto `using_api`

| `auto_using_api` | Behavior |
|------------------|----------|
| `off` (**default**) | Never auto-write; manual ops only |
| `on_403` | Probe/recheck may enable once per key **per run** on HTTP 403 |
| `on_fail` | Same, on 401/402/403 |

Manual ops action `using_api` is **never** gated by this setting.  
`SetUsingAPI` **write-then-read** verifies the field; failure returns error (no silent success).

### 1.6 Ops / install split

| Surface | Responsibility |
|---------|----------------|
| **运维台 → 编辑配置** | Daily probe/policy (`OpsSettingsKeys`) |
| **插件管理** | Enable + management key/url/`disable_via` (secrets) |
| **CPAMP** | Use **resource** write channel (`GET .../data|ops`); do **not** put `cpamp_...` panel key into this plugin |

### 1.7 Release artifacts (1.0 requirement)

- Git tag `vX.Y.Z`
- GitHub Release assets: `xai-autoban_<ver>_linux_amd64.zip`, `linux_arm64`, `windows_amd64` + `checksums.txt`
- `registry.json` version matches `pluginVersion` in `main.go`

Prefer **tag push** or **workflow_dispatch**; do not create an empty Release before CI finishes.

---

## 2. Non-guarantees (out of scope / no promise)

- Editing arbitrary auth fields (WebSockets, custom headers, proxy, model allowlists) beyond: disable, note, delete, reauth, `using_api`
- Multi-provider inspection or browser Device OAuth (use **cpa-auth-inspect**)
- Perfect parity with every future CPA internal build if host mapping of `disabled` / `using_api` changes
- Zero false positives on soft 403 / upstream flaps (mitigated by streak + usage grace, not eliminated)
- macOS CI binaries (local CGO build only)
- Lifetime cross-restart memory of soft-403 streak or auto-using_api tries (in-memory per process / per run)

---

## 3. Config freeze table (toward 1.0)

### 3.1 Frozen keys (must remain after 1.0; 0.x must not remove)

**Ops-console persistable (`OpsSettingsKeys`):**

| Key | Type (logical) | Default |
|-----|----------------|---------|
| `ban_401_seconds` | int | 86400 |
| `ban_402_seconds` | int | 604800 |
| `ban_403_seconds` | int | 86400 |
| `ban_429_fallback_seconds` | int | 1800 |
| `action_on_401` | ban\|disable\|delete | ban |
| `action_on_402` | ban\|disable\|delete | ban |
| `action_on_403` | ban\|disable\|delete | ban |
| `action_on_429` | ban (recommended) | ban |
| `probe_enabled` | bool | true |
| `probe_interval_seconds` | int | 600 |
| `probe_timeout_seconds` | int | 20 |
| `probe_concurrency` | int | 3 |
| `probe_qps` | float | 2 |
| `probe_mode` | responses_mini\|models | responses_mini |
| `probe_base_url` | string | "" |
| `probe_path` | string | /models |
| `probe_action` | ban\|disable\|delete | ban |
| `probe_on_success` | none\|unban\|reenable\|unban_and_reenable | unban |
| `probe_include_disabled` | bool | false |
| `probe_only_disabled` | bool | false |
| `auto_execute` | bool | true |
| `action_cooldown_seconds` | int | 60 |
| `fail_streak_403` | int | 3 |
| `fail_streak_window_seconds` | int | 1800 |
| `auto_using_api` | off\|on_403\|on_fail | off（更安全；旧默认 on_403） |
| `delete_fallback` | disable\|ban | disable |
| `scheduler_delegate` | round-robin\|fill-first | round-robin |
| `audit_max_events` | int | 200 |

**Install / management (plugin manage schema + runtime):**

| Key | Notes |
|-----|--------|
| `disable_via` | `host_auth` \| `management_api` |
| `management_url` | default `http://127.0.0.1:8317` |
| `management_key` / `management_key_env` | secret; never log |
| `management_timeout_seconds` | |
| `management_auth_failure_cooldown_seconds` | |
| `state_file` | default `xai-autoban-state.json` |

**Rules:**

- After **1.0.0**: removing or renaming a frozen key requires **major** version.
- Adding keys is allowed in minor/patch.
- Changing a **default** after 1.0 requires CHANGELOG callout; prefer opt-in new keys over silent default flips for safety-critical paths (isolate / delete / auto_using_api).

### 3.2 Apply-action vocabulary (frozen)

| `action` | Meaning |
|----------|---------|
| `ban` | Write isolation ledger |
| `disable` / `reenable` | CPA disabled flag |
| `delete` | Management delete (+ fallback) |
| `reauth` | refresh_token → access_token |
| `using_api` / `enable_api` / `api_mode` | Set `using_api=true` |

### 3.3 Resource ops (CPAMP) vocabulary (frozen)

Stable `op` names include: `settings`, `unban`, `unban_all`, `probe`, `apply`, `reauth`, `recheck_selected`, `recheck429`, `import`, `backup`, `probe_status`, `list_ids`, `data` (read).

---

## 4. 1.0 exit checklist

Ship **1.0.0** only when all are true:

- [x] This file reviewed and linked from README as the operator contract
- [x] No known open P0 false-isolate under normal traffic at ship time (soft 403 + usage grace + probe 402 skip covered by tests; continue monitoring)
- [x] Tests green on CI for: soft 403 streak, usage success unban, probe 402 no-isolate, `auto_using_api` gate, using_api write verify, delete fallback  
      → `internal/action/stability_contract_test.go`, `internal/usage/handle_test.go`, `internal/probe/using_api_test.go`, `internal/classify` 402→quota
- [x] Ops list shows: isolation/disabled, using_api, soft-403 progress, last probe (0.5.48+)
- [x] Disable + using_api write paths documented with Management key requirements (this file §1 + README)
- [x] Release workflow produces checksums for linux/windows without empty-asset races (tag / workflow_dispatch)
- [x] Version strings identical: `main.go` / `registry.json` / Release tag
- [x] CHANGELOG **1.0.0** states: *Stable contract per STABILITY.md*
- [x] Freeze window completed: **0.9.0** then **1.0.0**

### Version path (history)

| Version | Intent |
|---------|--------|
| **0.5.x** | Feature + contract tests |
| **0.9.0** | Config freeze |
| **1.0.0** | **Stable** — this release |

---

## 5. Operator quick start (stable entry points)

```text
运维台:  /v0/resource/plugins/xai-autoban/status
```

```yaml
plugins:
  configs:
    xai-autoban:
      enabled: true
      priority: 200
      disable_via: management_api
      management_key_env: CPA_MANAGEMENT_KEY   # CPA secret-key, NOT cpamp_...
```

State file (default): `xai-autoban-state.json` (bans + ops settings overlay; local artifact).

---

## 6. Document maintenance

- Update this file when adding frozen keys or changing guaranteed semantics.
- Historical designs: `docs/archive/`, `docs/superpowers/`.
- Product version always: `main.go` `pluginVersion`.
