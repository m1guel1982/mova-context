# install.ps1 -- Mova Context installer for Windows.
#
# Usage: install.ps1 [-InstallDir <path>]
#   Without -InstallDir the installer asks for a directory; pressing Enter keeps the
#   default ($GOPATH\bin, or %USERPROFILE%\go\bin).

param(
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

function Write-Info($msg) { Write-Host "[Mova Installer] $msg" }
function Write-Warn($msg) { Write-Host "[Mova Installer] WARNING: $msg" -ForegroundColor Yellow }
function Write-ErrorAndExit($msg) {
    Write-Host "[Mova Installer] ERROR: $msg" -ForegroundColor Red
    if ($script:tempBinary -and (Test-Path $script:tempBinary)) {
        Remove-Item -Force $script:tempBinary -ErrorAction SilentlyContinue
    }
    exit 1
}

Write-Info "Starting installation..."

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $arch = "arm64" }

$distBinary = Join-Path $repoRoot "dist\mova-windows-$arch.exe"
$builtBinary = $null
$script:tempBinary = $null

# ---------------------------------------------------------------------------
# Binary: prebuilt or build from source
# ---------------------------------------------------------------------------
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
    $script:tempBinary = Join-Path $env:TEMP "mova-build-$PID.exe"
    $buildExit = 1
    Push-Location $repoRoot
    try {
        & go build -ldflags="-s -w" -o $script:tempBinary $cliPath
        $buildExit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    if ($buildExit -ne 0) {
        Write-ErrorAndExit "Build failed. Check the Go output above."
    }
    $builtBinary = $script:tempBinary
    Write-Info "Build succeeded: $builtBinary"
}

# ---------------------------------------------------------------------------
# Install directory (default: $GOPATH\bin)
# ---------------------------------------------------------------------------
$gopath = $null
if ($env:GOPATH) {
    $gopath = $env:GOPATH
} elseif (Get-Command go -ErrorAction SilentlyContinue) {
    try { $gopath = (& go env GOPATH 2>$null) } catch { $gopath = $null }
}
if (-not $gopath) { $gopath = Join-Path $env:USERPROFILE "go" }
# GOPATH may hold several entries separated by ';' -> use the first one
$gopath = ($gopath -split ";")[0].Trim()
$defaultBinDir = Join-Path $gopath "bin"

