package renamer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
)

// ConvertSubtitleToSRT converts any supported subtitle format (VTT, ASS, SSA, TTML, STL, MicroDVD, SAMI) to SRT format.
// It uses github.com/asticode/go-astisub for parsing and writing.
func ConvertSubtitleToSRT(sourcePath, targetPath string) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("zielordner erstellen fehlgeschlagen: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))

	// If source is already SRT, copy it
	if ext == ".srt" {
		return copyFile(sourcePath, targetPath)
	}

	// Try astisub.OpenFile first (natively handles .vtt, .ass, .ssa, .stl, .ttml)
	sub, err := astisub.OpenFile(sourcePath)
	if err != nil {
		// Fallback: Read with encoding normalization (UTF-8, UTF-16, Latin1)
		sub, err = parseSubtitleWithFallback(sourcePath, ext)
	}

	if err != nil {
		return fmt.Errorf("untertitel-konvertierung fehlgeschlagen für %s: %w", filepath.Base(sourcePath), err)
	}

	// Write to target in .srt format
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("erstellen der srt-datei fehlgeschlagen: %w", err)
	}
	defer outFile.Close()

	if err := sub.WriteToSRT(outFile); err != nil {
		return fmt.Errorf("schreiben als srt fehlgeschlagen: %w", err)
	}

	return nil
}

// parseSubtitleWithFallback handles files with non-standard encodings, MicroDVD (.sub), or SAMI (.smi).
func parseSubtitleWithFallback(sourcePath, ext string) (*astisub.Subtitles, error) {
	content, err := readSubtitleSample(sourcePath, 15*1024*1024)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(content)

	switch ext {
	case ".vtt":
		return astisub.ReadFromWebVTT(reader)
	case ".ass", ".ssa":
		return astisub.ReadFromSSA(reader)
	case ".ttml", ".xml":
		return astisub.ReadFromTTML(reader)
	case ".stl":
		return astisub.ReadFromSTL(reader, astisub.STLOptions{})
	case ".sub":
		// MicroDVD format: {100}{250}Hello world
		return parseMicroDVD(content)
	case ".smi":
		// SAMI format: <SYNC Start=1000><P Class=ENCC>Hello
		return parseSAMI(content)
	default:
		// Attempt WebVTT or SSA as fallback
		if strings.Contains(strings.ToUpper(content), "WEBVTT") {
			return astisub.ReadFromWebVTT(strings.NewReader(content))
		}
		if strings.Contains(strings.ToUpper(content), "[SCRIPT INFO]") {
			return astisub.ReadFromSSA(strings.NewReader(content))
		}
		return nil, fmt.Errorf("unbekanntes Untertitelformat: %s", ext)
	}
}

// parseMicroDVD parses MicroDVD format subtitles ({frame_start}{frame_end}text).
func parseMicroDVD(content string) (*astisub.Subtitles, error) {
	fps := 23.976
	lines := strings.Split(content, "\n")
	subs := astisub.NewSubtitles()
	microDvdRegex := regexp.MustCompile(`^\{(\d+)\}\{(\d+)\}(.*)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := microDvdRegex.FindStringSubmatch(line)
		if m != nil {
			startFrame, _ := strconv.Atoi(m[1])
			endFrame, _ := strconv.Atoi(m[2])
			text := m[3]

			// Check if header sets fps, e.g. {1}{1}25.000 or {1}{1}23.976
			if startFrame == 1 && endFrame == 1 {
				if parsedFps, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil && parsedFps > 0 {
					fps = parsedFps
					continue
				}
			}

			startAt := time.Duration(float64(startFrame) / fps * float64(time.Second))
			endAt := time.Duration(float64(endFrame) / fps * float64(time.Second))

			// MicroDVD separates multiple lines with |
			parts := strings.Split(text, "|")
			var subLines []astisub.Line
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					subLines = append(subLines, astisub.Line{
						Items: []astisub.LineItem{
							{Text: p},
						},
					})
				}
			}

			if len(subLines) > 0 {
				subs.Items = append(subs.Items, &astisub.Item{
					Index:   len(subs.Items) + 1,
					StartAt: startAt,
					EndAt:   endAt,
					Lines:   subLines,
				})
			}
		}
	}

	if len(subs.Items) == 0 {
		return nil, fmt.Errorf("keine gültigen MicroDVD-Untertitel gefunden")
	}
	return subs, nil
}

// parseSAMI parses SAMI format subtitles (<SYNC Start=ms><P Class=... >text).
func parseSAMI(content string) (*astisub.Subtitles, error) {
	subs := astisub.NewSubtitles()
	syncRegex := regexp.MustCompile(`(?i)<SYNC\s+Start=(\d+)>`)
	tagStripRegex := regexp.MustCompile(`<[^>]+>`)

	matches := syncRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("keine gültigen SAMI-Synchronisationspunkte gefunden")
	}

	for i := 0; i < len(matches); i++ {
		startMs, err := strconv.Atoi(content[matches[i][2]:matches[i][3]])
		if err != nil {
			continue
		}
		var textChunk string
		if i+1 < len(matches) {
			textChunk = content[matches[i][1]:matches[i+1][0]]
		} else {
			textChunk = content[matches[i][1]:]
		}

		cleanText := tagStripRegex.ReplaceAllString(textChunk, " ")
		cleanText = strings.ReplaceAll(cleanText, "&nbsp;", " ")
		cleanText = strings.TrimSpace(cleanText)
		if cleanText == "" {
			continue
		}

		startAt := time.Duration(startMs) * time.Millisecond
		endAt := startAt + 3*time.Second
		if i+1 < len(matches) {
			if nextMs, err := strconv.Atoi(content[matches[i+1][2]:matches[i+1][3]]); err == nil {
				nextStart := time.Duration(nextMs) * time.Millisecond
				if nextStart > startAt {
					endAt = nextStart
				}
			}
		}

		subs.Items = append(subs.Items, &astisub.Item{
			Index:   len(subs.Items) + 1,
			StartAt: startAt,
			EndAt:   endAt,
			Lines: []astisub.Line{
				{Items: []astisub.LineItem{{Text: cleanText}}},
			},
		})
	}

	if len(subs.Items) == 0 {
		return nil, fmt.Errorf("keine gültigen SAMI-Untertitel gefunden")
	}
	return subs, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
