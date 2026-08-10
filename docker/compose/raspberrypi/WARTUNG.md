# Wartung — Storyden auf dem Raspberry Pi

Kurzreferenz für den laufenden Betrieb. Die ausführliche Ersteinrichtung mit
Begründungen und Fehlersuche steht in [README.md](README.md) — hier nur das,
was für den Alltag danach wichtig ist.

## Überblick: was läuft wo

| | Wo | Pfad |
| - | - | - |
| Code, Skripte | Windows-PC | `F:\Zahnmedizin\Forum\storyden\docker\compose\raspberrypi` |
| Laufende Container | Pi (`192.168.178.30`) | `~/storyden` |
| Datenbank | Pi, Docker-Volume | `storyden_postgres-data` |
| Uploads/Assets | Pi | `~/storyden/data` |
| DynDNS-Zugangsdaten | Pi | `~/storyden/ddns/config.json` |
| Konfiguration/Secrets | Pi | `~/storyden/.env` |

Vier Container unter dem Compose-Projekt `storyden`: `postgres`, `storyden`
(Backend+Frontend in einem Image), `caddy` (HTTPS), `ddns` (hält den
Domain-Record aktuell). Alle haben `restart: unless-stopped` — nach einem
`sudo reboot` auf dem Pi kommt alles von selbst wieder hoch.

---

## Code ändern (Frontend oder Backend)

Ein Image enthält beides — es gibt keinen separaten Deploy-Weg für
Frontend/Backend. Ablauf bei jeder Änderung:

**1. Auf dem PC bauen** (im Projekt-Root, wo der geänderte Code liegt):

```powershell
cd F:\Zahnmedizin\Forum\storyden\docker\compose\raspberrypi
.\build-image.ps1
```

Baut aus dem aktuellen Stand des Arbeitsverzeichnisses — kein bestimmter
Branch nötig, einfach das, was gerade ausgecheckt ist. 10–20 Minuten beim
ersten Mal, danach meist deutlich schneller durch den Build-Cache. Vergibt
zusätzlich einen Datums-Tag (`storyden-rpi:2026-08-10`) für Rollbacks.

**2. Nur das Image übertragen** (Config, Datenbank und Uploads bleiben
unangetastet):

```powershell
.\copy-to-pi.ps1 -PiUser pi -ImageOnly
```

**3. Auf dem Pi neu starten:**

```bash
cd ~/storyden && docker compose up -d
```

Datenbank-Schemaänderungen wendet die App beim Start selbst an, kein
separater Migrationsschritt nötig.

**Zurückrollen** auf eine ältere Version, falls ein Update Probleme macht:

```bash
docker images storyden-rpi
```

zeigt die verfügbaren Datums-Tags. Dann in `~/storyden/.env`:

```
STORYDEN_IMAGE=storyden-rpi:2026-08-05
```

```bash
docker compose up -d storyden
```

---

## Backups

**Manuell:**

```bash
cd ~/storyden && bash backup.sh
```

Schreibt `backups/db-<datum>.sql.gz` (Datenbank) und
`backups/assets-<datum>.tar.gz` (Uploads). Sicherungen älter als 14 Tage
werden automatisch gelöscht (`KEEP_DAYS` in `backup.sh` anpassen).

**Automatisch per Cron**, täglich um 03:30 Uhr:

```bash
crontab -e
```

```
30 3 * * * cd /home/pi/storyden && bash backup.sh >> backup.log 2>&1
```

**Wichtig:** die Backups liegen auf derselben SD-Karte wie die Daten — kein
Schutz gegen einen kaputten Datenträger. Regelmäßig nach draußen kopieren,
z. B. vom PC aus:

```powershell
scp -r pi@192.168.178.30:~/storyden/backups F:\Storyden-Backups\
```

**Wiederherstellen:**

