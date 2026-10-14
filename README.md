# SMRT - Media Renamer & Organizer (v0.3.0)

Ein schnelles Go-Tool mit einer modernen, responsiven Bubble Tea TUI, das Filme und Serien automatisch erkennt, bereinigt und strukturiert umbenennt.

Optimiert für **Desktop & Mobile** (z.B. Termius, Termux / SSH via Smartphone) mit voller **Maus- und Touch-Unterstützung**.

---

## Neu in Version 0.3.0

- 🗂 **Getrennte Pfade für Filme & Serien**: Direkt im Hauptmenü oder via CLI (`-m` / `-s`) separate Ordner für Filme und Serien angeben und in einem Durchgang verarbeiten.
- 🔢 **Pfad-Presets (1–99)**: Pfade dauerhaft mit Nummern (1 bis 99) verknüpfen. Im Hauptfenster oder per CLI einfach z. B. `1` oder `1 2` eingeben. Inklusive interaktivem Preset-Manager (`Strg + P` oder Touch-Button).
- 📺 **Intelligente Serien-Jahreszahlen**: Erscheinungsjahre im Serienordner (`Monster - The Lizzie Borden Story (2026)/`) bleiben erhalten, werden aber bei den Einzelepisoden weggelassen (`Monster - The Lizzie Borden Story - S01E01.mkv`). Titel, die mit einer Jahreszahl beginnen (z. B. `1999 - The best year ever`), bleiben stets intakt.
- 📱 **Mobile & Termius TUI Rework**:
  - Touch-Karten im vertikalen Stack mit Vollbreiten-Hitboxen.
  - Signalanroter Beenden-Button `[ ✖ Beenden ]` im Ergebnis-Screen.
  - Automatisches Schließen der Bildschirmtastatur auf Smartphones beim Wechsel zu Vorschau/Ergebnis (`\x1b[?25l`).
  - Schutz vor Schriftart-Drift bei Terminal-Buttons und responsiver mehrzeiliger Footer für schmale Displays (< 75 Spalten).

---

## Features

1. **Filme (Eigener Ordner pro Film)**:
   - **Einzelner Film oder mehrere Filme in einem Ordner**: Für jeden Film wird automatisch ein eigener Ordner erstellt, der exakt gleich heißt wie der Film:
     `The Film Title - Electric Buggalo (2009)/`
   - In diesem Ordner liegt dann die bereinigte Filmdatei:
     `The Film Title - Electric Buggalo (2009)/The Film Title - Electric Buggalo (2009).mkv`
   - Entfernt Codecs (`x264`, `hevc`), Scene-Tags (`1080p`, `web-dl`) und Release-Junk.

