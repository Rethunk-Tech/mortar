# Silent install, run, autostart, uninstall smoke for the Windows installer. Run as the interactive admin user:
#   powershell -File windows-installer-smoke.ps1 -Installer C:\path\mortar-amd64-installer.exe
# Exits non-zero when any check fails; arm64 installers need an ARM machine.
param([Parameter(Mandatory)][string]$Installer, [string]$Out = "$env:TEMP\mortar-smoke.txt")
$ErrorActionPreference = 'Continue'
$fails = 0
Remove-Item $Out -ErrorAction SilentlyContinue
function Check($name, [bool]$ok) {
    $line = '{0,-5} {1}' -f $(if ($ok) { 'PASS' } else { 'FAIL' }), $name
    if (-not $ok) { $script:fails++ }
    $line | Tee-Object -FilePath $Out -Append
}
function Uninstall-Entry { Get-ChildItem 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall', 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall' -ErrorAction SilentlyContinue |
    Where-Object { (Get-ItemProperty $_.PSPath).DisplayName -eq 'Mortar' } | Select-Object -First 1 }
$runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$startMenu = @("$env:ProgramData\Microsoft\Windows\Start Menu\Programs\Mortar.lnk", "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Mortar.lnk")

Start-Process $Installer '/S' -Wait
$entry = Uninstall-Entry
Check 'uninstall registry entry exists' ($null -ne $entry)
if (-not $entry) { exit 1 }
$props = Get-ItemProperty $entry.PSPath
$dir = Split-Path ($props.UninstallString -replace '"', '')
$exe = Join-Path $dir 'mortar.exe'
Check "mortar.exe installed in $dir" (Test-Path $exe)
Check 'Start Menu shortcut' ([bool]($startMenu | Where-Object { Test-Path $_ }))

$v = & $exe version 2>&1 | Out-String
Check "mortar version exits 0 ($($v.Trim()))" (($LASTEXITCODE -eq 0) -and ($v -match 'mortar \d'))

# Autostart is a setting of the running app; the CLI talks to it over its control socket. A cold first start takes
# a while to open that socket, and a set the app accepts while it is still starting can be lost, so poll the real
# condition under one deadline and set again until the setting reads back true: an autostart entry (Run key value or
# Startup shortcut) present and the app process running.
$startupLink = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\Mortar.lnk"
function Get-Autostart { if (Get-ItemProperty $runKey -Name Mortar -ErrorAction SilentlyContinue) { 'Run key' } elseif (Test-Path $startupLink) { 'Startup shortcut' } }
function Get-LaunchAtLogin { (& $exe settings get launchAtLogin 2>&1 | Out-String) -match '(?m)^launchAtLogin\s+true\s*$' }
Start-Process $exe
$set = $false
$autostart = $null
$running = $false
$deadline = (Get-Date).AddSeconds(120)
do {
    if (-not (Get-LaunchAtLogin)) {
        & $exe settings set launchAtLogin true *>$null
        if ($LASTEXITCODE -eq 0) { $set = $true }
    }
    $autostart = Get-Autostart
    $running = [bool](Get-Process -Name mortar -ErrorAction SilentlyContinue)
    if ($set -and $autostart -and $running) { break }
    Start-Sleep 1
} while ((Get-Date) -lt $deadline)
Check 'launchAtLogin set through the running app (within 120s)' $set
$state = if ($autostart) { $autostart } else { 'setting reads: ' + ((& $exe settings get launchAtLogin 2>&1 | Out-String) -replace '\s+', ' ').Trim() }
Check "autostart entry written ($state)" ($null -ne $autostart)
Check 'Mortar still running with autostart enabled' $running
Stop-Process -Name mortar -Force -ErrorAction SilentlyContinue
Start-Sleep 2

# The uninstaller copies itself to %TEMP% and returns at once, so wait for the directory to go.
Start-Process (Join-Path $dir 'uninstall.exe') '/S' -Wait
foreach ($i in 1..60) { if (-not (Test-Path $dir)) { break }; Start-Sleep 2 }
Check 'install dir removed' (-not (Test-Path $dir))
Check 'Start Menu shortcut removed' (-not ($startMenu | Where-Object { Test-Path $_ }))
Check 'autostart entry removed' ($null -eq (Get-Autostart))
Check 'uninstall registry entry removed' ($null -eq (Uninstall-Entry))
"failures: $fails" | Tee-Object -FilePath $Out -Append
exit $fails
