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

# Autostart is a setting of the running app; the CLI talks to it over its control socket, which a cold first start
# opens a few seconds in, so the set is issued until the app accepts it, once. Then poll under one deadline for the
# setting reading back true, an autostart entry (Run key value or Startup shortcut) and the app process. The output is piped
# to Out-String rather than assigned to $null: Windows PowerShell 5.1 leaves $LASTEXITCODE unset after `$null = & exe 2>&1`.
function Get-LaunchAtLogin { (& $exe settings get launchAtLogin 2>&1 | Out-String) -match '(?m)^launchAtLogin\s+true\s*$' }
$startupLink = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\Mortar.lnk"
function Get-Autostart { if (Get-ItemProperty $runKey -Name Mortar -ErrorAction SilentlyContinue) { 'Run key' } elseif (Test-Path $startupLink) { 'Startup shortcut' } }
Start-Process $exe
$set = $false
$autostart = $null
$running = $false
$value = $false
$deadline = (Get-Date).AddSeconds(120)
do {
    if (-not $set) {
        $null = (& $exe settings set launchAtLogin true 2>&1 | Out-String)
        $set = ($LASTEXITCODE -eq 0)
    }
    if ($set) { $value = Get-LaunchAtLogin }
    $autostart = Get-Autostart
    $running = [bool](Get-Process -Name mortar -ErrorAction SilentlyContinue)
    if ($set -and $value -and $autostart -and $running) { break }
    Start-Sleep 1
} while ((Get-Date) -lt $deadline)
Check 'launchAtLogin set through the running app (within 120s)' $set
$state = if ($autostart) { $autostart } else { 'setting reads: ' + ((& $exe settings get launchAtLogin 2>&1 | Out-String) -replace '\s+', ' ').Trim() }
Check 'launchAtLogin reads back true' $value
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
