# Windows acceptance for the antivirus scan of extracted mods (docs/architecture.md, "Antivirus scan"). Needs only the
# installed Mortar, running, with Defender on:
#   powershell -ExecutionPolicy Bypass -File scripts\av-smoke.ps1
# Installs a flagged-sample zip into a throwaway profile through the CLI, expects it refused as malware, then installs it with
# --allow-unscanned. The test strings are assembled from parts, so this file never holds it whole and Defender leaves the
# script alone; Defender may still quarantine the zip, so it is rewritten right before each install.
# Exits with the number of failed checks.
# -ZipDir is where the test zip is written. Give it a Defender exclusion first (an admin runs
# Add-MpPreference -ExclusionPath C:\avsmoke), so the zip survives until Mortar reads it; Mortar's own staging folder is
# not excluded, so the scan of the extracted files, or real-time protection on them, must still refuse the install.
param([string]$Exe = "$env:LOCALAPPDATA\Programs\Mortar\mortar.exe", [string]$Out = "$env:LOCALAPPDATA\Temp\mortar-av-smoke.txt", [string]$ZipDir = 'C:\avsmoke')
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
New-Item -ItemType Directory -Path $ZipDir -Force | Out-Null
$zip = Join-Path $ZipDir 'eicar-test.zip'

# The detection input depends on the scanner: Defender's AMSI provider reliably reports Microsoft's documented AMSI test
# sample, clamd reports EICAR. Both are assembled from parts, so this file never holds either whole.
function Get-Sample($scanner) {
    if ($scanner -eq 'amsi') { return @('sample.txt', (-join ('AMSI Test Sample: ', '7e72c3ce-861b-4339-8740-0ac1484c1386'))) }
    return @('eicar.com', (-join ('X5O!P%@AP[4\PZX54(P^)7CC)7}$', 'EICAR-STANDARD-ANTIVIRUS-TEST-FILE', '!$H+H*')))
}

function Write-Zip($scanner) {
    $name, $body = Get-Sample $scanner
    $work = Join-Path $env:TEMP ('mortar-av-smoke-' + [guid]::NewGuid().ToString('N'))
    $inner = Join-Path $work 'FlaggedMod'
    New-Item -ItemType Directory -Path $inner -Force | Out-Null
    Set-Content -Path (Join-Path $inner 'manifest.json') -Value '{"Name":"Flagged Test","UniqueID":"Mortar.FlaggedTest","Version":"1.0.0","MinimumApiVersion":"4.0.0"}'
    [IO.File]::WriteAllText((Join-Path $inner $name), $body)
    Remove-Item $zip -ErrorAction SilentlyContinue
    Compress-Archive -Path $inner -DestinationPath $zip
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}

function Note($text) { $text | Tee-Object -FilePath $Out -Append }

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

Note '== Defender'
$mp = Get-MpComputerStatus -ErrorAction SilentlyContinue
if ($mp) { Note (($mp | Select-Object AMServiceEnabled, AntivirusEnabled, RealTimeProtectionEnabled, IsTamperProtected | Format-List | Out-String).Trim()) } else { Note 'Get-MpComputerStatus: unavailable' }
$providers = Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\AMSI\Providers' -ErrorAction SilentlyContinue
Note ('AMSI providers registered: ' + $(if ($providers) { ($providers | ForEach-Object { $_.PSChildName }) -join ', ' } else { 'none' }))
Note '== Mortar antivirus status'
$status = (& $Exe antivirus status 2>&1 | Out-String)
Note $status.Trim()
$scanner = if ($status -match '(?m)^scanner:\s*(\S+)') { $Matches[1] } else { 'unknown' }
Check "Mortar's scanner is ready ($scanner)" ($status -match '(?m)^ready:\s*true')

Write-Zip $scanner
Check "test zip written ($zip)" (Test-Path $zip)

$refused = (& $Exe --json install $game $profile $zip 2>&1 | Out-String)
$refusedCode = $LASTEXITCODE
Note "install output: $($refused.Trim())"
Check "install refused (exit $refusedCode)" ($refusedCode -ne 0)
$refusal = try { $refused | ConvertFrom-Json } catch { $null }
Check 'install refused with kind malware' ($refusal -and $refusal.kind -eq 'malware')
# Either the AMSI scan of the extracted files or Windows real-time protection on a file Mortar wrote refuses the install;
# the typed detail of the error names which.
$caught = if ($refusal -and $refusal.detail.scanner) { $refusal.detail.scanner } else { 'unknown' }
Note "refused by: $caught"

if ($refusal -and $refusal.detail.removed) {
    Note '--allow-unscanned skipped: Windows removed the file itself, so there is nothing to install anyway'
} else {
    Write-Zip $scanner
    $anyway = (& $Exe install $game $profile $zip --allow-unscanned 2>&1 | Out-String)
    Check "install --allow-unscanned succeeds ($($anyway.Trim()))" (($LASTEXITCODE -eq 0) -and ($anyway -match 'Installed'))
    $history = (& $Exe profile history $game $profile 2>&1 | Out-String)
    Check 'history records the override' ($history -match 'although the antivirus flagged')
}

$null = (& $Exe profile delete $game $profile 2>&1 | Out-String)
Remove-Item $zip -ErrorAction SilentlyContinue
Finish
