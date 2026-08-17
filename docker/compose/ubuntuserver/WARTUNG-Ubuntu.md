# Wartung — Storyden auf dem Ubuntu-Server

Kurzreferenz für den laufenden Betrieb. Die ausführliche Ersteinrichtung mit
Begründungen steht in [Anleitung.md](Anleitung.md) — hier nur das, was für
den Alltag danach wichtig ist.

## Überblick: was läuft wo

| | Wo | Pfad |
| - | - | - |
| Code, Skripte | Windows-PC | `F:\Zahnmedizin\Forum\storyden\docker\compose\ubuntuserver` |
| Laufende Container | Ubuntu-Server (`192.168.178.105`), User `unidentist` | `~/storyden` |
| Datenbank | Server, Docker-Volume | `storyden_postgres-data` |
| Uploads/Assets | Server | `~/storyden/data` |
| DynDNS-Zugangsdaten (falls aktiv) | Server | `~/storyden/ddns/config.json` |
| Konfiguration/Secrets | Server | `~/storyden/.env` |

**Schnellzugriff:** `cd ~/storyden && ./menu.sh` öffnet ein interaktives
Menü für alle unten dokumentierten Alltagsaufgaben (Status, Logs, Neustart,
Backup, Rollback, Health-Check, ...) — praktisch per SSH vom Handy aus, wenn
man sich die Befehle nicht merken will. Die Befehle unten bleiben als
Referenz stehen und funktionieren weiterhin einzeln, z. B. für Cron oder wenn
kein interaktives Menü gewünscht ist.

Drei Container immer aktiv unter dem Compose-Projekt `storyden`: `postgres`,
`storyden` (Backend+Frontend in einem Image), `caddy` (HTTPS). Ein vierter,
`ddns`, ist optional und läuft nur nach `docker compose --profile ddns up -d`
(siehe Anleitung.md, Abschnitt 2d) — mit `docker compose ps` prüfen, ob er
gerade mitläuft. Alle haben `restart: unless-stopped` — nach `sudo reboot`
kommt alles von selbst wieder hoch.

**Login-User auf dem Server ist `unidentist`, nicht `ubuntu`** — Default in
`copy-to-server.ps1` (`$ServerUser`) und in den Beispielen unten.

---

## Code ändern (Frontend oder Backend)

Ein Image enthält beides — es gibt keinen separaten Deploy-Weg für
Frontend/Backend. Ablauf bei jeder Änderung:

**1. Auf dem PC bauen** (im Projekt-Root, wo der geänderte Code liegt):

```powershell
cd F:\Zahnmedizin\Forum\storyden\docker\compose\ubuntuserver
.\build-image.ps1
```

Baut standardmäßig für `linux/amd64` — auf dem Server `uname -m` prüfen, bei
`aarch64` stattdessen `-Platform linux/arm64` übergeben. Baut aus dem
aktuellen Stand des Arbeitsverzeichnisses, kein bestimmter Branch nötig.
Vergibt zusätzlich einen Datums-Tag (`storyden-server:2026-08-17`) für
Rollbacks.

**2. Nur das Image übertragen** (Config, Datenbank und Uploads bleiben
unangetastet):

```powershell
.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
```

**3. Auf dem Server neu starten:**

```bash
cd ~/storyden && docker compose up -d
```

Datenbank-Schemaänderungen wendet die App beim Start selbst an, kein
separater Migrationsschritt nötig.

**Zurückrollen** auf eine ältere Version, falls ein Update Probleme macht:

```bash
docker images storyden-server
```

zeigt die verfügbaren Datums-Tags. Dann in `~/storyden/.env`:

```
STORYDEN_IMAGE=storyden-server:2026-08-10
```

```bash
docker compose up -d storyden
```

(Das ältere Image muss dafür noch auf dem Server vorhanden sein — sonst vorher
noch einmal übertragen.)

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
30 3 * * * cd /home/unidentist/storyden && bash backup.sh >> backup.log 2>&1
```

**Wichtig:** die Backups liegen auf derselben Platte wie die Daten — kein
Schutz gegen einen Hardware-Defekt dort. Regelmäßig nach draußen kopieren,
z. B. vom PC aus:

```powershell
scp -r unidentist@192.168.178.105:~/storyden/backups F:\Storyden-Backups\
```

**Wiederherstellen:**

```bash
cd ~/storyden
gunzip -c backups/db-2026-08-17-0330.sql.gz | docker compose exec -T postgres psql -U storyden storyden
tar xzf backups/assets-2026-08-17-0330.tar.gz -C data
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

