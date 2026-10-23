# Domainwechsel: `unidentists.social` → `unidentists.de`

Schritt-für-Schritt-Anleitung für den Umzug des Forums auf dem Ubuntu-Server
(`192.168.178.105`) auf die neue Domain **`unidentists.de`**.

Die neue Domain ist bei **Hetzner** registriert, die alte (`unidentists.social`)
liegt weiterhin bei **name.com**.

Die Kurzfassung steht schon in [WARTUNG-Ubuntu.md](WARTUNG-Ubuntu.md),
Abschnitt „Domain ändern“. Diese Datei geht jeden Schritt einzeln durch,
inklusive Hetzner, FRITZ!Box, AdGuard und Weiterleitung der alten Domain.

---

## Fortschritt

- [x] 1a Nutzer informiert
- [x] 1b Backup auf dem Server (DB und Uploads vom 23.10. 22:10, `.env` und `Caddyfile` als `*.vor-domainwechsel`; `ddns/config.json` gab es nicht)
- [x] 1c Öffentliche IP notiert (`<ÖFFENTLICHE-IP>`)
- [x] 2 DNS bei Hetzner (A-Records `@` und `www`, AAAA entfernt)
- [x] 2f Alte Domain bleibt bei name.com
- [x] 3 Brevo: Domain authentifiziert, DNS-Einträge gesetzt
- [x] 4 FRITZ!Box und AdGuard (4a bis 4e)
- [x] 5a `.env` auf dem Server angepasst (`SITE_DOMAIN`, `PUBLIC_*`, `SMTP_FROM_ADDRESS=mail@unidentists.de`)
- [x] 5b DynDNS: nicht nötig, die IP ist fest
- [x] 5c `Caddyfile`: Weiterleitung der alten Domain angehängt, `Valid configuration`
- [x] 5d Firewall geprüft
- [x] 6 Container neu gestartet, Zertifikate da (`unidentists.de`, `www.unidentists.de`, `unidentists.social`); `www.unidentists.social` aus dem `Caddyfile` entfernt, weil es dafür nie einen DNS-Eintrag gab
- [x] 7 Windows-PC: lokale `.env` und `Caddyfile` angepasst (`Caddyfile` im Repo committet)
- [x] 8a Admin-Einstellungen: Links mit `unidentists.social` auf `unidentists.de` ändern (du, Login nötig)
- [x] 8b OAuth: nicht nötig, es ist kein OAuth-Anbieter aktiv (Login nur per Passwort, E-Mail-Code und Passkey)
- [x] 8c Passkey neu anlegen (du)
- [x] 9 Prüfen. Per `curl` bestätigt: `https://unidentists.de` 200 mit HSTS, `www.unidentists.de` und `unidentists.social/test` leiten um, die API meldet `api_address=https://unidentists.de`, Brevo-DNS (DKIM-CNAME, DMARC) vorhanden. Offen (du, Login nötig):
  - [x] Login mit Passwort, nach Neuladen noch eingeloggt
  - [x] Bilder, Avatare und alte Uploads werden angezeigt
  - [x] Neues Bild/PDF hochladen
  - [x] Passwort-vergessen-Mail kommt an, nicht im Spam, Link beginnt mit `https://unidentists.de/`
  - [x] Passkey neu registrieren und damit einloggen
  - [x] Test im WLAN und per Mobilfunk
  - [x] `sudo ufw status` (5d) und Neustart-Test mit `sudo reboot`

---

## Überblick

| Wo                        | Was                                                         | Pflicht?        |
| ------------------------- | ----------------------------------------------------------- | --------------- |
| Hetzner (Domainverwaltung)| A-Records für `unidentists.de` und `www.unidentists.de`      | ja              |
| Hetzner (Domainverwaltung)| DynDNS: nur möglich nach Nameserver-Wechsel (siehe 2d)       | bei wechselnder IP |
| Hetzner (Domainverwaltung)| SPF/DKIM/DMARC, falls Mails von `@unidentists.de` kommen     | für E-Mail ja   |
| FRITZ!Box                 | Portfreigaben prüfen, DNS-Rebind-Ausnahme                    | prüfen          |
| AdGuard Home (Pi)         | DNS-Rewrite für die neue Domain                              | ja (LAN-Zugriff)|
| Ubuntu-Server `.env`      | `SITE_DOMAIN`, `PUBLIC_*`, `SMTP_FROM_ADDRESS`               | ja              |
| Ubuntu-Server `ddns/`     | Neue Domain im DynDNS-Updater (Hetzner)                      | bei wechselnder IP |
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

