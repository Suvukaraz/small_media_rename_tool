# SMRT - Media Renamer & Organizer

Ein schnelles Go-Tool mit einer modernen, responsiven Bubble Tea TUI, das Filme und Serien automatisch erkennt, bereinigt und strukturiert umbenennt.

Optimiert für **Desktop & Mobile** (z.B. Termux / SSH via Smartphone) mit voller **Maus- und Touch-Unterstützung**.

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
   - **Kein Terminal-Scrollen (Zero-Drift)**: Alle Ansichten (Eingabe mit offenem/geschlossenem Dropdown, Vorschau mit Viewport, Ergebnis) passen ihre Höhe dynamisch so an, dass das Terminal niemals scrollt. Dadurch stimmen Touch- und Mauskoordinaten (`msg.Y`) zu 100% zeilengenau mit den Bedienelementen überein.
   - **Kein Doppel-Auslösen (Press/Release Deduplication)**: Taps auf Touchscreens oder Mausklicks lösen genau einmal aus. Das Heben des Fingers nach einem Klick schließt das Dropdown nicht mehr versehentlich und löst keine Folgebuttons aus.
   - **Hauptfenster (Eingabe)**:
     - Große Touch-Box für den Pfad inklusive direktem Cursor-Setzen per Fingertipp auf die gewünschte Textstelle.
     - Schlankes, fingerfreundliches Dropdown für Trennzeichen (`Leerzeichen`, `Punkt`, `Unterstrich`) mit Sofort-Auswahl per Touch und visueller Aktivitätsanzeige.
     - Schnelltaste `Strg + U` zum Durchwechseln ohne das Textfeld verlassen zu müssen.
     - Großer, kontrastreicher `[ ▶ LET'S GO ]` Touch-Button.
   - **Vorschaufenster (Screen 2)**:
     - **Vollflächige Touch-Buttons**: Auf Smartphones gestapelte Touch-Karten (`[ ✔ JETZT UMBENENNEN ]` und `[ ✖ Abbrechen / Zurück ]`) mit Daumen-Hitbox über die gesamte Bildschirmbreite. Auf Desktop mit exakt begrenzten Schaltflächen (keine Geisterklicks im leeren Rand).
     - **Touch-Scrolling**: Zusätzliche Touch-Buttons `[ ▲ Nach oben ]` & `[ ▼ Nach unten ]`, Scrollen per oberer/unterer Bildschirmhälfte sowie klassisches Mausrad.
     - **Dynamische Viewport-Höhe**: Berechnet die verfügbare Zeilenhöhe mathematisch exakt, sodass alle Buttons immer voll sichtbar und erreichbar bleiben.
   - **Ergebnisfenster (Screen 3)**:
     - Große, fingerfreundliche Touch-Karten für `[ ✔ Neuer Ordner ]` und `[ ✖ Beenden ]`.
   - **Mathematisch exakte Hitboxen**: Alle Schaltflächen, Dropdowns und Optionen reagieren verlässlich auf Desktop-Mäuse, Touchscreens und mobile Terminals (z.B. Termux).

---

## Installation & Start

```bash
# Bauen
go build -o smrt .

# Starten
./smrt

# Optional mit direkt übergebenem Pfad:
./smrt /pfad/zu/mediendateien
```

## Steuerung im TUI

- **Maus / Touch**:
  - Direkt auf das Eingabefeld tippen, um es zu fokussieren
  - Auf das Dropdown tippen, um die Optionen aufzuklappen und auszuwählen
  - In der Vorschau: `[ JETZT UMBENENNEN ]`, `[ Abbrechen ]` oder Scroll-Buttons per Fingertipp bedienen
  - Im Ergebnis: `[ Neuer Ordner ]` oder `[ Beenden ]` antippen
  - Mausrad oder Touch-Gesten zum Scrollen im Vorschaufenster
- **Tastatur**:
  - `Strg + U`: Trennzeichen direkt umschalten (Leerzeichen ➔ Punkt ➔ Unterstrich)
  - `Tab` / `↑` / `↓`: Zwischen Elementen bzw. im offenen Dropdown wechseln
  - `Enter`: Dropdown öffnen/schließen, Scan starten oder Umbenennung bestätigen
  - `Esc`: Dropdown schließen bzw. aus der Vorschau zurückkehren
  - `Strg + C`: Beenden
