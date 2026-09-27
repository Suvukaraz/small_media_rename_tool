package renamer

import (
	"testing"
)

func TestParseSubtitle(t *testing.T) {
	testCases := []struct {
		input          string
		expectedLang   string
		expectedForced bool
		expectedSuffix string
	}{
		{"The_Film_Title.ger.srt", "de", false, ".de"},
		{"The_Film_Title.german.forced.srt", "de", true, ".de.forced"},
		{"The_Film_Title.de.forced.srt", "de", true, ".de.forced"},
		{"The_Film_Title.en.srt", "en", false, ".en"},
		{"The_Film_Title.english.srt", "en", false, ".en"},
		{"The_Film_Title.srt", "", false, ""},
		{"The_Film_Title.forced.srt", "", true, ".forced"},
		{"Subs/2_German.forced.srt", "de", true, ".de.forced"},
		{"Subs/English.srt", "en", false, ".en"},
		{"TV_Show.S02E07.german.srt", "de", false, ".de"},
	}

	for _, tc := range testCases {
		info := ParseSubtitle(tc.input)
		if info.LangCode != tc.expectedLang {
			t.Errorf("[%s] Expected lang '%s', got '%s'", tc.input, tc.expectedLang, info.LangCode)
		}
		if info.IsForced != tc.expectedForced {
			t.Errorf("[%s] Expected forced %v, got %v", tc.input, tc.expectedForced, info.IsForced)
		}
		suffix := SubtitleSuffix(info)
		if suffix != tc.expectedSuffix {
			t.Errorf("[%s] Expected suffix '%s', got '%s'", tc.input, tc.expectedSuffix, suffix)
		}
	}
}

func TestTVSubtitleEpisodeDetection(t *testing.T) {
	sub := "TV_Show.S02E07.german.forced.srt"
	info := ParseSubtitle(sub)
	if !info.HasTVEpisode {
		t.Fatalf("Expected TV episode to be detected in subtitle")
	}
	if info.Season != 2 || info.Episode != 7 {
		t.Errorf("Expected S02E07, got S%02dE%02d", info.Season, info.Episode)
	}
	if info.LangCode != "de" || !info.IsForced {
		t.Errorf("Expected de.forced, got lang=%s forced=%v", info.LangCode, info.IsForced)
	}
}