Diese IP brauchst du gleich bei Hetzner (im Folgenden `<ÖFFENTLICHE-IP>`). Liefert der Befehl eine
Adresse zwischen `100.64.x.x` und `100.127.x.x`, hast du DS-Lite — dann siehe
[Anleitung.md](Anleitung.md), Schritt 2a.

---

## Schritt 2: DNS bei Hetzner

Die Domain ist bei Hetzner registriert und wird in der Hetzner-
**Domainverwaltung** (Bereich „Hetzner Domain“, Tabs *DNS-Records*,
*Nameserver*, *E-Mail Sicherheit*) gepflegt. Hetzner betreibt zwei getrennte
DNS-Systeme:

| System                            | Nameserver                                                                 |
| --------------------------------- | -------------------------------------------------------------------------- |
| Domainverwaltung (konsoleH/Robot) | `ns1.your-server.de`, `ns.second-ns.com`, `ns3.second-ns.de`               |
| Hetzner Console (DNS)             | `hydrogen.ns.hetzner.com`, `oxygen.ns.hetzner.com`, `helium.ns.hetzner.de` |

`unidentists.de` läuft im **ersten** System. Die drei Nameserver dort sind
korrekt, **an den Nameservern wird nichts geändert**. Die Einträge werden direkt
im Tab **DNS-Records** der Domainverwaltung bearbeitet. Die Hetzner Console und
ihre API-Tokens sind für diese Domain nicht zuständig.

### 2a. Nameserver kontrollieren (nur ansehen)

In der Domainverwaltung Tab **Nameserver** öffnen. Dort müssen stehen:

```
ns1.your-server.de.
ns.second-ns.com.
ns3.second-ns.de.
```

Zur Kontrolle vom PC oder Server:

```bash
dig +short NS unidentists.de
```

Muss dieselben drei Nameserver liefern. Nichts ändern.

### 2b. Vorhandene Standard-Einträge aufräumen

Tab **DNS-Records** öffnen. Dort stehen ab Werk mehrere Einträge (Parkseite,
Mail usw.). `NS`- und `SOA`-Einträge sowie vorhandene Mail-Einträge (`MX`,
SPF-`TXT`) **nicht anfassen**. Alle `A`- und `AAAA`-Einträge für `@` und `www`
(z. B. die Hetzner-Parkseite `88.198.219.246` bzw.
`2a01:4f8:d0a:27bd::2`) **löschen**.

**AAAA-Records nicht neu anlegen**, solange der Server nicht sauber über IPv6
erreichbar ist (FRITZ!Box-IPv6-Freigabe + `ufw` für IPv6). Sonst versucht
Let's Encrypt es über IPv6, scheitert, und Caddy bekommt kein Zertifikat.

### 2c. Neue Einträge anlegen

Im Tab **DNS-Records** zwei Einträge anlegen:

| Typ | Name  | Wert             | TTL          |
| --- | ----- | ---------------- | ------------ |
| A   | `@`   | `<ÖFFENTLICHE-IP>` | 300 (5 Min.) |
| A   | `www` | `<ÖFFENTLICHE-IP>` | 300 (5 Min.) |

Das ist dieselbe IP, auf die `unidentists.social` aktuell zeigt. Falls die TTL
im Formular nicht einstellbar ist, den Standardwert lassen.

### 2d. Wechselnde IP: DynDNS-Optionen

Ändert sich die öffentliche IP deines Anschlusses (normaler DSL/Kabel-Anschluss
ohne feste IP), muss der A-Record nachgeführt werden. Die Domainverwaltung
(`your-server.de`-Nameserver) bietet **keine API und keine DynDNS-Adresse**.
Der `ddns`-Container und die FRITZ!Box können `unidentists.de` dort deshalb
nicht aktualisieren. Zwei Wege:

