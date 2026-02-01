# Ditong Benchmark Suite Runner
# Usage: .\run.ps1 [-Group <name>] [-Iterations <n>]

param(
    [string]$Group = "",
    [int]$Iterations = 1
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

Write-Host "=================================="
Write-Host "Ditong Benchmark Suite"
Write-Host "=================================="
Write-Host ""

# Build ditong if needed
$ditongPath = "../go/ditong.exe"
if (-not (Test-Path $ditongPath)) {
    Write-Host "Building ditong..."
    Push-Location ../go
    go build -o ditong.exe ./cmd
    Pop-Location
}

# Run benchmarks
$args = @("run", "runner.go", "--iterations", $Iterations, "--force")
if ($Group) {
    Write-Host "Running group: $Group"
    $args += "--group"
    $args += $Group
} else {
    Write-Host "Running all groups..."
}

& go $args

Write-Host ""
Write-Host "Done!"
