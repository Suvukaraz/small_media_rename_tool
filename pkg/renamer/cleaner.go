package renamer

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Common video and subtitle extensions
var VideoExtensions = map[string]bool{
	".mkv":  true,
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".m4v":  true,
	".flv":  true,
	".webm": true,
	".ts":   true,
	".iso":  true,
}

var SubtitleExtensions = map[string]bool{
	".srt":  true,
	".sub":  true,
	".idx":  true,
	".vtt":  true,
	".ass":  true,
	".ssa":  true,
	".smi":  true,
	".stl":  true,
	".ttml": true,
}

// Scene tags and junk tokens that should be stripped
var sceneTagList = []string{
	// Video resolutions & standards
	"2160p", "4k", "uhd", "1080p", "1080i", "720p", "576p", "480p", "sd", "hd",
	// Codecs
	"x264", "x265", "h264", "h265", "h.264", "h.265", "hevc", "avc", "xvid", "divx", "av1", "10bit", "8bit",
	// Sources
	"bluray", "blu-ray", "bdrip", "brrip", "web-dl", "webdl", "web-rip", "webrip", "web", "hdtv", "pdtv", "dsr", "dvdrip", "dvd", "remux", "vhs",
	// Audio
	"ddp5.1", "dd5.1", "ddp", "dts-hd", "dts", "ac3", "eac3", "aac2.0", "aac", "flac", "truehd", "atmos", "mp3",
	// Languages & editions
	"german", "dl", "dubbed", "multi", "subbed", "eng", "ita", "french",
	// Release tags & junk
	"proper", "repack", "rerip", "internal", "unrated", "extended", "directors.cut", "remastered", "imax", "sample",
}

// Compile a unified delimiter-aware scene tag regex
var sceneTagPattern = func() *regexp.Regexp {
	var escaped []string
	for _, tag := range sceneTagList {
		escaped = append(escaped, regexp.QuoteMeta(tag))
	}
	pattern := `(?i)(?:[._\s\-\[(]|\A)(` + strings.Join(escaped, "|") + `)(?:[._\s\-\])]|\z)`
	return regexp.MustCompile(pattern)
}()

// Regex to detect TV episode notation (e.g., S01E02, s1e2, 1x02, S01E01-E02, S01E01E02)
var (
	tvPatternSxxExx    = regexp.MustCompile(`(?i)[._\s\-\[](?:s|season\s*)(\d{1,2})\s*(?:e|ep|episode\s*)(\d{1,3})(?:(?:\s*-\s*|\s*e|\s*ep)(\d{1,3}))?`)
	tvPatternNxNN      = regexp.MustCompile(`(?i)[._\s\-\[](\d{1,2})x(\d{1,3})(?:-(\d{1,3}))?`)
	tvSeasonTagPattern = regexp.MustCompile(`(?i)[._\s\-\[](?:s\d{1,2}(?:\s*-\s*s?\d{1,2})?|season\s*\d{1,2}(?:\s*-\s*\d{1,2})?|staffel\s*\d{1,2}(?:\s*-\s*\d{1,2})?|complete)(?:[._\s\-\]]|\z)`)
	yearPattern        = regexp.MustCompile(`(?i)(?:[-(._\s]|\A)(?:[\[(])?(19\d{2}|20\d{2})(?:[\])])?(?:[-(._\s]|\z)`)
)

// ExtractEpisodeInfo checks if a filename contains TV show season & episode markers.
func ExtractEpisodeInfo(name string) (season, episode, ep2 int, matched bool) {
	// First try S01E02 format
	if m := tvPatternSxxExx.FindStringSubmatch(name); len(m) > 2 {
		s, err1 := strconv.Atoi(m[1])
		e, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil {
			second := 0
			if len(m) > 3 && m[3] != "" {
				second, _ = strconv.Atoi(m[3])
			}
			return s, e, second, true
		}
	}

	// Try 1x02 format
	if m := tvPatternNxNN.FindStringSubmatch(name); len(m) > 2 {
		s, err1 := strconv.Atoi(m[1])
		e, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil {
			second := 0
			if len(m) > 3 && m[3] != "" {
				second, _ = strconv.Atoi(m[3])
			}
			return s, e, second, true
		}
	}

	return 0, 0, 0, false
}

// ExtractMovieYearAndTitle parses movie name and separates clean title from release year.
func ExtractMovieYearAndTitle(baseName string) (title string, year int) {
	// Find all 4-digit years between 1900 and 2099
	locs := yearPattern.FindAllStringSubmatchIndex(baseName, -1)
	if len(locs) > 0 {
		// Pick the match that looks most like a release year (prefer the last valid year if before scene tags, or year in brackets)
		bestIdx := -1
		bestYear := 0
		for _, loc := range locs {
			// Submatch 1 is the 4-digit year
			yearStr := baseName[loc[2]:loc[3]]
			y, err := strconv.Atoi(yearStr)
			if err == nil && y >= 1920 && y <= 2035 {
				bestIdx = loc[0]
				bestYear = y
			}
		}

		if bestYear > 0 && bestIdx >= 0 {
			rawTitle := baseName[:bestIdx]
			cleaned := CleanTitle(rawTitle)
			if cleaned != "" {
				return cleaned, bestYear
			}
		}
	}

	// If no year found, strip scene tags and clean title
	cleaned := CleanTitle(StripSceneTags(baseName))
	return cleaned, 0
}

