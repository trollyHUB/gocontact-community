# Activate version-controlled Git hooks for GoContact Studio
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

Write-Host "Configuring Git hooks path to .githooks..." -ForegroundColor Cyan
git config core.hooksPath .githooks

Write-Host "Verifying core.hooksPath configuration..." -ForegroundColor Yellow
$configuredPath = git config core.hooksPath
if ($configuredPath -eq ".githooks") {
    Write-Host "[SUCCESS] Git hooks successfully activated (.githooks)!" -ForegroundColor Green
    Write-Host "  - pre-commit: Anti-Leak Shield, gofmt, Clean Architecture linter" -ForegroundColor White
    Write-Host "  - commit-msg: Conventional Commits validator" -ForegroundColor White
    Write-Host "  - pre-push: Full go test suite verification" -ForegroundColor White
    Write-Host "  - post-merge: go mod tidy automation" -ForegroundColor White
} else {
    Write-Host "[ERROR] Failed to set Git hooks path!" -ForegroundColor Red
    exit 1
}
