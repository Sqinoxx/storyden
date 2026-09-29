# Update auf den Ubuntu-Server – ohne Datenbank-Transfer

Diese Anleitung bringt eine neue Version von Storyden auf den laufenden
Ubuntu-Server. **Nur das Programm (Docker-Image) wird ausgetauscht.**
Datenbank, Uploads und Konfiguration bleiben auf dem Server und werden nicht
angefasst.

| Was | Wo | Beim Update |
| - | - | - |
| Datenbank | Server, Docker-Volume `storyden_postgres-data` | bleibt |
| Uploads/Bilder | Server, `~/storyden/data` | bleibt |
| Konfiguration/Secrets | Server, `~/storyden/.env` | bleibt |
| Programm (Frontend + Backend) | Docker-Image `storyden-server` | **wird ersetzt** |

Neue Tabellen und Spalten legt die App beim Start selbst an. Ein eigener
Migrationsschritt ist nicht nötig.

Dauer: etwa 15–30 Minuten, der Server ist davon nur 1–2 Minuten offline.

---

## Schritt 0 – Code auf dem PC aktualisieren

In PowerShell auf dem Windows-PC:

```powershell
cd F:\Zahnmedizin\Forum\storyden
git checkout main
git pull
```

`build-image.ps1` baut aus genau diesem Stand. Was hier nicht aktuell ist,
landet auch nicht auf dem Server.

---

## Schritt 1 – Backup auf dem Server (Pflicht)

Auch wenn die Datenbank nicht übertragen wird: Beim Start passt die neue
Version das Datenbankschema an. Deshalb vorher sichern.

```powershell
ssh unidentist@192.168.178.105
```

```bash
cd ~/storyden && bash backup.sh
ls -lh backups | tail -2
```

Es müssen zwei frische Dateien mit dem heutigen Datum da sein:
`db-<datum>.sql.gz` und `assets-<datum>.tar.gz`.

Danach die Sicherung zusätzlich auf den PC holen (neues PowerShell-Fenster):

```powershell
scp -r unidentist@192.168.178.105:~/storyden/backups F:\Storyden-Backups\
```

---

## Schritt 2 – Image auf dem PC bauen

```powershell
cd F:\Zahnmedizin\Forum\storyden\docker\compose\ubuntuserver
.\build-image.ps1
```

- Baut für `linux/amd64`. Falls `uname -m` auf dem Server `aarch64` zeigt:
  `.\build-image.ps1 -Platform linux/arm64`
- Vergibt zusätzlich einen Datums-Tag (z. B. `storyden-server:2026-09-29`),
  damit man später zurückrollen kann.

---

## Schritt 3 – Nur das Image übertragen

```powershell
.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
```

`-ImageOnly` ist der entscheidende Schalter: **Es werden weder `db.sql` noch
`data.tar.gz` noch `.env` übertragen.** Das Skript kopiert nur das Image und
lädt es auf dem Server mit `docker load`.

> **Niemals ohne `-ImageOnly` aufrufen**, wenn der Server schon läuft. Sonst
> werden `.env` und alte Dumps vom PC mitkopiert.

---

## Schritt 4 – Auf dem Server neu starten

```bash
cd ~/storyden
docker compose up -d
docker compose ps
```

Alle Container sollten nach etwa einer Minute `Up` bzw. `healthy` zeigen. Die
Logs beim ersten Start beobachten (Beenden mit `Strg+C`):

```bash
docker compose logs -f storyden
```

Alternativ im Menü: `./menu.sh` → Health-Check.

---

## Schritt 5 – Kurz prüfen

1. Das Forum im Browser öffnen, mit dem Admin-Konto einloggen.
   Nach diesem Update müssen sich **alle Nutzer einmal neu anmelden**. Die
   Sitzungen werden jetzt sicherer gespeichert, alte Sitzungen gelten nicht
   mehr. Das ist gewollt.
2. Einen Beitrag öffnen, ein Bild ansehen. Bilder sind jetzt nur noch für
   eingeloggte Nutzer abrufbar.
3. Weiter mit Schritt 6 (nur beim ersten Update mit den Sicherheitsfixes
   nötig).

---

## Schritt 6 – Einmalig: Admin-Einstellungen setzen

Aufrufen: `https://<deine-domain>/admin?tab=system`
(Admin → Tab **System**)

### 6a) Client-IP: damit die App echte Besucher-IPs sieht

**Worum es geht:** Auf dem Server nimmt nicht Storyden die Besucher entgegen,
sondern Caddy. Caddy sorgt für HTTPS und reicht jede Anfrage an Storyden
weiter:

```
Besucher A (IP 91.12.34.56)  ┐
Besucher B (IP 84.98.76.54)  ├──►  Caddy (172.18.0.3)  ──►  Storyden
Angreifer  (IP 5.6.7.8)      ┘
```

