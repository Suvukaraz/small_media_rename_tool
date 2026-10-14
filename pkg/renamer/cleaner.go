package renamer

import (
	"fmt"
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
	"dts-hd.ma", "dts-hd", "dts-x", "dts", "truehd", "atmos", "ddp5.1", "dd5.1", "ddp", "ac3", "eac3", "aac2.0", "aac", "flac", "mp3", "5.1", "7.1",
	// HDR & Video details
	"hdr10plus", "hdr10+", "hdr10", "hdr", "dovi", "dv", "sdr",
	// Streaming sources
	"amzn", "nf", "dsnp", "atvp", "hmax", "itunes",
	// Languages & editions
	"german", "deutsch", "dl", "dubbed", "multi", "subbed", "eng", "english", "ita", "french",
	// Release tags & junk
	"proper", "repack", "rerip", "internal", "unrated", "extended", "directors.cut", "theatrical", "remastered", "imax", "sample", "complete",
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

// Regex to detect TV episode notation
var (
	// S01E02, s1e2, S01.E02, S01_E02, S01E01-E02, S01E01-02, S01E01E02, S01E01.E02
	tvPatternSxxExx = regexp.MustCompile(`(?i)[._\s\-\[](?:s|season\s*)(\d{1,2})[._\s\-]*(?:e|ep|episode\s*)(\d{1,3})(?:(?:\s*-\s*e?|\s*[._+]\s*e|\s*e|\s*ep)(\d{1,3}))?`)
	// 1x02, 01x02, 1x01-02
	tvPatternNxNN   = regexp.MustCompile(`(?i)[._\s\-\[](\d{1,2})x(\d{1,3})(?:-(\d{1,3}))?`)
	// German notation: Staffel 1 Folge 2, Staffel.1.Folge.02, St.1 Ep.2
	tvPatternGerman = regexp.MustCompile(`(?i)[._\s\-\[](?:staffel|st)[._\s\-]*(\d{1,2})[._\s\-]*(?:folge|episode|ep|flg)[._\s\-]*(\d{1,3})(?:[._\s\-]*(?:folge|episode|ep|flg|-)[._\s\-]*(\d{1,3}))?`)
	// Standalone episode: E01, EP01, Folge 01, Episode 01 (without season)
	tvPatternStandaloneEp = regexp.MustCompile(`(?i)(?:[._\s\-\[]|^)(?:e|ep|folge|episode)[._\s\-]*(\d{1,3})(?:[._\s\-]*(?:e|ep|folge|episode|-)[._\s\-]*(\d{1,3}))?(?:[._\s\-\]]|\z)`)
	// Leading number in filename (e.g. "01 - Title.mkv" or "01.mkv")
	tvPatternLeadingEp    = regexp.MustCompile(`^(\d{1,3})(?:[._\s\-]|$)`)
	tvPatternPartEp       = regexp.MustCompile(`(?i)(?:[._\s\-\[]|^)(?:part|teil|pt)[._\s\-]*(\d{1,3})(?:[._\s\-\]]|\z)`)
	tvPatternDashEp       = regexp.MustCompile(`(?i)(?:[._\s]|\A)-\s*(\d{1,3})(?:[._\s\-\]]|\z)`)
	tvSeasonFolderPattern = regexp.MustCompile(`(?i)(?:^|[._\s\-\[])(?:season|staffel|st|s)[._\s\-]*(\d{1,2})(?:[._\s\-\]]|$)`)
	tvSeasonTagPattern    = regexp.MustCompile(`(?i)[._\s\-\[](?:s\d{1,2}(?:\s*-\s*s?\d{1,2})?|season\s*\d{1,2}(?:\s*-\s*\d{1,2})?|staffel\s*\d{1,2}(?:\s*-\s*\d{1,2})?|complete)(?:[._\s\-\]]|\z)`)
	yearPattern           = regexp.MustCompile(`(?i)(?:[-(._\s]|\A)(?:[\[(])?(19\d{2}|20\d{2})(?:[\])])?(?:[-(._\s]|\z)`)
)

// ExtractSeasonFromFolder extracts season number if folder name matches "Season 1", "Staffel 02", "S01", "Staffel_1", etc.
func ExtractSeasonFromFolder(folderName string) int {
	clean := strings.TrimSpace(folderName)
	if m := tvSeasonFolderPattern.FindStringSubmatch(clean); len(m) > 1 {
		s, err := strconv.Atoi(m[1])
		if err == nil {
			return s
		}
	}
	return 0
}

// ExtractEpisodeInfo checks if a filename contains TV show season & episode markers.
func ExtractEpisodeInfo(name string) (season, episode, ep2 int, matched bool) {
	// 1. Try S01E02 format
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

	// 2. Try German notation (Staffel 1 Folge 2)
	if m := tvPatternGerman.FindStringSubmatch(name); len(m) > 2 {
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

	// 3. Try 1x02 format
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

// ExtractEpisodeInfoFromPath checks the filename and parent directory hierarchy to detect season & episode.
func ExtractEpisodeInfoFromPath(fullPath string) (season, episode, ep2 int, matched bool) {
	fileName := filepath.Base(fullPath)
	s, e, e2, ok := ExtractEpisodeInfo(fileName)
	if ok {
		return s, e, e2, true
	}

	// Try standalone episode in filename: e.g. "Show - E01.mkv" or "Show - Folge 02.mkv"
	if m := tvPatternStandaloneEp.FindStringSubmatch(fileName); len(m) > 1 {
		ep, err := strconv.Atoi(m[1])
		if err == nil {
			second := 0
			if len(m) > 2 && m[2] != "" {
				second, _ = strconv.Atoi(m[2])
			}
			parentDir := filepath.Base(filepath.Dir(fullPath))
			seasonNum := ExtractSeasonFromFolder(parentDir)
			if seasonNum <= 0 {
				seasonNum = 1 // Default to season 1
			}
			return seasonNum, ep, second, true
		}
	}

	// Try Part 1 / Teil 1 notation in filename
	if m := tvPatternPartEp.FindStringSubmatch(fileName); len(m) > 1 {
		ep, err := strconv.Atoi(m[1])
		if err == nil {
			parentDir := filepath.Base(filepath.Dir(fullPath))
			seasonNum := ExtractSeasonFromFolder(parentDir)
			if seasonNum <= 0 {
				seasonNum = 1
			}
			return seasonNum, ep, 0, true
		}
	}

	// Try dash episode: e.g. "Show - 01.mkv"
	if m := tvPatternDashEp.FindStringSubmatch(fileName); len(m) > 1 {
		ep, err := strconv.Atoi(m[1])
		if err == nil {
			parentDir := filepath.Base(filepath.Dir(fullPath))
			seasonNum := ExtractSeasonFromFolder(parentDir)
			if seasonNum <= 0 {
				seasonNum = 1
			}
			return seasonNum, ep, 0, true
		}
	}

	// Try parent directory as season folder (e.g. "Season 1/01 - Pilot.mkv" or "Staffel 2/05.mkv")
	parentDir := filepath.Base(filepath.Dir(fullPath))
	seasonNum := ExtractSeasonFromFolder(parentDir)
	if seasonNum > 0 {
		baseWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		if m := tvPatternLeadingEp.FindStringSubmatch(baseWithoutExt); len(m) > 1 {
			ep, err := strconv.Atoi(m[1])
			if err == nil {
				return seasonNum, ep, 0, true
			}
		}
	}

	return 0, 0, 0, false
}

// ExtractShowTitleFromFilename extracts the series title portion from an episode filename.
func ExtractShowTitleFromFilename(fileName string) string {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Find episode pattern position
	var loc []int
	if l := tvPatternSxxExx.FindStringIndex(base); l != nil {
		loc = l
	} else if l := tvPatternGerman.FindStringIndex(base); l != nil {
		loc = l
	} else if l := tvPatternNxNN.FindStringIndex(base); l != nil {
		loc = l
	} else if l := tvPatternStandaloneEp.FindStringIndex(base); l != nil {
		loc = l
	} else if l := tvPatternPartEp.FindStringIndex(base); l != nil {
		loc = l
	} else if l := tvPatternDashEp.FindStringIndex(base); l != nil {
		loc = l
	}


	if loc != nil && loc[0] > 0 {
		prefix := strings.Trim(base[:loc[0]], " ._-[]()")
		clean := CleanTitle(StripSceneTags(prefix))
		// If clean title ends with 4-digit year, e.g. "Doctor Who 2005", format as "Doctor Who (2005)"
		locs := yearPattern.FindAllStringSubmatchIndex(clean, -1)
		if len(locs) > 0 {
			lastLoc := locs[len(locs)-1]
			if lastLoc[1] == len(clean) {
				yearStr := clean[lastLoc[2]:lastLoc[3]]
				titlePart := strings.TrimSpace(clean[:lastLoc[0]])
				if titlePart != "" && !strings.HasSuffix(clean, "("+yearStr+")") {
					clean = fmt.Sprintf("%s (%s)", titlePart, yearStr)
				}
			}
		}
		if clean != "" {
			return clean
		}
	}

	return ""
}

// StripYearFromTitle removes release year in parentheses e.g. " (2026)" or " [2026]" or trailing year from title.
// For TV episodes, the year can be present in the folder name, but MUST NOT be in the episode filenames.
func StripYearFromTitle(title string) string {
	trimmed := strings.TrimSpace(title)
	// Match trailing (YYYY) or [YYYY]
	reParenYear := regexp.MustCompile(`(?i)\s*[\[(](?:19\d{2}|20\d{2})[\])]\s*$`)
	if reParenYear.MatchString(trimmed) {
		res := strings.TrimSpace(reParenYear.ReplaceAllString(trimmed, ""))
		if res != "" {
			return res
		}
	}
	// Match trailing year e.g. "Doctor Who 2005"
	reTrailYear := regexp.MustCompile(`(?i)\s+(?:19\d{2}|20\d{2})\s*$`)
	if reTrailYear.MatchString(trimmed) {
		res := strings.TrimSpace(reTrailYear.ReplaceAllString(trimmed, ""))
		if res != "" {
			return res
		}
	}
	return trimmed
}

// ExtractMovieYearAndTitle parses movie name and separates clean title from release year.
func ExtractMovieYearAndTitle(baseName string) (title string, year int) {
	// If the name explicitly matches TV episode notation, it is NOT a movie!
	if _, _, _, isTV := ExtractEpisodeInfo(baseName); isTV {
		return "", 0
	}

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

var partPattern = regexp.MustCompile(`(?i)[._\s\-\[](?:cd|part|pt|disc|disk)[._\s\-]*(\d{1,2})(?:[._\s\-\]]|\z)`)

// ExtractPartInfo checks if a video file has multi-part markers (e.g. CD1, Part 2, Disc 1).
func ExtractPartInfo(name string) int {
	if m := partPattern.FindStringSubmatch(name); len(m) > 1 {
		if p, err := strconv.Atoi(m[1]); err == nil && p > 0 {
			return p
		}
	}
	return 0
}

// IsSampleOrTrailer checks if a file is a sample clip or trailer.
func IsSampleOrTrailer(name string) (bool, string) {
	low := strings.ToLower(name)
	if strings.Contains(low, "sample") {
		return true, "Sample"
	}
	if strings.Contains(low, "trailer") {
		return true, "Trailer"
	}
	return false, ""
}
