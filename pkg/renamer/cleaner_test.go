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

func TestAdvancedEpisodeParsing(t *testing.T) {
	cases := []struct {
		filename  string
		season    int
		episode   int
		secondEp  int
		matched   bool
	}{
		{"Breaking.Bad.S01E01.720p.mkv", 1, 1, 0, true},
		{"Game.of.Thrones.S08E06.1080p.mkv", 8, 6, 0, true},
		{"Dark.Staffel.1.Folge.02.mkv", 1, 2, 0, true},
		{"Dark.Staffel 2 Folge 05.mkv", 2, 5, 0, true},
		{"Show.St.1.Ep.03.mkv", 1, 3, 0, true},
		{"Show.1x04.mkv", 1, 4, 0, true},
		{"Show.S02E01-E02.mkv", 2, 1, 2, true},
		{"Show.S02E01-02.mkv", 2, 1, 2, true},
		{"Show.S02E01E02.mkv", 2, 1, 2, true},
		{"Show.S02E01.E02.mkv", 2, 1, 2, true},
		{"Inception.2010.1080p.mkv", 0, 0, 0, false},
	}

	for _, c := range cases {
		s, ep, ep2, ok := ExtractEpisodeInfo(c.filename)
		if ok != c.matched {
			t.Errorf("[%s] matched=%v, want %v", c.filename, ok, c.matched)
			continue
		}
		if ok {
			if s != c.season || ep != c.episode || ep2 != c.secondEp {
				t.Errorf("[%s] got S%02dE%02d(second:%d), want S%02dE%02d(second:%d)",
					c.filename, s, ep, ep2, c.season, c.episode, c.secondEp)
			}
		}
	}
}

func TestEpisodeInfoFromPath(t *testing.T) {
	// 1. Standalone episode inside season folder
	p1 := "/media/series/Stranger Things/Season 2/01 - MADMAX.mkv"
	s, ep, _, ok := ExtractEpisodeInfoFromPath(p1)
	if !ok || s != 2 || ep != 1 {
		t.Errorf("ExtractEpisodeInfoFromPath(%q) = (%d, %d, %v), want (2, 1, true)", p1, s, ep, ok)
	}

	// 2. Standalone episode with E03
	p2 := "/media/series/Breaking Bad/Season 1/E03 - And the Bag's in the River.mkv"
	s, ep, _, ok = ExtractEpisodeInfoFromPath(p2)
	if !ok || s != 1 || ep != 3 {
		t.Errorf("ExtractEpisodeInfoFromPath(%q) = (%d, %d, %v), want (1, 3, true)", p2, s, ep, ok)
	}

	// 3. German Staffel folder
	p3 := "/media/series/Dark/Staffel 3/05.mkv"
	s, ep, _, ok = ExtractEpisodeInfoFromPath(p3)
	if !ok || s != 3 || ep != 5 {
		t.Errorf("ExtractEpisodeInfoFromPath(%q) = (%d, %d, %v), want (3, 5, true)", p3, s, ep, ok)
	}
}

func TestExtractShowTitleFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"Breaking.Bad.S01E01.720p.mkv", "Breaking Bad"},
		{"Spider-Man.S01E02.1080p.mkv", "Spider-Man"},
		{"Doctor.Who.2005.S02E03.mkv", "Doctor Who (2005)"},
		{"Dark.Staffel.1.Folge.02.mkv", "Dark"},
	}

	for _, tt := range tests {
		title := ExtractShowTitleFromFilename(tt.filename)
		if title != tt.expected {
			t.Errorf("ExtractShowTitleFromFilename(%q) = %q; want %q", tt.filename, title, tt.expected)
		}
	}
}

func TestEpisodeNotTreatedAsMovie(t *testing.T) {
	episodes := []string{
		"Breaking.Bad.S01E01.2008.720p",
		"Dark.Staffel.1.Folge.01.2017",
		"Doctor.Who.2005.S01E01",
	}

	for _, ep := range episodes {
		title, year := ExtractMovieYearAndTitle(ep)
		if title != "" || year != 0 {
			t.Errorf("Episode %q was classified as movie: title=%q, year=%d", ep, title, year)
		}
	}
}

func TestLeadingYearInTitle(t *testing.T) {
	// 1. Movie parsing tests with titles starting with year
	movieTests := []struct {
		input       string
		expectTitle string
		expectYear  int
	}{
		{"1999 - The best year ever.mkv", "1999 - The Best Year Ever", 0},
		{"1999 - The best year ever (2024).mkv", "1999 - The Best Year Ever", 2024},
		{"1999 - The best year ever (1999).mkv", "1999 - The Best Year Ever", 1999},
		{"1999.The.Best.Year.Ever.1080p.mkv", "1999 The Best Year Ever", 0},
		{"2001 - A Space Odyssey (1968).mkv", "2001 - A Space Odyssey", 1968},
		{"1984 (1984).mkv", "1984", 1984},
		{"1984.1080p.BluRay.mkv", "1984", 0},
		{"1917 (2019).mkv", "1917", 2019},
	}

	for _, tc := range movieTests {
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

	// 2. TV Show episode title extraction with leading year
	tvFilename := "1999 - The Best Year Ever - S01E01.mkv"
	tvTitle := ExtractShowTitleFromFilename(tvFilename)
	expectedTV := "1999 - The Best Year Ever"
	if tvTitle != expectedTV {
		t.Errorf("ExtractShowTitleFromFilename(%q) = %q; want %q", tvFilename, tvTitle, expectedTV)
	}

	// 3. StripYearFromTitle for TV shows (leading year MUST NOT be stripped, trailing year in parens MUST be stripped)
	if stripped := StripYearFromTitle("1999 - The Best Year Ever"); stripped != "1999 - The Best Year Ever" {
		t.Errorf("StripYearFromTitle('1999 - The Best Year Ever') = %q, want '1999 - The Best Year Ever'", stripped)
	}
	if stripped := StripYearFromTitle("1999 - The Best Year Ever (2024)"); stripped != "1999 - The Best Year Ever" {
		t.Errorf("StripYearFromTitle('1999 - The Best Year Ever (2024)') = %q, want '1999 - The Best Year Ever'", stripped)
	}
}

