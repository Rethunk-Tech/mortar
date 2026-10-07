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

# Autostart is a setting of the running app; the CLI talks to it over its control socket.
$app = Start-Process $exe -PassThru
$set = $false
foreach ($i in 1..30) {
    Start-Sleep 2
    & $exe settings set launchAtLogin true *>$null
    if ($LASTEXITCODE -eq 0) { $set = $true; break }
}
Check 'launchAtLogin set through the running app' $set
# A cold first start can take a while to apply the setting, so wait on state: the setting reads back true, then the Run key appears.
$reported = $false
if ($set) {
    foreach ($i in 1..30) {
        if ((& $exe settings get launchAtLogin 2>&1 | Out-String) -match '(?m)^launchAtLogin\s+true\s*$') { $reported = $true; break }
        Start-Sleep 2
    }
}
$run = $null
if ($reported) {
    foreach ($i in 1..30) {
        $run = Get-ItemProperty $runKey -Name Mortar -ErrorAction SilentlyContinue
        if ($run) { break }
        Start-Sleep 1
    }
}
$why = if (-not $set) { 'launchAtLogin never set' } elseif (-not $reported) { 'launchAtLogin never read back true within 60s' } else { 'Run key not written within 30s of launchAtLogin true' }
Check $(if ($null -ne $run) { 'autostart Run key written' } else { "autostart Run key written ($why)" }) ($null -ne $run)
Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue
Start-Sleep 2

# The uninstaller copies itself to %TEMP% and returns at once, so wait for the directory to go.
Start-Process (Join-Path $dir 'uninstall.exe') '/S' -Wait
foreach ($i in 1..60) { if (-not (Test-Path $dir)) { break }; Start-Sleep 2 }
Check 'install dir removed' (-not (Test-Path $dir))
Check 'Start Menu shortcut removed' (-not ($startMenu | Where-Object { Test-Path $_ }))
Check 'autostart Run key removed' ($null -eq (Get-ItemProperty $runKey -Name Mortar -ErrorAction SilentlyContinue))
Check 'uninstall registry entry removed' ($null -eq (Uninstall-Entry))
"failures: $fails" | Tee-Object -FilePath $Out -Append
exit $fails
