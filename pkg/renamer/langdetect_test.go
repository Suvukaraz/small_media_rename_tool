package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSubtitleLanguage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "langdetect_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name       string
		content    string
		expectLang string
	}{
		{
			name: "German Standard",
			content: "1\n00:00:01,000 --> 00:00:03,000\nGuten Tag, wir müssen jetzt gehen und das sehen.\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nIch habe keine Ahnung, warum sie nicht hier sind.",
			expectLang: "de",
		},
		{
			name: "German with Umlauts",
			content: "1\n00:00:01,000 --> 00:00:03,000\nÜber den Dächern von Nizza, für schöne Träume.\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nKönnen wir das bitte morgen machen?",
			expectLang: "de",
		},
		{
			name: "English Standard",
			content: "1\n00:00:01,000 --> 00:00:03,000\nHello, how are you doing? What are you looking for?\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nI don't think they're going to come back today.",
			expectLang: "en",
		},
		{
			name: "Spanish",
			content: "1\n00:00:01,000 --> 00:00:03,000\nHola señor, cómo estás? Gracias por todo.\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nEsto es muy bueno, pero no puedo hacer nada ahora.",
			expectLang: "other", // non-de/en
		},
		{
			name: "French",
			content: "1\n00:00:01,000 --> 00:00:03,000\nBonjour monsieur, où allez-vous? Merci beaucoup.\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nC'est tout ce que nous pouvons faire avec vous.",
			expectLang: "other", // non-de/en
		},
		{
			name: "Russian Cyrillic",
			content: "1\n00:00:01,000 --> 00:00:03,000\nПривет, как твои дела? Спасибо за помощь.\n\n" +
				"2\n00:00:04,000 --> 00:00:06,000\nМы должны идти туда прямо сейчас.",
			expectLang: "ru",
		},
		{
			name:       "Japanese CJK",
			content:    "1\n00:00:01,000 --> 00:00:03,000\nこんにちは、お元気ですか？\n2\n00:00:04,000 --> 00:00:06,000\n東京に行きます。",
			expectLang: "cjk",
		},
	}

	for _, tc := range testCases {
		filePath := filepath.Join(tempDir, tc.name+".srt")
		err := os.WriteFile(filePath, []byte(tc.content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", tc.name, err)
		}

		res := DetectSubtitleLanguage(filePath, SubtitleInfo{})
		if res.Language != tc.expectLang {
			t.Errorf("[%s] Expected language '%s', got '%s' (details: %s, conf: %.2f)",
				tc.name, tc.expectLang, res.Language, res.Details, res.Confidence)
		}
	}
}

func TestIdxLanguageDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "idx_lang_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	idxDe := filepath.Join(tempDir, "movie_de.idx")
	_ = os.WriteFile(idxDe, []byte("# VobSub index file\nid: de, index: 0\n"), 0644)
	resDe := DetectSubtitleLanguage(idxDe, SubtitleInfo{})
	if resDe.Language != "de" {
		t.Errorf("Expected 'de' for IDX, got '%s'", resDe.Language)
	}

	idxFr := filepath.Join(tempDir, "movie_fr.idx")
	_ = os.WriteFile(idxFr, []byte("# VobSub index file\nid: fr, index: 0\n"), 0644)
	resFr := DetectSubtitleLanguage(idxFr, SubtitleInfo{})
	if resFr.Language == "de" || resFr.Language == "en" {
		t.Errorf("Expected non-de/en for French IDX, got '%s'", resFr.Language)
	}
}
