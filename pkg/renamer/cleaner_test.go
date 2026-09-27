package renamer

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovieParsing(t *testing.T) {
	filename := "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla.mkv"
	ext := ".mkv"
	base := filename[:len(filename)-len(ext)]

	title, year := ExtractMovieYearAndTitle(base)
	expectedTitle := "The Film Title - Electric Buggalo"
	expectedYear := 2009

	if title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, title)
	}
	if year != expectedYear {
		t.Errorf("Expected year %d, got %d", expectedYear, year)
	}

	fullName := fmt.Sprintf("%s (%d)%s", title, year, ext)
	expectedFull := "The Film Title - Electric Buggalo (2009).mkv"
	if fullName != expectedFull {
		t.Errorf("Expected '%s', got '%s'", expectedFull, fullName)
	}

	// Test with dots
	withDots := FormatSeparator(fullName, SeparatorDot)
	expectedDots := "The.Film.Title.-.Electric.Buggalo.(2009).mkv"
	if withDots != expectedDots {
		t.Errorf("Expected '%s', got '%s'", expectedDots, withDots)
	}

	// Test with underscores
	withUnderscores := FormatSeparator(fullName, SeparatorUnderscore)
	expectedUnderscores := "The_Film_Title_-_Electric_Buggalo_(2009).mkv"
	if withUnderscores != expectedUnderscores {
		t.Errorf("Expected '%s', got '%s'", expectedUnderscores, withUnderscores)
	}
}

func TestTVShowFolderParsing(t *testing.T) {
	folder := "TV_Show_title_hevc-bla-bla_bla_2013"
	stripped := StripSceneTags(folder)
	clean := CleanTitle(stripped)
	expected := "TV Show Title"

	if clean != expected {
		t.Errorf("Expected '%s', got '%s'", expected, clean)
	}
}

func TestTVEpisodeParsing(t *testing.T) {
	epFilename := "TV_Shoe_Title-episodename.S02E07.hevc.mp4"
	season, ep, ep2, matched := ExtractEpisodeInfo(epFilename)
	if !matched {
		t.Fatalf("Expected episode to match")
	}
	if season != 2 || ep != 7 || ep2 != 0 {
		t.Errorf("Expected S02E07, got S%02dE%02d", season, ep)
	}

	showTitle := "TV Show Title"
	ext := ".mp4"
	expected := fmt.Sprintf("%s - S%02dE%02d%s", showTitle, season, ep, ext)
	if expected != "TV Show Title - S02E07.mp4" {
		t.Errorf("Expected 'TV Show Title - S02E07.mp4', got '%s'", expected)
	}
}

func TestMoreMovieParsing(t *testing.T) {
	testCases := []struct {
		input       string
		expectTitle string
		expectYear  int
	}{
		{"Inception.2010.1080p.BluRay.x264.mkv", "Inception", 2010},
		{"Avatar (2009) [1080p].mp4", "Avatar", 2009},
		{"The.Matrix.(1999).mkv", "The Matrix", 1999},
		{"Das_Boot_1981_Remastered.mkv", "Das Boot", 1981},
		{"König_der_Löwen-(1994).mkv", "König Der Löwen", 1994},
	}

	for _, tc := range testCases {
		ext := filepath.Ext(tc.input)
		base := strings.TrimSuffix(tc.input, ext)
		title, year := ExtractMovieYearAndTitle(base)
		if title != tc.expectTitle {
			t.Errorf("[%s] Expected title '%s', got '%s'", tc.input, tc.expectTitle, title)
		}
		if year != tc.expectYear {
			t.Errorf("[%s] Expected year %d, got %d", tc.input, tc.expectYear, year)
		}
	}
}

func TestUmlautsAndGerman(t *testing.T) {
	movie := "über_den_dächern_von_nizza-(1955).mkv"
	ext := ".mkv"
	base := movie[:len(movie)-len(ext)]
	title, year := ExtractMovieYearAndTitle(base)
	expectedTitle := "Über Den Dächern Von Nizza"
	if title != expectedTitle {
		t.Errorf("Expected '%s', got '%s'", expectedTitle, title)
	}
	if year != 1955 {
		t.Errorf("Expected year 1955, got %d", year)
	}
}

func TestSpiderManHyphen(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Spider-Man", "Spider-Man"},
		{"X-Men", "X-Men"},
		{"Sci-Fi", "Sci-Fi"},
		{"The_Film_Title-electric_Buggalo", "The Film Title - Electric Buggalo"},
		{"Spider-Man-No_Way_Home", "Spider-Man - No Way Home"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CleanTitle(tt.input)
			if result != tt.expected {
				t.Errorf("CleanTitle(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
