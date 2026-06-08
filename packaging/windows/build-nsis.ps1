param(
    [string]$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path,
    [string]$OutDir = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path "dist\installers")
)

$ErrorActionPreference = "Stop"

$manifest = Get-Content (Join-Path $RootDir "package.json") -Raw | ConvertFrom-Json
$version = $manifest.version

$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($null -eq $makensis) {
    throw "makensis was not found in PATH. Install NSIS 3 and retry."
}

foreach ($binary in @("qkbox.exe", "qkbox-window.exe", "qkbox-provider.exe")) {
    $path = Join-Path $RootDir "bin\$binary"
    if (-not (Test-Path $path)) {
        throw "Missing $binary. Run npm run build before packaging."
    }
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$installer = Join-Path $OutDir "qkbox-$version-setup.exe"

& $makensis.Source `
    "/DROOT_DIR=$RootDir" `
    "/DOUT_DIR=$OutDir" `
    "/DAPP_VERSION=$version" `
    (Join-Path $PSScriptRoot "qkbox.nsi")

if ($LASTEXITCODE -ne 0) {
    throw "makensis failed with exit code $LASTEXITCODE"
}

if (-not (Test-Path $installer)) {
    throw "NSIS completed but installer was not created: $installer"
}

Write-Host "Created $installer"
