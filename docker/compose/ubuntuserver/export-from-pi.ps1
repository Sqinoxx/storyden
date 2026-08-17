# Holt Datenbank-Dump, Uploads und Secrets vom Raspberry Pi (der aktuellen
# Live-Instanz von unidentists.social) und legt sie unter transfer/ ab.
#
#   .\export-from-pi.ps1 -PiUser pi
#
# Vorstufe zu build-image.ps1 + copy-to-server.ps1. Der Dump wird auf dem Pi
# selbst geschrieben und per scp geholt statt per PowerShell-Redirect ueber
# SSH - sonst werden Umlaute in Forenbeitraegen durch eine
# Textencoding-Umwandlung beschaedigt (siehe export-data.ps1 im
# raspberrypi-Verzeichnis fuer denselben Hinweis).

param(
    [string]$PiHost = "192.168.178.30",
    [string]$PiUser = "pi",
    [string]$RemoteDir = "",
    [switch]$SkipEnv
)

$ErrorActionPreference = "Stop"

if ($RemoteDir -eq "") { $RemoteDir = "/home/$PiUser/storyden" }
$target = "${PiUser}@${PiHost}"
$outDir = Join-Path $PSScriptRoot "transfer"

if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

function Read-RemoteEnv($sshTarget, $remotePath) {
    $map = @{}
    $lines = ssh $sshTarget "cat '$remotePath' 2>/dev/null"
    foreach ($line in $lines) {
        $t = $line.Trim()
        if ($t.Length -eq 0 -or $t.StartsWith("#")) { continue }
        $i = $t.IndexOf("=")
        if ($i -lt 1) { continue }
        $map[$t.Substring(0, $i)] = $t.Substring($i + 1)
    }
    return $map
}

Write-Host "Quelle: $target : $RemoteDir"
Write-Host ""

Write-Host "Lese .env des Pi ..."
$old = Read-RemoteEnv $target "$RemoteDir/.env"
if ($old.Count -eq 0) { throw "Konnte $RemoteDir/.env auf dem Pi nicht lesen - SSH-Zugang und RemoteDir pruefen." }

$pgUser = $old["POSTGRES_USER"]
if (-not $pgUser) { $pgUser = "storyden" }
$pgDb = $old["POSTGRES_DB"]
if (-not $pgDb) { $pgDb = "storyden" }

# --- 1. Datenbank ---

Write-Host "Dump auf dem Pi erzeugen (DB: $pgDb, User: $pgUser) ..."
ssh $target "cd '$RemoteDir' && docker compose exec -T postgres pg_dump -U '$pgUser' -d '$pgDb' --no-owner --no-privileges > /tmp/storyden-export.sql"
if ($LASTEXITCODE -ne 0) { throw "pg_dump auf dem Pi fehlgeschlagen (Exit $LASTEXITCODE)" }

$dbOut = Join-Path $outDir "db.sql"
scp "${target}:/tmp/storyden-export.sql" $dbOut
if ($LASTEXITCODE -ne 0) { throw "scp des Dumps fehlgeschlagen (Exit $LASTEXITCODE)" }
ssh $target "rm -f /tmp/storyden-export.sql"

$dbMb = [math]::Round((Get-Item $dbOut).Length / 1MB, 2)
Write-Host "  -> db.sql ($dbMb MB)"

# --- 2. Uploads (Assets, Avatare) ---

Write-Host "Packe Uploads auf dem Pi ..."
ssh $target "cd '$RemoteDir' && sudo tar czf /tmp/storyden-data.tar.gz -C data ."
if ($LASTEXITCODE -ne 0) { throw "tar auf dem Pi fehlgeschlagen (Exit $LASTEXITCODE)" }

$dataOut = Join-Path $outDir "data.tar.gz"
scp "${target}:/tmp/storyden-data.tar.gz" $dataOut
if ($LASTEXITCODE -ne 0) { throw "scp der Uploads fehlgeschlagen (Exit $LASTEXITCODE)" }
ssh $target "sudo rm -f /tmp/storyden-data.tar.gz"

$dataMb = [math]::Round((Get-Item $dataOut).Length / 1MB, 2)
Write-Host "  -> data.tar.gz ($dataMb MB)"

# --- 3. Secrets in die lokale .env uebernehmen ---

if (-not $SkipEnv) {
    # POSTGRES_PASSWORD/USER/DB und DATABASE_URL bewusst NICHT uebernommen:
    # der neue Server bekommt ein eigenes frisches Postgres-Volume mit
    # eigenem Passwort, der SQL-Dump enthaelt keine Rollen/Passwoerter
    # (--no-owner --no-privileges). JWT_SECRET dagegen MUSS uebereinstimmen,
    # sonst werden beim Umzug alle bestehenden Sessions ungueltig.
    $carry = @(
        "JWT_SECRET",
        "EMAIL_PROVIDER", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME",
        "SMTP_PASSWORD", "SMTP_FROM_ADDRESS", "SMTP_FROM_NAME", "SMTP_USE_TLS",
        "GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON", "GOOGLE_DRIVE_CACHE_TTL"
    )

    $envPath = Join-Path $PSScriptRoot ".env"
    if (-not (Test-Path $envPath)) {
        Copy-Item (Join-Path $PSScriptRoot ".env.example") $envPath
    }

    $lines = Get-Content -LiteralPath $envPath
    $result = New-Object System.Collections.Generic.List[string]
    $filled = New-Object System.Collections.Generic.List[string]

    foreach ($line in $lines) {
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

    $content = ($result -join "`n") + "`n"
    [System.IO.File]::WriteAllText($envPath, $content, (New-Object System.Text.UTF8Encoding($false)))

    Write-Host "  -> .env aktualisiert (uebernommen: $($filled -join ', '))"
    Write-Host ""
    Write-Host "Noch selbst pruefen/setzen in .env: POSTGRES_PASSWORD, ACME_EMAIL." -ForegroundColor Yellow
    Write-Host "SITE_DOMAIN/PUBLIC_* stehen schon auf unidentists.social." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Export fertig in $outDir"
Write-Host "transfer\ enthaelt Passwoerter und ist gitignored - nicht weitergeben."
Write-Host ""
Write-Host "Weiter mit: .\build-image.ps1"
