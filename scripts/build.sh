#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p dist
go test ./...
CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o "dist/xai-autoban" .

# normalize extension by GOOS
GOOS="$(go env GOOS)"
case "$GOOS" in
  windows)
    if [[ -f dist/xai-autoban ]]; then mv dist/xai-autoban dist/xai-autoban.dll; fi
    ;;
  darwin)
    if [[ -f dist/xai-autoban ]]; then mv dist/xai-autoban dist/xai-autoban.dylib; fi
    ;;
  *)
    if [[ -f dist/xai-autoban ]]; then mv dist/xai-autoban dist/xai-autoban.so; fi
    ;;
esac

echo "built artifacts in dist/"
ls -la dist || true
