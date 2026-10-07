<#
.SYNOPSIS
    Builds the taskd TUI and puts it on your user PATH, so `taskd` works from any terminal.

.DESCRIPTION
    1. Runs the Go tests (skip with -SkipTests).
    2. Builds cmd/taskd into %LOCALAPPDATA%\Programs\taskd\taskd.exe.
    3. Adds that folder to your *user* PATH, once. No admin rights needed.

    Run it again after pulling new code to rebuild. The PATH entry is only added the first time.
    Run with -Uninstall to remove the exe and the PATH entry.

.EXAMPLE
    .\scripts\install-taskd.ps1
.EXAMPLE
    .\scripts\install-taskd.ps1 -Uninstall
#>
[CmdletBinding()]
param(
    [switch]$SkipTests,
    [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'

$RepoRoot   = Split-Path -Parent $PSScriptRoot
$InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\taskd'
$ExePath    = Join-Path $InstallDir 'taskd.exe'

# Read the user PATH straight from the registry, *without* expanding entries
# like %USERPROFILE%\go\bin. [Environment]::GetEnvironmentVariable would expand
# them, and writing that back would silently replace them with fixed paths.
function Get-UserPath {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
    try {
        return [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    } finally {
        $key.Close()
    }
}

function Set-UserPath([string]$Value) {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    try {
        $key.SetValue('Path', $Value, [Microsoft.Win32.RegistryValueKind]::ExpandString)
    } finally {
        $key.Close()
    }
    # Writing the registry directly doesn't tell Explorer that PATH changed.
    # Setting and clearing a dummy variable through .NET sends that broadcast,
    # so terminals opened from now on pick up the new PATH.
    [Environment]::SetEnvironmentVariable('TASKD_PATH_REFRESH', '1', 'User')
    [Environment]::SetEnvironmentVariable('TASKD_PATH_REFRESH', $null, 'User')
}

function Split-PathList([string]$List) {
    return @($List -split ';' | Where-Object { $_.Trim() -ne '' })
}

# Compare entries after expanding variables and dropping a trailing slash,
# so "C:\x\" and "C:\x" count as the same folder.
function Test-SameDir([string]$A, [string]$B) {
    $x = [Environment]::ExpandEnvironmentVariables($A).TrimEnd('\')
    $y = [Environment]::ExpandEnvironmentVariables($B).TrimEnd('\')
    return $x -ieq $y
}

if ($Uninstall) {
    if (Test-Path $InstallDir) {
        Remove-Item -Recurse -Force $InstallDir
        Write-Host "Removed $InstallDir"
    }
    $entries = Split-PathList (Get-UserPath)
    $kept = @($entries | Where-Object { -not (Test-SameDir $_ $InstallDir) })
    if ($kept.Count -ne $entries.Count) {
        Set-UserPath ($kept -join ';')
        Write-Host "Removed $InstallDir from your user PATH."
    }
    Write-Host 'taskd uninstalled. Open a new terminal for the PATH change to apply.'
    return
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go is not on PATH. Install it first (scoop install go).'
}
if (-not (Test-Path (Join-Path $RepoRoot 'cmd\taskd\main.go'))) {
    throw "cmd\taskd\main.go not found under $RepoRoot. Is this script in the repo's scripts folder?"
}

Push-Location $RepoRoot
try {
    if (-not $SkipTests) {
        Write-Host 'Running tests...'
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Tests failed, not installing.' }
    }

    New-Item -ItemType Directory -Force $InstallDir | Out-Null

    Write-Host "Building $ExePath ..."
    # -trimpath keeps your local folder paths out of the binary.
    # -ldflags "-s -w" drops debug symbols, so the exe is smaller.
    go build -trimpath -ldflags '-s -w' -o $ExePath ./cmd/taskd
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
} finally {
    Pop-Location
}

$entries = Split-PathList (Get-UserPath)
if ($entries | Where-Object { Test-SameDir $_ $InstallDir }) {
    Write-Host "$InstallDir is already on your user PATH."
} else {
    Set-UserPath (($entries + $InstallDir) -join ';')
    Write-Host "Added $InstallDir to your user PATH."
}

# Make `taskd` work in this window too, without reopening it.
if (-not ((Split-PathList $env:Path) | Where-Object { Test-SameDir $_ $InstallDir })) {
    $env:Path = "$env:Path;$InstallDir"
}

Write-Host ''
Write-Host 'Done. Type `taskd` in any new terminal to start the TUI.'
