# Changelog

## 0.5.9 - 2026-07-15

### Fixes
- Exclusive probe flight lock: scheduled + manual/async cannot run concurrently
- UI delete copy: real Management DELETE with disable/ban fallback
- Ban list API includes `classification`
- Reauth uses direct no-proxy HTTP to token endpoint; post-refresh `/models` probe
- Probe path: single AuthGet for local expiry + upstream probe
- Recheck429 non-429 failures use body semantic classify
- Status list: sample AuthGet JSON for `token_expired` / `needs_refresh` flags
- `go vet` clean test helpers; scripts/build.sh ROOT fix

### Features (from 0.5.8 hardening)
- Semantic failure classifier (429 vs free-usage vs reauth)
- True Management DELETE
- Async probe job + progress polling
- `probe_include_disabled` / `probe_only_disabled`
- refresh_token reauth API + UI button
- Summary cards always show 0 + hover; 401–429 overview cards

## 0.5.8

- Package split (`internal/*`), usage/probe body classify baseline
