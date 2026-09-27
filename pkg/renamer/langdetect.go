package renamer

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Language detection results
type LangDetectionResult struct {
	Language   string  // "de", "en", "es", "fr", "it", "ru", "cjk", "other"
	Confidence float64 // 0.0 - 1.0
	Details    string
}

// Top German stop words
var germanWords = map[string]bool{
	"der": true, "die": true, "das": true, "und": true, "sein": true, "in": true, "ein": true,
	"zu": true, "haben": true, "ich": true, "sie": true, "nicht": true, "den": true, "mit": true,
	"sich": true, "auch": true, "auf": true, "für": true, "ist": true, "eine": true, "wir": true,
	"einen": true, "dem": true, "wenn": true, "dann": true, "aber": true, "oder": true, "ja": true,
	"nein": true, "bitte": true, "danke": true, "hallo": true, "warum": true, "hier": true, "jetzt": true,
	"doch": true, "schon": true, "alles": true, "nichts": true, "wieder": true, "immer": true, "sehen": true,
	"gehen": true, "können": true, "müssen": true, "wollen": true, "sollen": true, "heute": true,
	"kein": true, "keine": true, "viel": true, "mehr": true, "gut": true, "sehr": true, "mich": true,
	"dich": true, "mir": true, "dir": true, "uns": true, "euch": true, "ihr": true, "ihm": true,
	"ihn": true, "ihnen": true, "mein": true, "meine": true, "dein": true, "deine": true, "seinem": true,
	"seinen": true, "war": true, "waren": true, "hat": true, "hatte": true, "hatten": true, "wird": true,
	"werden": true, "wurde": true, "wurden": true, "bist": true, "bin": true, "seid": true, "gibt": true,
	"weiß": true, "weiss": true, "schön": true, "über": true, "unter": true, "nach": true, "von": true,
}

// Top English stop words
var englishWords = map[string]bool{
	"the": true, "and": true, "you": true, "that": true, "was": true, "for": true, "are": true,
	"with": true, "his": true, "they": true, "this": true, "have": true, "from": true, "one": true,
	"had": true, "word": true, "but": true, "not": true, "what": true, "all": true, "were": true,
	"when": true, "your": true, "can": true, "said": true, "there": true, "use": true, "each": true,
	"which": true, "she": true, "how": true, "their": true, "will": true, "other": true, "about": true,
	"out": true, "many": true, "then": true, "them": true, "these": true, "some": true, "her": true,
	"would": true, "make": true, "like": true, "him": true, "into": true, "time": true, "has": true,
	"look": true, "two": true, "more": true, "write": true, "see": true, "number": true, "way": true,
	"could": true, "people": true, "than": true, "first": true, "water": true, "been": true, "call": true,
	"who": true, "oil": true, "its": true, "now": true, "find": true, "long": true, "down": true,
	"day": true, "did": true, "get": true, "come": true, "made": true, "may": true, "part": true,
	"don't": true, "you're": true, "i'm": true, "didn't": true, "can't": true, "won't": true,
	"it's": true, "that's": true, "there's": true, "what's": true, "we're": true, "they're": true,
	"yeah": true, "okay": true, "hello": true, "thanks": true, "please": true, "sorry": true,
}

// Non-German/English stop words (Spanish, French, Italian, Portuguese)
var otherRomanceWords = map[string]string{
	"que": "es/fr", "para": "es/pt", "por": "es/pt", "con": "es/it", "una": "es/it", "como": "es/it",
	"pero": "es", "este": "es/pt", "esta": "es/pt", "estoy": "es", "hola": "es", "gracias": "es",
	"señor": "es", "bueno": "es", "hacer": "es", "ahora": "es", "todos": "es/pt", "nada": "es/pt",
	"dans": "fr", "pour": "fr", "avec": "fr", "cette": "fr", "sont": "fr", "mais": "fr", "tout": "fr",
	"faire": "fr", "merci": "fr", "monsieur": "fr", "vous": "fr", "nous": "fr", "elle": "fr", "ils": "fr",
	"c'est": "fr", "d'accord": "fr", "oui": "fr", "très": "fr", "aussi": "fr",
	"sono": "it", "questo": "it", "anche": "it", "perché": "it", "tutto": "it", "grazie": "it",
	"ciao": "it", "bene": "it", "cosa": "it", "quando": "it", "sempre": "it",
}

