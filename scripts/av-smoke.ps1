# Windows acceptance for the antivirus scan of extracted mods (docs/architecture.md, "Antivirus scan"). Run it in the
# test VM with Defender on, from the repository root:  powershell -ExecutionPolicy Bypass -File scripts\av-smoke.ps1
#
# 1. The AMSI scanner flags the EICAR test string (go test, skipped when no antivirus is registered).
# 2. It builds eicar-test.zip on the desktop for the manual half: add it to a profile in Mortar (Mods > Add archive, or
#    drop it on the window). Expect the install refused with "the antivirus (AMSI) reports ... in eicar.com", a toast or
#    a queue row with Install anyway, and, after the confirm, the mod installed and the profile's History showing
#    "Installed ... although the antivirus flagged it".
# The EICAR string is assembled below from parts, so this file never holds it whole and Defender leaves the checkout alone.
$ErrorActionPreference = 'Stop'

$defender = Get-MpComputerStatus -ErrorAction SilentlyContinue
if ($defender -and -not $defender.RealTimeProtectionEnabled) {
    Write-Warning 'Defender real-time protection is off; AMSI may report no detection.'
}

Write-Host '== AMSI scanner test'
$env:GOTMPDIR = $env:TEMP
go test -count=1 -run TestAMSIFlagsTheEicarString -v ./internal/avscan
if ($LASTEXITCODE -ne 0) { throw 'the AMSI scanner test failed' }

Write-Host '== Building the EICAR zip'
$parts = 'X5O!P%@AP[4\PZX54(P^)7CC)7}$', 'EICAR-STANDARD-ANTIVIRUS-TEST-FILE', '!$H+H*'
$work = Join-Path $env:TEMP ('mortar-av-smoke-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
try {
    $inner = Join-Path $work 'EicarMod'
    New-Item -ItemType Directory -Path $inner | Out-Null
    Set-Content -Path (Join-Path $inner 'manifest.json') -Value '{"Name":"Eicar Test","UniqueID":"Mortar.EicarTest","Version":"1.0.0","MinimumApiVersion":"4.0.0"}'
    [IO.File]::WriteAllText((Join-Path $inner 'eicar.com'), -join $parts)
    $zip = Join-Path ([Environment]::GetFolderPath('Desktop')) 'eicar-test.zip'
    if (Test-Path $zip) { Remove-Item $zip }
    Compress-Archive -Path $inner -DestinationPath $zip
    Write-Host "Wrote $zip. Defender may quarantine it on the desktop; add it to Mortar straight away, or restore it from Protection history."
} finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}