Storyden sieht deshalb bei **jeder** Anfrage nur Caddy als Absender, also
immer dieselbe interne Adresse `172.18.0.x`. Die echte IP des Besuchers
schreibt Caddy in einen Zusatz-Header namens `X-Forwarded-For`. Storyden liest
ihn aber nur, wenn man ihm sagt, dass Caddy vertrauenswürdig ist.

**Warum das wichtig ist:** Das Rate-Limit („höchstens N Anfragen pro Stunde“)
wird **pro IP-Adresse** gezählt. Wenn alle dieselbe IP haben, teilen sich
alle Besucher ein gemeinsames Kontingent. Ein einzelner Angreifer oder Bot,
der viele Anfragen schickt, verbraucht das Kontingent dann für alle, und das
ganze Forum antwortet mit „Too many requests“.

**Einstellung** unter *Client IP strategy*:

| Feld | Wert |
| - | - |
| Client IP mode | **X-Forwarded-For with trusted proxy CIDRs** |
| Trusted proxy CIDRs | `172.16.0.0/12` |

`172.16.0.0/12` ist der Adressbereich, aus dem Docker die internen Adressen
der Container vergibt, also auch Caddys Adresse. Sinngemäß heißt die
Einstellung: „Anfragen, die von Caddy kommen, darfst du die echte
Besucher-IP glauben, allen anderen nicht.“

Unten auf **Save** klicken.

**Kontrolle:** Weiter unten auf derselben Seite zeigt Storyden an, welche IP
es für dich erkennt (*resolved IP*). Dort muss jetzt deine öffentliche
Internet-IP stehen (zum Vergleich z. B. auf wieistmeineip.de nachsehen), nicht
mehr `172.x.x.x`. Im Heimnetz kann dort auch deine lokale Adresse
`192.168.178.x` stehen, das ist ebenfalls richtig.

Das lässt sich bewusst nicht über die `.env` einstellen. Diese Einstellung
steht in der Datenbank und bleibt bei allen künftigen Updates erhalten, sie
ist also nur **einmal** nötig.

### 6b) Login-Sperre einstellen (optional)

Unter *Login lockout*:

| Feld | Bedeutung | Standard |
| - | - | - |
| Failed logins before lockout | So viele falsche Passwörter in Folge darf ein Konto bekommen, bevor es gesperrt wird | 5 |
| Lockout duration | So lange bleibt das Konto nach dem letzten Fehlversuch gesperrt | 15 Minuten |

Die Sperre gilt **pro Konto** (E-Mail-Adresse oder Benutzername), egal von
welcher IP die Versuche kommen. Ein erfolgreicher Login vor Erreichen der
Grenze setzt den Zähler zurück. Wer gesperrt ist, bekommt auch mit dem
richtigen Passwort erst nach Ablauf der Sperrzeit wieder Zugang.

Die Rate-Limits pro IP stehen darüber unter *Rate limits*. Die
Standardwerte passen für ein Forum dieser Größe und müssen normalerweise nicht
geändert werden.

---

## Falls etwas schiefgeht: zurückrollen

**1. Altes Image wählen:**

```bash
docker images storyden-server
```

Den Datums-Tag der Vorversion in `~/storyden/.env` eintragen:

```
STORYDEN_IMAGE=storyden-server:2026-08-22
```

**2. Datenbank auf den Stand vor dem Update zurücksetzen.** Die neue Version
hat das Schema erweitert. Die alte Version kommt mit dem erweiterten Schema
nicht sicher zurecht, deshalb das Backup aus Schritt 1 einspielen:

```bash
cd ~/storyden
docker compose stop storyden
docker compose exec -T postgres psql -U storyden -d postgres -c "DROP DATABASE storyden WITH (FORCE);"
docker compose exec -T postgres psql -U storyden -d postgres -c "CREATE DATABASE storyden OWNER storyden;"
gunzip -c backups/db-<datum-aus-schritt-1>.sql.gz | docker compose exec -T postgres psql -U storyden storyden
docker compose up -d storyden
```

Achtung: Alles, was zwischen Update und Zurückrollen im Forum geschrieben
wurde, geht dabei verloren. Deshalb nach dem Update zügig prüfen (Schritt 5).

---

## Kurzfassung

```powershell
# PC
cd F:\Zahnmedizin\Forum\storyden; git pull
```
```bash
# Server
cd ~/storyden && bash backup.sh
```
```powershell
# PC
cd F:\Zahnmedizin\Forum\storyden\docker\compose\ubuntuserver
.\build-image.ps1
.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
```
```bash
# Server
cd ~/storyden && docker compose up -d && docker compose ps
```
