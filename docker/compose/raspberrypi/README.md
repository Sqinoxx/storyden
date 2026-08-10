# Storyden auf dem Raspberry Pi 4 — HTTPS auf Port 8443

Gebaut wird auf dem **Windows-PC**, laufen tut es auf dem **Pi** (`192.168.178.30`).
Der Pi baut nichts selbst — der Next.js-Build braucht mehr RAM als 4 GB
zuverlässig hergeben, dort hängt sich der Pi auf.

| Container  | Aufgabe                                                    |
| ---------- | ---------------------------------------------------------- |
| `caddy`    | HTTPS auf Port 8443, Let's-Encrypt-Zertifikat automatisch   |
| `storyden` | Backend (Go) + Frontend (Next.js) in einem Container        |
| `postgres` | Datenbank, übernommen vom PC                               |
| `ddns`     | hält den A-Record bei name.com auf der aktuellen IP        |

Caddy lauscht auf **8443**. Von außen kommt man über 443 herein, weil die
FRITZ!Box 443 → 8443 weiterleitet; im LAN ist 443 zusätzlich direkt auf dem Pi
veröffentlicht. Die Adresse ist dadurch überall dieselbe:
`https://unidentists.social`.

Ablauf in Kurzform:

```
PC:  export-data.ps1  →  build-image.ps1  →  copy-to-pi.ps1
Pi:  .env + ddns/config.json  →  Uploads entpacken  →  restore-db.sh  →  docker compose up -d
```

Was du brauchst:

- SSH-Zugang zum Pi und ein 64-Bit-Betriebssystem darauf (Schritt 1)
- Die Domain `unidentists.social` bei name.com, plus einen API-Token (Schritt 3c)
- Zugriff auf die FRITZ!Box (Portfreigaben 443 und 80)
- Docker Desktop auf dem PC

**Wo welcher Schritt läuft** — die Reihenfolge ist nicht beliebig:

| Schritt | Wo             |                                                    |
| ------- | -------------- | -------------------------------------------------- |
| 1–2     | Pi             | Vorbereiten, Ports freimachen                      |
| 3       | Browser        | FRITZ!Box, name.com, AdGuard                       |
| 4–6     | Windows-PC     | Export, Build, Übertragung                         |
| 7–8     | Pi             | Konfigurieren, starten, prüfen                     |

Auf dem Pi liegen die Dateien (`docker-compose.yml`, `.env`, `db.sql`,
`ddns-config.example.json`, …) **erst nach Schritt 6** und dann in
`~/storyden`, nicht im Home-Verzeichnis. Vorher gibt es dort nichts zu
kopieren oder zu starten.

---

## Aktueller Stand (10.08.2026)

Diesen Block löschen, sobald die Seite läuft.

Dieses Verzeichnis war zwischenzeitlich nur in einem git worktree
(`.claude/worktrees/website-raspberry-pi-deployment-d6bb73/...`); der Branch
ist inzwischen nach `main` gemergt. Ab jetzt hier weiterarbeiten:

```
F:\Zahnmedizin\Forum\storyden\docker\compose\raspberrypi
```

`transfer/` ist gitignored und beim Merge nicht mitgekommen — Image, Dump und
Uploads wurden von Hand aus dem Worktree hierher kopiert, kein erneuter Build
nötig.

| | Schritt | Stand |
| - | ------- | ----- |
| ☑ | 1. Pi: 64 Bit prüfen | bestätigt |
| ☑ | 2. Ports freimachen | apache2 deaktiviert, `ss` liefert nichts mehr |
| ☑ | 3a. Öffentliche IPv4 | `91.135.163.168`, kein DS-Lite |
| ☑ | 3b. Portfreigaben | bestätigt: extern 443 → Pi 8443 |
| ☑ | 3c. A-Record bei name.com | zeigt auf die öffentliche IP |
| ☑ | 3d. AdGuard DNS-Rewrite | `dig` auf dem Pi liefert `192.168.178.30` |
| ☑ | 4. Daten exportieren | `transfer/` enthält Dump, Uploads und `.env` |
| ☑ | 5. Image bauen | arm64, getestet, `transfer/storyden-arm64.tar` |
| ☐ | **`transfer/.env`: `ACME_EMAIL=` ausfüllen** | **noch leer — vor Schritt 6 nachholen** |
| **→** | **6. Auf den Pi kopieren** | **hier weitermachen** |
| ☐ | 7. Auf dem Pi starten | |
| ☐ | 8. Prüfen | |

