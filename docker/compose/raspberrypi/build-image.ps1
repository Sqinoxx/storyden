# Baut das Storyden-Fullstack-Image fuer den Raspberry Pi (arm64) auf dem
# Windows-PC und legt es als tar-Datei unter transfer/ ab.
#
#   .\build-image.ps1
#
# Go wird nativ nach arm64 cross-kompiliert, der Next.js-Build laeuft ebenfalls
# nativ (Ergebnis ist reines JavaScript), nur das Runtime-Image ist arm64.
# Beim ersten Mal 10-20 Minuten, danach greift der BuildKit-Cache.
#
# Voraussetzung: Docker Desktop laeuft und kann linux/arm64 bauen:
#   docker buildx ls
#   docker run --rm --platform linux/arm64 alpine uname -m   # -> aarch64

param(
    [string]$Platform = "linux/arm64",
    [string]$Repo = "storyden-rpi",
    [string]$Tag = (Get-Date -Format "yyyy-MM-dd"),
    [switch]$SkipSave
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
$outDir = Join-Path $PSScriptRoot "transfer"
$tarPath = Join-Path $outDir "storyden-arm64.tar"

if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

Write-Host "Projekt-Root : $repoRoot"
Write-Host "Plattform    : $Platform"
Write-Host "Tags         : ${Repo}:latest, ${Repo}:$Tag"
Write-Host ""

$started = Get-Date

# provenance/sbom aus: sonst legt buildx neben dem Image ein
# Attestation-Manifest ab, und "docker load" auf dem Pi bricht mit
# "does not support loading multi-platform images" ab.
docker buildx build `
    --platform $Platform `
    --provenance=false `
    --sbom=false `
    -t "${Repo}:latest" `
    -t "${Repo}:$Tag" `
    -f (Join-Path $repoRoot "docker\all\Dockerfile") `
    --load `
    $repoRoot

if ($LASTEXITCODE -ne 0) { throw "docker buildx build fehlgeschlagen (Exit $LASTEXITCODE)" }

Write-Host ""
Write-Host ("Build fertig in {0:hh\:mm\:ss}" -f ((Get-Date) - $started))

if ($SkipSave) { return }

Write-Host "Schreibe $tarPath ..."
docker save "${Repo}:latest" "${Repo}:$Tag" -o $tarPath
if ($LASTEXITCODE -ne 0) { throw "docker save fehlgeschlagen (Exit $LASTEXITCODE)" }

$sizeMb = [math]::Round((Get-Item $tarPath).Length / 1MB, 1)
Write-Host ""
Write-Host "Fertig: $tarPath ($sizeMb MB)"
Write-Host "Weiter mit: .\copy-to-pi.ps1"
