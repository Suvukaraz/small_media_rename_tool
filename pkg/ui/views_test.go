package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUIViews(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ui_views_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	movieFolder := filepath.Join(tempDir, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla")
	_ = os.MkdirAll(movieFolder, 0755)
	_ = os.WriteFile(filepath.Join(movieFolder, "The_Film_Title-electric_Buggalo-(2009)-x264_bla-bla_bla.mkv"), []byte("data"), 0644)

	m := InitialModel()
	// Set window size
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := m2.(Model)

	// Screen 1 View
	viewInput := model.View()
	if !strings.Contains(viewInput, "Ordnerpfad") || !strings.Contains(viewInput, "LET'S GO") {
		t.Errorf("ViewInput missing required elements: %s", viewInput)
	}

	// Go to Screen 2
	model.inputPath.SetValue(movieFolder)
	m3, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m3.(Model)
	if model.state != screenPreview {
		t.Fatalf("Expected screenPreview, got %v", model.state)
	}

	viewPreview := model.View()
	if !strings.Contains(viewPreview, "The Film Title - Electric Buggalo (2009)") {
		t.Errorf("ViewPreview missing expected movie title: %s", viewPreview)
	}

	// Go to Screen 3
	m4, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = m4.(Model)
	if model.state != screenResult {
		t.Fatalf("Expected screenResult, got %v", model.state)
	}

	viewResult := model.View()
	if !strings.Contains(viewResult, "Alles erledigt") {
		t.Errorf("ViewResult missing success message: %s", viewResult)
	}
}

func TestMobileViews(t *testing.T) {
	m := InitialModel()
	// Simulate narrow smartphone terminal (e.g. 40 columns x 25 rows)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 25})
	model := m2.(Model)

	// Closed dropdown view on mobile
	closedView := model.View()
	if !strings.Contains(closedView, "Trennzeichen") {
		t.Errorf("Mobile closed view missing Trennzeichen: %s", closedView)
	}

	// Open dropdown view on mobile
	model.dropdownOpen = true
	openView := model.View()
	if !strings.Contains(openView, "Leerzeichen") || !strings.Contains(openView, "Unterstrich") {
		t.Errorf("Mobile open view missing options: %s", openView)
	}
}