```bash
cd ~/storyden
gunzip -c backups/db-2026-08-10-0330.sql.gz | docker compose exec -T postgres psql -U storyden storyden
tar xzf backups/assets-2026-08-10-0330.tar.gz -C data
sudo chown -R 1001:1001 data
```

---

## Domain ändern

Betrifft mehrere Stellen, in dieser Reihenfolge:

**1. `~/storyden/.env`:**

```
SITE_DOMAIN=neue-domain.tld
PUBLIC_WEB_ADDRESS=https://neue-domain.tld
PUBLIC_API_ADDRESS=https://neue-domain.tld
```

**2. `~/storyden/ddns/config.json`:** das `domain`-Feld auf die neue Domain
setzen.

**3. Bei name.com:** A-Record für die neue Domain anlegen (Host leer,
aktuelle öffentliche IP, TTL 300). Der alte Record kann bleiben oder gelöscht
werden.

**4. AdGuard Home:** `Filter → DNS-Rewrites` — den bestehenden Eintrag auf die
neue Domain ändern (gleiche IP, `192.168.178.30`), sonst funktioniert der
Zugriff aus dem eigenen WLAN nicht mehr unter der neuen Adresse.

**5. Neu starten**, damit Caddy ein Zertifikat für die neue Domain holt:

```bash
cd ~/storyden && docker compose up -d
```

```bash
docker compose logs -f caddy
```

Dort muss wieder `certificate obtained successfully` erscheinen.

**Wichtig:** das loggt alle Nutzer aus und macht registrierte Passkeys
ungültig — die Origin ändert sich. Am besten außerhalb der Hauptnutzungszeit
durchführen und die Nutzer vorher informieren.

Die FRITZ!Box-Portfreigaben (443→8443, 80→80) sind domainunabhängig und
müssen nicht angefasst werden.

---

## IP-Adresse ändern

Zwei komplett unabhängige "IPs" — nicht verwechseln:

### Die öffentliche IP des Anschlusses (WAN)

Wechselt bei jeder Zwangstrennung von selbst. Darum kümmert sich der
`ddns`-Container automatisch — keine Aktion nötig.

### Die LAN-IP des Pi (`192.168.178.30`)

Ändert sich normalerweise nicht von selbst, außer die FRITZ!Box vergibt per
DHCP eine andere Adresse. **Empfehlung: einmalig eine feste IP-Reservierung
für den Pi einrichten**, dann stellt sich die Frage gar nicht erst:

`FRITZ!Box → Heimnetz → Netzwerk → Gerät bearbeiten → "Diesem Netzwerkgerät
immer die gleiche IPv4-Adresse zuweisen"`.

Ändert sie sich trotzdem (oder soll bewusst geändert werden), diese Stellen
anpassen:

- **AdGuard Home:** DNS-Rewrite für die Domain auf die neue IP ändern
  (Schritt 3d der Ersteinrichtung).
- **FRITZ!Box-Portfreigaben:** prüfen, dass 443→8443 und 80→80 noch auf den
  Pi zeigen (bei Geräteauswahl per MAC-Adresse folgt das meist automatisch,
  bei manueller IP-Eingabe nicht).
- **SSH-Zugriff und die Skripte auf dem PC:** entweder bei jedem Aufruf
  `-PiHost <neue-ip>` mitgeben —

  ```powershell
  .\copy-to-pi.ps1 -PiUser pi -PiHost 192.168.178.XX
  ```

  — oder den Default in `copy-to-pi.ps1` (Parameter `$PiHost`) dauerhaft
  ändern.

Die öffentliche Domain selbst ist von einer LAN-IP-Änderung nicht betroffen.

---

## Secrets ändern

**SMTP/Brevo-Zugangsdaten:** einfach in `.env` ändern, dann:

```bash
docker compose up -d storyden
```

(`restart` reicht hier **nicht** — das liest `.env` nicht neu ein, nur
`up -d` übernimmt geänderte Umgebungsvariablen.)

