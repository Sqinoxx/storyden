# Kopiert Image, Datenbank-Dump, Uploads und Konfiguration per SSH auf den Pi
# und laedt das Image dort in Docker.
#
#   .\copy-to-pi.ps1 -PiUser pi
#
# Fuer reine Code-Updates (Konfiguration und Daten bleiben unveraendert):
#   .\copy-to-pi.ps1 -PiUser pi -ImageOnly
#
# Startet auf dem Pi nichts - die Reihenfolge beim ersten Mal steht in
# README.md (erst Postgres, dann Restore, dann alles).

param(
    [string]$PiHost = "192.168.178.30",
    [string]$PiUser = "pi",
    [string]$RemoteDir = "",
    [switch]$ImageOnly,
    [switch]$SkipImage
)

$ErrorActionPreference = "Stop"

if ($RemoteDir -eq "") { $RemoteDir = "/home/$PiUser/storyden" }
$target = "${PiUser}@${PiHost}"
$transfer = Join-Path $PSScriptRoot "transfer"

function Send-File($localPath, $label) {
    if (-not (Test-Path $localPath)) {
        Write-Warning "$label nicht gefunden ($localPath) - uebersprungen."
        return
    }
    $mb = [math]::Round((Get-Item $localPath).Length / 1MB, 1)
    Write-Host "-> $label ($mb MB)"
    # -C komprimiert unterwegs, das spart beim Image-tar deutlich Zeit.
    scp -C $localPath "${target}:$RemoteDir/"
    if ($LASTEXITCODE -ne 0) { throw "scp von $label fehlgeschlagen (Exit $LASTEXITCODE)" }
}

Write-Host "Ziel: $target : $RemoteDir"
Write-Host ""

ssh $target "mkdir -p '$RemoteDir'"
if ($LASTEXITCODE -ne 0) { throw "SSH-Verbindung zu $target fehlgeschlagen" }

if (-not $ImageOnly) {
    Send-File (Join-Path $PSScriptRoot "docker-compose.yml") "docker-compose.yml"
    Send-File (Join-Path $PSScriptRoot "Caddyfile") "Caddyfile"
    Send-File (Join-Path $PSScriptRoot "restore-db.sh") "restore-db.sh"
    Send-File (Join-Path $PSScriptRoot "backup.sh") "backup.sh"
    Send-File (Join-Path $PSScriptRoot "ddns-config.example.json") "ddns-config.example.json"
    Send-File (Join-Path $transfer ".env") ".env"
    Send-File (Join-Path $transfer "db.sql") "db.sql"
    Send-File (Join-Path $transfer "data.tar.gz") "data.tar.gz"
}

if (-not $SkipImage) {
    Send-File (Join-Path $transfer "storyden-arm64.tar") "storyden-arm64.tar"

    Write-Host ""
    Write-Host "Lade Image auf dem Pi (dauert 1-3 Minuten) ..."
    ssh $target "docker load -i '$RemoteDir/storyden-arm64.tar'"
    if ($LASTEXITCODE -ne 0) { throw "docker load auf dem Pi fehlgeschlagen (Exit $LASTEXITCODE)" }
}

if (-not $ImageOnly) {
    ssh $target "chmod 600 '$RemoteDir/.env' 2>/dev/null; chmod +x '$RemoteDir'/*.sh 2>/dev/null; true"
}

Write-Host ""
Write-Host "Uebertragung fertig."
Write-Host ""
if ($ImageOnly) {
    Write-Host "Naechster Schritt auf dem Pi:"
    Write-Host "  cd $RemoteDir && docker compose up -d"
} else {
    Write-Host "Naechste Schritte auf dem Pi (siehe README.md, Schritt 7):"
    Write-Host "  ssh $target"
    Write-Host "  cd $RemoteDir"
    Write-Host "  nano .env                        # Domain-Zeilen pruefen"
    Write-Host "  mkdir -p ddns && cp ddns-config.example.json ddns/config.json && nano ddns/config.json"
    Write-Host "  sudo chown -R 1000:1000 ddns"
    Write-Host "  mkdir -p data && sudo tar xzf data.tar.gz -C data && sudo chown -R 1001:1001 data"
    Write-Host "  docker compose up -d postgres"
    Write-Host "  bash restore-db.sh db.sql"
    Write-Host "  docker compose up -d"
}
