param(
    [string]$MoeAtelierRepo = "https://github.com/JoeyLearnsToCode/moe-atelier.git",
    [string]$MoeAtelierDir = "moe-atelier-tmp",
    [string]$OutputDir = "web/moe-atelier-dist"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptPath

$moeDir = Join-Path $projectRoot $MoeAtelierDir
$outDir = Join-Path $projectRoot $OutputDir

if (Test-Path $moeDir) { Remove-Item -Recurse -Force $moeDir }
if (Test-Path $outDir) { Remove-Item -Recurse -Force $outDir }

Write-Host "=== Cloning moe-atelier ===" -ForegroundColor Green
git clone --depth 1 $MoeAtelierRepo $moeDir
if ($LASTEXITCODE -ne 0) { throw "git clone failed with exit code $LASTEXITCODE" }

try {
    Push-Location $moeDir

    Write-Host "=== Installing dependencies ===" -ForegroundColor Green
    bun install
    if ($LASTEXITCODE -ne 0) { throw "bun install failed with exit code $LASTEXITCODE" }

    Write-Host "=== Building frontend ===" -ForegroundColor Green
    bun run build
    if ($LASTEXITCODE -ne 0) { throw "bun run build failed with exit code $LASTEXITCODE" }
    if (-not (Test-Path "dist/index.html")) { throw "Build failed: dist/index.html not found" }

} finally {
    Pop-Location
}

Write-Host "=== Copying dist to $OutputDir ===" -ForegroundColor Green
New-Item -ItemType Directory -Path $outDir -Force
Remove-Item -Recurse -Force $outDir -ErrorAction SilentlyContinue
Move-Item -Path "$moeDir/dist" -Destination $outDir -Force
if ($LASTEXITCODE -ne 0) { throw "moving dist failed with exit code $LASTEXITCODE" }

Remove-Item -Recurse -Force $moeDir

Write-Host "=== Done! moe-atelier frontend built at $OutputDir ===" -ForegroundColor Green