- **Option A: Nameserver auf die Hetzner Console umstellen.** In der Hetzner
  Console (console.hetzner.com) in einem Projekt unter **DNS** die Zone
  `unidentists.de` anlegen, dieselben A-Records (und Mail-Einträge) dort
  eintragen, unter **Sicherheit → API-Tokens** einen Token mit „Lesen &
  Schreiben“ erzeugen, und dann in der Domainverwaltung im Tab **Nameserver**
  auf `hydrogen.ns.hetzner.com`, `oxygen.ns.hetzner.com` und
  `helium.ns.hetzner.de` umstellen. Danach funktioniert Schritt 5b mit Provider
  `hetznercloud`. Die Umstellung kann bis zu 48 Stunden dauern, deshalb vor dem
  eigentlichen Umzug erledigen. Alle Einträge vorher in die neue Zone
  kopieren, sonst fallen Mails oder Subdomains aus.
- **Option B: Einträge bei Bedarf von Hand ändern.** Ändert sich die IP selten,
  den A-Record im Tab **DNS-Records** manuell anpassen, sobald das Forum
  nicht mehr erreichbar ist. Schritt 5b dann nur für `unidentists.social`
  (name.com) ausfüllen.

Hat der Anschluss eine feste IP, ist beides nicht nötig.

### 2e. Prüfen

Hetzner übernimmt Änderungen meist innerhalb weniger Minuten. Vom PC oder
Server:

```bash
dig +short unidentists.de @ns1.your-server.de
dig +short unidentists.de @1.1.1.1
dig +short www.unidentists.de @1.1.1.1
```

Alle müssen die IP aus Schritt 1c liefern. Unter Windows geht auch:

```powershell
nslookup unidentists.de 1.1.1.1
```

**Erst weitermachen, wenn das stimmt** — sonst schlägt in Schritt 6 das
Zertifikat fehl, und nach mehreren Fehlversuchen sperrt Let's Encrypt
vorübergehend (Rate Limit).

### 2f. Alte Domain behalten

`unidentists.social` bei name.com **nicht kündigen** und den A-Record dort
**nicht löschen**. Die alte Domain soll noch mindestens ein Jahr auf den
Server zeigen, damit alte Links, Lesezeichen und E-Mail-Links weitergeleitet
werden (Schritt 5c).

---

## Schritt 3: E-Mail-DNS (nur wenn Mails von `@unidentists.de` kommen sollen)

Der Absender wechselt von `mail@unidentists.social` auf
`mail@unidentists.de`. Dafür muss die neue Domain bei **Brevo** authentifiziert
werden. Sonst lehnt Brevo den Absender ab oder die Mails (Registrierung,
Passwort-Reset) landen im Spam.

Der SMTP-Zugang (`SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD`) bleibt
unverändert. Es ändert sich nur die Absenderadresse.

### 3a. Domain bei Brevo hinzufügen

