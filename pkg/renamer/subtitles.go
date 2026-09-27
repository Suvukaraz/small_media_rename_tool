package renamer

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SubtitleInfo stores parsed language and metadata for a subtitle file.
type SubtitleInfo struct {
	OriginalPath string
	Filename     string
	Extension    string
	LangCode     string // e.g. "de", "en", "fr"
	IsForced     bool
	IsSDH        bool
	Season       int
	Episode      int
	HasTVEpisode bool
}

var langMap = map[string]string{
	// German
	"german": "de", "deutsch": "de", "ger": "de", "de": "de",
	// English
	"english": "en", "eng": "en", "en": "en",
	// French
	"french": "fr", "francais": "fr", "fra": "fr", "fre": "fr", "fr": "fr",
	// Spanish
	"spanish": "es", "espanol": "es", "spa": "es", "es": "es",
	// Italian
	"italian": "it", "italiano": "it", "ita": "it", "it": "it",
	// Dutch
	"dutch": "nl", "nederlands": "nl", "dut": "nl", "nl": "nl",
	// Russian
	"russian": "ru", "rus": "ru", "ru": "ru",
	// Polish
	"polish": "pl", "pol": "pl", "pl": "pl",
	// Turkish
	"turkish": "tr", "tur": "tr", "tr": "tr",
	// Japanese
	"japanese": "ja", "jpn": "ja", "ja": "ja",
	// Korean
	"korean": "ko", "kor": "ko", "ko": "ko",
	// Chinese
	"chinese": "zh", "chi": "zh", "zho": "zh", "zh": "zh",
	// Portuguese
	"portuguese": "pt", "por": "pt", "pt": "pt", "pob": "pt", "pt-br": "pt",
	// Scandinavian
	"swedish": "sv", "swe": "sv", "sv": "sv",
	"norwegian": "no", "nor": "no", "no": "no",
	"danish": "da", "dan": "da", "da": "da",
	"finnish": "fi", "fin": "fi", "fi": "fi",
}

var tokenDelimRegex = regexp.MustCompile(`[._\s\-\[\]\(\)]+`)

// ParseSubtitle analyzes a subtitle path and extracts language, forced/sdh tags, and season/episodes.
func ParseSubtitle(path string) SubtitleInfo {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	info := SubtitleInfo{
		OriginalPath: path,
		Filename:     filename,
		Extension:    ext,
	}

	// Check if this subtitle has TV episode markers
	s, e, _, matched := ExtractEpisodeInfo(base)
	if matched {
		info.Season = s
		info.Episode = e
		info.HasTVEpisode = true
	}

	// Split tokens by delimiters
	tokens := tokenDelimRegex.Split(strings.ToLower(base), -1)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}

		// Check for forced
		if tok == "forced" || tok == "forziert" || tok == "force" {
			info.IsForced = true
			continue
		}

		// Check for hearing impaired / sdh
		if tok == "sdh" || tok == "hi" || tok == "cc" {
			info.IsSDH = true
			continue
		}

		// Check for language if not yet found
		if info.LangCode == "" {
			if code, ok := langMap[tok]; ok {
				info.LangCode = code
			}
		}
	}

	return info
}

// SubtitleSuffix returns the formatted suffix for the target subtitle filename.
// Examples:
// - ".de" -> target: "Movie (2009).de.srt"
// - ".de.forced" -> target: "Movie (2009).de.forced.srt"
// - "" -> target: "Movie (2009).srt"
func SubtitleSuffix(info SubtitleInfo) string {
	var parts []string
	if info.LangCode != "" {
		parts = append(parts, info.LangCode)
	}
	if info.IsForced {
		parts = append(parts, "forced")
	} else if info.IsSDH {
		parts = append(parts, "sdh")
	}

	if len(parts) == 0 {
		return ""
	}
	return "." + strings.Join(parts, ".")
}
