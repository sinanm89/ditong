# Initial GitHub push script - creates a clean squashed history
# Run this ONCE to initialize the GitHub repo with clean history
# Usage: .\scripts\github-init.ps1

$ErrorActionPreference = "Stop"

$GITHUB_REMOTE = "git@github.com:sinanm89/ditong.git"
$TEMP_DIR = Join-Path $env:TEMP "ditong-github-init-$(Get-Random)"

Write-Host "=== ditong GitHub Initial Push ===" -ForegroundColor Cyan
Write-Host "This creates a clean, squashed history for the GitHub mirror."
Write-Host ""

# Ensure we're in the repo root
if (-not (Test-Path "go/go.mod")) {
    Write-Host "Error: Run this from the ditong repository root" -ForegroundColor Red
    exit 1
}

$CURRENT_DIR = Get-Location

try {
    Write-Host "Step 1: Creating temporary clean copy..." -ForegroundColor Yellow

    # Create temp directory and copy files (excluding .git)
    New-Item -ItemType Directory -Path $TEMP_DIR -Force | Out-Null

    # Copy all files except .git, sources, dicts_test, bin
    Get-ChildItem -Path . -Exclude ".git", "sources", "dicts_test", "bin", "__pycache__", "*.egg-info", ".venv", "venv" |
        Copy-Item -Destination $TEMP_DIR -Recurse -Force

    Set-Location $TEMP_DIR

    Write-Host "Step 2: Initializing fresh git repo..." -ForegroundColor Yellow
    git init
    git config user.email "sinanm89@gmail.com"
    git config user.name "Sinan Midillili"

    Write-Host "Step 3: Creating single initial commit..." -ForegroundColor Yellow
    git add -A

    $commitMessage = @"
feat: initial release - multi-language lexicon toolkit

ditong is a multi-language lexicon toolkit for building cross-language
word dictionaries with comprehensive metadata tracking.

Features:
- Multi-language normalization (Turkish, German, French, Spanish, etc.)
- Pluggable ingestors (Hunspell, plain text)
- Per-word metadata tracking (sources, categories, languages)
- Synthesis builder for cross-language word unions
- Pluggable IPA transcription backends
- Dual implementation (Python and Go)

Licensed under GPL-3.0 with commercial licensing available.
"@

    git commit -m $commitMessage

    Write-Host "Step 4: Adding GitHub remote..." -ForegroundColor Yellow
    git remote add origin $GITHUB_REMOTE

    Write-Host "Step 5: Pushing to GitHub..." -ForegroundColor Yellow
    git branch -M main
    git push -u origin main --force

    Write-Host ""
    Write-Host "=== Done! ===" -ForegroundColor Green
    Write-Host "GitHub repository initialized: https://github.com/sinanm89/ditong" -ForegroundColor Green
    Write-Host ""
    Write-Host "Future updates will be pushed automatically on version tags via CI."
}
finally {
    # Cleanup
    Set-Location $CURRENT_DIR
    if (Test-Path $TEMP_DIR) {
        Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    }
}
