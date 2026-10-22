$ErrorActionPreference = 'Stop'
$Version = $env:NORIA_VERSION
if (-not $Version) { $Version = (Invoke-RestMethod 'https://api.github.com/repos/amxv/noria/releases/latest').tag_name }
$Version = $Version.TrimStart('v')
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Could not resolve a stable release' }
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
$Asset = "noria_windows_${Arch}.tar.gz"
$Root = if ($env:NORIA_INSTALL_DIR) { $env:NORIA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Noria\bin' }
if (-not [System.IO.Path]::IsPathRooted($Root)) { throw 'NORIA_INSTALL_DIR must be absolute' }
$Binary = Join-Path $Root 'noria.exe'
$Tray = Join-Path $Root 'noria-tray.exe'
if ((Test-Path $Binary) -or (Test-Path $Tray)) { throw 'Noria is already installed. Run noria upgrade to update it safely.' }
$Temp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $Temp | Out-Null
try {
    $Base = "https://github.com/amxv/noria/releases/download/v$Version"
    Invoke-WebRequest "$Base/SHA256SUMS" -OutFile (Join-Path $Temp 'SHA256SUMS')
    Invoke-WebRequest "$Base/$Asset" -OutFile (Join-Path $Temp $Asset)
    $Rows = @(Get-Content (Join-Path $Temp 'SHA256SUMS') | Where-Object { ($_ -split '\s+')[1] -eq $Asset })
    if ($Rows.Count -ne 1) { throw 'Missing or ambiguous checksum entry' }
    $Expected = ($Rows[0] -split '\s+')[0]
    if ($Expected -notmatch '^[0-9a-fA-F]{64}$') { throw 'Invalid release checksum' }
    $Actual = (Get-FileHash (Join-Path $Temp $Asset) -Algorithm SHA256).Hash
    if ($Actual -ine $Expected) { throw 'Checksum mismatch; nothing was installed' }
    if (-not (Get-Command 'tar.exe' -ErrorAction SilentlyContinue)) { throw 'Windows tar.exe is required (included in supported Windows 10/11 releases)' }
    $Archive = Join-Path $Temp $Asset
    $Names = @(& tar.exe -tzf $Archive)
    if ($LASTEXITCODE -ne 0) { throw 'Invalid release archive' }
    $Allowed = @('noria.exe', 'noria-tray.exe', 'LICENSE', 'THIRD_PARTY_NOTICES.md')
    if ($Names.Count -ne $Allowed.Count -or @($Names | Where-Object { $_ -notin $Allowed }).Count -ne 0) { throw 'Unexpected release archive contents' }
    $Unpacked = Join-Path $Temp 'unpacked'
    New-Item -ItemType Directory -Path $Unpacked | Out-Null
    & tar.exe -xzf $Archive -C $Unpacked
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path (Join-Path $Unpacked 'noria.exe')) -or -not (Test-Path (Join-Path $Unpacked 'noria-tray.exe'))) { throw 'Incomplete release archive' }
    New-Item -ItemType Directory -Path $Root -Force | Out-Null
    if (Test-Path $Binary) { throw 'Noria is already installed. Run noria upgrade to update it safely.' }
    Move-Item (Join-Path $Unpacked 'noria-tray.exe') $Tray
    Move-Item (Join-Path $Unpacked 'noria.exe') $Binary
    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($UserPath -split ';') -notcontains $Root) { [Environment]::SetEnvironmentVariable('Path', "$Root;$UserPath", 'User') }
    $env:Path = "$Root;$env:Path"
    & $Binary init
    & $Binary --version
    if ($env:NORIA_NO_LAUNCH -ne '1') {
        try {
            & $Binary preferences auto-update on | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'Automatic update preference failed' }
        } catch { Write-Warning 'Automatic updates could not be enabled. Set them from Configure in the tray.' }
        try {
            & $Binary preferences tray-at-login on | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'Tray-at-login registration failed' }
        } catch { Write-Warning 'Tray startup could not be registered. Enable it later with: noria preferences tray-at-login on' }
        & $Binary tray open
        if ($LASTEXITCODE -ne 0) { Write-Warning 'Tray did not open. Run: noria tray open' }
    }
    Write-Host 'Add your accounts: noria account add codex (or claude)'
    Write-Host 'Configure once: noria integrate codex --yes'
    Write-Host 'Then use codex or claude normally. noria stop restores direct settings.'
} finally { Remove-Item -Recurse -Force $Temp }
