\xef\xbb\xbf# uninstall.ps1 -- Mova Context uninstaller for Windows (undoes installers\windows\install.ps1).
# Removes: mova.exe (+ mova.exe.old), the user PATH entry, the user variable MOVA_PROJECT_ROOT, temp leftovers.
# The repo folder (your projects / memory) is deleted ONLY if you confirm or pass -RemoveRepo.
# Usage: uninstall.ps1 [-InstallDir <path>] [-RemoveRepo | -KeepRepo] [-PurgePath] [-Yes] [-DryRun]
#   -PurgePath  also drop the PATH entry when the bin folder still holds other programs (default: keep it)
param(
    [string]$InstallDir = "",
    [switch]$RemoveRepo, [switch]$KeepRepo, [switch]$PurgePath, [switch]$Yes, [switch]$DryRun
)
$ErrorActionPreference = "Continue"
function Info($m) { Write-Host "[Mova Uninstaller] $m" }
function Warn($m) { Write-Host "[Mova Uninstaller] WARNING: $m" -ForegroundColor Yellow }
function Do-It([string]$what, [scriptblock]$act) { if ($DryRun) { Info "[dry-run] $what" } else { Info $what; & $act } }

$selfRepo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# --- locate binary folder
$binDir = $null
if ($InstallDir) { $binDir = $InstallDir.Trim('"').TrimEnd('\') }
else {
    $cmd = Get-Command mova -All -ErrorAction SilentlyContinue | Where-Object { $_.CommandType -eq "Application" } | Select-Object -First 1
    if ($cmd) { $binDir = Split-Path -Parent $cmd.Source }
    else {
        $gp = $env:GOPATH; if (-not $gp) { $gp = Join-Path $env:USERPROFILE "go" }
        $cand = Join-Path (($gp -split ";")[0]) "bin"
        if (Test-Path (Join-Path $cand "mova.exe")) { $binDir = $cand }
    }
}
if ($binDir) { Info "Binary folder: $binDir" } else { Warn "No installed mova.exe found; cleaning the rest." }

# --- locate repo (MOVA_PROJECT_ROOT, user scope)
$repo = [Environment]::GetEnvironmentVariable("MOVA_PROJECT_ROOT", "User")
if (-not $repo) { $repo = $env:MOVA_PROJECT_ROOT }
if (-not $repo -and (Test-Path (Join-Path $selfRepo "workflow.md"))) { $repo = $selfRepo }
Info "Repo folder:   $(if ($repo) { $repo } else { '(none found)' })"

# --- 1. stop running mova
foreach ($p in @(Get-Process -Name "mova" -ErrorAction SilentlyContinue)) {
    Do-It "Stopping mova (PID $($p.Id))" { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
}
if (-not $DryRun) { Start-Sleep -Milliseconds 500 }

# --- 2. binary
$others = 0
if ($binDir -and (Test-Path $binDir)) {
    foreach ($n in @("mova.exe", "mova.exe.old")) {
        $f = Join-Path $binDir $n
        if (Test-Path $f) { Do-It "Deleting $f" { Remove-Item -Force $f -ErrorAction SilentlyContinue } }
    }
    $others = @(Get-ChildItem -Force $binDir -ErrorAction SilentlyContinue | Where-Object { $_.Name -notin @("mova.exe", "mova.exe.old") }).Count
}

# --- 3. user environment: PATH entry + MOVA_PROJECT_ROOT
if ($binDir) {
    if ($others -eq 0 -or $PurgePath) {
        $cur = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($cur) {
            $norm = $binDir.TrimEnd('\')
            $keep = @($cur.Split(";") | Where-Object { $_ -and ([Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') -ine $norm) })
            if ($keep.Count -ne @($cur.Split(";") | Where-Object { $_ }).Count) {
                Do-It "Removing $binDir from the user PATH" { [Environment]::SetEnvironmentVariable("Path", ($keep -join ";"), "User") }
            }
        }
    } else { Info "Kept $binDir on PATH: it holds $others other item(s). Use -PurgePath to drop it." }
}
if ([Environment]::GetEnvironmentVariable("MOVA_PROJECT_ROOT", "User")) {
    Do-It "Removing user variable MOVA_PROJECT_ROOT" { [Environment]::SetEnvironmentVariable("MOVA_PROJECT_ROOT", $null, "User") }
}
if (-not $DryRun) { Remove-Item Env:\MOVA_PROJECT_ROOT -ErrorAction SilentlyContinue }
if (-not $DryRun -and $binDir -and (Test-Path $binDir) -and -not (Get-ChildItem -Force $binDir -ErrorAction SilentlyContinue)) {
    Info "Removing empty folder $binDir"; Remove-Item -Force $binDir -ErrorAction SilentlyContinue
}

# --- 4. temp leftovers
foreach ($t in @(Get-ChildItem -Path $env:TEMP -Force -ErrorAction SilentlyContinue | Where-Object { $_.Name -like "mova-trace-*" -or $_.Name -like "mova-build-*" })) {
    Do-It "Deleting temp $($t.FullName)" { Remove-Item -Recurse -Force $t.FullName -ErrorAction SilentlyContinue }
}

# --- 5. repo folder (explicit)
if ($repo -and (Test-Path $repo) -and -not $KeepRepo) {
    $isRepo = (Test-Path (Join-Path $repo "workflow.md")) -and (Test-Path (Join-Path $repo "src"))
    $isRoot = ([System.IO.Path]::GetPathRoot($repo).TrimEnd('\') -ieq $repo.TrimEnd('\')) -or ($repo.TrimEnd('\') -ieq $env:USERPROFILE.TrimEnd('\'))
    if (-not $isRepo -or $isRoot) { Warn "$repo does not look like a mova repo (workflow.md + src\ required) -> NOT deleted." }
    else {
        $go = $RemoveRepo
        if (-not $go -and -not $Yes) { $a = Read-Host "Delete repo $repo (your projects\ and memory.md inside are lost)? [y/N]"; $go = ($a -match '^(y|yes)$') }
        if ($go) { Set-Location $env:USERPROFILE; Do-It "Deleting $repo" { Remove-Item -Recurse -Force $repo -ErrorAction SilentlyContinue } }
        else { Info "Kept $repo" }
    }
}

if (-not $DryRun) {
    $left = $false
    if ([Environment]::GetEnvironmentVariable("MOVA_PROJECT_ROOT", "User")) { Warn "MOVA_PROJECT_ROOT still set"; $left = $true }
    if ($binDir -and (Test-Path (Join-Path $binDir "mova.exe"))) { Warn "mova.exe still at $binDir"; $left = $true }
    if (-not $left) { Info "Clean: no mova.exe or MOVA_PROJECT_ROOT left." }
    Info "Open a NEW terminal for the environment change to apply."
}
