package renamer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovieFolderEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_movie_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla")
	err = os.MkdirAll(movieFolder, 0755)
	if err != nil {
		t.Fatalf("Failed to create movie folder: %v", err)
	}

	movieFile := filepath.Join(movieFolder, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla.mkv")
	err = os.WriteFile(movieFile, []byte("fake video"), 0644)
	if err != nil {
		t.Fatalf("Failed to write movie file: %v", err)
	}

	subFile := filepath.Join(movieFolder, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla.de.srt")
	_ = os.WriteFile(subFile, []byte("fake sub"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(movieFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(res.Actions) == 0 {
		t.Fatalf("Expected actions, got none")
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "The Film Title - Electric Buggalo (2009)")
	expectedFile := filepath.Join(expectedFolder, "The Film Title - Electric Buggalo (2009).mkv")
	expectedSub := filepath.Join(expectedFolder, "The Film Title - Electric Buggalo (2009).de.srt")

	if _, err := os.Stat(expectedFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", expectedFolder)
	}
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedFile)
	}
	if _, err := os.Stat(expectedSub); os.IsNotExist(err) {
		t.Errorf("Expected subtitle %s does not exist", expectedSub)
	}
}

func TestMultipleMoviesInOneFolder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_multimovie_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create folder containing multiple movies and subtitles
	m1 := filepath.Join(tempDir, "Gladiator_2000_1080p.mkv")
	m1Sub := filepath.Join(tempDir, "Gladiator_2000_1080p.ger.forced.srt")
	m2 := filepath.Join(tempDir, "The_Matrix_1999.mkv")
	m2Sub := filepath.Join(tempDir, "The_Matrix_1999.en.srt")

	_ = os.WriteFile(m1, []byte("m1"), 0644)
	_ = os.WriteFile(m1Sub, []byte("m1sub"), 0644)
	_ = os.WriteFile(m2, []byte("m2"), 0644)
	_ = os.WriteFile(m2Sub, []byte("m2sub"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	// For EACH movie, a dedicated folder must be created!
	gladiatorFolder := filepath.Join(tempDir, "Gladiator (2000)")
	gladiatorVideo := filepath.Join(gladiatorFolder, "Gladiator (2000).mkv")
	gladiatorSub := filepath.Join(gladiatorFolder, "Gladiator (2000).de.forced.srt")

	matrixFolder := filepath.Join(tempDir, "The Matrix (1999)")
	matrixVideo := filepath.Join(matrixFolder, "The Matrix (1999).mkv")
	matrixSub := filepath.Join(matrixFolder, "The Matrix (1999).en.srt")

	if _, err := os.Stat(gladiatorFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", gladiatorFolder)
	}
	if _, err := os.Stat(gladiatorVideo); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", gladiatorVideo)
	}
	if _, err := os.Stat(gladiatorSub); os.IsNotExist(err) {
		t.Errorf("Expected sub %s does not exist", gladiatorSub)
	}

	if _, err := os.Stat(matrixFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", matrixFolder)
	}
	if _, err := os.Stat(matrixVideo); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", matrixVideo)
	}
	if _, err := os.Stat(matrixSub); os.IsNotExist(err) {
		t.Errorf("Expected sub %s does not exist", matrixSub)
	}
}

func TestMovieWithSubsSubfolder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_subsdir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "Inception_2010_BluRay")
	subsFolder := filepath.Join(movieFolder, "Subs")
	_ = os.MkdirAll(subsFolder, 0755)

	mFile := filepath.Join(movieFolder, "Inception_2010_BluRay.mkv")
	sub1 := filepath.Join(subsFolder, "2_German.forced.srt")
	sub2 := filepath.Join(subsFolder, "English.srt")

	_ = os.WriteFile(mFile, []byte("movie"), 0644)
	_ = os.WriteFile(sub1, []byte("sub1"), 0644)
	_ = os.WriteFile(sub2, []byte("sub2"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(movieFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetFolder := filepath.Join(tempDir, "Inception (2010)")
	targetMovie := filepath.Join(targetFolder, "Inception (2010).mkv")
	targetSub1 := filepath.Join(targetFolder, "Inception (2010).de.forced.srt")
	targetSub2 := filepath.Join(targetFolder, "Inception (2010).en.srt")

	if _, err := os.Stat(targetMovie); os.IsNotExist(err) {
		t.Errorf("Expected movie %s does not exist", targetMovie)
	}
	if _, err := os.Stat(targetSub1); os.IsNotExist(err) {
		t.Errorf("Expected sub1 %s does not exist", targetSub1)
	}
	if _, err := os.Stat(targetSub2); os.IsNotExist(err) {
		t.Errorf("Expected sub2 %s does not exist", targetSub2)
	}

	// Verify Subs/ directory was cleaned up
	oldSubs := filepath.Join(targetFolder, "Subs")
	if _, err := os.Stat(oldSubs); !os.IsNotExist(err) {
		t.Errorf("Subs/ directory should have been removed, but still exists")
	}
}

func TestTVShowMultipleSeasonsEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_tv_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showFolder := filepath.Join(tempDir, "TV_Show_title_hevc-bla-bla_bla_2013")
	err = os.MkdirAll(showFolder, 0755)
	if err != nil {
		t.Fatalf("Failed to create show folder: %v", err)
	}

	ep1 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S01E01.hevc.mp4")
	ep1Sub := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S01E01.hevc.de.srt")
	ep2 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S02E07.hevc.mp4")
	ep2Sub := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S02E07.hevc.german.forced.srt")

	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep1Sub, []byte("ep1sub"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)
	_ = os.WriteFile(ep2Sub, []byte("ep2sub"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(showFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "TV Show Title")
	expectedS01 := filepath.Join(expectedFolder, "S01", "TV Show Title - S01E01.mp4")
	expectedS01Sub := filepath.Join(expectedFolder, "S01", "TV Show Title - S01E01.de.srt")
	expectedS02 := filepath.Join(expectedFolder, "S02", "TV Show Title - S02E07.mp4")
	expectedS02Sub := filepath.Join(expectedFolder, "S02", "TV Show Title - S02E07.de.forced.srt")

	if _, err := os.Stat(expectedFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", expectedFolder)
	}
	if _, err := os.Stat(expectedS01); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS01)
	}
	if _, err := os.Stat(expectedS01Sub); os.IsNotExist(err) {
		t.Errorf("Expected subtitle %s does not exist", expectedS01Sub)
	}
	if _, err := os.Stat(expectedS02); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS02)
	}
	if _, err := os.Stat(expectedS02Sub); os.IsNotExist(err) {
		t.Errorf("Expected subtitle %s does not exist", expectedS02Sub)
	}
}

func TestTVShowSingleSeasonEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_tv_single_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showFolder := filepath.Join(tempDir, "Single_Season_Show_720p")
	err = os.MkdirAll(showFolder, 0755)
	if err != nil {
		t.Fatalf("Failed to create show folder: %v", err)
	}

	ep1 := filepath.Join(showFolder, "Single_Season_Show.S01E01.mp4")
	ep2 := filepath.Join(showFolder, "Single_Season_Show.S01E02.mp4")
	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(showFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "Single Season Show")
	expectedS01Ep1 := filepath.Join(expectedFolder, "Single Season Show - S01E01.mp4")
	expectedS01Ep2 := filepath.Join(expectedFolder, "Single Season Show - S01E02.mp4")

	if _, err := os.Stat(expectedFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", expectedFolder)
	}
	if _, err := os.Stat(expectedS01Ep1); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS01Ep1)
	}
	if _, err := os.Stat(expectedS01Ep2); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS01Ep2)
	}
}

func TestTVShowNoSpaces(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_tv_nospaces_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showFolder := filepath.Join(tempDir, "TV_Show_title_hevc-bla-bla_bla_2013")
	_ = os.MkdirAll(showFolder, 0755)
	ep1 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S01E01.hevc.mp4")
	ep2 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S02E07.hevc.mp4")
	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)

	scanner := NewScanner(SeparatorDot)
	res, err := scanner.Scan(showFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "TV.Show.Title")
	expectedS01 := filepath.Join(expectedFolder, "S01", "TV.Show.Title.-.S01E01.mp4")
	expectedS02 := filepath.Join(expectedFolder, "S02", "TV.Show.Title.-.S02E07.mp4")

	if _, err := os.Stat(expectedFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", expectedFolder)
	}
	if _, err := os.Stat(expectedS01); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS01)
	}
	if _, err := os.Stat(expectedS02); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS02)
	}
}

func TestTVShowUnderscore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_tv_underscore_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showFolder := filepath.Join(tempDir, "TV_Show_title_hevc-bla-bla_bla_2013")
	_ = os.MkdirAll(showFolder, 0755)
	ep1 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S01E01.hevc.mp4")
	ep2 := filepath.Join(showFolder, "TV_Shoe_Title-episodename.S02E07.hevc.mp4")
	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)

	scanner := NewScanner(SeparatorUnderscore)
	res, err := scanner.Scan(showFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "TV_Show_Title")
	expectedS01 := filepath.Join(expectedFolder, "S01", "TV_Show_Title_-_S01E01.mp4")
	expectedS02 := filepath.Join(expectedFolder, "S02", "TV_Show_Title_-_S02E07.mp4")

	if _, err := os.Stat(expectedFolder); os.IsNotExist(err) {
		t.Errorf("Expected folder %s does not exist", expectedFolder)
	}
	if _, err := os.Stat(expectedS01); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS01)
	}
	if _, err := os.Stat(expectedS02); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedS02)
	}
}