**2. Falls DynDNS aktiv:** `~/storyden/ddns/config.json` — das `domain`-Feld
auf die neue Domain setzen.

**3. Beim Domain-Provider (name.com):** A-Record für die neue Domain anlegen
(Host leer, aktuelle öffentliche IP, TTL 300).

**4. Falls im LAN ein AdGuard Home o.ä. mit Split-Horizon-DNS läuft** (der Pi
hat `unidentists.social` bisher per DNS-Rewrite auf seine eigene LAN-IP
`192.168.178.30` aufgelöst): den Rewrite-Eintrag auf die neue Domain **und**
auf `192.168.178.105` anpassen, sonst lösen Geräte im eigenen WLAN die Domain
falsch auf, während externer Zugriff normal funktioniert.

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

Die Router-Portfreigaben (443/80 → Server) sind domainunabhängig und müssen
nicht angefasst werden.

---

## IP-Adresse ändern

Zwei komplett unabhängige "IPs" — nicht verwechseln:

### Die öffentliche IP des Anschlusses (WAN)

Wechselt bei jeder Zwangstrennung von selbst. Ist der `ddns`-Service aktiv
(`docker compose ps` zeigt ihn), kümmert der sich automatisch darum — keine
Aktion nötig. Ist er inaktiv und die IP nicht fest, muss der A-Record beim
Provider manuell nachgeführt werden.

### Die LAN-IP des Servers (`192.168.178.105`)

Ändert sich normalerweise nicht von selbst, außer der Router vergibt per DHCP
eine andere Adresse. **Empfehlung: einmalig eine feste IP-Reservierung
einrichten**, dann stellt sich die Frage gar nicht erst
(`Router → Heimnetz/Netzwerk → Gerät bearbeiten → feste IPv4-Adresse
zuweisen`).

Ändert sie sich trotzdem, diese Stellen anpassen:

- **Router-Portfreigaben:** prüfen, dass 443→443 und 80→80 noch auf den
  Server zeigen (Schritt 2b der Anleitung).
- **Split-Horizon-DNS im LAN** (falls vorhanden, z. B. AdGuard Home): Rewrite
  für die Domain auf die neue IP anpassen.
- **SSH-Zugriff und die Skripte auf dem PC:** entweder bei jedem Aufruf
  `-ServerHost <neue-ip>` mitgeben —

  ```powershell
  .\copy-to-server.ps1 -ServerUser unidentist -ServerHost 192.168.178.XX
  ```

  — oder den Default in `copy-to-server.ps1` (Parameter `$ServerHost`)
  dauerhaft ändern.

Die öffentliche Domain selbst ist von einer LAN-IP-Änderung nicht betroffen.

---

## Secrets ändern

**SMTP/Zugangsdaten:** einfach in `.env` ändern, dann:

```bash
docker compose up -d storyden
```

(`restart` reicht hier **nicht** — das liest `.env` nicht neu ein, nur
`up -d` übernimmt geänderte Umgebungsvariablen.)

**Postgres-Passwort:** muss an **zwei** Stellen synchron geändert werden — in
der Datenbank selbst und in `.env`, an **zwei** Variablen dort
(`POSTGRES_PASSWORD` und das eingebettete Passwort in `DATABASE_URL`). Wird
nur eine der beiden Stellen geändert, verbindet sich die App mit dem falschen
Passwort und schlägt mit `password authentication failed` fehl — genau das
ist beim Ersteinrichten schon einmal passiert.

```bash
docker compose exec -T postgres psql -U storyden -c "ALTER ROLE storyden WITH PASSWORD 'neues-passwort'"
```

Dann in `.env` **beide** Stellen anpassen:

```
POSTGRES_PASSWORD=neues-passwort
DATABASE_URL=postgresql://storyden:neues-passwort@postgres:5432/storyden?sslmode=disable
```

