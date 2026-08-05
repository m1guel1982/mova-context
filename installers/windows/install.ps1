# install.ps1 - Mova Context installer for Windows.

$ErrorActionPreference = "Stop"

function Write-Info($msg) { Write-Host "[Mova Installer] $msg" }
function Write-ErrorAndExit($msg) {
    Write-Host "[Mova Installer] ERROR: $msg" -ForegroundColor Red
    exit 1
}

Write-Info "Starting installation..."

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $arch = "arm64" }

$distBinary = Join-Path $repoRoot "dist\mova-windows-$arch.exe"
$builtBinary = $null

if (Test-Path $distBinary) {
    Write-Info "Found prebuilt binary: $distBinary"
    $builtBinary = $distBinary
} else {
    Write-Info "No prebuilt binary found - building from source (requires Go)..."
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Write-ErrorAndExit "Go is not installed or not on PATH. Install Go from https://go.dev/dl and run this installer again, or run 'make build-all' first."
    }
    $cliPath = Join-Path $repoRoot "src\cli"
    $tempBinary = Join-Path $env:TEMP "mova.exe"
    Push-Location $repoRoot
    & go build -ldflags="-s -w" -o $tempBinary $cliPath
    Pop-Location
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorAndExit "Build failed. Check the Go output above."
    }
    $builtBinary = $tempBinary
    Write-Info "Build succeeded: $builtBinary"
}

$gopath = (& go env GOPATH 2>$null)
if (-not $gopath) { $gopath = Join-Path $env:USERPROFILE "go" }
$binDir = Join-Path $gopath "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$target = Join-Path $binDir "mova.exe"
Copy-Item -Force $builtBinary $target
Write-Info "Installed: $target"

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $currentPath) { $currentPath = "" }
if (-not ($currentPath.Split(";") -contains $binDir)) {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$binDir", "User")
    Write-Info "Added $binDir to your user PATH. Open a NEW terminal window for this to take effect."
} else {
    Write-Info "$binDir is already on your PATH."
}

$currentMovaRoot = [Environment]::GetEnvironmentVariable("MOVA_PROJECT_ROOT", "User")
if (-not $currentMovaRoot) {
    [Environment]::SetEnvironmentVariable("MOVA_PROJECT_ROOT", $repoRoot, "User")
    Write-Info "Set MOVA_PROJECT_ROOT to $repoRoot - mova now works from any folder or drive."
} else {
    Write-Info "MOVA_PROJECT_ROOT is already set to $currentMovaRoot - leaving it as-is."
}

Write-Info "Done."

Write-Host ""
Write-Host "Which console would you like to open, ready to use mova?"
Write-Host "  [1] PowerShell (default)"
Write-Host "  [2] Command Prompt (CMD)"
Write-Host "  [3] Don't open one"
$choice = Read-Host "Choose 1-3 and press Enter (default: 1)"
if ([string]::IsNullOrWhiteSpace($choice)) { $choice = "1" }

switch ($choice) {
    "2" {
        Start-Process cmd.exe -ArgumentList "/K", "set PATH=%PATH%;$binDir&& set MOVA_PROJECT_ROOT=$repoRoot&& cd /d `"$repoRoot`" && mova"
    }
    "3" {
        Write-Info "OK - remember to open a NEW terminal window for the PATH/MOVA_PROJECT_ROOT change to apply."
    }
    default {
        $pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
        $shell = if ($pwsh) { "pwsh" } else { "powershell" }
        Start-Process $shell -ArgumentList "-NoExit", "-Command", "`$env:PATH += ';$binDir'; `$env:MOVA_PROJECT_ROOT = '$repoRoot'; Set-Location '$repoRoot'; mova"
    }
}