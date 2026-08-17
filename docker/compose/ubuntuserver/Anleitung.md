# Storyden auf dem Ubuntu-Server (192.168.178.105) — loest den Pi ab

Gebaut wird auf dem **Windows-PC**, laufen tut es auf dem **Ubuntu-Server**
(`192.168.178.105`). Das Image wird dort nicht gebaut, sondern per SSH/`scp`
uebertragen und mit `docker load` eingespielt — genau wie beim
Raspberry-Pi-Setup, nur fuer `amd64` statt `arm64`.

**Dieser Server ersetzt den Raspberry Pi** (`192.168.178.30`) als Host fuer
`unidentists.social`. Beide haengen am selben Anschluss (oeffentliche IP laut
[raspberrypi/README.md](../raspberrypi/README.md) `91.135.163.168`,
hier `91.135.163.16` genannt — nahezu identisch, vermutlich derselbe Wert).
Deshalb kann die Portfreigabe im Router immer nur auf **ein** Geraet zeigen:
sie wird in Schritt 2b von Pi:8443 auf Ubuntu-Server:443 umgehaengt, danach
liefert der Pi unter dieser Domain nichts mehr aus. Die anderen Dienste auf
dem Pi (AdGuard, Nextcloud, Home Assistant, wg-easy, Portainer) sind davon
nicht betroffen.

**Bestehende Daten (Accounts, Beitraege, Uploads) werden vom Pi
uebernommen**, nicht neu angelegt. Schritt 3 holt Dump und Uploads direkt
vom Pi, Schritt 7 spielt sie auf dem neuen Server ein, bevor die App zum
ersten Mal startet.

| Container  | Aufgabe                                            |
| ---------- | --------------------------------------------------- |
| `caddy`    | Reverse Proxy, HTTPS-Zertifikat automatisch          |
| `storyden` | Backend (Go) + Frontend (Next.js) in einem Prozess   |
| `postgres` | Datenbank                                            |
| `ddns`     | optional: haelt den A-Record aktuell (siehe unten)   |

Ablauf in Kurzform:

```
PC:      .\export-from-pi.ps1  →  .\build-image.ps1  →  .\copy-to-server.ps1
Server:  Uploads entpacken  →  restore-db.sh  →  docker compose up -d
```

Was du brauchst:

- SSH-Zugang zum Ubuntu-Server UND zum Pi (fuer den Datenexport in Schritt 3)
- Zugriff auf den Router (Portfreigaben 80 und 443)
- Docker Desktop auf dem PC (fuer den Cross-Build nach `amd64`)

---

## 1. Server vorbereiten (einmalig)

Einloggen:

```bash
ssh unidentist@192.168.178.105
```

**Architektur pruefen** — steuert, mit welcher `-Platform` `build-image.ps1`
laeuft:

```bash
uname -m
```

`x86_64` → Standard (`linux/amd64`, ist im Skript schon voreingestellt).
`aarch64` → beim Aufruf `.\build-image.ps1 -Platform linux/arm64` verwenden.

**Docker installieren:**

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
```

Einmal ab-/anmelden (`exit`, dann wieder `ssh`), sonst braucht jeder
`docker`-Befehl `sudo`. Pruefen:

```bash
docker run --rm hello-world
docker compose version
```

**Firewall:**

```bash
sudo apt update && sudo apt install -y ufw
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

**Pruefen, dass 80/443 noch frei sind** (laut deiner Angabe ist der Server
frisch, also sollte hier nichts erscheinen):

```bash
sudo ss -tulpn | grep -E ':(80|443)\s'
```

---

## 2. Domain, Router und DNS

### 2a. Oeffentliche IP des Anschlusses ermitteln

Vom Server aus:

```bash
curl -4 ifconfig.me
```

