# Kopiert Image, Datenbank-Dump, Uploads und Konfiguration per SSH auf den
# Ubuntu-Server und laedt das Image dort in Docker.
#
#   .\copy-to-server.ps1 -ServerUser unidentist
#
# Fuer reine Code-Updates (Konfiguration und Daten bleiben unveraendert):
#   .\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
#
# db.sql und data.tar.gz kommen aus transfer/ - vorher .\export-from-pi.ps1
# ausfuehren. Startet auf dem Server nichts - die Schritte danach stehen in
# Anleitung.md.

param(
    [string]$ServerHost = "192.168.178.105",
    [string]$ServerUser = "unidentist",
    [string]$RemoteDir = "",
    [switch]$ImageOnly,
    [switch]$SkipImage
)

$ErrorActionPreference = "Stop"

if ($RemoteDir -eq "") { $RemoteDir = "/home/$ServerUser/storyden" }
$target = "${ServerUser}@${ServerHost}"
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
    Send-File (Join-Path $PSScriptRoot "backup.sh") "backup.sh"
    Send-File (Join-Path $PSScriptRoot "restore-db.sh") "restore-db.sh"
    Send-File (Join-Path $PSScriptRoot "ddns-config.example.json") "ddns-config.example.json"
    Send-File (Join-Path $PSScriptRoot ".env") ".env"
    Send-File (Join-Path $transfer "db.sql") "db.sql"
    Send-File (Join-Path $transfer "data.tar.gz") "data.tar.gz"
}

if (-not $SkipImage) {
    Send-File (Join-Path $transfer "storyden-amd64.tar") "storyden-amd64.tar"

    Write-Host ""
    Write-Host "Lade Image auf dem Server (dauert 1-3 Minuten) ..."
    ssh $target "docker load -i '$RemoteDir/storyden-amd64.tar'"
    if ($LASTEXITCODE -ne 0) { throw "docker load auf dem Server fehlgeschlagen (Exit $LASTEXITCODE)" }
}

if (-not $ImageOnly) {
    ssh $target "chmod 600 '$RemoteDir/.env' 2>/dev/null; chmod +x '$RemoteDir'/*.sh 2>/dev/null; true"
}

Write-Host ""
Write-Host "Uebertragung fertig."
Write-Host ""
if ($ImageOnly) {
    Write-Host "Naechster Schritt auf dem Server:"
    Write-Host "  cd $RemoteDir && docker compose up -d"
} else {
    Write-Host "Naechste Schritte auf dem Server (siehe Anleitung.md):"
    Write-Host "  ssh $target"
    Write-Host "  cd $RemoteDir"
    Write-Host "  nano .env                        # Werte pruefen"
    Write-Host "  mkdir -p data && sudo tar xzf data.tar.gz -C data && sudo chown -R 1001:1001 data"
    Write-Host "  bash restore-db.sh db.sql"
    Write-Host "  docker compose up -d"
}