```bash
docker compose up -d
```

**JWT_SECRET:** nur im Notfall ändern (z. B. bei Verdacht auf Kompromittierung)
— das invalidiert **sofort alle** aktiven Sessions und offenen
Passwort-Reset-Links. Kein Rollback möglich, alle Nutzer müssen sich neu
einloggen.

---

## Bekannte Stolperfallen (bereits gefixt, aber gut zu wissen)

- **CRLF in `.env` oder den `.sh`-Skripten.** Git-Checkout auf Windows
  (`core.autocrlf=true`) wandelt Textdateien standardmäßig auf
  Windows-Zeilenenden um; `scp` überträgt sie danach 1:1 auf den Linux-Server.
  Symptome: `set: pipefail: Ungültiger Optionsname` (Skript selbst betroffen)
  oder `role "storyden\r" does not exist` (`.env` betroffen — der `\r` landet
  im Wert, den Bash-Skripte per `grep`/`cut` daraus lesen; Docker Compose
  selbst parst `.env` robuster und ist davon nicht betroffen). Abgesichert
  durch `.gitattributes` (`*.sh`, `.env.example` bleiben LF), durch
  `export-from-pi.ps1` (schreibt `.env` explizit mit LF) und defensiv in
  `restore-db.sh`/`backup.sh` (`tr -d '\r'` beim Auslesen von `.env`). Sollte
  trotzdem mal wieder CRLF auftauchen (z. B. nach Bearbeiten mit einem
  Windows-Editor): `tr -d '\r' < .env > .env.tmp && mv .env.tmp .env` auf dem
  Server, oder lokal auf dem PC vor dem nächsten `copy-to-server.ps1`.
- **`DATABASE_URL` und `POSTGRES_PASSWORD` laufen auseinander** — siehe
  Abschnitt "Secrets ändern" oben.

---

## Alltag: Status, Logs, Neustarts

```bash
docker compose ps                    # laeuft alles, ist alles "healthy"?
docker compose logs -f storyden      # App-Logs live
docker compose logs -f caddy         # Zertifikat/HTTPS-Probleme
docker compose logs -f postgres      # DB-Probleme
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

**Server neu starten** — alles kommt automatisch wieder hoch:

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
Images wie `storyden-server:2026-08-10` für Rollbacks unangetastet):

```bash
docker image prune
```

Auf dem PC sammeln sich in `docker\compose\ubuntuserver\transfer\` alte
`storyden-amd64.tar`-Dateien (mehrere hundert MB) — der Ordner ist
gitignored, kann bedenkenlos aufgeräumt werden, sobald ein Deploy
durchgelaufen ist.

---

## Zertifikat und DynDNS — was normal ist

- Let's Encrypt läuft automatisch etwa 30 Tage vor Ablauf über Caddy neu ab.
  Keine Aktion nötig, außer Port 443/80 wird dauerhaft unerreichbar (siehe
  Anleitung.md, Abschnitt "Fehlersuche").
- Falls `ddns` aktiv ist: prüft alle 5 Minuten und schickt bei jedem Neustart
  des Containers einmalig ein Update (auch wenn sich nichts geändert hat) —
  das ist normal. Danach sollte er zwischen den Aktualisierungszyklen ruhig
  bleiben.
- Gelegentlich stichprobenartig prüfen:

  ```bash
  docker compose ps
  ```

  Alle Container sollten `Up` bzw. `healthy` zeigen.

---

## Kurzreferenz

| Aufgabe | Befehl |
| - | - |
| Status prüfen | `docker compose ps` |
| Logs live | `docker compose logs -f <service>` |
| Config-Änderung anwenden | `docker compose up -d` |
| Nur einen Dienst neu starten | `docker compose restart <service>` |
| Backup jetzt | `bash backup.sh` |
| Code-Update ausrollen (PC) | `.\build-image.ps1` → `.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly` |
| Code-Update übernehmen (Server) | `docker compose up -d` |
| Auf alte Version zurück | `.env`: `STORYDEN_IMAGE=storyden-server:<datum>` → `docker compose up -d storyden` |
| Postgres-Shell | `docker compose exec -T postgres psql -U storyden storyden` |
