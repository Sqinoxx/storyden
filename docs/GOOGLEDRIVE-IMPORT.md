# GoogleDrive-Downloads in die Library importieren

Das Archiv aus der Zeit vor der Website (`GoogleDrive-Downloads/`, rund 4.000
Dateien in 45 GB) wird als Library-Baum in die Seite überführt: jede Datei
bekommt eine eigene Seite mit Fach, Semester und Typ, der Volltext aller PDFs
wird durchsuchbar, und die Ordnerstruktur wird dabei auf das vorhandene
Tag-Vokabular abgebildet statt eins zu eins übernommen.

Gesteuert wird das über zwei Dateien im Repo-Wurzelverzeichnis:

| Datei | Zweck |
|---|---|
| `import-manifest.yaml` | Regeln: welcher Quellpfad wird zu welchem Knoten, mit welchen Tags und Properties |
| `import-vocab.yaml` | Vokabular: Abschnitte, Fächer, Typen samt Schreibweisen, plus die Tag-Umbenennungen |

## Ablauf

Die sechs Phasen bauen aufeinander auf und sind alle wiederaufnehmbar. `scan`
und `plan` brauchen weder Datenbank noch Konfiguration.

### 1. Inventar und Dubletten

```bash
go run ./cmd/import --phase scan --root "F:/Zahnmedizin/Forum/storyden/GoogleDrive-Downloads/GoogleDrive-Downloads"
```

Läuft über den Baum, bildet je Datei einen SHA-256 und schreibt
`import-work/inventory.jsonl` sowie `import-work/duplicates.report.txt`.

Bei Inhaltsgleichheit gewinnt der Pfad, der **nicht** unter einem in
`defaults.demote` genannten Präfix liegt — der Zweig
`Zusammenfassungen, Vorlesungen, Anki-Decks/Vorlesungen & Unterlagen` ist
weitgehend eine Zweitkopie und tritt deshalb zurück.

Für einen schnellen Blick auf die Manifest-Abdeckung ohne Hashing (Sekunden
statt Minuten) gibt es `--no-hash`. Dubletten werden dabei nicht erkannt.

### 2. Zielstruktur prüfen

```bash
go run ./cmd/import --phase plan --dry-run
```

Gibt den entstehenden Baum mit Dateizahl je Knoten aus und schreibt nichts. Die
zwei Zahlen am Ende sind die Qualitätskennzahlen:

- **Treffer im Auffangnetz** — Dateien, die nur auf die letzte Regel `**`
  passten. Sollte null sein.
- **Ordnernamen ohne Vokabulareintrag** — werden zu Ordnern, erzeugen aber
  bewusst *kein* Tag, damit ein einzelnes Streuverzeichnis das geschlossene
  Vokabular nicht verwässert.

Beides sinkt, indem Regeln in `import-manifest.yaml` oder Aliase in
`import-vocab.yaml` ergänzt werden. Diesen Schritt so lange wiederholen, bis das
Ergebnis passt — hier ist der Import noch kostenlos korrigierbar.

### 3. Vokabular angleichen

```bash
go run ./cmd/import --phase vocab --dry-run
go run ./cmd/import --phase vocab
```

Benennt die drei Tippfehler um (`parodondologie` → `parodontologie`,
`pharmalogie` → `pharmakologie`, `gedächnisprotokoll` → `gedächtnisprotokoll`)
und legt fehlende Fächer an. Die Umbenennung nimmt bestehende Verknüpfungen mit,
vorhandene Beiträge verlieren ihre Tags also nicht.

Muss **vor** `enrich` laufen, sonst klassifiziert das Modell gegen ein
unvollständiges Vokabular.

### 4. Import

```bash
OCR_ENABLED=false go run ./cmd/import --phase apply --owner sqinox \
  --root "F:/Zahnmedizin/Forum/storyden/GoogleDrive-Downloads/GoogleDrive-Downloads"
```

Legt Knoten an, lädt die Dateien als Assets hoch und hängt sie an. Der Upload
geht direkt über den Uploader statt über die HTTP-API, deshalb greift weder
`MAX_UPLOAD_SIZE_MB` noch das `max_size` im Caddyfile — die 300-MB-Scans passen
durch.

`OCR_ENABLED=false` hält die Texterkennung während des Imports aus dem Weg; die
Assets bleiben auf `pending` und werden in Phase 5 abgearbeitet.

Der Fortschritt liegt in `import-work/import-state.jsonl`. Ein Abbruch ist
unkritisch: derselbe Befehl noch einmal überspringt alles bereits Erledigte.

Für einen Probelauf auf einem Teilbaum:

```bash
go run ./cmd/import --phase apply --owner sqinox --limit-root "Altklausuren" \
  --root "F:/Zahnmedizin/Forum/storyden/GoogleDrive-Downloads/GoogleDrive-Downloads"
```

### 5. Texterkennung

