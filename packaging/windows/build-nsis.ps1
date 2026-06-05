param(
    [string]$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path,
    [string]$OutDir = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path "dist\installers")
)

$ErrorActionPreference = "Stop"

$manifest = Get-Content (Join-Path $RootDir "package.json") -Raw | ConvertFrom-Json
$version = $manifest.version

foreach ($binary in @("qkbox.exe", "qkboxd.exe", "qkbox-provider.exe")) {
    $path = Join-Path $RootDir "bin\$binary"
    if (-not (Test-Path $path)) {
        throw "Missing $binary. Run npm run build before packaging."
    }
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

& makensis `
    "/DROOT_DIR=$RootDir" `
    "/DOUT_DIR=$OutDir" `
    "/DAPP_VERSION=$version" `
    (Join-Path $PSScriptRoot "qkbox.nsi")

if ($LASTEXITCODE -ne 0) {
    throw "makensis failed with exit code $LASTEXITCODE"
}