1. [app.brevo.com](https://app.brevo.com) öffnen und einloggen.
2. Oben rechts auf den Kontonamen klicken → **Senders, Domains & Dedicated IPs**
   (je nach Version auch „Absender, Domains & dedizierte IPs“).
3. Reiter **Domains** → **Add a domain** (Domain hinzufügen).
4. `unidentists.de` eintragen (ohne `www.` und ohne `https://`) und bestätigen.
5. Brevo zeigt jetzt eine Liste von DNS-Einträgen. Diese Seite offen lassen.
   Falls Brevo „Authenticate the domain“ mit Auswahl „Automatisch“ (Domain
   Connect) anbietet: **nicht** benutzen, Hetzner unterstützt das nicht.
   „Manuell“ bzw. „Manually“ wählen.

### 3b. Einträge bei Hetzner anlegen

Hetzner Domainverwaltung → `unidentists.de` → Tab **DNS-Records** → Eintrag
hinzufügen. Die **genauen Werte kommen aus Brevo** (jeder Account hat eigene).
Typischerweise sind es diese drei Gruppen:

| Typ   | Name (Host)                | Wert (aus Brevo kopieren)                          |
| ----- | -------------------------- | -------------------------------------------------- |
| TXT   | `@`                        | `brevo-code:…` (Verifizierungscode)                |
| CNAME | `brevo1._domainkey`        | `b1.unidentists-de.dkim.brevo.com` (aus Brevo)     |
| CNAME | `brevo2._domainkey`        | `b2.unidentists-de.dkim.brevo.com` (aus Brevo)     |
| TXT   | `_dmarc`                   | `v=DMARC1; p=none; rua=mailto:rua@dmarc.brevo.com` |

Regeln dabei:

- Werte aus Brevo **exakt kopieren**. Zeigt Brevo andere Namen oder Typen
  (z. B. DKIM als TXT statt CNAME), gilt, was Brevo anzeigt.
- Beim Namen nur den Teil **vor** `.unidentists.de` eintragen, z. B.
  `brevo1._domainkey` statt `brevo1._domainkey.unidentists.de`. Wenn das
  Hetzner-Formular den Domainnamen automatisch anhängt, sonst entstehen
  doppelte Namen. Bei `@` das Feld so lassen, wie Hetzner es für die
  Hauptdomain vorsieht (`@` oder leer).
- TXT-Werte ohne zusätzliche Anführungszeichen einfügen, sofern Hetzner sie
  nicht selbst verlangt.
- Für `@` existiert bereits ein TXT-Eintrag
  `v=spf1 +a +mx ?all` (SPF). **Nicht löschen und keinen zweiten SPF-Eintrag
  anlegen.** Brevo braucht keinen SPF-Eintrag auf der Hauptdomain. Zwei
  `v=spf1`-Einträge würden SPF ungültig machen. Der neue `brevo-code`-Eintrag
  ist ein **zusätzlicher** TXT-Eintrag neben dem SPF-Eintrag.
- Den bestehenden `MX`-Eintrag (`www4.your-server.de`) nicht ändern.
- Gibt es schon einen `_dmarc`-Eintrag, diesen nicht doppelt anlegen. Bei
  Bedarf den vorhandenen anpassen.

### 3c. Bei Brevo verifizieren

1. Warten, bis die Einträge sichtbar sind (meist 5 bis 30 Minuten). Prüfen vom
   PC:

   ```powershell
   nslookup -type=TXT unidentists.de 1.1.1.1
   nslookup -type=CNAME brevo1._domainkey.unidentists.de 1.1.1.1
   nslookup -type=CNAME brevo2._domainkey.unidentists.de 1.1.1.1
   nslookup -type=TXT _dmarc.unidentists.de 1.1.1.1
   ```

2. Zurück in Brevo bei der Domain auf **Authenticate this domain** bzw.
   **Verify** klicken.
3. Warten, bis die Domain als **Authenticated** (grün) angezeigt wird.
   Schlägt es fehl, bei den Einträgen auf Tippfehler und doppelte Domainnamen
   im Host prüfen und nach einigen Minuten erneut versuchen.

### 3d. Absenderadresse bei Brevo anlegen

Im selben Bereich Reiter **Senders** → **Add a sender** (Absender hinzufügen):

- Name: `Unidentists`
- E-Mail: `mail@unidentists.de`

Da die Domain authentifiziert ist, ist keine Bestätigungs-Mail nötig. Wird
trotzdem eine verlangt, kommt sie an die Adresse `mail@unidentists.de`. Diese
Adresse muss dann Mails empfangen können (Weiterleitung bei Hetzner einrichten
oder die Absenderadresse vorübergehend auf eine erreichbare Adresse setzen).

**Willst du das jetzt noch nicht machen**, in Schritt 5a `SMTP_FROM_ADDRESS`
auf `mail@unidentists.social` lassen. Das funktioniert weiter, solange diese
Domain bei Brevo authentifiziert bleibt.

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

### 4d. DynDNS in der FRITZ!Box: nicht benutzen

Die Hetzner-Domainverwaltung bietet keine DynDNS-Update-Adresse, die die
FRITZ!Box aufrufen könnte. Die neue Domain wird daher, falls nötig, vom
`ddns`-Container auf dem Server aktuell gehalten (Schritt 2d, Option A, und
Schritt 5b). Unter
`Internet → Freigaben → DynDNS` für `unidentists.de` **nichts** eintragen —
zwei Updater für dieselbe Domain stören sich.

MyFRITZ! kann eingeschaltet bleiben, es betrifft die Domain nicht.

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
SMTP_FROM_ADDRESS=mail@unidentists.de
```

Wichtig:

- `https://` davor, **kein** `/` am Ende, **kein** `www.`
- `JWT_SECRET`, `POSTGRES_*` und `DATABASE_URL` **nicht** anfassen.

Kontrolle:

```bash
grep -E '^(SITE_DOMAIN|PUBLIC_WEB_ADDRESS|PUBLIC_API_ADDRESS|SMTP_FROM_ADDRESS)=' .env
```

### 5b. DynDNS-Container

Ist die IP deines Anschlusses fest, diesen Schritt überspringen. Die
Hetzner-Blöcke unten funktionieren nur mit Schritt 2d, Option A (Zone in der
Hetzner Console). Bei Option B nur den name.com-Block für `unidentists.social`
eintragen.

Sonst die Konfiguration öffnen (bzw. anlegen, falls es sie noch nicht gibt):

```bash
mkdir -p ddns
[ -f ddns/config.json ] || cp ddns-config.example.json ddns/config.json
nano ddns/config.json
```

Inhalt: die neue Domain über **Hetzner**, die alte weiterhin über
**name.com** — ein Block pro Hostname:

```json
{
  "settings": [
    {
      "provider": "hetznercloud",
      "domain": "unidentists.de",
      "token": "DEIN-HETZNER-API-TOKEN",
      "ttl": 300,
      "ip_version": "ipv4"
    },
    {
      "provider": "hetznercloud",
      "domain": "www.unidentists.de",
      "token": "DEIN-HETZNER-API-TOKEN",
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

- Den Hetzner-Token aus Schritt 2d einsetzen.
- **`hetznercloud`** verwenden, nicht `hetzner` — das ist die alte, abgeschaltete
  DNS-Console-Schnittstelle.
- Den name.com-Block mit den Werten aus deiner bisherigen `config.json`
  füllen (Sicherung aus Schritt 1b: `ddns/config.json.vor-domainwechsel`).
  Gab es bisher keinen DynDNS-Container, den name.com-Block weglassen.

Rechte setzen (der Container läuft als UID 1000) und die Datei schützen, weil
Tokens darin stehen:

```bash
sudo chown -R 1000:1000 ddns
sudo chmod 600 ddns/config.json
```

### 5c. Caddyfile: alte Domain weiterleiten

```bash
nano Caddyfile
```

Ganz unten **anfügen** (der Rest bleibt unverändert, `{$SITE_DOMAIN}` zeigt
nach Schritt 5a automatisch auf die neue Domain):

```
# Alte Domain: alles dauerhaft auf die neue Domain umleiten.
unidentists.social {
	redir https://unidentists.de{uri} permanent
}
```

Nur Hostnamen eintragen, für die bei name.com ein DNS-Eintrag existiert.
`www.unidentists.social` hat keinen, deshalb steht es nicht im Block. Sonst
scheitert das Zertifikat für diesen Namen und Caddy versucht es ständig neu.

Caddy holt dafür ein eigenes Zertifikat für die alte Domain, damit auch
`https://unidentists.social/...` sauber weitergeleitet wird. Voraussetzung:
die alte Domain zeigt weiterhin auf deinen Anschluss (Schritt 2f).

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

Erst wenn Schritt 2e die richtige IP zeigt:

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
für `unidentists.de`, `www.unidentists.de` und `unidentists.social`.

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

### 8b. Login mit Google / GitHub / Discord (nur falls eingerichtet; bei Unidentists nicht der Fall, überspringen)

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
- [ ] `docker compose logs ddns` (falls aktiv) meldet für alle drei Hostnamen
      keinen Fehler, in der Hetzner Console steht die aktuelle IP
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

**DynDNS: „unauthorized“ / „zone not found“**
Token falsch, nur Lesezugriff oder aus einem anderen Projekt als die Zone.
Neuen Token mit „Lesen & Schreiben“ im Projekt der Zone erzeugen und prüfen,
dass `provider` auf `hetznercloud` steht. Liegt die Domain noch bei den
`your-server.de`-Nameservern, kann der Updater sie nicht ändern (Schritt 2d).

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