var (
	timestampRegex = regexp.MustCompile(`(?m)^\d{1,4}$|^\d{1,2}:\d{2}:\d{2}[,.]\d{1,3}.*$|-->.*$`)
	tagRegex       = regexp.MustCompile(`<[^>]+>|{[^}]+}|\[[^\]]+\]`)
	wordCleanRegex = regexp.MustCompile(`[^\p{L}']+`)
)

// DetectSubtitleLanguage analyzes the content and metadata of a subtitle file.
// It returns "de", "en", or other language codes (e.g. "es", "fr", "ru", "other").
func DetectSubtitleLanguage(filePath string, filenameInfo SubtitleInfo) LangDetectionResult {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Special case: .idx files (VobSub metadata)
	if ext == ".idx" {
		return detectFromIdxFile(filePath, filenameInfo)
	}
	if ext == ".sub" {
		idxPath := strings.TrimSuffix(filePath, ext) + ".idx"
		if _, err := os.Stat(idxPath); err == nil {
			return detectFromIdxFile(idxPath, filenameInfo)
		}
	}

	// Read content sample (up to 96 KB)
	content, err := readSubtitleSample(filePath, 98304)
	if err != nil || len(strings.TrimSpace(content)) == 0 {
		// Fallback to filename if content cannot be read
		if filenameInfo.LangCode == "de" || filenameInfo.LangCode == "en" {
			return LangDetectionResult{Language: filenameInfo.LangCode, Confidence: 0.5, Details: "Dateiname"}
		}
		if filenameInfo.LangCode != "" {
			return LangDetectionResult{Language: filenameInfo.LangCode, Confidence: 0.5, Details: "Dateiname (Nicht DE/EN)"}
		}
		return LangDetectionResult{Language: "unknown", Confidence: 0.0, Details: "Datei leer oder unlesbar"}
	}

	// 1. Script checks (Cyrillic, CJK, Arabic)
	var cyrillicCount, cjkCount, arabicCount int
	for _, r := range content {
		if unicode.In(r, unicode.Cyrillic) {
			cyrillicCount++
		} else if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			cjkCount++
		} else if unicode.In(r, unicode.Arabic) {
			arabicCount++
		}
	}

	if cyrillicCount > 15 {
		return LangDetectionResult{Language: "ru", Confidence: 0.95, Details: "Kyrillische Schrift"}
	}
	if cjkCount > 15 {
		return LangDetectionResult{Language: "cjk", Confidence: 0.95, Details: "Asiatische Schrift (CJK)"}
	}
	if arabicCount > 15 {
		return LangDetectionResult{Language: "ar", Confidence: 0.95, Details: "Arabische Schrift"}
	}

	// 2. Clean subtitle formatting
	cleaned := timestampRegex.ReplaceAllString(content, " ")
	cleaned = tagRegex.ReplaceAllString(cleaned, " ")

	// Count German specific letters (ä, ö, ü, ß)
	germanCharCount := 0
	for _, r := range strings.ToLower(cleaned) {
		if r == 'ä' || r == 'ö' || r == 'ü' || r == 'ß' {
			germanCharCount++
		}
	}

	// 3. Word frequency scoring
	words := wordCleanRegex.Split(strings.ToLower(cleaned), -1)

	var scoreGerman, scoreEnglish, scoreOther int
	for _, w := range words {
		w = strings.Trim(w, "'")
		if len(w) < 2 {
			continue
		}
		if germanWords[w] {
			scoreGerman += 2
		}
		if englishWords[w] {
			scoreEnglish += 2
		}
		if _, ok := otherRomanceWords[w]; ok {
			scoreOther += 3
		}
	}

	// Add strong weight for German special characters
	scoreGerman += germanCharCount * 4

	// If filename indicates German/English, give it a small prior boost
	if filenameInfo.LangCode == "de" {
		scoreGerman += 3
	} else if filenameInfo.LangCode == "en" {
		scoreEnglish += 3
	} else if filenameInfo.LangCode != "" {
		scoreOther += 5
	}

	// Determine winner
	if scoreOther > scoreGerman && scoreOther > scoreEnglish && scoreOther >= 6 {
		return LangDetectionResult{
			Language:   "other",
			Confidence: 0.85,
			Details:    "Romanische/Andere Fremdsprache erkannt",
		}
	}

	if scoreGerman > scoreEnglish && scoreGerman >= 5 {
		confidence := float64(scoreGerman) / float64(scoreGerman+scoreEnglish+1)
		return LangDetectionResult{
			Language:   "de",
			Confidence: confidence,
			Details:    "Deutsch (Wortschatz & Umlaute)",
		}
	}

	if scoreEnglish > scoreGerman && scoreEnglish >= 5 {
		confidence := float64(scoreEnglish) / float64(scoreGerman+scoreEnglish+1)
		return LangDetectionResult{
			Language:   "en",
			Confidence: confidence,
			Details:    "Englisch (Wortschatz)",
		}
	}

	// If inconclusive by score, check filename metadata
	if filenameInfo.LangCode == "de" {
		return LangDetectionResult{Language: "de", Confidence: 0.6, Details: "Dateiname (DE)"}
	}
	if filenameInfo.LangCode == "en" {
		return LangDetectionResult{Language: "en", Confidence: 0.6, Details: "Dateiname (EN)"}
	}
	if filenameInfo.LangCode != "" {
		return LangDetectionResult{Language: filenameInfo.LangCode, Confidence: 0.6, Details: "Dateiname (" + filenameInfo.LangCode + ")"}
	}

	return LangDetectionResult{Language: "unknown", Confidence: 0.0, Details: "Nicht zweifelsfrei DE oder EN"}
}