2. **Untertitel-Lösung mit automatischer SRT-Konvertierung (`astisub`), Sprachprüfung & Duplikat-Erkennung**:
   - **Intelligente Duplikat-Erkennung (`[DUPLIKAT]`)**: Erkennt automatisch doppelt vorhandene Untertitel (auch bei völlig unterschiedlicher Benennung wie z.B. `Movie.2020.de.srt` und `Movie.2020.German.srt` oder `Subs/track1_ger.srt`). Ein Text-Normalisierer entfernt Timestamps, Sequenznummern und Formatierungs-Tags (`<b>`, `{\\an8}` etc.) und berechnet einen Inhalts-Hash (SHA-256). Bei Duplikaten pro Sprache wird automatisch die vollständigste/größte Datei behalten und alle anderen werden als `[DUPLIKAT]` sicher entfernt.
   - **Nur `.srt` als Zielformat**: Alle Untertitelformate außer `.srt` (z.B. `.vtt`, `.ass`, `.ssa`, `.sub`, `.smi`, `.stl`, `.ttml`) werden automatisch mittels [`go-astisub`](https://pkg.go.dev/github.com/asticode/go-astisub) in saubere, standardkonforme `.srt`-Dateien konvertiert (`[KONVERTIEREN]`). Nach erfolgreicher Konvertierung werden die alten Nicht-SRT-Dateien (sowie evtl. zugehörige `.idx`-Dateien) gelöscht, sodass ausschließlich `.srt`-Dateien verbleiben.
   - **Nur Deutsch & Englisch behalten**: Es werden **ausschließlich deutsche (`.de`) und englische (`.en`)** Untertitel behalten. Alle anderen Sprachen (z.B. Spanisch, Französisch, Russisch, Chinesisch etc.) werden automatisch zum Löschen vorgemerkt und entfernt (`[LÖSCHEN]`).
   - **Inhaltsbasierte Sprachprüfung**: Vertraut nicht nur auf Dateinamen! Eine integrierte Textanalyse prüft die tatsächlichen Untertitel-Dateiinhalte (deutsche und englische Stoppwörter, Umlautgewichtung, Zeichensätze wie Kyrillisch/CJK) sowie Metadaten (z.B. VobSub `.idx` / `.sub`), um die Sprache zweifelsfrei zu erkennen. Selbst Untertitel ohne Sprachangabe im Dateinamen (z.B. `Subs/track1.vtt`) werden durch ihren Inhalt korrekt erkannt und als `.de.srt` bzw. `.en.srt` übernommen.
   - **Forced- & SDH-Erkennung**: Erkennt Kennzeichnungen wie `forced` / `forziert` oder `sdh` / `cc` und formatiert standardkonform:
     `The Film Title - Electric Buggalo (2009).de.forced.srt`
   - **Ordner-Organisation**:
     - Bei Filmen: Alle zum Film gehörenden DE/EN-Untertitel werden direkt in den Filmordner verschoben bzw. konvertiert.
     - Auch Untertitel aus Unterordnern wie `Subs/` oder `Subtitles/` werden erkannt, geprüft, sauber benannt, konvertiert und herausgezogen. Leere Unterordner werden automatisch bereinigt.
     - Bei Serien: Untertitel werden der passenden Episode zugeordnet (`S02E07`) und direkt neben die Episode (z.B. in `S02/`) gelegt:
       `TV Show Title/S02/TV Show Title - S02E07.de.forced.srt`
   - **Kollisionsschutz**: Falls mehrere unterschiedliche Untertitel derselben Sprache vorliegen, werden sie nummeriert (`.de.2.srt`), damit keine Datei überschrieben wird. Duplikate werden vorher entfernt.

3. **Serien**:
   - Erkennt Serientitel, Staffel und Episode (`S02E07`).
   - Bereinigt den Serienordner:
     `TV_Show_title_hevc-bla-bla_bla_2013` ➔ `TV Show Title`
   - **Mehrere Staffeln**: Legt automatisch Staffelordner an (`S01`, `S02`) und verschiebt Episoden und Untertitel hinein:
     `TV Show Title/S02/TV Show Title - S02E07.mp4`
   - **Nur eine Staffel**: Verbleibt flach im Serienordner:
     `TV Show Title/TV Show Title - S01E01.mp4`

4. **Umlaute, Bindestrich-Wörter & Sicherheit**:
   - Vollständige UTF-8-Unterstützung für deutsche Umlaute (`ä`, `ö`, `ü`, `ß`, `Ä`, `Ö`, `Ü`).
   - Korrekter Erhalt von echten Bindestrich-Wörtern wie `Spider-Man`, `X-Men`, `Sci-Fi` (keine Zerstörung zu `Spider - Man`).
   - Schutz vor versehentlichem Umbenennen von System- und Download-Ordnern (`Downloads`, `Movies`, `Series` etc.).
   - Dateisystem-Überschreibschutz vor unbeabsichtigtem Überschreiben vorhandener Dateien.

5. **Perfekte Touch- & Maus-Bedienung über alle Fenster**:
   - **Kein Terminal-Scrollen (Zero-Drift)**: Alle Ansichten passen ihre Höhe dynamisch an die Terminalgröße an, sodass Koordinaten immer zeilengenau stimmen.
   - **Kein Doppel-Auslösen (Press/Release Deduplication)**: Touch-Taps und Klicks lösen verlässlich genau einmal aus.
   - **Hauptfenster (Eingabe)**:
     - Große Touch-Boxen für Film- und Serienpfad.
     - Schlankes Dropdown für Trennzeichen (`Leerzeichen`, `Punkt`, `Unterstrich`).
     - Schnellzugriff auf Presets via `[ ⚡ Presets (1-99) ]` oder `Strg + P`.
     - Großer `[ ▶ LET'S GO ]` Touch-Button.
   - **Vorschaufenster (Screen 2)**:
     - Gestapelte Touch-Karten (`[ ✔ JETZT UMBENENNEN ]` und `[ ✖ Abbrechen / Zurück ]`) mit Daumen-Hitbox über die gesamte Bildschirmbreite.
     - Scrollen per Buttons `[ ▲ ]` / `[ ▼ ]`, Touch-Gesten oder Mausrad.
   - **Ergebnisfenster (Screen 3)**:
     - Touch-Karten für `[ ✔ Neuer Scan / Ordner ]` und roter `[ ✖ Beenden ]` Button.

---

## Installation & Start

```bash
# Bauen
go build -o smrt .

# Starten (Interaktive TUI)
./smrt

# Mit Pfad oder Preset-Nummer (1-99):
./smrt 1
./smrt /pfad/zu/filmen

# Beide Pfade (Filme & Serien) übergeben:
./smrt 1 2
./smrt -m /pfad/zu/filmen -s /pfad/zu/serien

# Version und Hilfe anzeigen:
./smrt -v
./smrt -h
```

## Steuerung im TUI

- **Maus / Touch**:
  - Direkt auf das jeweilige Eingabefeld tippen, um es zu fokussieren
  - Auf das Dropdown tippen, um die Optionen aufzuklappen und auszuwählen
  - `[ ⚡ Presets (1-99) ]` tippen für das Preset-Untermenü
  - In der Vorschau: `[ JETZT UMBENENNEN ]`, `[ Abbrechen ]` oder Scroll-Buttons antippen
  - Im Ergebnis: `[ Neuer Scan / Ordner ]` oder `[ Beenden ]` antippen
- **Tastatur**:
  - `Strg + P`: Preset-Manager öffnen / schließen
  - `Strg + U`: Trennzeichen direkt umschalten (Leerzeichen ➔ Punkt ➔ Unterstrich)
  - `Tab` / `Shift + Tab` / `↑` / `↓`: Zwischen Feldern wechseln
  - `Enter`: Aktion ausführen / Scan starten
  - `Esc`: Zurück zum Hauptfenster
  - `Strg + C`: Beenden