**Vor Schritt 6 außerdem noch einmal kurz durchgehen — leicht zu vergessen:**

- **`transfer\.env` einmal komplett durchlesen**, nicht nur `ACME_EMAIL`. Die
  Datei wurde automatisch aus dem alten Setup befüllt; `SITE_DOMAIN` und die
  beiden `PUBLIC_*`-Zeilen darin auf Tippfehler prüfen (`unidentists.social`
  mit **s**, nicht `unidentist.social`) — ein falscher Wert dort loggt später
  alle Nutzer aus.
- **name.com-API-Token besorgt?** Wird erst in Schritt 7 gebraucht
  (`ddns/config.json`), aber jetzt schon holen spart später einen
  Tab-Wechsel: [name.com → Account → API](https://www.name.com/account/settings/api).
- **SSH-Key statt Passwort** — optional, aber `copy-to-pi.ps1` fragt sonst bei
  jeder Datei einzeln nach dem Passwort. Befehl steht unten bei Schritt 6.
- Auf dem Pi läuft evtl. noch ein **certbot-Timer** von der alten Seite (siehe
  Schritt 2) — der scheitert jetzt an Port 80/443 und mailt Fehler.

---

## 1. Pi vorbereiten

Einloggen (Benutzername ggf. anpassen):

```bash
ssh pi@192.168.178.30
```

**64 Bit prüfen** — das ist die wichtigste Voraussetzung:

```bash
uname -m
```

Muss `aarch64` ergeben. Kommt `armv7l`, läuft ein 32-Bit-System und das
arm64-Image startet nicht. Dann entweder Raspberry Pi OS (64-bit) neu
aufsetzen oder in `build-image.ps1` `-Platform linux/arm/v7` verwenden — die
64-Bit-Variante ist deutlich schneller und der empfohlene Weg.

**Docker installieren** (falls noch nicht vorhanden):

```bash
curl -fsSL https://get.docker.com | sh
```

```bash
sudo usermod -aG docker $USER
```

Danach einmal ab- und neu anmelden (`exit`, dann wieder `ssh`), sonst braucht
jeder Docker-Befehl `sudo`. Prüfen:

```bash
docker run --rm hello-world
```

**Swap prüfen.** Gebaut wird nichts, aber Postgres und Node zusammen sind auf
4 GB knapp — 1–2 GB Swap als Puffer schadet nicht:

```bash
free -h
```

Bei weniger als 1 GB Swap:

```bash
sudo dphys-swapfile swapoff && sudo sed -i 's/^CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile && sudo dphys-swapfile setup && sudo dphys-swapfile swapon
```

---

## 2. Ports freimachen

Storyden braucht auf dem Host **80**, **443** und **8443**. Was dort lauscht:

```bash
sudo ss -tulpn | grep -E ':(80|443|8443)\s'
```

Auf diesem Pi ist es `apache2` — ein systemd-Dienst, kein Container. Er belegt
80 und 443 und muss weg:

```bash
sudo systemctl disable --now apache2
```

`disable` verhindert, dass er beim nächsten Neustart wiederkommt. Gelöscht wird
nichts: `sudo systemctl enable --now apache2` bringt die alte Seite zurück.
Vorher nachsehen, was Apache überhaupt ausliefert:

```bash
sudo apache2ctl -S 2>/dev/null | grep -E 'port|namevhost'
```

Kontrolle — der Befehl darf jetzt keine Zeile mehr ausgeben:

```bash
sudo ss -tulpn | grep -E ':(80|443|8443)\s'
```

Hatte die alte Seite ein Let's-Encrypt-Zertifikat, läuft im Hintergrund
wahrscheinlich noch ein certbot-Timer. Der versucht weiter zu verlängern,
scheitert an Port 80 (den hat jetzt Caddy) und schickt Fehlermails:

```bash
systemctl list-timers | grep -i certbot
```

Falls vorhanden und nicht mehr gebraucht:

```bash
sudo systemctl disable --now certbot.timer
```

### Die anderen Container bleiben, wo sie sind

Keiner der übrigen Container auf diesem Pi belegt 80, 443 oder 8443 — sie
veröffentlichen alle auf hohen Host-Ports:

| Container       | Host-Ports                                    | Konflikt |
| --------------- | --------------------------------------------- | -------- |
| `adguardhome`   | 53, 67, 180→80, 784, 853, 1443→443, 3001, 8853 | nein     |
| `nextcloud`     | 5080→80, 5443→443                             | nein     |
| `pi-website`    | 5441→80                                       | nein     |
| `portainer`     | 8000, 9443                                    | nein     |
| `wg-easy`       | 51820, 51821                                  | nein     |
| `home-assistant`| host-Netzwerk                                 | nein     |

Storyden selbst veröffentlicht **keinen** Port 8000 auf dem Host — der Container
spricht intern mit Caddy. Portainer darf 8000 also behalten.

Ist `pi-website` (nginx, Stack `my-website`) die alte Seite und soll auch weg:

```bash
docker stop pi-website && docker update --restart=no pi-website
```

Die Volumes bleiben erhalten, die Daten sind nicht verloren.

---

## 3. Domain, DynDNS und Router

### 3a. Zuerst: gibt es überhaupt eine öffentliche IPv4?

Vom Pi aus:

```bash
curl -4 ifconfig.me
```

Kommt ein Fehler oder eine Adresse, die mit `100.64.` bis `100.127.` beginnt,
hast du **DS-Lite** — keine eigene öffentliche IPv4. Dann ist die Seite von
außen über IPv4 grundsätzlich nicht erreichbar, unabhängig von Portfreigaben.
Lösung: beim Provider eine echte IPv4 beantragen (bei vielen kostenlos auf
Anfrage), und für das Zertifikat **Anhang A** (DNS-01).

Die Zahl aus diesem Befehl muss mit der übereinstimmen, die die FRITZ!Box unter
`Übersicht → Verbindungen` anzeigt.

### 3b. Portfreigaben in der FRITZ!Box

`Internet → Freigaben → Portfreigaben → Gerät für Freigaben hinzufügen → Pi`

| Extern | → | Pi   | Protokoll | Wofür                                             |
| ------ | - | ---- | --------- | ------------------------------------------------- |
| 443    | → | 8443 | TCP       | HTTPS **und** das Zertifikat (TLS-ALPN-Challenge)  |
| 80     | → | 80   | TCP       | Weiterleitung von `http://` + Ersatz-Challenge     |

**Der externe Port muss exakt 443 sein, nicht 433.** Let's Encrypt verbindet
sich für die TLS-ALPN-Challenge immer auf 443; auf 433 kommt weder die
Prüfung an noch ein normaler Besucher. Das ist die Stelle, an der sich ein
Zahlendreher am längsten versteckt.

Mit der 443er-Freigabe reicht das für das Zertifikat schon aus — Port 80 ist
nur noch Komfort (wer `unidentists.social` ohne `https://` eintippt) und
Rückfallweg. Fehlt eine der beiden Freigaben, kannst du im `Caddyfile` den
jeweils nicht erreichbaren Challenge-Typ abschalten, dann probiert Caddy ihn
nicht erst erfolglos aus. Die Zeilen stehen dort kommentiert.

### 3c. Domain bei name.com auf den Anschluss zeigen lassen

Deine IP zu Hause wechselt, also muss der A-Record automatisch nachgeführt
werden. **Die FRITZ!Box kann das für name.com nicht selbst**: ihr DynDNS ruft
nur eine einfache GET-URL auf, die name.com-API erwartet einen JSON-PUT mit
API-Token. Deshalb übernimmt das der `ddns`-Container auf dem Pi.

**1. API-Token holen:** [name.com → Account → API](https://www.name.com/account/settings/api).
Token erzeugen und notieren, dazu deinen name.com-Benutzernamen.

**2. A-Record anlegen** unter `MY DOMAINS → unidentists.social → Manage DNS
Records`, falls noch keiner existiert:

| Type | Host    | Answer                       | TTL |
| ---- | ------- | ---------------------------- | --- |
| A    | (leer)  | die IP aus Schritt 3a        | 300 |

Host leer bedeutet die Domain selbst (Apex). Trage die aktuelle IP direkt ein —
damit funktioniert die Seite sofort, auch bevor der `ddns`-Container läuft. Der
Container korrigiert den Wert später bei jedem IP-Wechsel. Die TTL muss auf
300 stehen (name.coms Minimum), sonst hängen Besucher nach einem IP-Wechsel
lange auf der alten Adresse.

**3. Den Container konfigurieren — erst in Schritt 7.** Die Datei
`ddns-config.example.json` liegt bis dahin nur auf dem PC und kommt mit
`copy-to-pi.ps1` nach `~/storyden/`. Vorher gibt es auf dem Pi nichts zu
kopieren.

**Wenn das nicht klappt:** Der Weg ganz ohne Zusatz-Container steht in
**Anhang B** (MyFRITZ! plus ANAME-Record).

### 3d. Zugriff aus dem eigenen LAN

Die FRITZ!Box leitet Anfragen aus dem LAN an die eigene öffentliche IP nicht
zuverlässig zurück (NAT-Hairpinning). Weil du AdGuard Home als DNS im Netz
betreibst, ist die saubere Lösung ein Split-Horizon-Eintrag:

`AdGuard Home → Filter → DNS-Rewrites → Hinzufügen`

| Domain               | IP-Adresse       |
| -------------------- | ---------------- |
| `unidentists.social` | `192.168.178.30` |

Damit landen Geräte im WLAN direkt auf dem Pi, unter derselben Adresse wie von
außen. Das funktioniert, weil Caddy zusätzlich auf Host-Port 443 liegt (siehe
`docker-compose.yml`) — sonst müsste im LAN `:8443` angehängt werden, und die
API-Aufrufe des Frontends würden ins Leere laufen.

### 3e. Prüfen

```bash
dig +short unidentists.social
```

Muss die IP aus 3a liefern. Von außen testen — im Handynetz, nicht im WLAN:

```bash
curl -I https://unidentists.social
```

---

## 4. Daten vom PC exportieren

Auf dem **PC**, in PowerShell. Der Postgres-Container des aktuellen Setups muss
laufen (`docker ps` → `compose-postgres-1`):

Alle Skripte laufen aus **diesem** Verzeichnis (`docker/compose/raspberrypi`)
und finden das Projekt-Root selbst. Also dorthin wechseln, wo diese README
liegt:

```powershell
cd <projektverzeichnis>\docker\compose\raspberrypi
```

```powershell
.\export-data.ps1
```

Läuft das Ganze aus einem git worktree, liegen `.env` und `data/` des alten
Setups nur im Haupt-Checkout. Dann zusätzlich den Quellpfad angeben:

```powershell
.\export-data.ps1 -SourceRoot F:\Zahnmedizin\Forum\storyden
```

Legt in `transfer\` ab:

| Datei          | Inhalt                                                     |
| -------------- | ---------------------------------------------------------- |
| `db.sql`       | kompletter Dump der aktuellen Postgres-Datenbank           |
| `data.tar.gz`  | Assets und Avatare aus `docker/compose/data`               |
| `.env`         | Pi-Konfiguration, Secrets aus dem alten Setup übernommen   |

Danach `transfer\.env` öffnen und die Domain-Zeilen prüfen:

```
SITE_DOMAIN=unidentists.social
ACME_EMAIL=deine@mail.de
PUBLIC_WEB_ADDRESS=https://unidentists.social
PUBLIC_API_ADDRESS=https://unidentists.social
```

Ohne Port, weil die FRITZ!Box extern 443 auf Pi:8443 weiterleitet. Fehlt diese
Freigabe, gilt stattdessen Variante B aus der `.env` (mit `:8443`) — und im
`docker-compose.yml` muss beim `caddy`-Service die Zeile `"443:8443"` raus.

`JWT_SECRET`, die Brevo-Zugangsdaten und `POSTGRES_PASSWORD` sind schon
eingetragen — die kommen 1:1 aus `docker/compose/.env`. Das ist wichtig:
mit einem neuen `JWT_SECRET` wären alle bestehenden Sessions ungültig.

`transfer\` enthält Passwörter und ist gitignored.

---

## 5. Image auf dem PC bauen

```powershell
.\build-image.ps1
```

Erzeugt `transfer\storyden-arm64.tar` und die Tags `storyden-rpi:latest`
sowie `storyden-rpi:<datum>`.

Beim ersten Mal 10–20 Minuten, danach greift der BuildKit-Cache und es sind
meist wenige Minuten.

Emuliert wird dabei so gut wie nichts: Go wird nativ nach arm64
cross-kompiliert, der Next.js-Build läuft ebenfalls nativ (sein Ergebnis ist
reines JavaScript und damit architekturunabhängig), und nur das Runtime-Image
mit Node, tesseract und dem Go-Binary ist arm64. Ein vollständig emulierter
arm64-Build funktioniert an dieser Stelle **nicht** — Turbopack läuft unter
QEMU in interne Timeouts (`TurbopackInternalError ... deadline has elapsed`).

**Optional: Image vor dem Transfer lokal testen.** Docker Desktop kann das
arm64-Image per QEMU ausführen. Läuft auf Port 18000 mit einer
Wegwerf-SQLite-Datenbank und stört das bestehende Setup nicht:

```powershell
docker run -d --name storyden-smoke --platform linux/arm64 -p 18000:8000 -e JWT_SECRET=nur-zum-testen storyden-rpi:latest
```

Nach etwa einer Minute muss `http://localhost:18000/healthz` mit 200 antworten
und `http://localhost:18000/` fertiges HTML liefern. Danach aufräumen:

```powershell
docker rm -f storyden-smoke
```

---

## 6. Alles auf den Pi kopieren  ← hier weitermachen

```powershell
.\copy-to-pi.ps1 -PiUser pi
```

Kopiert Image, Dump, Uploads, `.env`, `docker-compose.yml`, `Caddyfile` und die
Skripte nach `/home/pi/storyden` und lädt das Image dort in Docker
(`docker load`). Gestartet wird noch nichts.

Zu übertragen sind rund 310 MB (Image 185 MB, Uploads 125 MB, Dump 0,5 MB) —
im LAN etwa eine Minute. Bei Updates später nur noch das Image.

Bei jedem `scp`/`ssh` fragt der Pi nach dem Passwort. Wer das öfter macht,
legt einmal einen SSH-Key an:

```powershell
ssh-keygen -t ed25519; type $env:USERPROFILE\.ssh\id_ed25519.pub | ssh pi@192.168.178.30 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

---

## 7. Auf dem Pi starten

```bash
ssh pi@192.168.178.30 && cd ~/storyden
```

Konfiguration noch einmal ansehen:

```bash
nano .env
```

DynDNS einrichten — Benutzername und API-Token von name.com eintragen
(Schritt 3c):

```bash
mkdir -p ddns && cp ddns-config.example.json ddns/config.json && nano ddns/config.json
```

```bash
sudo chown -R 1000:1000 ddns
```

Uploads entpacken. Der Storyden-Container läuft als UID 1001 und kann sonst
nicht in das Verzeichnis schreiben:

```bash
mkdir -p data && sudo tar xzf data.tar.gz -C data && sudo chown -R 1001:1001 data
```

Datenbank einspielen. Das Skript startet Postgres, wartet, prüft dass die
Datenbank leer ist, spielt den Dump ein und zeigt danach die Zeilenzahlen:

```bash
bash restore-db.sh db.sql
```

Alles starten:

```bash
docker compose up -d
```

Beim ersten Start dauert es 1–2 Minuten, bis Next.js auf dem Pi bereit ist.
Mitlesen:

```bash
docker compose logs -f
```

Im Caddy-Log muss eine Zeile wie `certificate obtained successfully` stehen,
im `ddns`-Log eine Bestätigung, dass der A-Record aktuell ist. Danach:

```
https://unidentists.social
```

---

## 8. Nach dem Start prüfen

- [ ] Aufruf **aus dem Handynetz** (nicht WLAN): Seite lädt über HTTPS, das
      Schloss im Browser ist grün, keine Zertifikatswarnung
- [ ] Aufruf **aus dem WLAN** unter derselben Adresse (setzt den
      AdGuard-DNS-Rewrite aus Schritt 3d voraus)
- [ ] Login mit einem bestehenden Account funktioniert
- [ ] Alte Beiträge, Threads und Bilder sind da
- [ ] Registrierung: kommt die Brevo-Bestätigungsmail an, und ist der Link
      darin `https://unidentists.social/...`?
- [ ] Bild und PDF hochladen, danach nach Text aus dem PDF suchen (OCR)
- [ ] Google-Drive-Bibliothek in den Admin-Einstellungen erreichbar
- [ ] `http://unidentists.social` leitet auf HTTPS um
- [ ] `docker compose logs ddns` zeigt den A-Record als aktuell
- [ ] Nextcloud, AdGuard, Home Assistant und wg-easy laufen weiter
- [ ] Neustart-Test: `sudo reboot`, danach kommt alles von selbst wieder hoch

Wenn alles läuft, das alte Setup auf dem PC herunterfahren, damit nicht
versehentlich zwei Instanzen mit auseinanderlaufenden Daten existieren:

```powershell
docker compose -f F:\Zahnmedizin\Forum\storyden\docker\compose\docker-compose.yml down
```

Das Volume mit der alten Datenbank bleibt dabei bestehen — als Sicherheitsnetz
für die ersten Tage.

**Passkeys müssen neu registriert werden.** Sie sind an die Origin gebunden,
unter der sie angelegt wurden (vorher `localhost`, jetzt die Domain). Passwörter
und Sessions sind davon nicht betroffen, solange `JWT_SECRET` übernommen wurde.

---

## 9. Update nach Code-Änderungen

Auf dem PC:

```powershell
.\build-image.ps1
```

```powershell
.\copy-to-pi.ps1 -PiUser pi -ImageOnly
```

Auf dem Pi:

```bash
cd ~/storyden && docker compose up -d
```

`-ImageOnly` lässt `.env`, Dump und Uploads unangetastet — es wird nur das
Image ersetzt. Datenbank-Schemaänderungen wendet die App beim Start selbst an.

**Zurückrollen** auf eine ältere Version: in `.env` die Zeile
`STORYDEN_IMAGE=storyden-rpi:latest` auf einen Datums-Tag ändern
(`docker images storyden-rpi` zeigt, was da ist), dann `docker compose up -d`.

---

## 10. Backups

```bash
bash backup.sh
```

Schreibt Datenbank-Dump und Uploads nach `./backups` und löscht Sicherungen,
die älter als 14 Tage sind. Täglich per cron:

```bash
crontab -e
```

```
30 3 * * * cd /home/pi/storyden && bash backup.sh >> backup.log 2>&1
```

Die Backups liegen damit auf derselben SD-Karte wie die Daten. SD-Karten
sterben — zusätzlich woanders ablegen, z.B. per `rsync` auf den PC.

Wiederherstellen:

```bash
gunzip -c backups/db-2026-08-11-0330.sql.gz | docker compose exec -T postgres psql -U storyden storyden
```

---

## Fehlersuche

```bash
docker compose ps
docker compose logs -f storyden
docker compose logs -f caddy
docker compose exec storyden wget -qO- http://127.0.0.1:8000/healthz
```

**`cp: cannot stat 'ddns-config.example.json'` oder `no configuration file
provided`** — die Dateien sind noch nicht auf dem Pi oder du stehst im falschen
Verzeichnis. Alles liegt nach Schritt 6 in `~/storyden`:

```bash
cd ~/storyden && ls
```

Ist das Verzeichnis leer oder fehlt, wurde `copy-to-pi.ps1` (Schritt 6) noch
nicht ausgeführt.

**`bind: address already in use` beim Start von `caddy`** — auf dem Host lauscht
noch etwas auf 80 oder 443, in der Regel apache2 (Schritt 2). Prüfen mit
`sudo ss -tulpn | grep -E ':(80|443|8443)\s'`. Kommt apache2 nach einem
Neustart wieder, wurde nur `stop` statt `disable --now` benutzt.

**`certificate has expired`, obwohl Storyden noch nicht läuft** — dann antwortet
noch der alte Webserver auf 443 mit seinem abgelaufenen Zertifikat. Wer es ist:

```bash
echo | openssl s_client -connect 192.168.178.30:443 -servername unidentists.social 2>/dev/null | openssl x509 -noout -subject -issuer -dates
```

Zeigt der `issuer` nicht `Let's Encrypt`, ist es nicht Caddy. Schritt 2
nachholen.

**`dig` auf dem Pi liefert 192.168.178.30 statt der öffentlichen IP** — das ist
richtig so: der DNS-Rewrite in AdGuard greift, und der Pi benutzt AdGuard selbst
als Resolver. Den öffentlichen Record über einen externen Resolver prüfen:

```bash
dig +short unidentists.social @1.1.1.1
```

**`exec format error` beim Start von `storyden`** — das Image ist für die
falsche Architektur. `uname -m` auf dem Pi prüfen und mit dem `-Platform` aus
`build-image.ps1` vergleichen.

**Compose sucht das Image in einer Registry** (`pull access denied`) — das
`docker load` hat nicht funktioniert. `docker images storyden-rpi` auf dem Pi
prüfen, ggf. `docker load -i storyden-arm64.tar` erneut ausführen.

**Kein Zertifikat.** Caddy-Log lesen, dann der Reihe nach:

1. Zeigt der DNS auf die aktuelle IP? `dig +short unidentists.social` gegen
   `curl -4 ifconfig.me` auf dem Pi vergleichen. Weicht es ab, ist der
   `ddns`-Container das Problem: `docker compose logs ddns`.
2. Steht in der FRITZ!Box extern wirklich **443** und nicht 433? Auf 433
   kommt die TLS-ALPN-Challenge nie an.
3. Kommt der Port von außen durch? Im Handynetz testen, nicht im WLAN:

```bash
curl -I https://unidentists.social
```

Ist nur eine der beiden Freigaben (443 oder 80) aktiv, im `Caddyfile` den
nicht erreichbaren Challenge-Typ abschalten — sonst verbrennt Caddy Versuche
an einem Weg, der nie funktionieren kann.

Let's Encrypt drosselt nach mehreren Fehlversuchen für eine Stunde. Zum
Ausprobieren deshalb im `Caddyfile` die `acme_ca`-Zeile mit der Staging-URL
einkommentieren; das Zertifikat ist dann im Browser nicht vertrauenswürdig,
aber die Ausstellung selbst wird real durchgespielt. Vor dem Scharfstellen die
Zeile wieder auskommentieren **und** `docker volume rm storyden_caddy_data`,
damit Caddy das Staging-Zertifikat nicht behält.

**Aus dem WLAN nicht erreichbar, von außen schon.** NAT-Hairpinning der
FRITZ!Box. DNS-Rewrite in AdGuard Home setzen (Schritt 3d) und prüfen, dass die
Clients AdGuard wirklich als DNS benutzen: `nslookup unidentists.social` muss
`192.168.178.30` liefern.

**Sofort wieder ausgeloggt / Login geht nicht.** `PUBLIC_WEB_ADDRESS` stimmt
nicht exakt mit der aufgerufenen URL überein (Port vergessen, Slash am Ende,
`www.` davor).

**`ddns` aktualisiert nichts.** `docker compose logs ddns`. Häufig: der
A-Record existiert bei name.com noch nicht (der Updater ändert nur, er legt
nicht an), das Token ist abgelaufen, oder `ddns/` gehört nicht UID 1000 —
dann kann der Container seinen Status nicht schreiben.

**Uploads schlagen bei großen Dateien fehl.** `MAX_UPLOAD_SIZE_MB` in `.env`
und `max_size` im `Caddyfile` angleichen.

**Pi ist dauerhaft am Anschlag.** `docker stats` ansehen. Ist es der
Storyden-Container direkt nach einem Neustart, ist es meist der OCR-Backfill —
`OCR_BACKFILL_ENABLED=false` ist in `.env` bereits der Standard.

**Alte Website zurückholen.** Storyden stoppen, apache2 wieder aktivieren:

```bash
cd ~/storyden && docker compose down
```

```bash
sudo systemctl enable --now apache2
```

---

## Anhang A: Zertifikat ohne offene Ports (DNS-01)

Nötig bei DS-Lite oder wenn der Provider 80 und 443 blockiert. Caddy beweist den
Domain-Besitz dann über einen TXT-Record bei name.com statt über eine
eingehende Verbindung. Dafür braucht Caddy ein Provider-Plugin, das im
Standard-Image nicht enthalten ist.

**Wichtig:** Das löst nur das Zertifikat. Damit Besucher den Pi überhaupt
erreichen, braucht es trotzdem eine erreichbare öffentliche Adresse — bei
DS-Lite also entweder eine echte IPv4 vom Provider, IPv6-only-Betrieb oder
einen Tunnel.

`Dockerfile.caddy` neben der `docker-compose.yml`:

```dockerfile
FROM caddy:2-builder AS builder
RUN xcaddy build --with github.com/caddy-dns/namedotcom

FROM caddy:2-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

Im `docker-compose.yml` beim `caddy`-Service `image: caddy:2-alpine` ersetzen
durch:

```yaml
    build:
      context: .
      dockerfile: Dockerfile.caddy
```

und in dessen `environment` ergänzen:

```yaml
      - NAMEDOTCOM_USER=${NAMEDOTCOM_USER}
      - NAMEDOTCOM_TOKEN=${NAMEDOTCOM_TOKEN}
```

Im `Caddyfile` den kompletten `tls`-Block ersetzen:

```
	tls {
		dns namedotcom {
			user {$NAMEDOTCOM_USER}
			token {$NAMEDOTCOM_TOKEN}
		}
	}
```

Es sind dieselben Zugangsdaten wie für den `ddns`-Container. Die Portfreigabe
für 80 entfällt damit, ebenso der `http://`-Block im `Caddyfile`. Dieses
Caddy-Image ist klein und lässt sich ausnahmsweise direkt auf dem Pi bauen.

---

## Anhang B: DynDNS ohne Zusatz-Container (MyFRITZ! + ANAME)

Für den Fall, dass der `ddns`-Container nicht läuft — falsches Token, API
gesperrt, oder du willst einfach keinen API-Schlüssel auf dem Pi liegen haben.

### Wie es funktioniert

Die FRITZ!Box kennt ihre öffentliche IP immer als Erste und meldet sie
zuverlässig an genau **einen** Dienst: MyFRITZ!. Daraus entsteht ein
Hostname wie `abc123def456.myfritz.net`, der immer auf deinen Anschluss zeigt.

Damit `unidentists.social` demselben Ziel folgt, verweist die Domain bei
name.com auf diesen Hostnamen. Ein normaler **CNAME** ginge dafür nicht: am
Apex einer Zone ist ein CNAME laut DNS-Standard verboten, weil dort schon
SOA- und NS-Records stehen und ein CNAME keine Geschwister-Records neben sich
duldet. name.com bietet deshalb **ANAME** an: das löst das Ziel auf den
eigenen Nameservern auf und liefert nach außen eine ganz normale A-Antwort.

Die Kette ist dann: Besucher → name.com (ANAME) → myfritz.net → deine IP.
Nichts davon läuft auf dem Pi.

### 1. MyFRITZ! aktivieren

`Internet → MyFRITZ!-Konto` → E-Mail-Adresse eintragen → Bestätigungsmail
öffnen und den Link anklicken.

Danach steht der Hostname unter `Internet → MyFRITZ!-Konto`, in der Form
`xxxxxxxxxxxx.myfritz.net`. Genau notieren.

Kontrolle vom Pi aus — muss deine öffentliche IP liefern:

```bash
dig +short xxxxxxxxxxxx.myfritz.net
```

Kommt hier nichts, ist MyFRITZ! nicht fertig eingerichtet. Erst weitermachen,
wenn dieser Befehl die richtige IP zeigt.

### 2. ANAME bei name.com setzen

`MY DOMAINS → unidentists.social → Manage DNS Records`

Den vorhandenen **A-Record der Domain löschen** — sonst konkurrieren zwei
Antworten für denselben Namen. Dann anlegen:

| Type  | Host   | Answer                        | TTL |
| ----- | ------ | ----------------------------- | --- |
| ANAME | (leer) | `xxxxxxxxxxxx.myfritz.net`    | 300 |

Nimmt die Oberfläche den leeren Host bei ANAME nicht an, unterstützt name.com
den Apex an dieser Stelle nicht. Dann zwei Möglichkeiten:

- Subdomain benutzen, z.B. Host `forum` → `forum.unidentists.social`. Dafür in
  der `.env` `SITE_DOMAIN` und beide `PUBLIC_*`-Zeilen auf die Subdomain
  ändern. Achtung: das ändert die Origin, alle Nutzer werden ausgeloggt und
  Passkeys müssen neu registriert werden — also besser vor dem Start
  entscheiden.
- Beim `ddns`-Container bleiben (Schritt 3c), der schreibt einen echten
  A-Record und braucht kein ANAME.

### 3. Container entfernen

```bash
cd ~/storyden && docker compose stop ddns && docker compose rm -f ddns
```

Danach den `ddns`-Service aus der `docker-compose.yml` löschen, sonst kommt er
beim nächsten `docker compose up -d` wieder.

### 4. Prüfen

```bash
dig +short unidentists.social
```

Muss dieselbe IP liefern wie `dig +short xxxxxxxxxxxx.myfritz.net`. Nach dem
Anlegen kann das bis zu einer Stunde dauern, in Einzelfällen bis 24 Stunden.

### Was du dafür in Kauf nimmst

- **Ein Glied mehr in der Kette.** Fällt MyFRITZ! aus, ist die Domain nicht
  erreichbar, obwohl der Pi läuft.
- **Träger bei IP-Wechseln.** Zur TTL der Domain kommt die Auflösung von
  myfritz.net dazu.
- **Die Umstellung selbst ist nicht sofort sichtbar.** Ein gelöschter
  A-Record kann noch in Resolvern stecken; Let's Encrypt sieht dann kurzzeitig
  die alte Antwort.

### Variante: anderer DynDNS-Anbieter statt MyFRITZ!

Dasselbe Muster funktioniert mit jedem Anbieter, den die FRITZ!Box unter
`Internet → Freigaben → DynDNS` bedienen kann (dynv6, DuckDNS, deSEC, No-IP —
dort auch „Benutzerdefiniert" mit Update-URL). Der ANAME bei name.com zeigt
dann eben auf `deinname.dynv6.net` statt auf myfritz.net. Sinnvoll, wenn du
den DynDNS-Namen selbst kontrollieren willst; technisch ist es dieselbe Kette
mit denselben Nachteilen.