// StripSceneTags trims any junk / release info starting from known scene tags.
func StripSceneTags(name string) string {
	if loc := tvSeasonTagPattern.FindStringIndex(name); loc != nil && loc[0] > 0 {
		name = name[:loc[0]]
	}
	loc := sceneTagPattern.FindStringSubmatchIndex(name)
	if len(loc) > 2 {
		// Submatch 1 is the tag. We cut right before the tag (accounting for leading delimiter).
		// loc[0] is the start of the match (which includes the leading delimiter, unless at start of string)
		cutPos := loc[0]
		// If delimiter at start of match, keep the string before that delimiter
		res := name[:cutPos]
		return strings.Trim(res, " ._-")
	}
	return strings.Trim(name, " ._-")
}

// CleanTitle takes a raw string (e.g. "The_Film_Title-electric_Buggalo") and cleans it:
// - Replaces underscores and dots with spaces
// - Formats hyphens separating phrases with spaced hyphens " - "
// - Capitalizes words (preserving existing capitalization like "TV" or "Buggalo", and properly handling Umlauts)
func CleanTitle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Split on hyphens
	parts := strings.Split(raw, "-")
	
	// Clean each part individually
	var cleanedParts []string
	for _, part := range parts {
		p := strings.Trim(part, " ._")
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, "_", " ")
		p = strings.ReplaceAll(p, ".", " ")
		p = cleanSpaces(p)
		p = capitalizeWords(p)
		if p != "" {
			cleanedParts = append(cleanedParts, p)
		}
	}

	if len(cleanedParts) == 0 {
		return ""
	}

	// Reconstruct: decide whether to use " - " (phrase separator) or "-" (compound word)
	var result strings.Builder
	result.WriteString(cleanedParts[0])
	for i := 1; i < len(cleanedParts); i++ {
		prev := cleanedParts[i-1]
		curr := cleanedParts[i]
		// If both sides are single words (no spaces), it's a compound like Spider-Man
		prevIsMultiWord := strings.Contains(prev, " ")
		currIsMultiWord := strings.Contains(curr, " ")
		if !prevIsMultiWord && !currIsMultiWord {
			result.WriteString("-")
		} else {
			result.WriteString(" - ")
		}
		result.WriteString(curr)
	}

	return result.String()
}

// capitalizeWords capitalizes the first letter of each word in a string,
// preserving existing uppercase letters (e.g., "electric" -> "Electric", "Buggalo" -> "Buggalo", "TV" -> "TV").
// It fully supports UTF-8 runes including German Umlauts (ä, ö, ü, ß).
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = capitalizeWord(w)
	}
	return strings.Join(words, " ")
}

func capitalizeWord(w string) string {
	if w == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(w)
	// If first letter is lowercase, uppercase it
	if unicode.IsLower(r) {
		return string(unicode.ToUpper(r)) + w[size:]
	}
	return w
}

func cleanSpaces(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// SeparatorMode specifies how words and elements are separated in filenames and folders.
type SeparatorMode string

const (
	SeparatorSpace      SeparatorMode = "space"      // " " (Standard Leerzeichen)
	SeparatorDot        SeparatorMode = "dot"        // "." (Punkte)
	SeparatorUnderscore SeparatorMode = "underscore" // "_" (Unterstriche)
)

// FormatSeparator applies the separator rule (spaces, dots, or underscores).
func FormatSeparator(name string, mode SeparatorMode) string {
	switch mode {
	case SeparatorDot:
		res := strings.ReplaceAll(name, " ", ".")
		for strings.Contains(res, "..") {
			res = strings.ReplaceAll(res, "..", ".")
		}
		return res
	case SeparatorUnderscore:
		res := strings.ReplaceAll(name, " ", "_")
		for strings.Contains(res, "__") {
			res = strings.ReplaceAll(res, "__", "_")
		}
		return res
	default: // SeparatorSpace
		return name
	}
}

// FormatSpaces applies the spacing rule (kept for backward compatibility).
func FormatSpaces(name string, useSpaces bool) string {
	if useSpaces {
		return FormatSeparator(name, SeparatorSpace)
	}
	return FormatSeparator(name, SeparatorDot)
}

// IsMediaFile returns true if the extension matches a supported video file.
func IsMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return VideoExtensions[ext]
}

// IsSubtitleFile returns true if the extension matches a supported subtitle file.
func IsSubtitleFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return SubtitleExtensions[ext]
}
