# Windows acceptance for the antivirus scan of extracted mods (docs/architecture.md, "Antivirus scan"). Needs only the
# installed Mortar, running, with Defender on:
#   powershell -ExecutionPolicy Bypass -File scripts\av-smoke.ps1
# Installs an EICAR zip into a throwaway profile through the CLI, expects it refused as malware, then installs it with
# --allow-unscanned. The EICAR string is assembled from parts, so this file never holds it whole and Defender leaves the
# script alone; Defender may still quarantine the zip, so it is rewritten right before each install.
# Exits with the number of failed checks.
param([string]$Exe = "$env:LOCALAPPDATA\Programs\Mortar\mortar.exe", [string]$Out = "$env:LOCALAPPDATA\Temp\mortar-av-smoke.txt")
$ErrorActionPreference = 'Continue'
$fails = 0
Remove-Item $Out -ErrorAction SilentlyContinue
function Check($name, [bool]$ok) {
    $line = '{0,-5} {1}' -f $(if ($ok) { 'PASS' } else { 'FAIL' }), $name
    if (-not $ok) { $script:fails++ }
    $line | Tee-Object -FilePath $Out -Append
}
function Finish { "failures: $fails" | Tee-Object -FilePath $Out -Append; exit $fails }

$game = 'stardew'
$profile = 'av-smoke-' + [guid]::NewGuid().ToString('N').Substring(0, 8)
$zip = Join-Path $env:TEMP 'eicar-test.zip'

function Write-Zip {
    $parts = 'X5O!P%@AP[4\PZX54(P^)7CC)7}$', 'EICAR-STANDARD-ANTIVIRUS-TEST-FILE', '!$H+H*'
    $work = Join-Path $env:TEMP ('mortar-av-smoke-' + [guid]::NewGuid().ToString('N'))
    $inner = Join-Path $work 'EicarMod'
    New-Item -ItemType Directory -Path $inner -Force | Out-Null
    Set-Content -Path (Join-Path $inner 'manifest.json') -Value '{"Name":"Eicar Test","UniqueID":"Mortar.EicarTest","Version":"1.0.0","MinimumApiVersion":"4.0.0"}'
    [IO.File]::WriteAllText((Join-Path $inner 'eicar.com'), -join $parts)
    Remove-Item $zip -ErrorAction SilentlyContinue
    Compress-Archive -Path $inner -DestinationPath $zip
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}

Check "mortar.exe present ($Exe)" (Test-Path $Exe)
if (-not (Test-Path $Exe)) { Finish }
if (-not (Get-Process -Name mortar -ErrorAction SilentlyContinue)) { Start-Process $Exe }

# The CLI talks to the running app over its control socket, which a cold start opens a few seconds in.
$created = $false
foreach ($i in 1..60) {
    $null = (& $Exe profile create $game $profile 2>&1 | Out-String)
    if ($LASTEXITCODE -eq 0) { $created = $true; break }
    Start-Sleep 2
}
Check "profile '$profile' created through the running app" $created
if (-not $created) { Finish }

Write-Zip
Check "eicar-test.zip written ($zip)" (Test-Path $zip)

$refused = (& $Exe --json install $game $profile $zip 2>&1 | Out-String)
$refusedCode = $LASTEXITCODE
Check "install refused (exit $refusedCode)" ($refusedCode -ne 0)
Check 'refusal has kind malware' ($refused -match '"kind":\s*"malware"')

Write-Zip
$anyway = (& $Exe install $game $profile $zip --allow-unscanned 2>&1 | Out-String)
Check "install --allow-unscanned succeeds ($($anyway.Trim()))" (($LASTEXITCODE -eq 0) -and ($anyway -match 'Installed'))
$history = (& $Exe profile history $game $profile 2>&1 | Out-String)
Check 'history records the override' ($history -match 'although the antivirus flagged')

$null = (& $Exe profile delete $game $profile 2>&1 | Out-String)
Remove-Item $zip -ErrorAction SilentlyContinue
Finish