Kommt ein Fehler oder eine Adresse von `100.64.` bis `100.127.`, hast du
**DS-Lite** — keine eigene oeffentliche IPv4. Dann ist der Server von aussen
ueber IPv4 nicht direkt erreichbar; entweder beim Provider eine echte IPv4
anfordern oder mit DNS-01-Zertifikaten arbeiten (siehe Anhang im
[Raspberry-Pi-Setup](../raspberrypi/README.md), Anhang A — dasselbe Verfahren
gilt hier).

### 2b. Portfreigaben im Router

`Internet → Freigaben → Portfreigaben → Geraet hinzufuegen → Ubuntu-Server`

| Extern | → | Server | Protokoll | Wofuer                                    |
| ------ | - | ------ | --------- | ------------------------------------------ |
| 443    | → | 443    | TCP       | HTTPS und Zertifikat (TLS-ALPN-Challenge)   |
| 80     | → | 80     | TCP       | Umleitung http → https, Ersatz-Challenge    |

Die bestehende Freigabe `443 → Pi:8443` dabei **aendern**, nicht zusaetzlich
anlegen — sonst konkurrieren zwei Ziele um denselben externen Port. Danach
ist der Pi unter `unidentists.social` nicht mehr erreichbar (siehe Hinweis
oben); Caddy auf dem Pi laeuft ggf. weiter, bekommt aber keine Anfragen mehr.

**Reihenfolge beachten:** die Portfreigabe erst umhaengen, nachdem der neue
Server in Schritt 7 erfolgreich laeuft — sonst ist `unidentists.social`
zwischenzeitlich fuer niemanden erreichbar. Bis dahin kann der neue Server
uebergangsweise z.B. per `curl -I --resolve unidentists.social:443:192.168.178.105 https://unidentists.social`
aus dem LAN getestet werden (Zertifikat schlaegt dabei erwartungsgemaess fehl,
solange die Portfreigabe noch auf den Pi zeigt).

### 2c. A-Record bei deinem Domain-Provider

Zeigt bereits auf die IP aus 2a (der Pi nutzt dieselbe Domain). Nur pruefen:

```bash
dig +short unidentists.social
```

### 2d. Aendert sich die IP des Anschlusses? → DynDNS

Der Pi haelt den A-Record aktuell vermutlich schon per `ddns`-Container
(siehe [raspberrypi/README.md](../raspberrypi/README.md), Abschnitt 3c). Wird
dieser Server zum neuen dauerhaften Host, uebernimmt sinnvollerweise er diese
Aufgabe, damit nicht zwei Updater um denselben Record konkurrieren. Der
`ddns`-Service hier ist per `profiles: ["ddns"]` deaktiviert, bis du ihn
brauchst:

```bash
mkdir -p ddns && cp ddns-config.example.json ddns/config.json
nano ddns/config.json      # Benutzername + API-Token von name.com eintragen
sudo chown -R 1000:1000 ddns
docker compose --profile ddns up -d
```

Danach auf dem Pi den dortigen `ddns`-Service stoppen (Schritt 8), sonst
schreiben beide denselben Record und es kommt zu Race Conditions bei jedem
IP-Wechsel.

Ist die IP wirklich fest (kein DynDNS noetig), diesen Schritt ueberspringen.

### 2e. Pruefen

```bash
dig +short unidentists.social
```

Muss die IP aus 2a liefern.

---

## 3. Daten vom Pi exportieren

Auf dem **PC**, in PowerShell. Der Pi muss laufen und per SSH erreichbar
sein:

```powershell
cd <projektverzeichnis>\docker\compose\ubuntuserver
```

```powershell
.\export-from-pi.ps1 -PiUser pi
```

Holt vom Pi:

| Datei          | Inhalt                                                     |
| -------------- | ---------------------------------------------------------- |
| `db.sql`       | Dump der aktuellen Postgres-Datenbank (`--no-owner --no-privileges`) |
| `data.tar.gz`  | Assets und Avatare aus `~/storyden/data` auf dem Pi         |

