package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"smrt/pkg/renamer"
)

func TestUIFlowAndCtrlU(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ui_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	showDir := filepath.Join(tempDir, "TV_Show_title_hevc-bla-bla_bla_2013")
	_ = os.MkdirAll(showDir, 0755)
	_ = os.WriteFile(filepath.Join(showDir, "TV_Shoe_Title-episodename.S01E01.hevc.mp4"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(showDir, "TV_Shoe_Title-episodename.S02E07.hevc.mp4"), []byte("data"), 0644)

	m := InitialModel()
	if m.state != screenInput {
		t.Errorf("Expected initial state to be screenInput, got %v", m.state)
	}
	if m.separator != renamer.SeparatorSpace {
		t.Errorf("Expected separator default to be SeparatorSpace, got %v", m.separator)
	}

	// Test Ctrl+U: Space -> Dot
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model := m2.(Model)
	if model.separator != renamer.SeparatorDot {
		t.Errorf("Expected separator to be SeparatorDot after Ctrl+U, got %v", model.separator)
	}

	// Test Ctrl+U: Dot -> Underscore
	m3, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = m3.(Model)
	if model.separator != renamer.SeparatorUnderscore {
		t.Errorf("Expected separator to be SeparatorUnderscore after Ctrl+U, got %v", model.separator)
	}

	// Test Ctrl+U: Underscore -> Space
	m4, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = m4.(Model)
	if model.separator != renamer.SeparatorSpace {
		t.Errorf("Expected separator to cycle back to SeparatorSpace, got %v", model.separator)
	}

	// Spacebar when input is focused should NOT start scan ("Nicht mit leertaste + start!")
	m5, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	model = m5.(Model)
	if model.state != screenInput {
		t.Errorf("Spacebar should not trigger scan, state is %v", model.state)
	}

	// Set input path
	model.inputPath.SetValue(showDir)

	// Focus dropdown with Tab
	m6, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = m6.(Model)
	if model.focus != fieldDropdown {
		t.Errorf("Expected focus to be fieldDropdown, got %v", model.focus)
	}

	// Enter opens dropdown
	m7, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m7.(Model)
	if !model.dropdownOpen {
		t.Errorf("Expected dropdown to be open")
	}

	// Down moves option
	m8, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = m8.(Model)
	if model.dropdownIdx != 1 {
		t.Errorf("Expected dropdownIdx to be 1, got %d", model.dropdownIdx)
	}

	// Enter selects option and closes dropdown
	m9, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m9.(Model)
	if model.dropdownOpen {
		t.Errorf("Expected dropdown to be closed after selection")
	}
	if model.separator != renamer.SeparatorDot {
		t.Errorf("Expected separator to be SeparatorDot, got %v", model.separator)
	}

	// Tab to button and Enter to start scan
	m10, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = m10.(Model)
	if model.focus != fieldButtonGo {
		t.Errorf("Expected focus to be fieldButtonGo, got %v", model.focus)
	}

	m11, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m11.(Model)
	if model.state != screenPreview {
		t.Errorf("Expected state to be screenPreview, got %v", model.state)
	}

	// Enter in preview executes
	m12, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m12.(Model)
	if model.state != screenResult {
		t.Errorf("Expected state to be screenResult, got %v", model.state)
	}

	// Verify files with dot separator
	targetShow := filepath.Join(tempDir, "TV.Show.Title")
	ep1 := filepath.Join(targetShow, "S01", "TV.Show.Title.-.S01E01.mp4")
	ep2 := filepath.Join(targetShow, "S02", "TV.Show.Title.-.S02E07.mp4")

	if _, err := os.Stat(ep1); os.IsNotExist(err) {
		t.Errorf("File %s does not exist on disk", ep1)
	}
	if _, err := os.Stat(ep2); os.IsNotExist(err) {
		t.Errorf("File %s does not exist on disk", ep2)
	}
}

func TestMouseAndTouchInteraction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ui_mouse_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla")
	_ = os.MkdirAll(movieFolder, 0755)
	_ = os.WriteFile(filepath.Join(movieFolder, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla.mkv"), []byte("data"), 0644)

	m := InitialModel()
	m.inputPath.SetValue(movieFolder)
	// Render view to populate clickAreas
	_ = m.View()

	if m.clickAreas == nil || len(*m.clickAreas) == 0 {
		t.Fatalf("Expected clickAreas to be populated after View()")
	}

	// Find dropdown_header click area and simulate tap
	var dropdownArea *clickArea
	for _, a := range *m.clickAreas {
		if a.id == "dropdown_header" {
			dropdownArea = &a
			break
		}
	}
	if dropdownArea == nil {
		t.Fatalf("Could not find dropdown_header click area")
	}

	// Click to open dropdown
	clickMsg := tea.MouseMsg{
		X:      dropdownArea.minX + 2,
		Y:      dropdownArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	m2, _ := m.Update(clickMsg)
	model := m2.(Model)
	if !model.dropdownOpen {
		t.Fatalf("Expected dropdown to open on mouse click")
	}

	// Render view with open dropdown
	_ = model.View()

	// Find opt_2 (Unterstrich) area and click it
	var opt2Area *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "opt_2" {
			opt2Area = &a
			break
		}
	}
	if opt2Area == nil {
		t.Fatalf("Could not find opt_2 click area")
	}

	m3, _ := model.Update(tea.MouseMsg{
		X:      opt2Area.minX + 2,
		Y:      opt2Area.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m3.(Model)
	if model.dropdownOpen {
		t.Errorf("Expected dropdown to close after clicking option")
	}
	if model.separator != renamer.SeparatorUnderscore {
		t.Errorf("Expected separator to be SeparatorUnderscore, got %v", model.separator)
	}

	// Render view and click Let's Go button
	_ = model.View()
	var goArea *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "btn_go" {
			goArea = &a
			break
		}
	}
	if goArea == nil {
		t.Fatalf("Could not find btn_go click area")
	}

	m4, _ := model.Update(tea.MouseMsg{
		X:      goArea.minX + 1,
		Y:      goArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m4.(Model)
	if model.state != screenPreview {
		t.Fatalf("Expected state to be screenPreview after clicking Let's Go, got %v", model.state)
	}

	// Render preview and click Confirm button
	_ = model.View()
	var confirmArea *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "btn_confirm" {
			confirmArea = &a
			break
		}
	}
	if confirmArea == nil {
		t.Fatalf("Could not find btn_confirm click area")
	}

	m5, _ := model.Update(tea.MouseMsg{
		X:      confirmArea.minX + 1,
		Y:      confirmArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m5.(Model)
	if model.state != screenResult {
		t.Fatalf("Expected state to be screenResult after clicking Confirm, got %v", model.state)
	}

	// Verify renamed files with underscores
	expectedFolder := filepath.Join(tempDir, "The_Film_Title_-_Electric_Buggalo_(2009)")
	expectedFile := filepath.Join(expectedFolder, "The_Film_Title_-_Electric_Buggalo_(2009).mkv")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist on disk", expectedFile)
	}
}

func TestTouchInPreviewAndResultScreens(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ui_touch_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "The_Film_Title-(2009)")
	_ = os.MkdirAll(movieFolder, 0755)
	_ = os.WriteFile(filepath.Join(movieFolder, "The_Film_Title-(2009).mkv"), []byte("data"), 0644)

	m := InitialModel()
	// Set mobile dimensions
	m.width = 40
	m.height = 24
	m.inputPath.SetValue(movieFolder)

	// Transition to screenPreview
	m2, _ := m.startScan()
	model := m2.(Model)
	if model.state != screenPreview {
		t.Fatalf("Expected screenPreview, got %v", model.state)
	}

	// 1. Test Preview Screen Touch Elements
	_ = model.View()
	areas := make(map[string]clickArea)
	for _, a := range *model.clickAreas {
		areas[a.id] = a
	}

	// Verify required touch areas exist
	if _, ok := areas["btn_confirm"]; !ok {
		t.Errorf("Missing btn_confirm touch area in screenPreview")
	}
	if _, ok := areas["btn_cancel"]; !ok {
		t.Errorf("Missing btn_cancel touch area in screenPreview")
	}

	// In mobile mode, hitboxes must span across the entire terminal width
	confirmArea := areas["btn_confirm"]
	if confirmArea.minX != 0 || confirmArea.maxX < 39 {
		t.Errorf("btn_confirm hitbox does not cover full screen width: minX=%d, maxX=%d", confirmArea.minX, confirmArea.maxX)
	}

	cancelArea := areas["btn_cancel"]
	if cancelArea.minX != 0 || cancelArea.maxX < 39 {
		t.Errorf("btn_cancel hitbox does not cover full screen width: minX=%d, maxX=%d", cancelArea.minX, cancelArea.maxX)
	}

	// Simulate touch on Cancel button -> should return to screenInput
	m3, _ := model.Update(tea.MouseMsg{
		X:      10,
		Y:      cancelArea.minY + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m3.(Model)
	if model.state != screenInput {
		t.Errorf("Expected state to be screenInput after tapping cancel, got %v", model.state)
	}

	// Return to preview
	m4, _ := model.startScan()
	model = m4.(Model)
	_ = model.View()

	// Find confirm area again
	confirmArea = clickArea{}
	for _, a := range *model.clickAreas {
		if a.id == "btn_confirm" {
			confirmArea = a
			break
		}
	}

	// Tap confirm button -> transitions to screenResult
	m5, _ := model.Update(tea.MouseMsg{
		X:      15,
		Y:      confirmArea.minY + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m5.(Model)
	if model.state != screenResult {
		t.Fatalf("Expected state to be screenResult after tapping confirm, got %v", model.state)
	}

	// 2. Test Result Screen Touch Elements
	_ = model.View()
	resAreas := make(map[string]clickArea)
	for _, a := range *model.clickAreas {
		resAreas[a.id] = a
	}

	if _, ok := resAreas["btn_new"]; !ok {
		t.Errorf("Missing btn_new touch area in screenResult")
	}
	if _, ok := resAreas["btn_quit"]; !ok {
		t.Errorf("Missing btn_quit touch area in screenResult")
	}

	newArea := resAreas["btn_new"]
	if newArea.minX != 0 || newArea.maxX < 39 {
		t.Errorf("btn_new hitbox does not cover full screen width: minX=%d, maxX=%d", newArea.minX, newArea.maxX)
	}

	quitArea := resAreas["btn_quit"]
	if quitArea.minX != 0 || quitArea.maxX < 39 {
		t.Errorf("btn_quit hitbox does not cover full screen width: minX=%d, maxX=%d", quitArea.minX, quitArea.maxX)
	}

	// Simulate touch on [Neuer Ordner] -> should return to screenInput
	m6, _ := model.Update(tea.MouseMsg{
		X:      5,
		Y:      newArea.minY + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m6.(Model)
	if model.state != screenInput {
		t.Errorf("Expected state to be screenInput after tapping btn_new, got %v", model.state)
	}

	// 3. Test empty scan / no actions touch button
	emptyDir, _ := os.MkdirTemp("", "empty_ui_*")
	defer os.RemoveAll(emptyDir)

	model.inputPath.SetValue(emptyDir)
	m7, _ := model.startScan()
	model = m7.(Model)
	_ = model.View()

	var emptyCancelArea *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "btn_cancel" {
			emptyCancelArea = &a
			break
		}
	}
	if emptyCancelArea == nil {
		t.Fatalf("Missing btn_cancel area in empty preview screen")
	}

	m8, _ := model.Update(tea.MouseMsg{
		X:      8,
		Y:      emptyCancelArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model = m8.(Model)
	if model.state != screenInput {
		t.Errorf("Expected return to screenInput after tapping back on empty preview, got %v", model.state)
	}
}

func TestPreviewConvertSubtitleTag(t *testing.T) {
	m := InitialModel()
	m.state = screenPreview
	m.scanResult = &renamer.ScanResult{
		Actions: []renamer.Action{
			{
				Type:        renamer.ActionConvertSubtitle,
				SourcePath:  "/path/to/movie.de.vtt",
				TargetPath:  "/path/to/Movie (2020)/Movie (2020).de.srt",
				DisplayOld:  "movie.de.vtt",
				DisplayNew:  "Movie (2020)/Movie (2020).de.srt",
				Description: "Untertitel nach SRT konvertieren (de)",
			},
		},
	}
	content := m.renderPreviewContent()
	if !strings.Contains(content, "[KONVERTIEREN]") {
		t.Errorf("Expected [KONVERTIEREN] in preview content, got: %s", content)
	}
	if !strings.Contains(content, "Untertitel nach SRT konvertieren (de)") {
		t.Errorf("Expected description in preview content, got: %s", content)
	}
}

func TestPreviewDuplicateSubtitleTag(t *testing.T) {
	m := InitialModel()
	m.state = screenPreview
	m.scanResult = &renamer.ScanResult{
		Actions: []renamer.Action{
			{
				Type:        renamer.ActionDeleteDuplicate,
				SourcePath:  "/path/to/movie.German.srt",
				TargetPath:  "",
				DisplayOld:  "movie.German.srt",
				DisplayNew:  "(Wird gelöscht)",
				Description: "Doppelter Untertitel (de, identisch mit movie.de.srt)",
			},
		},
	}
	content := m.renderPreviewContent()
	if !strings.Contains(content, "[DUPLIKAT]") {
		t.Errorf("Expected [DUPLIKAT] in preview content, got: %s", content)
	}
	if !strings.Contains(content, "Doppelter Untertitel (de, identisch mit movie.de.srt)") {
		t.Errorf("Expected description in preview content, got: %s", content)
	}
}

func TestTouchPressReleaseDeduplication(t *testing.T) {
	m := InitialModel()
	m.width = 60
	m.height = 25
	_ = m.View()

	// Find dropdown_header area
	var dropdownArea *clickArea
	for _, a := range *m.clickAreas {
		if a.id == "dropdown_header" {
			dropdownArea = &a
			break
		}
	}
	if dropdownArea == nil {
		t.Fatalf("Could not find dropdown_header click area")
	}

	// 1. Send MouseActionPress -> dropdown should OPEN
	m2, _ := m.Update(tea.MouseMsg{
		X:      dropdownArea.minX + 2,
		Y:      dropdownArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model := m2.(Model)
	if !model.dropdownOpen {
		t.Fatalf("Expected dropdown to open on MouseActionPress")
	}

	// 2. Immediately send MouseActionRelease (finger lifted) -> dropdown MUST STAY OPEN!
	// It should NOT toggle closed because release is consumed by the prior press!
	m3, _ := model.Update(tea.MouseMsg{
		X:      dropdownArea.minX + 2,
		Y:      dropdownArea.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = m3.(Model)
	if !model.dropdownOpen {
		t.Errorf("Dropdown was closed by MouseActionRelease! Release must not double-trigger.")
	}

	// 3. Test terminal that ONLY sends MouseActionRelease (e.g. touch emulators)
	_ = model.View()
	var opt1Area *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "opt_1" {
			opt1Area = &a
			break
		}
	}
	if opt1Area == nil {
		t.Fatalf("Could not find opt_1 click area")
	}

	// Wait 250ms to simulate touch without prior press
	model.lastClickTime = model.lastClickTime.Add(-300 * time.Millisecond)
	model.pressHandled = false

	m4, _ := model.Update(tea.MouseMsg{
		X:      opt1Area.minX + 2,
		Y:      opt1Area.minY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = m4.(Model)
	if model.dropdownOpen {
		t.Errorf("Expected dropdown to close after selecting option via pure Release")
	}
	if model.separator != renamer.SeparatorDot {
		t.Errorf("Expected separator to be SeparatorDot, got %v", model.separator)
	}
}

func TestInputTapCursorPositioning(t *testing.T) {
	m := InitialModel()
	m.inputPath.SetValue("/home/user/downloads/my_movie")
	m.focus = fieldButtonGo // unfocus input first
	_ = m.View()

	var inputArea *clickArea
	for _, a := range *m.clickAreas {
		if a.id == "input" {
			inputArea = &a
			break
		}
	}
	if inputArea == nil {
		t.Fatalf("Could not find input click area")
	}

	// Tap at position 10 characters into the input box
	tapX := inputArea.minX + 4 + 10
	m2, _ := m.Update(tea.MouseMsg{
		X:      tapX,
		Y:      inputArea.minY + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	model := m2.(Model)
	if model.focus != fieldPath {
		t.Errorf("Expected focus to be fieldPath after tapping input, got %v", model.focus)
	}
	if model.inputPath.Position() != 10 {
		t.Errorf("Expected cursor position to be 10, got %d", model.inputPath.Position())
	}
}

func TestTerminalHeightFittingAndNoScroll(t *testing.T) {
	// Test Screen 1 on standard phone height (24 rows) and small phone height (20 rows)
	heights := []int{20, 24, 30}
	for _, h := range heights {
		m := InitialModel()
		m2, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: h})
		model := m2.(Model)

		// Closed dropdown
		viewClosed := model.View()
		lineCountClosed := strings.Count(viewClosed, "\n")
		if lineCountClosed > h {
			t.Errorf("Height %d: Screen 1 (closed) has %d lines, exceeds terminal height!", h, lineCountClosed)
		}

		// Open dropdown
		model.dropdownOpen = true
		viewOpen := model.View()
		lineCountOpen := strings.Count(viewOpen, "\n")
		if lineCountOpen > h {
			t.Errorf("Height %d: Screen 1 (open dropdown) has %d lines, exceeds terminal height!", h, lineCountOpen)
		}

		// Screen 2 Preview with actions
		model.state = screenPreview
		model.dropdownOpen = false
		model.scanResult = &renamer.ScanResult{
			TotalMovies: 1,
			TotalShows:  0,
			Actions: []renamer.Action{
				{Type: renamer.ActionRenameFile, DisplayOld: "old1.mkv", DisplayNew: "new1.mkv", Description: "Film 1"},
				{Type: renamer.ActionMoveFile, DisplayOld: "old1.de.srt", DisplayNew: "new1.de.srt", Description: "Untertitel 1"},
				{Type: renamer.ActionDeleteDuplicate, DisplayOld: "old1.German.srt", DisplayNew: "(Gelöscht)", Description: "Duplikat 1"},
			},
		}
		model.viewport.SetContent(model.renderPreviewContent())
		viewPreview := model.View()
		lineCountPreview := strings.Count(viewPreview, "\n")
		if lineCountPreview > h {
			t.Errorf("Height %d: Screen 2 (preview) has %d lines, exceeds terminal height %d!", h, lineCountPreview, h)
		}
	}
}

func TestDesktopMarginSafety(t *testing.T) {
	m := InitialModel()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model := m2.(Model)
	model.state = screenResult
	model.execResult = &renamer.ExecutionResult{SuccessCount: 5}
	_ = model.View()

	// On desktop (width 120), clicking at column 110 (empty right margin) should NOT trigger btn_quit or btn_new
	m3, cmd := model.Update(tea.MouseMsg{
		X:      110,
		Y:      20,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	_ = m3
	if cmd != nil {
		t.Errorf("Click in empty right margin triggered command (likely btn_quit): %v", cmd)
	}
}

func TestAndroidTouchButtonNoneInPreview(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "android_touch_*")
	defer os.RemoveAll(tempDir)
	_ = os.WriteFile(filepath.Join(tempDir, "Show.S01E01.mkv"), []byte("video"), 0644)

	m := InitialModel()
	m.width = 38
	m.height = 24
	m.inputPath.SetValue(tempDir)
	m2, _ := m.startScan()
	model := m2.(Model)
	_ = model.View()

	var confirmArea *clickArea
	for _, a := range *model.clickAreas {
		if a.id == "btn_confirm" {
			confirmArea = &a
			break
		}
	}
	if confirmArea == nil {
		t.Fatalf("Could not find btn_confirm area")
	}

	// Android touch: sends MouseButtonNone on Press
	m3, _ := model.Update(tea.MouseMsg{
		X:      20,
		Y:      confirmArea.minY + 1,
		Button: tea.MouseButtonNone,
		Action: tea.MouseActionPress,
	})
	model = m3.(Model)
	if model.state != screenResult {
		t.Errorf("Expected transition to screenResult with Android MouseButtonNone, got %v", model.state)
	}
}




