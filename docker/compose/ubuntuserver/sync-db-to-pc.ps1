# Holt einen frischen Datenbank-Dump vom Ubuntu-Server und spielt ihn in die
# lokale Postgres-Instanz auf diesem PC ein (docker-compose.yml in diesem
# Ordner). Ersetzt die Klick-Strecke aus Daten_von_Ubuntu.txt durch einen
# Aufruf:
#
#   .\sync-db-to-pc.ps1
#
# Setzt lokal laufendes Docker Desktop voraus. Die lokale Postgres-Datenbank
# wird dabei komplett neu aufgesetzt (Volume geloescht) - lokale
# Testaenderungen an der DB gehen verloren, der Dump vom Server bleibt aber
# unberuehrt.
#
# Nach dem Lauf: Backend neu starten (go run ./cmd/backend), damit es die
# neuen Daten sieht.

param(
    [string]$ServerHost = "192.168.178.105",
    [string]$ServerUser = "unidentist",
    [string]$RemoteDir = "~/storyden",
    [string]$PgUser = "storyden",
    [string]$PgDb = "storyden"
)

$ErrorActionPreference = "Stop"
$target = "${ServerUser}@${ServerHost}"
$dumpPath = Join-Path $PSScriptRoot "db.sql"

Write-Host "1/3 Dump auf dem Server erzeugen ($target) ..."
ssh $target "cd $RemoteDir && docker compose exec -T postgres pg_dump -U $PgUser -d $PgDb --no-owner --no-privileges > /tmp/db.sql"
if ($LASTEXITCODE -ne 0) { throw "pg_dump auf dem Server fehlgeschlagen (Exit $LASTEXITCODE)" }

Write-Host "2/3 Dump herunterladen ..."
scp "${target}:/tmp/db.sql" $dumpPath
if ($LASTEXITCODE -ne 0) { throw "scp fehlgeschlagen (Exit $LASTEXITCODE)" }
ssh $target "rm -f /tmp/db.sql"

$dbMb = [math]::Round((Get-Item $dumpPath).Length / 1MB, 2)
Write-Host "  -> db.sql ($dbMb MB)"

Write-Host "3/3 Lokale Postgres-DB neu aufsetzen und Dump einspielen ..."
Push-Location $PSScriptRoot
try {
    docker compose down postgres
    docker volume rm storyden_postgres-data 2>$null

    # Plain "bash" greift auf Windows zuerst WSL ab statt Git Bash - falls WSL
    # nicht eingerichtet ist, schlaegt das fehl. Git Bash direkt verwenden.
    $gitBash = "C:\Program Files\Git\bin\bash.exe"
    if (-not (Test-Path $gitBash)) { throw "Git Bash nicht gefunden unter $gitBash" }
    & $gitBash restore-db.sh db.sql
    if ($LASTEXITCODE -ne 0) { throw "restore-db.sh fehlgeschlagen (Exit $LASTEXITCODE)" }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Fertig. Jetzt Backend neu starten: go run ./cmd/backend" -ForegroundColor Yellow