und legt beides unter `transfer\` ab. Zusaetzlich wird die lokale `.env` in
diesem Verzeichnis angelegt (aus `.env.example`, falls sie noch nicht
existiert) und mit `JWT_SECRET` sowie den SMTP-/Google-Drive-Werten vom Pi
befuellt — `JWT_SECRET` **muss** identisch bleiben, sonst werden beim Umzug
alle bestehenden Sessions ungueltig.

`SITE_DOMAIN` und `PUBLIC_*` stehen in `.env.example` schon auf
`unidentists.social`, weil es dieselbe Domain bleibt. `POSTGRES_PASSWORD`
wird bewusst **nicht** vom Pi uebernommen — dieser Server bekommt ein neues,
eigenes Postgres-Volume mit eigenem Passwort; der SQL-Dump enthaelt keine
Rollen oder Passwoerter.

`transfer\` enthaelt Nutzerdaten und ist gitignored.

---

## 4. Auf dem Windows-PC: Image bauen

```powershell
.\build-image.ps1
```

Erzeugt `transfer\storyden-amd64.tar` und die Tags `storyden-server:latest`
sowie `storyden-server:<datum>`. Beim ersten Mal ein paar Minuten, danach
greift der BuildKit-Cache.

**Optional: Image vor dem Transfer lokal testen.**

```powershell
docker run -d --name storyden-smoke -p 18000:8000 -e JWT_SECRET=nur-zum-testen storyden-server:latest
```

Nach etwa einer Minute muss `http://localhost:18000/healthz` mit 200
antworten. Danach aufraeumen:

```powershell
docker rm -f storyden-smoke
```

---

## 5. Konfiguration (.env) pruefen

`.env` existiert bereits aus Schritt 3 (teilweise vorausgefuellt). Einmal
komplett durchlesen, nicht nur die neuen Werte:

```powershell
notepad .env
```

Noch selbst setzen:

| Variable            | Hinweis                                    |
| -------------------- | ------------------------------------------- |
| `ACME_EMAIL`         | Kontaktadresse fuer Let's-Encrypt-Warnungen |
| `POSTGRES_PASSWORD`  | `openssl rand -hex 24` (z.B. in Git Bash), muss zu `DATABASE_URL` passen |

Vom Pi automatisch uebernommen (Schritt 3): `JWT_SECRET`, `SMTP_*`,
`GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON`. Schon korrekt in `.env.example`:
`SITE_DOMAIN`, `PUBLIC_WEB_ADDRESS`, `PUBLIC_API_ADDRESS` (alle
`unidentists.social`, da die Domain gleich bleibt).

Compose bricht spaeter mit einer klaren Fehlermeldung ab, wenn eine
Pflichtvariable fehlt.

`.env` liegt nur lokal und wird von `copy-to-server.ps1` mit uebertragen —
sie ist gitignored, landet also nicht im Repo.

---

## 6. Auf den Server kopieren

```powershell
.\copy-to-server.ps1 -ServerUser unidentist
```

Kopiert Image, Dump, Uploads, `.env`, `docker-compose.yml`, `Caddyfile`,
`backup.sh`, `restore-db.sh` und `ddns-config.example.json` nach
`/home/unidentist/storyden` und laedt das Image dort in Docker. Gestartet wird
noch nichts.

Zu uebertragen sind je nach Forumsgroesse einige hundert MB (Image, Uploads,
Dump) — im LAN typischerweise unter einer Minute.

Bei jedem `scp`/`ssh` fragt der Server nach dem Passwort. Fuer einen
SSH-Key:

