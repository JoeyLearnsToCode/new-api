param(
    [string]$MoeAtelierRepo = "https://github.com/JoeyLearnsToCode/moe-atelier.git"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptPath

$version = if (Test-Path (Join-Path $projectRoot "VERSION")) {
    Get-Content (Join-Path $projectRoot "VERSION") -Raw | ForEach-Object { $_.Trim() }
} else {
    "dev-$(git -C $projectRoot rev-parse --short HEAD 2>$null)"
}
if (-not $version) { $version = "dev" }

Write-Host "=== Building new-api frontend ===" -ForegroundColor Green
Push-Location (Join-Path $projectRoot "web")
bun install
if ($LASTEXITCODE -ne 0) { throw "bun install failed" }
$env:DISABLE_ESLINT_PLUGIN = 'true'
$env:VITE_REACT_APP_VERSION = $version
bun run build
if ($LASTEXITCODE -ne 0) { throw "bun run build (frontend) failed" }
Pop-Location

Write-Host "=== Building moe-atelier frontend ===" -ForegroundColor Green
& (Join-Path $scriptPath "build-moe-atelier.ps1") -MoeAtelierRepo $MoeAtelierRepo
if (-not $?) { throw "moe-atelier build failed" }

Write-Host "=== Building Go binary ===" -ForegroundColor Green
Push-Location $projectRoot
go build -ldflags "-s -w -X 'one-api/common.Version=$version'" -o new-api.exe
if ($LASTEXITCODE -ne 0) { throw "go build failed" }
Pop-Location

$binaryPath = Join-Path $projectRoot "new-api.exe"
if (Test-Path $binaryPath) {
    $size = (Get-Item $binaryPath).Length
    Write-Host "=== Done! new-api.exe built ($([math]::Round($size / 1MB, 1)) MB) ===" -ForegroundColor Green
} else {
    throw "Binary not found at $binaryPath"
}
