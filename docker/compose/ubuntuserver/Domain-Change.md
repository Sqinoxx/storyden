# Domainwechsel: `unidentists.social` → `unidentists.de`

Schritt-für-Schritt-Anleitung für den Umzug des Forums auf dem Ubuntu-Server
(`192.168.178.105`) auf die neue Domain **`unidentists.de`**.

Die Kurzfassung steht schon in [WARTUNG-Ubuntu.md](WARTUNG-Ubuntu.md),
Abschnitt „Domain ändern“. Diese Datei geht jeden Schritt einzeln durch,
inklusive Domain-Provider, FRITZ!Box, AdGuard und Weiterleitung der alten
Domain.

---

## Überblick

| Wo                        | Was                                                         | Pflicht?        |
| ------------------------- | ----------------------------------------------------------- | --------------- |
| Domain-Provider (neu)     | A-Records für `unidentists.de` und `www.unidentists.de`      | ja              |
| Domain-Provider (neu)     | SPF/DKIM/DMARC, falls Mails von `@unidentists.de` kommen     | für E-Mail ja   |
| FRITZ!Box                 | Portfreigaben prüfen, DNS-Rebind-Ausnahme                    | prüfen          |
| AdGuard Home (Pi)         | DNS-Rewrite für die neue Domain                              | ja (LAN-Zugriff)|
| Ubuntu-Server `.env`      | `SITE_DOMAIN`, `PUBLIC_*`, `SMTP_FROM_ADDRESS`               | ja              |
| Ubuntu-Server `ddns/`     | Domain im DynDNS-Updater                                     | falls aktiv     |
| Ubuntu-Server `Caddyfile` | Alte Domain auf neue weiterleiten                            | empfohlen       |
| Windows-PC `.env`         | Gleiche Änderungen wie auf dem Server                        | ja              |
| OAuth-Apps (Google usw.)  | Neue Redirect-URL                                            | falls genutzt   |

**Was der Wechsel für die Nutzer bedeutet:**

- Alle werden einmal **ausgeloggt** (Cookies hängen an der Domain).
- **Passkeys funktionieren nicht mehr** und müssen neu registriert werden —
  sie sind fest an die Domain gebunden. Login per Passwort/E-Mail geht weiter.
- Daten (Accounts, Beiträge, Uploads) bleiben vollständig erhalten. Die
  Datenbank speichert keine Domain.

Am besten außerhalb der Hauptnutzungszeit umstellen und die Nutzer ein paar
Tage vorher informieren.

---

## Schritt 1: Vorbereitung

### 1a. Nutzer informieren

Kurze Ankündigung im Forum, z. B.:

> Am `<Datum>` zieht das Forum auf **unidentists.de** um. Die alte Adresse
> leitet automatisch weiter. Ihr müsst euch einmal neu einloggen. Wer einen
> Passkey benutzt, meldet sich mit Passwort an und legt den Passkey danach neu an.

Wer bisher **nur** einen Passkey und kein Passwort hat: vorher ein Passwort
setzen lassen, sonst bleibt nur „Passwort vergessen“ per E-Mail.

### 1b. Backup auf dem Server

```bash
ssh unidentist@192.168.178.105
cd ~/storyden
bash backup.sh
cp .env .env.vor-domainwechsel
cp Caddyfile Caddyfile.vor-domainwechsel
[ -f ddns/config.json ] && cp ddns/config.json ddns/config.json.vor-domainwechsel
```

Damit ist jederzeit ein Zurückrollen möglich (siehe Schritt 10).

### 1c. Öffentliche IP notieren

Auf dem Server:

```bash
curl -4 ifconfig.me
```

Diese IP brauchst du gleich beim Domain-Provider. Liefert der Befehl eine
Adresse zwischen `100.64.x.x` und `100.127.x.x`, hast du DS-Lite — dann siehe
[Anleitung.md](Anleitung.md), Schritt 2a.

---

## Schritt 2: DNS beim Domain-Provider der neuen Domain

Im Kundenbereich des Anbieters, bei dem du `unidentists.de` gekauft hast
(name.com, IONOS, Strato, INWX, …), die DNS-Verwaltung der Domain öffnen.

### 2a. Vorhandene Standard-Einträge aufräumen

Viele Anbieter legen bei einer neuen Domain automatisch eine
„Parkseite“ oder Weiterleitung an. Diese entfernen:

- vorhandene **A**- und **AAAA**-Records für `@` (leer) und `www` löschen
- eine eingerichtete „Domain-Weiterleitung“ / „Parking“ deaktivieren