# Strips quotes, expands %VARS% and ~, makes the path absolute, removes trailing '\'
function Resolve-InstallDir([string]$raw) {
    $d = $raw.Trim().Trim('"').Trim("'")
    $d = [Environment]::ExpandEnvironmentVariables($d)
    if ($d -eq "~") {
        $d = $env:USERPROFILE
    } elseif ($d.StartsWith("~\") -or $d.StartsWith("~/")) {
        $d = Join-Path $env:USERPROFILE $d.Substring(2)
    }
    if (-not [System.IO.Path]::IsPathRooted($d)) {
        $d = Join-Path (Get-Location).Path $d
    }
    $d = [System.IO.Path]::GetFullPath($d)
    $root = [System.IO.Path]::GetPathRoot($d)
    if ($d.Length -gt $root.Length) { $d = $d.TrimEnd('\', '/') }
    return $d
}

# Creates the directory if needed and checks that it is writable
function Test-PrepareDir([string]$dir) {
    try {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        $probe = Join-Path $dir ".mova-write-test-$PID"
        Set-Content -Path $probe -Value "" -ErrorAction Stop
        Remove-Item -Force $probe -ErrorAction SilentlyContinue
        return $true
    } catch {
        return $false
    }
}

$binDir = $null
if ($InstallDir) {
    $binDir = Resolve-InstallDir $InstallDir
    if (-not (Test-PrepareDir $binDir)) {
        Write-ErrorAndExit "Cannot create or write to $binDir."
    }
} else {
    Write-Host ""
    while (-not $binDir) {
        $answer = $null
        $gotInput = $true
        try { $answer = Read-Host "Install directory (press Enter for default: $defaultBinDir)" } catch { $gotInput = $false }

        if (-not $gotInput -or [string]::IsNullOrWhiteSpace($answer)) {
            $candidate = $defaultBinDir
        } else {
            try { $candidate = Resolve-InstallDir $answer } catch {
                Write-Warn "Invalid path. Try another one."
                continue
            }
        }
        if (Test-PrepareDir $candidate) {
            $binDir = $candidate
        } else {
            if (-not $gotInput) { Write-ErrorAndExit "Cannot create or write to $candidate." }
            Write-Warn "Cannot create or write to $candidate. Try another path."
        }
    }
}
Write-Info "Install directory: $binDir"

$target = Join-Path $binDir "mova.exe"

# Warn if another mova.exe elsewhere on PATH could shadow the new one
$existing = Get-Command mova -All -ErrorAction SilentlyContinue |
    Where-Object { $_.CommandType -eq "Application" -and $_.Source -and ($_.Source -ine $target) } |
    Select-Object -First 1
if ($existing) {
    Write-Warn "Another mova was found at $($existing.Source). It may take precedence over $target depending on your PATH order."
}

# ---------------------------------------------------------------------------
# Stop any running mova before replacing the executable
# ---------------------------------------------------------------------------
function Stop-RunningMova {
    $procs = @(Get-Process -Name "mova" -ErrorAction SilentlyContinue)
    if ($procs.Count -eq 0) {
        Write-Info "No running mova process found."
        return
    }
    foreach ($p in $procs) {
        $procPath = "unknown path"
        try { if ($p.Path) { $procPath = $p.Path } } catch { }
        Write-Info "Stopping running mova (PID $($p.Id)) from $procPath ..."
        try {
            Stop-Process -Id $p.Id -Force -ErrorAction Stop
        } catch {
            Write-Warn "Could not stop PID $($p.Id): $($_.Exception.Message)"
        }
    }
    foreach ($p in $procs) {
        try { [void]$p.WaitForExit(5000) } catch { }
    }
    Start-Sleep -Milliseconds 500   # let Windows / antivirus release the file handle
}

# Copies with retries. If the file is still locked, Windows still lets us RENAME a
# running .exe, so we move the old one aside (mova.exe.old) and copy the new one.
function Install-Binary([string]$src, [string]$dst) {
    $old = "$dst.old"
    if (Test-Path $old) { Remove-Item -Force $old -ErrorAction SilentlyContinue }
    $maxAttempts = 8
    for ($i = 1; $i -le $maxAttempts; $i++) {
        try {
            Copy-Item -Force -Path $src -Destination $dst -ErrorAction Stop
            return
        } catch {
            if ($i -eq $maxAttempts) { throw }
            if ($i -ge 3 -and (Test-Path $dst)) {
                try { Move-Item -Force -Path $dst -Destination $old -ErrorAction Stop } catch { }
            }
            Start-Sleep -Milliseconds 500
        }
    }
}

Stop-RunningMova

try {
    Install-Binary $builtBinary $target
} catch {
    Write-ErrorAndExit "Could not install to $target. $($_.Exception.Message) Close any program using mova.exe and run the installer again."
}
Write-Info "Installed: $target"

if ($script:tempBinary -and (Test-Path $script:tempBinary)) {
    Remove-Item -Force $script:tempBinary -ErrorAction SilentlyContinue
}

# ---------------------------------------------------------------------------
# PATH and MOVA_PROJECT_ROOT (user scope)
# ---------------------------------------------------------------------------
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $currentPath) { $currentPath = "" }
$normBin = $binDir.TrimEnd('\')
$onPath = $currentPath.Split(";") |
    Where-Object { $_ } |
    ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') } |
    Where-Object { $_ -ieq $normBin }
if (-not $onPath) {
    $newPath = if ($currentPath) { $currentPath.TrimEnd(';') + ";" + $binDir } else { $binDir }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
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
$choice = "1"
try { $choice = Read-Host "Choose 1-3 and press Enter (default: 1)" } catch { $choice = "3" }
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
        $binDirQ = $binDir.Replace("'", "''")
        $repoRootQ = $repoRoot.Replace("'", "''")
        Start-Process $shell -ArgumentList "-NoExit", "-Command", "`$env:PATH += ';$binDirQ'; `$env:MOVA_PROJECT_ROOT = '$repoRootQ'; Set-Location '$repoRootQ'; mova"
    }
}