```bash
OCR_ENABLED=true OCR_CONCURRENCY=8 OCR_MAX_FILE_SIZE_MB=512 OCR_PDF_MAX_PAGES=200 \
  go run ./cmd/import --phase ocr
```

Bei PDFs wird zuerst die eingebettete Textebene gelesen — sofort, verlustfrei
und für digitale Vorlesungsfolien vollständig. Nur echte Scans gehen über
`pdftoppm` durch Tesseract.

`OCR_CONCURRENCY` ist neu und steht standardmäßig auf 1, damit sich am Verhalten
des Servers nichts ändert. Auf dem Arbeitsrechner darf der Wert der Kernzahl
entsprechen.

**Voraussetzung:** `tesseract` und `pdftoppm` müssen im Pfad liegen. Unter
Windows ist der zuverlässigere Weg, den Import in einem Container auf Basis von
`docker/api/Dockerfile` laufen zu lassen — dort sind beide samt deutscher
Sprachdaten bereits installiert.

Wiederaufnahme ergibt sich von selbst über die Spalte `ocr_status`.

### 6. Properties füllen

Schickt Dateiname, Ordner und die ersten 4.000 Zeichen des erkannten Texts an
ein Sprachmodell und füllt Fach, Semester, Jahr und Dozent. Tags werden
ausschließlich aus dem Vokabular vergeben; alles andere wird verworfen.

**Der vom Import gesetzte `Typ` wird nicht überschrieben.** Er stammt
deterministisch aus dem Ordnerpfad und ist damit verlässlicher als eine
Modellschätzung. Nur leere Felder werden befüllt — `--overwrite` hebt das auf.

Diese Phase ist die einzige, die Daten nach außen gibt. Ohne sie bleibt der
Import vollständig lokal, nur die Property-Spalten bleiben leer.

**Mit Gemini** (kostenloses Kontingent, kein Zahlungsmittel nötig). Key auf
<https://aistudio.google.com/apikey> erzeugen:

```powershell
$env:LANGUAGE_MODEL_PROVIDER = "google"
$env:GOOGLE_AI_API_KEY = "..."
go run ./cmd/import --phase enrich --owner sqinox --limit 5 --dry-run
```

**Mit OpenAI** stattdessen `LANGUAGE_MODEL_PROVIDER=openai` und `OPENAI_API_KEY`.

| Variable | Default | Zweck |
|---|---|---|
| `GOOGLE_AI_MODEL` | `gemini-2.5-flash-lite` | Wird am 16.10.2026 abgeschaltet, Nachfolger `gemini-3.1-flash-lite` |
| `GOOGLE_AI_MAX_RPM` | `15` | Gratis-Kontingent. Mit aktivierter Abrechnung deutlich höher setzbar |
| `GOOGLE_AI_MAX_RETRIES` | `3` | Backoff-Versuche bei 429 und 5xx |

Das Gratis-Kontingent erlaubt 15 Anfragen/Minute und 1.000/Tag. Bei rund 2.400
Dokumenten verteilt sich der Lauf also über etwa drei Tage. Ist das Tageslimit
erreicht, **bricht der Lauf sauber ab** statt den Rest in denselben Fehler
laufen zu lassen, und meldet, wie weit er gekommen ist. Am nächsten Tag genügt
derselbe Befehl.

Bereits verarbeitete Dokumente stehen in `import-work/enrich-state.jsonl` und
werden übersprungen — auch die, bei denen das Modell nichts Verwertbares
geliefert hat. Ohne diese Datei würden Dokumente ohne erkennbaren Dozenten bei
jedem Lauf erneut Kontingent kosten.

Am Ende wird der Tokenverbrauch ausgegeben. Zur Einordnung: der gesamte Bestand
liegt bei Flash-Lite grob bei 3,5 Mio Input- und 0,3 Mio Output-Tokens, also
unter einem halben Dollar, falls doch mit Abrechnung gefahren wird.

## Danach

- Suchindex neu aufbauen, damit die erkannten Texte in der Volltextsuche landen.
- Assets per `rsync -avP` auf den Server übertragen, nicht als ein Tarball —
  bei 30 GB ist Wiederaufnehmbarkeit den Unterschied wert.
- Auf dem Zielsystem `OCR_BACKFILL_ENABLED=false` setzen, sonst schickt der
  Backfill nach jedem Neustart den gesamten Bestand erneut durch die Erkennung.
  Im Ubuntu-Compose steht der Standard derzeit auf `true`.

## Grenzen

- Die Erkennung deckt PNG, JPEG und PDF ab. Office-Dateien und Anki-Decks sind
  über Name, Tags und Properties auffindbar, aber nicht im Inhalt durchsuchbar.
- Library-Knoten haben keine Kategorien. Die Semesterachse existiert im Baum und
  in den Tags, aber ein Knoten kann nicht wie ein Beitrag in „5. Semester" liegen.
- Uploads über die Weboberfläche bleiben bei `MAX_UPLOAD_SIZE_MB` gedeckelt; nur
  der Importer umgeht das.
