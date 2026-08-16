# Baut das Storyden-Fullstack-Image fuer den Ubuntu-Server (amd64) auf dem
# Windows-PC und legt es als tar-Datei unter transfer/ ab.
#
#   .\build-image.ps1
#
# Go wird nativ nach amd64 kompiliert, der Next.js-Build laeuft ebenfalls
# nativ. Beim ersten Mal ein paar Minuten, danach greift der BuildKit-Cache.
#
# Voraussetzung: Docker Desktop laeuft.
#
# Ist der Server KEIN x86_64/amd64-System (z.B. eine ARM-VM), stattdessen
# -Platform linux/arm64 uebergeben. Pruefen mit "uname -m" auf dem Server:
# x86_64 -> amd64, aarch64 -> arm64.

param(
    [string]$Platform = "linux/amd64",
    [string]$Repo = "storyden-server",
    [string]$Tag = (Get-Date -Format "yyyy-MM-dd"),
    [switch]$SkipSave
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
$outDir = Join-Path $PSScriptRoot "transfer"
$tarPath = Join-Path $outDir "storyden-amd64.tar"

if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

Write-Host "Projekt-Root : $repoRoot"
Write-Host "Plattform    : $Platform"
Write-Host "Tags         : ${Repo}:latest, ${Repo}:$Tag"
Write-Host ""

$started = Get-Date

# provenance/sbom aus: sonst legt buildx neben dem Image ein
# Attestation-Manifest ab, und "docker load" auf dem Server bricht mit
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
Write-Host "Weiter mit: .\copy-to-server.ps1"