```powershell
ssh-keygen -t ed25519
type $env:USERPROFILE\.ssh\id_ed25519.pub | ssh unidentist@192.168.178.105 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

---

## 7. Auf dem Server starten

```bash
ssh unidentist@192.168.178.105
cd ~/storyden
```

Konfiguration noch einmal ansehen:

```bash
nano .env
```

Falls DynDNS aus Schritt 2d gebraucht wird, `ddns/config.json` jetzt anlegen
(siehe dort).

Uploads entpacken. Der Storyden-Container laeuft als UID 1001 und kann sonst
nicht in das Verzeichnis schreiben:

```bash
mkdir -p data && sudo tar xzf data.tar.gz -C data && sudo chown -R 1001:1001 data
```

Datenbank einspielen. Das Skript startet Postgres, wartet, prueft dass die
Datenbank leer ist, spielt den Dump ein und zeigt danach die Zeilenzahlen:

```bash
bash restore-db.sh db.sql
```

Alles starten:

```bash
docker compose up -d
```

Mit DynDNS stattdessen:

```bash
docker compose --profile ddns up -d
```

Mitlesen:

```bash
docker compose logs -f
```

Im Caddy-Log muss eine Zeile wie `certificate obtained successfully` stehen
— vorausgesetzt, die Portfreigabe aus Schritt 2b zeigt inzwischen auf diesen
Server.

Danach:

```
https://unidentists.social
```

---

## 8. Nach dem ersten Start pruefen

- [ ] Aufruf ueber HTTPS, Schloss im Browser gruen, keine Zertifikatswarnung
- [ ] Login mit einem bestehenden Account funktioniert
- [ ] Alte Beitraege, Threads und Bilder sind da
- [ ] Bild und PDF hochladen, danach nach Text aus dem PDF suchen (OCR)
- [ ] Google-Drive-Bibliothek in den Admin-Einstellungen erreichbar
- [ ] `https://www.unidentists.social` leitet auf die Hauptdomain um
- [ ] `http://unidentists.social` leitet automatisch auf `https://` um
- [ ] `docker compose logs ddns` (falls aktiv) zeigt den A-Record als aktuell
- [ ] Neustart-Test: `sudo reboot`, danach kommt alles von selbst wieder hoch

Wenn alles laeuft, auf dem **Pi** die Storyden-Instanz stilllegen, damit
nicht versehentlich zwei Instanzen mit auseinanderlaufenden Daten
existieren:

```bash
ssh pi@192.168.178.30
cd ~/storyden && docker compose down
```

Die anderen Dienste auf dem Pi (AdGuard, Nextcloud, Home Assistant, wg-easy,
Portainer) laufen weiter — nur der Storyden-Stack (inkl. dessen `ddns`, falls
dort noch aktiv) wird gestoppt. Die Volumes bleiben als Sicherheitsnetz fuer
die ersten Tage erhalten.

**Passkeys muessen nicht neu registriert werden**, solange die Domain
gleich bleibt (`unidentists.social` auf beiden Seiten) — Passkeys sind an
die Origin gebunden, nicht an den Server dahinter.

---

## 9. Update nach Code-Aenderungen

Auf dem PC:

```powershell
.\build-image.ps1
.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
```

Auf dem Server:

```bash
cd ~/storyden && docker compose up -d
```

`-ImageOnly` laesst `.env` und `data/` unangetastet — es wird nur das Image
ersetzt. Datenbank-Schemaaenderungen wendet die App beim Start selbst an.

**Zurueckrollen** auf eine aeltere Version: in `.env` die Zeile
`STORYDEN_IMAGE=storyden-server:latest` auf einen Datums-Tag aendern
(`docker images storyden-server` auf dem Server zeigt, was da ist — dafuer
muesste das aeltere Image aber noch einmal uebertragen werden, falls es
zwischenzeitlich ueberschrieben wurde), dann `docker compose up -d`.

---

## 10. Backups

```bash
bash backup.sh
```

Schreibt Datenbank-Dump und Uploads nach `./backups` und loescht
Sicherungen, die aelter als 14 Tage sind. Taeglich per cron:

```bash
crontab -e
```

```
30 3 * * * cd /home/unidentist/storyden && bash backup.sh >> backup.log 2>&1
```

Zusaetzlich woanders ablegen (z.B. `rsync` auf den PC), Backups auf demselben
Server schuetzen nicht vor einem Hardware-Defekt dort.

Wiederherstellen:

```bash
gunzip -c backups/db-2026-08-16-0330.sql.gz | docker compose exec -T postgres psql -U storyden storyden
tar xzf backups/assets-2026-08-16-0330.tar.gz
```

