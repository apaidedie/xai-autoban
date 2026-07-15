$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location -LiteralPath $Root
New-Item -ItemType Directory -Force -Path dist | Out-Null
Write-Host "Running tests..."
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$env:CGO_ENABLED = "1"
Write-Host "Building c-shared DLL..."
go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o "dist/xai-autoban.dll" .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "built dist/xai-autoban.dll"
Get-ChildItem dist | Format-Table Name, Length
