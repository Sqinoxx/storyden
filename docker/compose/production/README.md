# Produktions-Deployment: unidentists.social

Setup: eine einzelne Linux-VM mit Docker. Drei Container:

| Container  | Aufgabe                                            |
| ---------- | -------------------------------------------------- |
| `caddy`    | Reverse Proxy, HTTPS-Zertifikat automatisch         |
| `storyden` | Backend (Go) + Frontend (Next.js) in einem Prozess  |
| `postgres` | Datenbank                                           |

## Voraussetzungen

- VM mit mindestens **2 vCPU und 4 GB RAM**. Der Next.js-Build im Docker-Image
  braucht diesen Speicher; auf einer 1-GB-Micro-Instanz bricht `docker compose
  build` mit OOM ab. Oracle Ampere A1 (4 OCPU / 24 GB, Free Tier) passt.
- Docker Engine + Compose Plugin
- Die Domain, mit Zugriff auf die DNS-Einstellungen

## 1. DNS setzen (zuerst!)

Beim Domain-Provider zwei A-Records auf die oeffentliche IP der VM:

```
unidentists.social       A    <SERVER-IP>
www.unidentists.social   A    <SERVER-IP>
```

Pruefen (kann bis zu einer Stunde dauern):

```bash
dig +short unidentists.social
```

Caddy holt das Let's-Encrypt-Zertifikat beim ersten Start. Zeigt der DNS noch
nicht auf die VM, schlaegt das fehl und Let's Encrypt drosselt nach mehreren
Versuchen fuer eine Stunde.

## 2. Firewall

Bei Oracle Cloud auf **zwei** Ebenen:

1. Im Oracle-Dashboard: Networking → VCN → Security List → Ingress Rules →
   TCP 80 und 443 fuer `0.0.0.0/0` freigeben.
2. Auf der VM selbst:

```bash
sudo bash oracle-firewall.sh
```

Bei anderen Anbietern reicht in der Regel die Cloud-Firewall bzw. `ufw allow
80,443/tcp`.

## 3. Konfiguration

```bash
git clone <repo-url> storyden
cd storyden/docker/compose/production

cp .env.example .env
chmod 600 .env
nano .env
```

Pflichtwerte in `.env`:

| Variable                                | Hinweis                                   |
| --------------------------------------- | ----------------------------------------- |
| `SITE_DOMAIN`, `ACME_EMAIL`             | Domain und Kontaktadresse fuer Let's Encrypt |
| `PUBLIC_WEB_ADDRESS`, `PUBLIC_API_ADDRESS` | `https://unidentists.social`, ohne Slash am Ende |
| `POSTGRES_PASSWORD`                     | `openssl rand -hex 24`                     |
| `DATABASE_URL`                          | dasselbe Passwort eintragen                |
| `JWT_SECRET`                            | `openssl rand -hex 32`                     |
| `SMTP_*`                                | Zugangsdaten des Mail-Providers            |

Compose bricht mit einer klaren Fehlermeldung ab, wenn eine Pflichtvariable
fehlt.

Datenverzeichnis anlegen. Der Container laeuft als UID 1001 und kann sonst
nicht in das Volume schreiben:

```bash
mkdir -p data
sudo chown -R 1001:1001 data
```

## 4. Start

```bash
docker compose build      # dauert beim ersten Mal 5-15 Minuten
docker compose up -d
docker compose logs -f
```

Das Datenbankschema legt die App beim ersten Start selbst an, ein separater
Migrationsschritt ist nicht noetig.

Danach `https://unidentists.social` aufrufen. Der erste registrierte Account
wird automatisch Administrator.

## 5. Nach dem ersten Start pruefen

- [ ] Registrierung: kommt die Bestaetigungsmail an, und nicht im Spam?
- [ ] Login und Logout
- [ ] Bild und PDF hochladen, danach nach Text aus dem PDF suchen (OCR)
- [ ] Google-Drive-Bibliothek in den Admin-Einstellungen verbinden
- [ ] `https://www.unidentists.social` leitet auf die Hauptdomain um
- [ ] Aufruf ueber `http://` landet automatisch bei `https://`

## Update nach Code-Aenderungen

```bash
git pull
docker compose build
docker compose up -d
```

## Backups

`backup.sh` sichert Datenbank und Uploads nach `./backups` und loescht
Sicherungen aelter als 14 Tage.

```bash
bash backup.sh
```

Taeglich per cron:

```bash
crontab -e
# 30 3 * * * cd ~/storyden/docker/compose/production && bash backup.sh >> backup.log 2>&1
```

Backups liegen damit auf derselben VM. Fuer echten Schutz zusaetzlich auf
externen Speicher kopieren (`rclone`, `scp`, Object Storage).

Wiederherstellen:

```bash
gunzip -c backups/db-2026-08-08-0330.sql.gz | \
  docker compose exec -T postgres psql -U storyden storyden
tar xzf backups/assets-2026-08-08-0330.tar.gz
```

## Bestehende SQLite-Daten nach Postgres migrieren

Nur noetig, wenn die lokale Entwicklungsdatenbank uebernommen werden soll.

```bash
docker compose up -d postgres

# Schema anlegen
DATABASE_URL="postgresql://storyden:PASSWORT@localhost:5432/storyden?sslmode=disable" \
  go run ../../../cmd/migrate

# Probelauf
go run ../../../cmd/sqlite2postgres \
  --source /pfad/zur/alten/data.db \
  --target "postgresql://storyden:PASSWORT@localhost:5432/storyden?sslmode=disable" \
  --dry-run

# Echter Lauf: derselbe Befehl ohne --dry-run
```

Die Uploads aus dem alten `data/`-Verzeichnis zusaetzlich nach `./data`
kopieren und danach `sudo chown -R 1001:1001 data` erneut ausfuehren.

## Fehlersuche

```bash
docker compose ps                       # laeuft alles?
docker compose logs -f storyden         # App-Logs
docker compose logs -f caddy            # Zertifikatsprobleme
docker compose exec storyden wget -qO- http://127.0.0.1:8000/healthz
```

**Kein Zertifikat:** DNS pruefen (`dig +short unidentists.social`) und ob Port
80 von aussen erreichbar ist. Caddy braucht Port 80 fuer die ACME-Challenge.

**Login funktioniert nicht / man wird sofort ausgeloggt:** `PUBLIC_WEB_ADDRESS`
stimmt nicht mit der aufgerufenen Domain ueberein. Session-Cookies und Passkeys
haengen an diesem Wert.

**Uploads schlagen bei grossen Dateien fehl:** `MAX_UPLOAD_SIZE_MB` in `.env`
und `max_size` im `Caddyfile` angleichen.

**Hohe CPU-Last nach Neustart:** der OCR-Backfill verarbeitet alte Uploads
nach. `OCR_BACKFILL_ENABLED=false` setzen.