**Postgres-Passwort:** muss an zwei Stellen synchron geändert werden — in der
Datenbank selbst und in `.env`:

```bash
docker compose exec -T postgres psql -U storyden -c "ALTER ROLE storyden WITH PASSWORD 'neues-passwort'"
```

Dann in `.env` `POSTGRES_PASSWORD` und das Passwort in `DATABASE_URL`
anpassen, danach:

```bash
docker compose up -d
```

**JWT_SECRET:** nur im Notfall ändern (z. B. bei Verdacht auf Kompromittierung)
— das invalidiert **sofort alle** aktiven Sessions und offenen
Passwort-Reset-Links. Kein Rollback möglich, alle Nutzer müssen sich neu
einloggen.

---

## Alltag: Status, Logs, Neustarts

```bash
docker compose ps                    # laeuft alles, ist alles "healthy"?
docker compose logs -f storyden      # App-Logs live
docker compose logs -f caddy         # Zertifikat/HTTPS-Probleme
docker compose logs -f ddns          # DynDNS-Updates
```

**Einzelnen Dienst neu starten** (z. B. wenn die App mal hängt, ohne dass
sich an der Config etwas geändert hat):

```bash
docker compose restart storyden
```

**Nach Änderungen an `.env` oder `docker-compose.yml`** immer `up -d` statt
`restart` — nur das liest die neue Konfiguration ein:

```bash
docker compose up -d
```

**Pi neu starten** — alles kommt automatisch wieder hoch:

```bash
sudo reboot
```

**Betriebssystem aktuell halten**, gelegentlich:

```bash
sudo apt update && sudo apt full-upgrade -y
```

Nach einem Kernel-Update: `sudo reboot`.

---

## Speicherplatz im Blick behalten

```bash
df -h /
```

```bash
docker system df
```

Alte, nicht mehr getaggte Image-Layer aufräumen (lässt aktuell getaggte
Images wie `storyden-rpi:2026-08-05` für Rollbacks unangetastet):

```bash
docker image prune
```

Auf dem PC sammeln sich in `docker\compose\raspberrypi\transfer\` alte
`storyden-arm64.tar`-Dateien (jeweils ~150–200 MB) — der Ordner ist
gitignored, kann bedenkenlos aufgeräumt werden, sobald ein Deploy
durchgelaufen ist.

---

## Zertifikat und DynDNS — was normal ist

- Let's Encrypt läuft automatisch etwa 30 Tage vor Ablauf über Caddy neu ab.
  Keine Aktion nötig, außer Port 443/80 wird dauerhaft unerreichbar (siehe
  README, Abschnitt Fehlersuche).
- Der `ddns`-Container prüft alle 5 Minuten und schickt bei jedem Neustart
  des Containers einmalig ein Update (auch wenn sich nichts geändert hat) —
  das ist normal. Danach sollte er zwischen den Aktualisierungszyklen ruhig
  bleiben.
- Beide gelegentlich stichprobenartig prüfen:

  ```bash
  docker compose ps
  ```

  Alle vier Container sollten `Up` bzw. `healthy` zeigen.

---

## Kurzreferenz

| Aufgabe | Befehl |
| - | - |
| Status prüfen | `docker compose ps` |
| Logs live | `docker compose logs -f <service>` |
| Config-Änderung anwenden | `docker compose up -d` |
| Nur einen Dienst neu starten | `docker compose restart <service>` |
| Backup jetzt | `bash backup.sh` |
| Code-Update ausrollen (PC) | `.\build-image.ps1` → `.\copy-to-pi.ps1 -ImageOnly` |
| Code-Update übernehmen (Pi) | `docker compose up -d` |
| Auf alte Version zurück | `.env`: `STORYDEN_IMAGE=storyden-rpi:<datum>` → `docker compose up -d storyden` |
| Postgres-Shell | `docker compose exec -T postgres psql -U storyden storyden` |