// readSubtitleSample reads and normalizes subtitle text encoding (UTF-8, UTF-16 LE/BE, Latin1).
func readSubtitleSample(path string, maxBytes int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, maxBytes)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	buf = buf[:n]

	// Handle UTF-16 Little Endian BOM
	if len(buf) >= 2 && buf[0] == 0xFF && buf[1] == 0xFE {
		return decodeUTF16(buf[2:], binary.LittleEndian), nil
	}
	// Handle UTF-16 Big Endian BOM
	if len(buf) >= 2 && buf[0] == 0xFE && buf[1] == 0xFF {
		return decodeUTF16(buf[2:], binary.BigEndian), nil
	}
	// Handle UTF-8 BOM
	if len(buf) >= 3 && buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF {
		buf = buf[3:]
	}

	// If valid UTF-8
	if utf8.Valid(buf) {
		return string(buf), nil
	}

	// Fallback: convert bytes preserving Latin1/Windows-1252 runes
	var sb strings.Builder
	for _, b := range buf {
		sb.WriteRune(rune(b))
	}
	return sb.String(), nil
}

func decodeUTF16(b []byte, order binary.ByteOrder) string {
	u16s := make([]uint16, len(b)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = order.Uint16(b[i*2:])
	}
	return string(utf16Runes(u16s))
}

func utf16Runes(s []uint16) []rune {
	return bytes.Runes([]byte(string(utf8StringFromU16(s))))
}

func utf8StringFromU16(s []uint16) string {
	var sb strings.Builder
	for _, v := range s {
		sb.WriteRune(rune(v))
	}
	return sb.String()
}

// detectFromIdxFile parses .idx metadata files for language entries.
func detectFromIdxFile(path string, filenameInfo SubtitleInfo) LangDetectionResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return LangDetectionResult{Language: filenameInfo.LangCode, Confidence: 0.5, Details: "IDX Dateiname"}
	}
	content := strings.ToLower(string(data))

	// Look for "id: de" or "id: ger" or "id: en" or "id: eng"
	if strings.Contains(content, "id: de") || strings.Contains(content, "id: ger") || strings.Contains(content, "deutsch") {
		return LangDetectionResult{Language: "de", Confidence: 0.9, Details: "IDX Metadaten (Deutsch)"}
	}
	if strings.Contains(content, "id: en") || strings.Contains(content, "id: eng") || strings.Contains(content, "english") {
		return LangDetectionResult{Language: "en", Confidence: 0.9, Details: "IDX Metadaten (Englisch)"}
	}

	// Check if another language is listed
	idRegex := regexp.MustCompile(`id:\s*([a-z]{2,3})`)
	if m := idRegex.FindStringSubmatch(content); len(m) > 1 {
		return LangDetectionResult{Language: m[1], Confidence: 0.8, Details: "IDX Metadaten (" + m[1] + ")"}
	}

	return LangDetectionResult{Language: "other", Confidence: 0.5, Details: "IDX Metadaten (Nicht DE/EN)"}
}