**AAAA-Records nicht neu anlegen**, solange der Server nicht sauber über IPv6
erreichbar ist (FRITZ!Box-IPv6-Freigabe + `ufw` für IPv6). Sonst versucht
Let's Encrypt es über IPv6, scheitert, und Caddy bekommt kein Zertifikat.

### 2b. Neue Einträge anlegen

| Typ | Host / Name        | Wert / Ziel                  | TTL |
| --- | ------------------ | ---------------------------- | --- |
| A   | `@` bzw. leer      | öffentliche IP aus Schritt 1c | 300 |
| A   | `www`              | öffentliche IP aus Schritt 1c | 300 |

> **Alternative ohne DynDNS-Container (MyFRITZ!):** Statt einer festen IP
> kann `www` als **CNAME** auf `xxxxxxxxxxxx.myfritz.net` zeigen, und der Apex
> (`@`) als **ANAME/ALIAS** auf denselben Hostnamen — falls der Anbieter
> ANAME/ALIAS kann. Details in
> [raspberrypi/README.md, Anhang B](../raspberrypi/README.md).

### 2c. Prüfen

DNS braucht ein paar Minuten (bei `.de` manchmal bis zu einer Stunde, bis die
Delegation bei der DENIC aktiv ist). Vom PC oder Server:

```bash
dig +short unidentists.de @1.1.1.1
dig +short www.unidentists.de @1.1.1.1
```

Beide müssen die IP aus Schritt 1c liefern. Unter Windows geht auch:

```powershell
nslookup unidentists.de 1.1.1.1
```

**Erst weitermachen, wenn das stimmt** — sonst schlägt in Schritt 6 das
Zertifikat fehl, und nach mehreren Fehlversuchen sperrt Let's Encrypt
vorübergehend (Rate Limit).

### 2d. Alte Domain behalten

`unidentists.social` **nicht kündigen** und den A-Record dort **nicht
löschen**. Die alte Domain soll noch mindestens ein Jahr auf den Server
zeigen, damit alte Links, Lesezeichen und E-Mail-Links weitergeleitet
werden (Schritt 5c).

---

## Schritt 3: E-Mail-DNS (nur wenn Mails von `@unidentists.de` kommen sollen)

Soll der Absender von `no-reply@unidentists.social` auf
`no-reply@unidentists.de` wechseln, muss die neue Domain beim SMTP-Anbieter
(Brevo, Mailjet, Gmail-SMTP, …) **verifiziert** werden. Sonst landen
Registrierungs- und Passwort-Reset-Mails im Spam oder werden abgelehnt.

1. Beim SMTP-Anbieter `unidentists.de` als Absender-Domain hinzufügen.
2. Die dort angezeigten DNS-Einträge beim Domain-Provider anlegen,
   typischerweise:

   | Typ | Host                     | Wert (vom SMTP-Anbieter)                 |
   | --- | ------------------------ | ---------------------------------------- |
   | TXT | `@`                      | `v=spf1 include:<anbieter> ~all`         |
   | TXT / CNAME | `<selector>._domainkey` | DKIM-Schlüssel                    |
   | TXT | `_dmarc`                 | `v=DMARC1; p=none; rua=mailto:…`         |

3. Beim SMTP-Anbieter auf „Verifizieren“ klicken und warten, bis alles grün ist.

**Willst du das jetzt noch nicht machen**, einfach in Schritt 5a
`SMTP_FROM_ADDRESS` auf der alten Adresse lassen — das funktioniert weiter,
solange die alte Domain verifiziert bleibt.

---

## Schritt 4: FRITZ!Box

### 4a. Portfreigaben prüfen (nichts ändern)

Portfreigaben hängen an der IP, nicht an der Domain. Nur kontrollieren:

`Internet → Freigaben → Portfreigaben → Ubuntu-Server`

| Extern | → | Server (`192.168.178.105`) | Protokoll |
| ------ | - | -------------------------- | --------- |
| 443    | → | 443                        | TCP       |
| 80     | → | 80                         | TCP       |

Optional zusätzlich `443 → 443 UDP` (HTTP/3), ist aber nicht nötig.

### 4b. Feste IP für den Server prüfen

`Heimnetz → Netzwerk → Netzwerkverbindungen → Ubuntu-Server → Bearbeiten`
→ Haken bei **„Diesem Netzwerkgerät immer die gleiche IPv4-Adresse zuweisen“**.

### 4c. DNS-Rebind-Schutz: Ausnahme eintragen

