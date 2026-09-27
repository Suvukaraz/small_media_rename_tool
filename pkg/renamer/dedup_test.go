package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeSubtitleContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "SRT with timestamps and numbers",
			input: `1
00:00:01,000 --> 00:00:03,000
Hello World

2
00:00:05,000 --> 00:00:07,000
Goodbye World
`,
			expected: "hello world goodbye world",
		},
		{
			name:     "With formatting tags",
			input:    "1\n00:00:01,000 --> 00:00:03,000\n<b>Hello</b> <i>World</i>\n",
			expected: "hello world",
		},
		{
			name:     "With ASS style tags",
			input:    "1\n00:00:01,000 --> 00:00:03,000\n{\\an8}Hello World\n",
			expected: "hello world",
		},
		{
			name:     "With BOM",
			input:    "\xef\xbb\xbf1\n00:00:01,000 --> 00:00:03,000\nHallo Welt\n",
			expected: "hallo welt",
		},
		{
			name:     "WebVTT header",
			input:    "WEBVTT\n\n00:00:01.000 --> 00:00:03.000\nTest subtitle\n",
			expected: "test subtitle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSubtitleContent(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSubtitleContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	// Same content should produce same hash
	hash1 := ContentHash("hello world")
	hash2 := ContentHash("hello world")
	if hash1 != hash2 {
		t.Errorf("Same content should produce same hash")
	}

	// Different content should produce different hash
	hash3 := ContentHash("hello world!")
	if hash1 == hash3 {
		t.Errorf("Different content should produce different hash")
	}
}

func TestDeduplicateSubtitles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_dedup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create 3 subtitle files: 2 identical German, 1 unique English
	srtContentA := "1\n00:00:01,000 --> 00:00:03,000\nWillkommen in der Welt.\n\n2\n00:00:05,000 --> 00:00:07,000\nDanke schön.\n"
	// Same content but with different formatting (extra tags, different timestamps slightly)
	srtContentB := "1\n00:00:01,000 --> 00:00:03,500\n<b>Willkommen</b> in der Welt.\n\n2\n00:00:05,000 --> 00:00:07,500\nDanke schön.\n"
	srtContentEn := "1\n00:00:01,000 --> 00:00:03,000\nWelcome to the world.\n\n2\n00:00:05,000 --> 00:00:07,000\nThank you.\n"

	fileA := filepath.Join(tempDir, "Movie.2020.de.srt")
	fileB := filepath.Join(tempDir, "Movie.2020.German.srt")
	fileEn := filepath.Join(tempDir, "Movie.2020.en.srt")

	_ = os.WriteFile(fileA, []byte(srtContentA), 0644)
	_ = os.WriteFile(fileB, []byte(srtContentB), 0644)
	_ = os.WriteFile(fileEn, []byte(srtContentEn), 0644)

	subs := []SubtitleInfo{
		{OriginalPath: fileA, Filename: "Movie.2020.de.srt", Extension: ".srt", LangCode: "de"},
		{OriginalPath: fileB, Filename: "Movie.2020.German.srt", Extension: ".srt", LangCode: "de"},
		{OriginalPath: fileEn, Filename: "Movie.2020.en.srt", Extension: ".srt", LangCode: "en"},
	}

	keep, dups := DeduplicateSubtitles(subs)

	// Should keep 2 (1 de + 1 en) and have 1 duplicate
	if len(keep) != 2 {
		t.Errorf("Expected 2 kept subtitles, got %d", len(keep))
	}
	if len(dups) != 1 {
		t.Errorf("Expected 1 duplicate, got %d", len(dups))
	}

	// The English sub should definitely be kept
	enKept := false
	for _, k := range keep {
		if k.LangCode == "en" {
			enKept = true
		}
	}
	if !enKept {
		t.Errorf("English subtitle should be kept")
	}

	// The duplicate should be German
	if len(dups) > 0 && dups[0].Duplicate.LangCode != "de" {
		t.Errorf("Duplicate should be German, got %s", dups[0].Duplicate.LangCode)
	}
}

func TestDeduplicateNoDuplicates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_dedup_nodup_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srtDe := "1\n00:00:01,000 --> 00:00:03,000\nHallo Welt.\n"
	srtEn := "1\n00:00:01,000 --> 00:00:03,000\nHello World.\n"

	fileDe := filepath.Join(tempDir, "movie.de.srt")
	fileEn := filepath.Join(tempDir, "movie.en.srt")

	_ = os.WriteFile(fileDe, []byte(srtDe), 0644)
	_ = os.WriteFile(fileEn, []byte(srtEn), 0644)

	subs := []SubtitleInfo{
		{OriginalPath: fileDe, Filename: "movie.de.srt", Extension: ".srt", LangCode: "de"},
		{OriginalPath: fileEn, Filename: "movie.en.srt", Extension: ".srt", LangCode: "en"},
	}

	keep, dups := DeduplicateSubtitles(subs)

	if len(keep) != 2 {
		t.Errorf("Expected 2 kept, got %d", len(keep))
	}
	if len(dups) != 0 {
		t.Errorf("Expected 0 duplicates, got %d", len(dups))
	}
}

func TestDeduplicateKeepsLargestFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_dedup_largest_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Same dialog text, but file B has extra formatting making it larger
	srtSmall := "1\n00:00:01,000 --> 00:00:03,000\nHallo Welt.\n"
	srtLarge := "1\n00:00:01,000 --> 00:00:03,000\n<b><i>Hallo</i></b> Welt.\n\n2\n00:00:04,000 --> 00:00:04,500\n \n"

	fileSmall := filepath.Join(tempDir, "movie.de.srt")
	fileLarge := filepath.Join(tempDir, "movie.deutsch.srt")

	_ = os.WriteFile(fileSmall, []byte(srtSmall), 0644)
	_ = os.WriteFile(fileLarge, []byte(srtLarge), 0644)

	subs := []SubtitleInfo{
		{OriginalPath: fileSmall, Filename: "movie.de.srt", Extension: ".srt", LangCode: "de"},
		{OriginalPath: fileLarge, Filename: "movie.deutsch.srt", Extension: ".srt", LangCode: "de"},
	}

	keep, dups := DeduplicateSubtitles(subs)

	if len(keep) != 1 {
		t.Errorf("Expected 1 kept, got %d", len(keep))
	}
	if len(dups) != 1 {
		t.Errorf("Expected 1 duplicate, got %d", len(dups))
	}

	// The larger file should be kept
	if len(keep) > 0 && keep[0].OriginalPath != fileLarge {
		t.Errorf("Expected larger file %s to be kept, but got %s", fileLarge, keep[0].OriginalPath)
	}
}

func TestDeduplicateSingleFile(t *testing.T) {
	subs := []SubtitleInfo{
		{OriginalPath: "/fake/path.srt", Filename: "path.srt", Extension: ".srt", LangCode: "de"},
	}

	keep, dups := DeduplicateSubtitles(subs)
	if len(keep) != 1 {
		t.Errorf("Expected 1 kept, got %d", len(keep))
	}
	if len(dups) != 0 {
		t.Errorf("Expected 0 duplicates, got %d", len(dups))
	}
}

func TestDeduplicateEmptyList(t *testing.T) {
	keep, dups := DeduplicateSubtitles(nil)
	if len(keep) != 0 {
		t.Errorf("Expected 0 kept, got %d", len(keep))
	}
	if len(dups) != 0 {
		t.Errorf("Expected 0 duplicates, got %d", len(dups))
	}
}