---

## Fehlersuche

```bash
docker compose ps
docker compose logs -f storyden
docker compose logs -f caddy
docker compose exec storyden wget -qO- http://127.0.0.1:8000/healthz
```

**`bind: address already in use` beim Start von `caddy`** — auf dem Host
lauscht noch etwas auf 80 oder 443:

```bash
sudo ss -tulpn | grep -E ':(80|443)\s'
```

**Kein Zertifikat.** Caddy-Log lesen, dann der Reihe nach:

1. Zeigt der DNS auf die aktuelle IP? `dig +short unidentists.social` gegen
   `curl -4 ifconfig.me` auf dem Server vergleichen. Weicht es ab und du
   nutzt DynDNS: `docker compose logs ddns`.
2. Zeigt die Portfreigabe im Router wirklich schon auf diesen Server und
   nicht mehr auf den Pi (Schritt 2b)?
3. Kommt der Port von aussen durch? Im Mobilfunknetz testen, nicht im WLAN:

```bash
curl -I https://unidentists.social
```

Let's Encrypt drosselt nach mehreren Fehlversuchen fuer eine Stunde. Zum
Ausprobieren im `Caddyfile` die `acme_ca`-Zeile mit der Staging-URL
einkommentieren (dann ist das Zertifikat im Browser nicht vertrauenswuerdig,
aber die Ausstellung wird real durchgespielt). Vor dem Scharfstellen wieder
auskommentieren **und** `docker volume rm storyden_caddy_data`, damit Caddy
das Staging-Zertifikat nicht behaelt.

**`export-from-pi.ps1` bricht bei `pg_dump` oder `tar` ab.** Wahrscheinlich
stimmt `-RemoteDir` nicht (Standard `/home/pi/storyden`) oder der Pi-Nutzer
darf nicht ohne Passwort `sudo tar` ausfuehren. Manuell auf dem Pi pruefen:
`cd ~/storyden && docker compose ps`.

**`restore-db.sh` meldet, dass schon Tabellen existieren.** Entweder lief
der Restore schon einmal, oder die App ist bereits gestartet und hat das
Schema selbst angelegt (dann in der falschen Reihenfolge vorgegangen —
Restore muss vor dem ersten `docker compose up -d` ohne `postgres`-Filter
laufen). Sauber neu aufsetzen: `docker compose down && docker volume rm
storyden_postgres-data`, danach `bash restore-db.sh db.sql` erneut.

**`exec format error` beim Start von `storyden`** — das Image ist fuer die
falsche Architektur gebaut. `uname -m` auf dem Server pruefen und mit der
`-Platform` aus `build-image.ps1` vergleichen (`x86_64` → `linux/amd64`,
`aarch64` → `linux/arm64`).

**Compose sucht das Image in einer Registry** (`pull access denied`) — das
`docker load` hat nicht funktioniert:

```bash
docker images storyden-server
docker load -i storyden-amd64.tar
```

**Login funktioniert nicht / man wird sofort ausgeloggt.**
`PUBLIC_WEB_ADDRESS` stimmt nicht exakt mit der aufgerufenen URL ueberein
(Slash am Ende, `www.` davor), oder `JWT_SECRET` weicht vom Pi ab (Schritt 3
haette es automatisch uebernehmen sollen — in `.env` gegenpruefen).

**Uploads schlagen bei grossen Dateien fehl.** `MAX_UPLOAD_SIZE_MB` in `.env`
und `max_size` im `Caddyfile` angleichen.

**Hohe CPU-Last nach Neustart.** Der OCR-Backfill verarbeitet alte Uploads
nach. `OCR_BACKFILL_ENABLED=false` in `.env` setzen.

**`ddns` aktualisiert nichts, oder zwei `ddns`-Container schreiben denselben
Record.** Sicherstellen, dass nach Schritt 8 nur noch der `ddns`-Service auf
diesem Server laeuft, nicht mehr der auf dem Pi.