Die FRITZ!Box blockiert DNS-Antworten, die auf eine **private** IP
(`192.168.x.x`) zeigen. Genau das liefert aber der AdGuard-Rewrite aus
Schritt 4e für Geräte im LAN. Falls Geräte die FRITZ!Box als DNS benutzen und
die FRITZ!Box an AdGuard weiterleitet, wird die Antwort sonst verworfen.

`Heimnetz → Netzwerk → Netzwerkeinstellungen → Weitere Einstellungen →
IPv4-Einstellungen` bzw. **„DNS-Rebind-Schutz“** →
im Feld „Hostnamen-Ausnahmen“ ergänzen (eine Zeile pro Eintrag):

```
unidentists.de
www.unidentists.de
```

Steht dort schon `unidentists.social`, kann das bleiben.

### 4d. DynDNS in der FRITZ!Box (nur falls du das statt des Containers nutzt)

`Internet → Freigaben → DynDNS`

Benutzt du bisher den `ddns`-Container auf dem Server, hier **nichts**
eintragen — zwei Updater für dieselbe Domain stören sich.

Willst du stattdessen die FRITZ!Box aktualisieren lassen und unterstützt dein
Anbieter eine Update-URL (z. B. IONOS, Strato, INWX, DuckDNS):

- DynDNS benutzen: ☑
- Anbieter: „Benutzerdefiniert“ (oder den Anbieter aus der Liste)
- Update-URL: laut Anleitung deines Domain-Anbieters
- Domainname: `unidentists.de`
- Benutzername / Kennwort: vom Anbieter

### 4e. AdGuard Home (läuft auf dem Pi): DNS-Rewrite

Damit das Forum auch **im eigenen WLAN** unter der neuen Domain erreichbar
ist (die FRITZ!Box kann kein zuverlässiges NAT-Hairpinning):

`http://192.168.178.30:180` (AdGuard-Oberfläche) →
`Filter → DNS-Umschreibungen → DNS-Umschreibung hinzufügen`

| Domain                | Antwort            |
| --------------------- | ------------------ |
| `unidentists.de`      | `192.168.178.105`  |
| `www.unidentists.de`  | `192.168.178.105`  |

Die Einträge für `unidentists.social` **stehen lassen** (bzw. auf
`192.168.178.105` korrigieren, falls sie noch auf den Pi `192.168.178.30`
zeigen) — die alte Domain soll im LAN ja ebenfalls weiterleiten.

Prüfen von einem Gerät im WLAN:

```powershell
nslookup unidentists.de
```

Muss `192.168.178.105` liefern.

---

## Schritt 5: Ubuntu-Server

```bash
ssh unidentist@192.168.178.105
cd ~/storyden
```

### 5a. `.env` anpassen

```bash
nano .env
```

Diese Zeilen ändern:

```
SITE_DOMAIN=unidentists.de
PUBLIC_WEB_ADDRESS=https://unidentists.de
PUBLIC_API_ADDRESS=https://unidentists.de
```

Falls Schritt 3 erledigt ist, auch:

```
SMTP_FROM_ADDRESS=no-reply@unidentists.de
```

Wichtig:

- `https://` davor, **kein** `/` am Ende, **kein** `www.`
- `JWT_SECRET`, `POSTGRES_*` und `DATABASE_URL` **nicht** anfassen.

Kontrolle:

```bash
grep -E '^(SITE_DOMAIN|PUBLIC_WEB_ADDRESS|PUBLIC_API_ADDRESS|SMTP_FROM_ADDRESS)=' .env
```

### 5b. DynDNS-Container (nur falls aktiv)

Ob er läuft:

```bash
docker compose ps ddns
```

Falls ja:

```bash
nano ddns/config.json
```

Den bestehenden Eintrag für die alte Domain **behalten** und die neue
zusätzlich eintragen — ein Block pro Hostname. Liegen beide Domains bei
**name.com**:

```json
{
  "settings": [
    {
      "provider": "name.com",
      "domain": "unidentists.de",
      "username": "DEIN-NAME.COM-BENUTZERNAME",
      "token": "DEIN-NAME.COM-API-TOKEN",
      "ttl": 300,
      "ip_version": "ipv4"
    },
    {
      "provider": "name.com",
      "domain": "www.unidentists.de",
      "username": "DEIN-NAME.COM-BENUTZERNAME",
      "token": "DEIN-NAME.COM-API-TOKEN",
      "ttl": 300,
      "ip_version": "ipv4"
    },
    {
      "provider": "name.com",
      "domain": "unidentists.social",
      "username": "DEIN-NAME.COM-BENUTZERNAME",
      "token": "DEIN-NAME.COM-API-TOKEN",
      "ttl": 300,
      "ip_version": "ipv4"
    }
  ]
}
```

