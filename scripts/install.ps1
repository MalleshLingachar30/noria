$ErrorActionPreference = 'Stop'
$Version = $env:NORIA_VERSION
if (-not $Version) { $Version = (Invoke-RestMethod 'https://api.github.com/repos/amxv/noria/releases/latest').tag_name }
$Version = $Version.TrimStart('v')
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Could not resolve a stable release' }
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
$Asset = "noria_windows_${Arch}.exe"
$Root = if ($env:NORIA_INSTALL_DIR) { $env:NORIA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Noria\bin' }
if (-not [System.IO.Path]::IsPathRooted($Root)) { throw 'NORIA_INSTALL_DIR must be absolute' }
$Binary = Join-Path $Root 'noria.exe'
if (Test-Path $Binary) { throw 'Noria is already installed. Run noria upgrade to update it safely.' }
$Temp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $Temp | Out-Null
try {
    $Base = "https://github.com/amxv/noria/releases/download/v$Version"
    Invoke-WebRequest "$Base/SHA256SUMS" -OutFile (Join-Path $Temp 'SHA256SUMS')
    Invoke-WebRequest "$Base/$Asset" -OutFile (Join-Path $Temp $Asset)
    $Rows = @(Get-Content (Join-Path $Temp 'SHA256SUMS') | Where-Object { ($_ -split '\s+')[1] -eq $Asset })
    if ($Rows.Count -ne 1) { throw 'Missing or ambiguous checksum entry' }
    $Expected = ($Rows[0] -split '\s+')[0]
    $Actual = (Get-FileHash (Join-Path $Temp $Asset) -Algorithm SHA256).Hash
    if ($Actual -ine $Expected) { throw 'Checksum mismatch; nothing was installed' }
    New-Item -ItemType Directory -Path $Root -Force | Out-Null
    if (Test-Path $Binary) { throw 'Noria is already installed. Run noria upgrade to update it safely.' }
    Move-Item (Join-Path $Temp $Asset) $Binary -Force
    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($UserPath -split ';') -notcontains $Root) { [Environment]::SetEnvironmentVariable('Path', "$Root;$UserPath", 'User') }
    $env:Path = "$Root;$env:Path"
    & $Binary init
    & $Binary --version
    Write-Host 'Add your accounts: noria account add codex (or claude)'
    Write-Host 'Configure once: noria integrate codex --yes'
    Write-Host 'Then use codex or claude normally. noria stop restores direct settings.'
} finally { Remove-Item -Recurse -Force $Temp }
