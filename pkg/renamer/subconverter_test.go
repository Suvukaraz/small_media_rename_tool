package renamer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertVTTToSRT(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_vtt_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vttContent := `WEBVTT

1
00:00:01.000 --> 00:00:04.000
Hallo Welt, dies ist ein Untertitel.
`
	vttFile := filepath.Join(tempDir, "test.vtt")
	_ = os.WriteFile(vttFile, []byte(vttContent), 0644)

	srtFile := filepath.Join(tempDir, "test.srt")
	err = ConvertSubtitleToSRT(vttFile, srtFile)
	if err != nil {
		t.Fatalf("ConvertSubtitleToSRT failed: %v", err)
	}

	data, err := os.ReadFile(srtFile)
	if err != nil {
		t.Fatalf("Failed to read converted srt file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "00:00:01,000 --> 00:00:04,000") {
		t.Errorf("Expected SRT timestamp format with comma, got:\n%s", content)
	}
	if !strings.Contains(content, "Hallo Welt, dies ist ein Untertitel.") {
		t.Errorf("Expected text in converted SRT, got:\n%s", content)
	}
}

func TestConvertASSToSRT(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_ass_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	assContent := `[Script Info]
Title: Test
ScriptType: v4.00+

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:02.50,0:00:05.80,Default,,0,0,0,,Dies ist ein ASS Untertitel.
`
	assFile := filepath.Join(tempDir, "test.ass")
	_ = os.WriteFile(assFile, []byte(assContent), 0644)

	srtFile := filepath.Join(tempDir, "test.srt")
	err = ConvertSubtitleToSRT(assFile, srtFile)
	if err != nil {
		t.Fatalf("ConvertSubtitleToSRT failed for ASS: %v", err)
	}

	data, err := os.ReadFile(srtFile)
	if err != nil {
		t.Fatalf("Failed to read converted srt file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Dies ist ein ASS Untertitel.") {
		t.Errorf("Expected text in converted SRT, got:\n%s", content)
	}
}

func TestConvertMicroDVDToSRT(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_microdvd_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subContent := `{1}{1}25.000
{25}{100}Erste Zeile|Zweite Zeile
{150}{250}Dritte Zeile
`
	subFile := filepath.Join(tempDir, "test.sub")
	_ = os.WriteFile(subFile, []byte(subContent), 0644)

	srtFile := filepath.Join(tempDir, "test.srt")
	err = ConvertSubtitleToSRT(subFile, srtFile)
	if err != nil {
		t.Fatalf("ConvertSubtitleToSRT failed for MicroDVD: %v", err)
	}

	data, err := os.ReadFile(srtFile)
	if err != nil {
		t.Fatalf("Failed to read converted srt file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Erste Zeile") || !strings.Contains(content, "Zweite Zeile") {
		t.Errorf("Expected converted text from MicroDVD, got:\n%s", content)
	}
}

func TestConvertSAMIToSRT(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_sami_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	samiContent := `<SAMI>
<HEAD>
<TITLE>Test</TITLE>
</HEAD>
<BODY>
<SYNC Start=1000>
<P Class=ENCC>Hallo aus SAMI!
<SYNC Start=4000>
<P Class=ENCC>Zweiter Satz.
</BODY>
</SAMI>`

	smiFile := filepath.Join(tempDir, "test.smi")
	_ = os.WriteFile(smiFile, []byte(samiContent), 0644)

	srtFile := filepath.Join(tempDir, "test.srt")
	err = ConvertSubtitleToSRT(smiFile, srtFile)
	if err != nil {
		t.Fatalf("ConvertSubtitleToSRT failed for SAMI: %v", err)
	}

	data, err := os.ReadFile(srtFile)
	if err != nil {
		t.Fatalf("Failed to read converted srt file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Hallo aus SAMI!") {
		t.Errorf("Expected converted text from SAMI, got:\n%s", content)
	}
}