Liegt `unidentists.de` bei einem **anderen Anbieter**, in den ersten beiden
Blöcken `provider` und die Zugangsfelder passend setzen. Die unterstützten Anbieter
und ihre Felder stehen in der Doku von
[ddns-updater](https://github.com/qdm12/ddns-updater/tree/master/docs).

### 5c. Caddyfile: alte Domain weiterleiten

```bash
nano Caddyfile
```

Ganz unten **anfügen** (der Rest bleibt unverändert, `{$SITE_DOMAIN}` zeigt
nach Schritt 5a automatisch auf die neue Domain):

```
# Alte Domain: alles dauerhaft auf die neue Domain umleiten.
unidentists.social, www.unidentists.social {
	redir https://unidentists.de{uri} permanent
}
```

Caddy holt dafür ein eigenes Zertifikat für die alte Domain, damit auch
`https://unidentists.social/...` sauber weitergeleitet wird. Voraussetzung:
die alte Domain zeigt weiterhin auf deinen Anschluss (Schritt 2d).

Syntax prüfen:

```bash
docker compose run --rm --no-deps -e SITE_DOMAIN=unidentists.de -e ACME_EMAIL=test@example.com caddy caddy validate --config /etc/caddy/Caddyfile
```

Muss mit `Valid configuration` enden.

### 5d. Firewall (nur prüfen)

```bash
sudo ufw status
```

`80/tcp` und `443/tcp` müssen `ALLOW` sein. Domainunabhängig, keine Änderung
nötig.

---

## Schritt 6: Neu starten

Erst wenn Schritt 2c die richtige IP zeigt:

```bash
cd ~/storyden
docker compose up -d --force-recreate storyden caddy
```

Mit aktivem DynDNS zusätzlich:

```bash
docker compose --profile ddns up -d --force-recreate ddns
```

`--force-recreate` ist nötig, weil Änderungen am `Caddyfile` sonst nicht
zuverlässig übernommen werden.

Mitlesen, bis die Zertifikate da sind (mit `Strg+C` beenden):

```bash
docker compose logs -f caddy
```

Erwartet wird je Domain eine Zeile `certificate obtained successfully`, also
für `unidentists.de`, `www.unidentists.de`, `unidentists.social` und
`www.unidentists.social`.

Die App selbst:

```bash
docker compose logs --tail=50 storyden
docker compose ps
```

Alle Container `healthy` bzw. `running`.

---

## Schritt 7: Windows-PC anpassen

**Wichtig, sonst wird der Wechsel beim nächsten Update rückgängig gemacht:**
`copy-to-server.ps1` (ohne `-ImageOnly`) kopiert die lokale `.env` vom PC auf
den Server und würde die alte Domain zurückschreiben.

Im Projektordner auf dem PC:

```powershell
cd <projektverzeichnis>\docker\compose\ubuntuserver
notepad .env
```

Dieselben Zeilen wie in Schritt 5a ändern. Am einfachsten die fertige Datei
vom Server zurückholen:

```powershell
scp unidentist@192.168.178.105:~/storyden/.env .env
scp unidentist@192.168.178.105:~/storyden/Caddyfile Caddyfile
```

Achtung: Die zweite Zeile überschreibt das `Caddyfile` im Git-Repo. Willst du
die Weiterleitung dauerhaft im Repo haben, diese Änderung committen; sonst
`git checkout Caddyfile` und das Caddyfile künftig nicht mehr mit
`copy-to-server.ps1` überschreiben lassen (immer `-ImageOnly` benutzen).

---

## Schritt 8: In Storyden selbst

### 8a. Admin-Einstellungen

Einloggen auf `https://unidentists.de` → **Admin → Settings**:

- **Client IP / Trusted Proxies:** bleibt wie es ist, domainunabhängig.
- **Links im Seitentext** (Startseite, Beschreibung, Regeln, Menü-Links):
  Einträge mit `unidentists.social` auf `unidentists.de` ändern. Durch die
  Weiterleitung funktionieren sie zwar weiter, sauberer ist es aber direkt.

### 8b. Login mit Google / GitHub / Discord (nur falls eingerichtet)

Für jeden aktiven OAuth-Anbieter in dessen Entwickler-Konsole die erlaubte
Redirect-/Callback-URL **ergänzen** (alte vorerst stehen lassen):

```
https://unidentists.de/auth/google/callback
https://unidentists.de/auth/github/callback
https://unidentists.de/auth/discord/callback
```

Bei Google zusätzlich unter „Autorisierte JavaScript-Quellen“
`https://unidentists.de` eintragen.

### 8c. Passkey neu anlegen

Mit Passwort einloggen → Profil/Konto → Passkey hinzufügen. Alten Passkey für
`unidentists.social` im Passwortmanager/Handy löschen.

---

## Schritt 9: Prüfen

Am besten einmal **im WLAN** und einmal **über Mobilfunk** (WLAN am Handy aus)
testen:

- [ ] `https://unidentists.de` lädt, Schloss im Browser, keine Warnung
- [ ] `http://unidentists.de` → leitet auf `https://unidentists.de` um
- [ ] `https://www.unidentists.de` → leitet auf `https://unidentists.de` um
- [ ] `https://unidentists.social/irgendein/pfad` → landet auf
      `https://unidentists.de/irgendein/pfad`
- [ ] Login mit Passwort funktioniert, Seite bleibt nach Neuladen eingeloggt
- [ ] Bilder, Avatare und alte Uploads werden angezeigt
- [ ] Neues Bild/PDF hochladen funktioniert
- [ ] „Passwort vergessen“ schicken: Mail kommt an, **nicht im Spam**, und der
      Link darin beginnt mit `https://unidentists.de/`
- [ ] Passkey neu registrieren und damit einloggen
- [ ] OAuth-Login (falls genutzt) funktioniert
- [ ] Neustart-Test: `sudo reboot`, danach kommt alles von selbst wieder hoch

Schnelltest per Kommandozeile:

```bash
curl -sI https://unidentists.de | head -1
curl -sI https://www.unidentists.de | grep -i location
curl -sI https://unidentists.social/test | grep -i location
```

Erwartet: `HTTP/2 200`, dann zweimal ein `location: https://unidentists.de/…`.

---

## Schritt 10: Fehlerbehebung

**Caddy: „no such host“ / „NXDOMAIN“ / Zertifikat schlägt fehl**
DNS noch nicht aktiv. `dig +short unidentists.de @1.1.1.1` prüfen, warten,
dann `docker compose restart caddy`.

**Caddy: „Timeout during connect“ / „connection refused“**
Portfreigabe 80/443 in der FRITZ!Box prüfen (Schritt 4a) und ob die
öffentliche IP im A-Record noch stimmt (`curl -4 ifconfig.me` vergleichen).

**Caddy: „too many failed authorizations“ / „rateLimited“**
Let's Encrypt hat nach zu vielen Fehlversuchen gesperrt. Eine Stunde warten,
Ursache (meist DNS) beheben, dann neu starten.

**Im WLAN geht's nicht, über Mobilfunk schon**
AdGuard-Rewrite (Schritt 4e) und DNS-Rebind-Ausnahme (Schritt 4c) prüfen.
Am PC den DNS-Cache leeren: `ipconfig /flushdns`.

**Login klappt, aber nach dem Neuladen ist man wieder ausgeloggt**
`PUBLIC_WEB_ADDRESS` passt nicht exakt zur Adresse im Browser (Tippfehler,
`www.`, `/` am Ende, `http` statt `https`). Schritt 5a prüfen, dann
`docker compose up -d --force-recreate storyden`.

**E-Mails kommen nicht an oder landen im Spam**
Schritt 3 — Domain beim SMTP-Anbieter noch nicht verifiziert. Übergangsweise
`SMTP_FROM_ADDRESS` auf die alte Adresse zurückstellen.

**Zurückrollen auf die alte Domain:**

```bash
cd ~/storyden
cp .env.vor-domainwechsel .env
cp Caddyfile.vor-domainwechsel Caddyfile
[ -f ddns/config.json.vor-domainwechsel ] && cp ddns/config.json.vor-domainwechsel ddns/config.json
docker compose up -d --force-recreate storyden caddy
```

---

## Schritt 11: Später aufräumen

- **Nach ein paar Wochen:** in den OAuth-Konsolen die alten
  `unidentists.social`-Redirect-URLs entfernen.
- **Frühestens nach einem Jahr:** Weiterleitung der alten Domain aus dem
  Caddyfile nehmen und `unidentists.social` auslaufen lassen. Bis dahin die
  alte Domain unbedingt verlängern — sonst kann sie jemand anderes kaufen und
  deine alten Links übernehmen.
