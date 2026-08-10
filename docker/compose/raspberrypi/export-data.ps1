# Exportiert den aktuellen Stand vom Windows-PC nach transfer/:
#
#   db.sql        Datenbank-Dump aus dem laufenden Postgres-Container
#   data.tar.gz   Assets und Avatare aus docker/compose/data
#   .env          vorgefuellte Pi-Konfiguration (Secrets aus docker/compose/.env)
#
#   .\export-data.ps1
#
# Der Dump wird im Container geschrieben und per docker cp geholt, damit
# PowerShell die Datei nicht in UTF-16 umkodiert - psql wuerde daran scheitern.

param(
    [string]$PostgresContainer = "compose-postgres-1",
    # Projekt-Root des laufenden Setups. Muss angegeben werden, wenn dieses
    # Skript aus einem git worktree heraus laeuft - .env und data/ sind
    # gitignored und liegen nur im Haupt-Checkout.
    [string]$SourceRoot = "",
    [switch]$SkipEnv
)

$ErrorActionPreference = "Stop"

if ($SourceRoot -eq "") {
    $repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
} else {
    $repoRoot = (Resolve-Path $SourceRoot).Path
}

$sourceEnv = Join-Path $repoRoot "docker\compose\.env"
$sourceData = Join-Path $repoRoot "docker\compose\data"
$outDir = Join-Path $PSScriptRoot "transfer"

if (-not (Test-Path $sourceEnv)) {
    Write-Warning "$sourceEnv nicht gefunden - .env wird nicht vorgefuellt. Bei Bedarf -SourceRoot <Projekt-Root> angeben."
}

if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

function Read-EnvFile($path) {
    $map = @{}
    if (-not (Test-Path $path)) { return $map }
    foreach ($line in Get-Content -LiteralPath $path) {
        $t = $line.Trim()
        if ($t.Length -eq 0) { continue }
        if ($t.StartsWith("#")) { continue }
        $i = $t.IndexOf("=")
        if ($i -lt 1) { continue }
        $map[$t.Substring(0, $i)] = $t.Substring($i + 1)
    }
    return $map
}

$old = Read-EnvFile $sourceEnv

$pgUser = $old["POSTGRES_USER"]
if (-not $pgUser) { $pgUser = "storyden" }
$pgDb = $old["POSTGRES_DB"]
if (-not $pgDb) { $pgDb = "storyden" }

# --- 1. Datenbank ---

$running = docker ps --filter "name=$PostgresContainer" --format "{{.Names}}"
if ($running -notcontains $PostgresContainer) {
    throw "Container '$PostgresContainer' laeuft nicht. Vorher starten: docker compose -f $repoRoot\docker\compose\docker-compose.yml up -d postgres"
}

Write-Host "Dump aus '$PostgresContainer' (DB: $pgDb, User: $pgUser) ..."
docker exec $PostgresContainer sh -c "pg_dump -U '$pgUser' -d '$pgDb' --no-owner --no-privileges -f /tmp/storyden-export.sql"
if ($LASTEXITCODE -ne 0) { throw "pg_dump fehlgeschlagen (Exit $LASTEXITCODE)" }

$dbOut = Join-Path $outDir "db.sql"
docker cp "${PostgresContainer}:/tmp/storyden-export.sql" $dbOut
if ($LASTEXITCODE -ne 0) { throw "docker cp fehlgeschlagen (Exit $LASTEXITCODE)" }
docker exec $PostgresContainer rm -f /tmp/storyden-export.sql | Out-Null

$dbMb = [math]::Round((Get-Item $dbOut).Length / 1MB, 2)
Write-Host "  -> db.sql ($dbMb MB)"

# --- 2. Assets und Avatare ---

if (Test-Path $sourceData) {
    $dataOut = Join-Path $outDir "data.tar.gz"
    Write-Host "Packe Uploads aus $sourceData ..."
    # data.db ist die alte SQLite-Datei und auf dem Pi nicht mehr relevant.
    tar -czf $dataOut -C $sourceData --exclude=data.db --exclude=.perm_check .
    if ($LASTEXITCODE -ne 0) { throw "tar fehlgeschlagen (Exit $LASTEXITCODE)" }
    $dataMb = [math]::Round((Get-Item $dataOut).Length / 1MB, 2)
    Write-Host "  -> data.tar.gz ($dataMb MB)"
} else {
    Write-Warning "$sourceData nicht gefunden - keine Uploads exportiert."
}

# --- 3. Vorgefuellte .env fuer den Pi ---

if (-not $SkipEnv) {
    $carry = @(
        "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB", "DATABASE_URL",
        "JWT_SECRET",
        "EMAIL_PROVIDER", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME",
        "SMTP_PASSWORD", "SMTP_FROM_ADDRESS", "SMTP_FROM_NAME", "SMTP_USE_TLS",
        "GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON"
    )

    $template = Get-Content -LiteralPath (Join-Path $PSScriptRoot ".env.example")
    $result = New-Object System.Collections.Generic.List[string]
    $filled = New-Object System.Collections.Generic.List[string]

    foreach ($line in $template) {
        $t = $line.Trim()
        $handled = $false
        if ($t.Length -gt 0 -and -not $t.StartsWith("#") -and $t.Contains("=")) {
            $key = $t.Substring(0, $t.IndexOf("="))
            if ($carry -contains $key -and $old.ContainsKey($key) -and $old[$key].Length -gt 0) {
                $result.Add("$key=" + $old[$key])
                $filled.Add($key)
                $handled = $true
            }
        }
        if (-not $handled) { $result.Add($line) }
    }

    $envOut = Join-Path $outDir ".env"
    [System.IO.File]::WriteAllLines($envOut, $result, (New-Object System.Text.UTF8Encoding($false)))

    Write-Host "  -> .env (uebernommen: $($filled -join ', '))"
    Write-Host ""
    Write-Host "In transfer\.env noch selbst setzen: SITE_DOMAIN, ACME_EMAIL," -ForegroundColor Yellow
    Write-Host "PUBLIC_WEB_ADDRESS, PUBLIC_API_ADDRESS." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Export fertig in $outDir"
Write-Host "transfer\ enthaelt Passwoerter und ist gitignored - nicht weitergeben."