func TestSubtitleLanguageFilteringAndContentCheckMovie(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_subfilter_movie_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "Inception.2010.1080p")
	_ = os.MkdirAll(movieFolder, 0755)

	movieFile := filepath.Join(movieFolder, "Inception.2010.1080p.mkv")
	_ = os.WriteFile(movieFile, []byte("movie content"), 0644)

	// 1. German sub WITHOUT language in filename -> content check must detect German!
	germanSubContent := `1
00:00:01,000 --> 00:00:04,000
Guten Tag, wie geht es Ihnen heute? Wir haben hier kein Problem.
`
	subGerman := filepath.Join(movieFolder, "Inception.2010.1080p.srt")
	_ = os.WriteFile(subGerman, []byte(germanSubContent), 0644)

	// 2. English sub
	englishSubContent := `1
00:00:01,000 --> 00:00:04,000
Hello, how are you today? We have no problem here, that's what I said.
`
	subEnglish := filepath.Join(movieFolder, "Inception.2010.1080p.en.srt")
	_ = os.WriteFile(subEnglish, []byte(englishSubContent), 0644)

	// 3. Spanish sub
	spanishSubContent := `1
00:00:01,000 --> 00:00:04,000
Hola señor, cómo estás hoy? Gracias por venir con nosotros ahora.
`
	subSpanish := filepath.Join(movieFolder, "Inception.2010.1080p.es.srt")
	_ = os.WriteFile(subSpanish, []byte(spanishSubContent), 0644)

	// 4. French sub
	frenchSubContent := `1
00:00:01,000 --> 00:00:04,000
Bonjour monsieur, comment allez-vous? Merci beaucoup pour votre aide aujourd'hui avec nous.
`
	subFrench := filepath.Join(movieFolder, "Inception.2010.1080p.french.srt")
	_ = os.WriteFile(subFrench, []byte(frenchSubContent), 0644)

	// 5. Russian / Cyrillic sub (no lang in name)
	russianSubContent := `1
00:00:01,000 --> 00:00:04,000
Привет, как дела сегодня? Мы здесь ничего не знаем об этом фильме.
`
	subRussian := filepath.Join(movieFolder, "Inception.2010.1080p.custom_track.srt")
	_ = os.WriteFile(subRussian, []byte(russianSubContent), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(movieFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify planned actions:
	// We expect 3 delete actions (Spanish, French, Russian)
	var deleteActions []Action
	var moveSubActions []Action
	for _, act := range res.Actions {
		if act.Type == ActionDeleteFile {
			deleteActions = append(deleteActions, act)
		} else if act.Type == ActionMoveFile && filepath.Ext(act.TargetPath) == ".srt" {
			moveSubActions = append(moveSubActions, act)
		}
	}

	if len(deleteActions) != 3 {
		t.Fatalf("Expected 3 delete actions for non-DE/EN subtitles, got %d: %+v", len(deleteActions), deleteActions)
	}
	if len(moveSubActions) != 2 {
		t.Fatalf("Expected 2 move subtitle actions for DE/EN subtitles, got %d: %+v", len(moveSubActions), moveSubActions)
	}

	// Execute actions
	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetFolder := filepath.Join(tempDir, "Inception (2010)")
	expectedMovie := filepath.Join(targetFolder, "Inception (2010).mkv")
	expectedGermanSub := filepath.Join(targetFolder, "Inception (2010).de.srt")
	expectedEnglishSub := filepath.Join(targetFolder, "Inception (2010).en.srt")

	if _, err := os.Stat(expectedMovie); os.IsNotExist(err) {
		t.Errorf("Expected movie %s does not exist", expectedMovie)
	}
	if _, err := os.Stat(expectedGermanSub); os.IsNotExist(err) {
		t.Errorf("Expected German subtitle %s does not exist", expectedGermanSub)
	}
	if _, err := os.Stat(expectedEnglishSub); os.IsNotExist(err) {
		t.Errorf("Expected English subtitle %s does not exist", expectedEnglishSub)
	}

	// Verify unwanted subtitles were completely deleted from disk
	if _, err := os.Stat(subSpanish); !os.IsNotExist(err) {
		t.Errorf("Spanish subtitle %s was not deleted!", subSpanish)
	}
	if _, err := os.Stat(subFrench); !os.IsNotExist(err) {
		t.Errorf("French subtitle %s was not deleted!", subFrench)
	}
	if _, err := os.Stat(subRussian); !os.IsNotExist(err) {
		t.Errorf("Russian subtitle %s was not deleted!", subRussian)
	}
}

func TestSubtitleLanguageFilteringTVShow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_subfilter_tv_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showFolder := filepath.Join(tempDir, "Dark.S01-S02.German.1080p")
	_ = os.MkdirAll(showFolder, 0755)

	ep1 := filepath.Join(showFolder, "Dark.S01E01.mkv")
	ep2 := filepath.Join(showFolder, "Dark.S02E01.mkv")
	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)

	// S01E01 German sub
	s01German := filepath.Join(showFolder, "Dark.S01E01.de.srt")
	_ = os.WriteFile(s01German, []byte("1\n00:00:01,000 --> 00:00:03,000\nWo ist der Junge? Wir müssen ihn finden.\n"), 0644)

	// S01E01 Spanish sub (should be deleted)
	s01Spanish := filepath.Join(showFolder, "Dark.S01E01.es.srt")
	_ = os.WriteFile(s01Spanish, []byte("1\n00:00:01,000 --> 00:00:03,000\nDónde está el niño? Tenemos que encontrarlo por favor.\n"), 0644)

	// S02E01 English sub
	s02English := filepath.Join(showFolder, "Dark.S02E01.en.srt")
	_ = os.WriteFile(s02English, []byte("1\n00:00:01,000 --> 00:00:03,000\nWhere is the boy? We have to find him now.\n"), 0644)

	// S02E01 Russian sub (should be deleted)
	s02Russian := filepath.Join(showFolder, "Dark.S02E01.ru.srt")
	_ = os.WriteFile(s02Russian, []byte("1\n00:00:01,000 --> 00:00:03,000\nГде этот мальчик? Мы должны найти его сейчас.\n"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(showFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetFolder := filepath.Join(tempDir, "Dark")
	expectedS01Sub := filepath.Join(targetFolder, "S01", "Dark - S01E01.de.srt")
	expectedS02Sub := filepath.Join(targetFolder, "S02", "Dark - S02E01.en.srt")

	if _, err := os.Stat(expectedS01Sub); os.IsNotExist(err) {
		t.Errorf("Expected German subtitle %s does not exist", expectedS01Sub)
	}
	if _, err := os.Stat(expectedS02Sub); os.IsNotExist(err) {
		t.Errorf("Expected English subtitle %s does not exist", expectedS02Sub)
	}

	// Verify unwanted subtitles deleted
	if _, err := os.Stat(s01Spanish); !os.IsNotExist(err) {
		t.Errorf("Spanish subtitle %s was not deleted!", s01Spanish)
	}
	if _, err := os.Stat(s02Russian); !os.IsNotExist(err) {
		t.Errorf("Russian subtitle %s was not deleted!", s02Russian)
	}
}

func TestSubtitleLanguageFilteringLooseFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_subfilter_loose_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloads := filepath.Join(tempDir, "Downloads")
	_ = os.MkdirAll(downloads, 0755)

	movieFile := filepath.Join(downloads, "The.Matrix.1999.mkv")
	_ = os.WriteFile(movieFile, []byte("matrix"), 0644)

	deSub := filepath.Join(downloads, "The.Matrix.1999.de.srt")
	_ = os.WriteFile(deSub, []byte("1\n00:00:01,000 --> 00:00:03,000\nWillkommen in der realen Welt, mein Freund.\n"), 0644)

	itSub := filepath.Join(downloads, "The.Matrix.1999.it.srt")
	_ = os.WriteFile(itSub, []byte("1\n00:00:01,000 --> 00:00:03,000\nBenvenuto nel mondo reale, grazie a tutti per questo.\n"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(downloads)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetFolder := filepath.Join(downloads, "The Matrix (1999)")
	expectedMovie := filepath.Join(targetFolder, "The Matrix (1999).mkv")
	expectedDeSub := filepath.Join(targetFolder, "The Matrix (1999).de.srt")

	if _, err := os.Stat(expectedMovie); os.IsNotExist(err) {
		t.Errorf("Expected movie %s does not exist", expectedMovie)
	}
	if _, err := os.Stat(expectedDeSub); os.IsNotExist(err) {
		t.Errorf("Expected German sub %s does not exist", expectedDeSub)
	}
	if _, err := os.Stat(itSub); !os.IsNotExist(err) {
		t.Errorf("Italian sub %s was not deleted!", itSub)
	}
}

func TestSubtitleConversionEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_subconv_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "Inception_2010_1080p")
	_ = os.MkdirAll(movieFolder, 0755)

	videoFile := filepath.Join(movieFolder, "Inception_2010_1080p.mkv")
	_ = os.WriteFile(videoFile, []byte("fake video content"), 0644)

	// German VTT subtitle
	vttContent := "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nWas ist der widerstandsfähigste Parasit? Ein Gedanke.\n"
	deVtt := filepath.Join(movieFolder, "Inception_2010_1080p.de.vtt")
	_ = os.WriteFile(deVtt, []byte(vttContent), 0644)

	// English ASS subtitle
	assContent := `[Script Info]
Title: English Subtitle
ScriptType: v4.00+

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:04.00,Default,,0,0,0,,What is the most resilient parasite? An idea.
`
	enAss := filepath.Join(movieFolder, "Inception_2010_1080p.en.ass")
	_ = os.WriteFile(enAss, []byte(assContent), 0644)

	// Spanish VTT subtitle (should be deleted because only de and en are kept)
	esVttContent := "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\n¿Cuál es el parásito más resistente? Una idea.\n"
	esVtt := filepath.Join(movieFolder, "Inception_2010_1080p.es.vtt")
	_ = os.WriteFile(esVtt, []byte(esVttContent), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(movieFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	var convertDeFound, convertEnFound, deleteEsFound bool
	for _, act := range res.Actions {
		if act.Type == ActionConvertSubtitle && filepath.Base(act.TargetPath) == "Inception (2010).de.srt" {
			convertDeFound = true
		}
		if act.Type == ActionConvertSubtitle && filepath.Base(act.TargetPath) == "Inception (2010).en.srt" {
			convertEnFound = true
		}
		if act.Type == ActionDeleteFile && filepath.Base(act.SourcePath) == "Inception_2010_1080p.es.vtt" {
			deleteEsFound = true
		}
	}

	if !convertDeFound {
		t.Errorf("Expected ActionConvertSubtitle for German VTT to Inception (2010).de.srt")
	}
	if !convertEnFound {
		t.Errorf("Expected ActionConvertSubtitle for English ASS to Inception (2010).en.srt")
	}
	if !deleteEsFound {
		t.Errorf("Expected ActionDeleteFile for Spanish VTT")
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedFolder := filepath.Join(tempDir, "Inception (2010)")
	expectedVideo := filepath.Join(expectedFolder, "Inception (2010).mkv")
	expectedDeSrt := filepath.Join(expectedFolder, "Inception (2010).de.srt")
	expectedEnSrt := filepath.Join(expectedFolder, "Inception (2010).en.srt")

	if _, err := os.Stat(expectedVideo); os.IsNotExist(err) {
		t.Errorf("Expected video %s does not exist", expectedVideo)
	}
	if _, err := os.Stat(expectedDeSrt); os.IsNotExist(err) {
		t.Errorf("Expected converted German SRT %s does not exist", expectedDeSrt)
	} else {
		content, _ := os.ReadFile(expectedDeSrt)
		if !strings.Contains(string(content), "Parasit") {
			t.Errorf("Expected converted German SRT to contain 'Parasit', got: %s", string(content))
		}
	}

	if _, err := os.Stat(expectedEnSrt); os.IsNotExist(err) {
		t.Errorf("Expected converted English SRT %s does not exist", expectedEnSrt)
	} else {
		content, _ := os.ReadFile(expectedEnSrt)
		if !strings.Contains(string(content), "parasite") {
			t.Errorf("Expected converted English SRT to contain 'parasite', got: %s", string(content))
		}
	}

	// Verify original non-srt files and deleted foreign subtitle no longer exist
	if _, err := os.Stat(deVtt); !os.IsNotExist(err) {
		t.Errorf("Original German VTT %s should have been removed", deVtt)
	}
	if _, err := os.Stat(enAss); !os.IsNotExist(err) {
		t.Errorf("Original English ASS %s should have been removed", enAss)
	}
	if _, err := os.Stat(esVtt); !os.IsNotExist(err) {
		t.Errorf("Foreign Spanish VTT %s should have been removed", esVtt)
	}
}

func TestTVShowSubtitleConversionEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_tvsubconv_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showDir := filepath.Join(tempDir, "Breaking_Bad_720p")
	_ = os.MkdirAll(showDir, 0755)

	ep1 := filepath.Join(showDir, "Breaking.Bad.S01E01.720p.mkv")
	_ = os.WriteFile(ep1, []byte("fake video"), 0644)

	// MicroDVD format (.sub) German subtitle
	microDvdContent := "{1}{50}Pilotfolge\n{60}{120}Mein Name ist Walter Hartwell White.\n"
	subFile := filepath.Join(showDir, "Breaking.Bad.S01E01.ger.sub")
	_ = os.WriteFile(subFile, []byte(microDvdContent), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(showDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	var convertFound bool
	for _, act := range res.Actions {
		if act.Type == ActionConvertSubtitle && filepath.Base(act.TargetPath) == "Breaking Bad - S01E01.de.srt" {
			convertFound = true
		}
	}
	if !convertFound {
		t.Errorf("Expected ActionConvertSubtitle for Breaking Bad - S01E01.de.srt")
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetShow := filepath.Join(tempDir, "Breaking Bad")
	expectedSrt := filepath.Join(targetShow, "Breaking Bad - S01E01.de.srt")
	if _, err := os.Stat(expectedSrt); os.IsNotExist(err) {
		t.Errorf("Expected converted SRT %s does not exist", expectedSrt)
	} else {
		content, _ := os.ReadFile(expectedSrt)
		if !strings.Contains(string(content), "Walter Hartwell White") {
			t.Errorf("Expected SRT content to contain 'Walter Hartwell White', got: %s", string(content))
		}
	}

	if _, err := os.Stat(subFile); !os.IsNotExist(err) {
		t.Errorf("Original .sub file %s was not deleted", subFile)
	}
}

func TestMonsterLizzieBorden(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_lizzie_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	serienDir := filepath.Join(tempDir, "serien")
	showDir := filepath.Join(serienDir, "Monster - The Lizzie Borden Story (2026)")
	_ = os.MkdirAll(showDir, 0755)

	// Episode files
	ep1 := filepath.Join(showDir, "Monster.The.Lizzie.Borden.Story.S01E01.720p.mkv")
	ep2 := filepath.Join(showDir, "Monster.The.Lizzie.Borden.Story.S01E02.720p.mkv")
	_ = os.WriteFile(ep1, []byte("ep1"), 0644)
	_ = os.WriteFile(ep2, []byte("ep2"), 0644)

	scanner := NewScanner(SeparatorSpace)

	// Case A: scanning serenDir via ScanPaths("", serienDir)
	resA, err := scanner.ScanPaths("", serienDir)
	if err != nil {
		t.Fatalf("ScanPaths failed: %v", err)
	}
	t.Logf("Case A Actions (%d):", len(resA.Actions))
	for _, a := range resA.Actions {
		t.Logf("  Action: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}

	// Case B: scanning showDir directly via ScanPaths("", showDir)
	resB, err := scanner.ScanPaths("", showDir)
	if err != nil {
		t.Fatalf("ScanPaths B failed: %v", err)
	}
	t.Logf("Case B Actions (%d):", len(resB.Actions))
	for _, a := range resB.Actions {
		t.Logf("  Action: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}

	// Case D: files inside Season 1 subfolder, scanned via Scan(showDir)
	seasonDir := filepath.Join(showDir, "Season 1")
	_ = os.MkdirAll(seasonDir, 0755)
	ep3 := filepath.Join(seasonDir, "Monster.The.Lizzie.Borden.Story.S01E03.mkv")
	_ = os.WriteFile(ep3, []byte("ep3"), 0644)
	resD, err := scanner.Scan(showDir)
	if err != nil {
		t.Fatalf("Scan D failed: %v", err)
	}
	t.Logf("Case D Actions (%d):", len(resD.Actions))
	for _, a := range resD.Actions {
		t.Logf("  Action: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}

	// Case F: showDir has NO files in root, ONLY files in Season 1 subfolder, scanned with Scan(showDir2)
	showDir2 := filepath.Join(serienDir, "Monster - The Lizzie Borden Story (2026) 2")
	s1Dir := filepath.Join(showDir2, "Staffel 1")
	_ = os.MkdirAll(s1Dir, 0755)
	epS1 := filepath.Join(s1Dir, "Monster.The.Lizzie.Borden.Story.S01E01.mkv")
	_ = os.WriteFile(epS1, []byte("eps1"), 0644)
	resF, err := scanner.Scan(showDir2)
	if err != nil {
		t.Fatalf("Scan F failed: %v", err)
	}
	t.Logf("Case F Actions (%d):", len(resF.Actions))
	for _, a := range resF.Actions {
		t.Logf("  Action F: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}
	// Case G: showDir2 scanned via ScanPaths("", showDir2)
	resG, err := scanner.ScanPaths("", showDir2)
	if err != nil {
		t.Fatalf("ScanPaths G failed: %v", err)
	}
	t.Logf("Case G Actions (%d):", len(resG.Actions))
	for _, a := range resG.Actions {
		t.Logf("  Action G: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}
	// Case H: Files named with " - 01" or "Part 1"
	showDir3 := filepath.Join(serienDir, "Monster - The Lizzie Borden Story (2026) 3")
	_ = os.MkdirAll(showDir3, 0755)
	epH1 := filepath.Join(showDir3, "Monster - The Lizzie Borden Story - 01.mkv")
	epH2 := filepath.Join(showDir3, "Monster - The Lizzie Borden Story - Part 2.mkv")
	_ = os.WriteFile(epH1, []byte("h1"), 0644)
	_ = os.WriteFile(epH2, []byte("h2"), 0644)
	resH, err := scanner.Scan(showDir3)
	if err != nil {
		t.Fatalf("Scan H failed: %v", err)
	}
	t.Logf("Case H Actions (%d):", len(resH.Actions))
	for _, a := range resH.Actions {
		t.Logf("  Action H: %v | %s -> %s", a.Type, a.SourcePath, a.TargetPath)
	}
}

func TestGenericFolderNotRenamed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_generic_folder_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloads := filepath.Join(tempDir, "Downloads")
	_ = os.MkdirAll(downloads, 0755)

	// Add episodes
	ep1 := filepath.Join(downloads, "Show.S01E01.mkv")
	_ = os.WriteFile(ep1, []byte("fake video"), 0644)
	ep2 := filepath.Join(downloads, "Show.S01E02.mkv")
	_ = os.WriteFile(ep2, []byte("fake video"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(downloads)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, act := range res.Actions {
		if act.Type == ActionRenameFolder {
			t.Errorf("Downloads folder should not be renamed to a TV show, got ActionRenameFolder to %s", act.TargetPath)
		}
	}
}

func TestSubtitleDeduplicationEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_test_sub_dedup_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "The_Matrix_1999_1080p")
	_ = os.MkdirAll(movieFolder, 0755)

	video := filepath.Join(movieFolder, "The_Matrix_1999_1080p.mkv")
	_ = os.WriteFile(video, []byte("fake matrix video"), 0644)

	// Two German subtitles with identical dialogue, slightly different timestamps or formatting
	sub1 := filepath.Join(movieFolder, "The.Matrix.1999.de.srt")
	sub1Content := "1\n00:00:01,000 --> 00:00:04,000\nWillkommen in der realen Welt, Neo.\n\n2\n00:00:05,000 --> 00:00:08,000\nNimm die rote Pille.\n"
	_ = os.WriteFile(sub1, []byte(sub1Content), 0644)

	sub2 := filepath.Join(movieFolder, "The.Matrix.1999.German.srt")
	// slightly longer/different tags, but same normalized dialogue
	sub2Content := "1\n00:00:01,000 --> 00:00:04,500\n<b>Willkommen</b> in der realen Welt, Neo.\n\n2\n00:00:05,000 --> 00:00:08,500\nNimm die rote Pille.\n"
	_ = os.WriteFile(sub2, []byte(sub2Content), 0644)

	// An English subtitle (unique content)
	subEn := filepath.Join(movieFolder, "The.Matrix.1999.en.srt")
	subEnContent := "1\n00:00:01,000 --> 00:00:04,000\nWelcome to the real world, Neo.\n\n2\n00:00:05,000 --> 00:00:08,000\nTake the red pill.\n"
	_ = os.WriteFile(subEn, []byte(subEnContent), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.Scan(movieFolder)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	var duplicateDeleted bool
	var germanKeptCount int
	var englishKeptCount int

	for _, act := range res.Actions {
		if act.Type == ActionDeleteDuplicate {
			duplicateDeleted = true
			if !strings.Contains(act.Description, "Doppelter Untertitel") {
				t.Errorf("Expected description to mention 'Doppelter Untertitel', got: %s", act.Description)
			}
		}
		if (act.Type == ActionMoveFile || act.Type == ActionRenameFile) && strings.HasSuffix(act.TargetPath, ".de.srt") {
			germanKeptCount++
		}
		if (act.Type == ActionMoveFile || act.Type == ActionRenameFile) && strings.HasSuffix(act.TargetPath, ".en.srt") {
			englishKeptCount++
		}
	}

	if !duplicateDeleted {
		t.Errorf("Expected one duplicate German subtitle to be marked for deletion (ActionDeleteDuplicate)")
	}
	if germanKeptCount != 1 {
		t.Errorf("Expected exactly 1 German subtitle to be kept, got %d", germanKeptCount)
	}
	if englishKeptCount != 1 {
		t.Errorf("Expected exactly 1 English subtitle to be kept, got %d", englishKeptCount)
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	targetFolder := filepath.Join(tempDir, "The Matrix (1999)")
	expectedGerman := filepath.Join(targetFolder, "The Matrix (1999).de.srt")
	expectedEnglish := filepath.Join(targetFolder, "The Matrix (1999).en.srt")

	if _, err := os.Stat(expectedGerman); os.IsNotExist(err) {
		t.Errorf("Expected kept German sub %s does not exist", expectedGerman)
	}
	if _, err := os.Stat(expectedEnglish); os.IsNotExist(err) {
		t.Errorf("Expected kept English sub %s does not exist", expectedEnglish)
	}

	// Verify only 1 German sub exists (no collision numbering like .de.2.srt)
	collisionSub := filepath.Join(targetFolder, "The Matrix (1999).de.2.srt")
	if _, err := os.Stat(collisionSub); !os.IsNotExist(err) {
		t.Errorf("Collision sub %s should NOT exist because duplicate was removed", collisionSub)
	}
}

func TestScanPathsMoviesAndShows(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_scanpaths_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	moviesDir := filepath.Join(tempDir, "Movies")
	showsDir := filepath.Join(tempDir, "Shows")
	_ = os.MkdirAll(moviesDir, 0755)
	_ = os.MkdirAll(showsDir, 0755)

	// Create movie in moviesDir
	m1 := filepath.Join(moviesDir, "Inception.2010.1080p.mkv")
	_ = os.WriteFile(m1, []byte("movie"), 0644)

	// Create series episode in showsDir
	s1 := filepath.Join(showsDir, "Breaking.Bad.S01E01.720p.mkv")
	_ = os.WriteFile(s1, []byte("show"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.ScanPaths(moviesDir, showsDir)
	if err != nil {
		t.Fatalf("ScanPaths failed: %v", err)
	}

	if res.TotalMovies != 1 {
		t.Errorf("Expected 1 movie, got %d", res.TotalMovies)
	}
	if res.TotalShows != 1 {
		t.Errorf("Expected 1 show, got %d", res.TotalShows)
	}
	if len(res.Actions) == 0 {
		t.Fatalf("Expected actions for both movie and show")
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedMovie := filepath.Join(moviesDir, "Inception (2010)", "Inception (2010).mkv")
	expectedShow := filepath.Join(showsDir, "Breaking Bad", "Breaking Bad - S01E01.mkv")

	if _, err := os.Stat(expectedMovie); os.IsNotExist(err) {
		t.Errorf("Expected movie %s does not exist", expectedMovie)
	}
	if _, err := os.Stat(expectedShow); os.IsNotExist(err) {
		t.Errorf("Expected show %s does not exist", expectedShow)
	}
}

func TestSeriesPathNeverTreatsEpisodeAsMovie(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_series_protect_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showsDir := filepath.Join(tempDir, "MySeries")
	_ = os.MkdirAll(showsDir, 0755)

	// Single episode with a year in title: Doctor Who (2005) - S01E01
	// In the past, single episodes with a year were often mistaken for movies!
	epFile := filepath.Join(showsDir, "Doctor.Who.2005.S01E01.720p.mkv")
	_ = os.WriteFile(epFile, []byte("data"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.ScanPaths("", showsDir)
	if err != nil {
		t.Fatalf("ScanPaths failed: %v", err)
	}

	if res.TotalMovies != 0 {
		t.Errorf("Series path should have 0 movies, got %d", res.TotalMovies)
	}
	if res.TotalShows != 1 {
		t.Errorf("Expected 1 show, got %d", res.TotalShows)
	}

	// Verify target path does NOT contain movie format or single-episode movie folder
	for _, act := range res.Actions {
		if act.Type == ActionCreateFolder && strings.Contains(act.TargetPath, "S01E01") {
			t.Errorf("Episode was treated as movie folder: %s", act.TargetPath)
		}
	}

	executor := NewExecutor()
	execRes := executor.Execute(res.Actions)
	if execRes.ErrorCount > 0 {
		t.Fatalf("Execution had errors: %v", execRes.Errors)
	}

	expectedShow := filepath.Join(showsDir, "Doctor Who (2005)", "Doctor Who - S01E01.mkv")
	if _, err := os.Stat(expectedShow); os.IsNotExist(err) {
		t.Errorf("Expected show file %s does not exist", expectedShow)
	}
}

func TestMoviePathSkipsSeriesEpisodesWithWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_movie_warn_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	moviesDir := filepath.Join(tempDir, "Movies")
	_ = os.MkdirAll(moviesDir, 0755)

	// A valid movie
	m1 := filepath.Join(moviesDir, "Gladiator.2000.mkv")
	_ = os.WriteFile(m1, []byte("movie"), 0644)

	// An accidental TV episode in the movies directory
	s1 := filepath.Join(moviesDir, "Dark.S01E01.mkv")
	_ = os.WriteFile(s1, []byte("tv episode"), 0644)

	scanner := NewScanner(SeparatorSpace)
	res, err := scanner.ScanPaths(moviesDir, "")
	if err != nil {
		t.Fatalf("ScanPaths failed: %v", err)
	}

	// Should process the movie, but skip the TV episode and add a warning
	if res.TotalMovies != 1 {
		t.Errorf("Expected 1 movie processed, got %d", res.TotalMovies)
	}
	if len(res.Warnings) == 0 {
		t.Errorf("Expected warning for TV episode in movie folder, got none")
	}

	// Ensure no action treats Dark.S01E01 as a movie
	for _, act := range res.Actions {
		if strings.Contains(act.SourcePath, "Dark.S01E01") {
			t.Errorf("Accidental TV episode in movie folder should not have movie action: %+v", act)
		}
	}
}


