package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutorTargetCollisionDisambiguation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_exec_collision_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetDir := filepath.Join(tempDir, "Target")
	_ = os.MkdirAll(targetDir, 0755)

	// Existing file in target directory
	existingFile := filepath.Join(targetDir, "Movie (2020).mkv")
	_ = os.WriteFile(existingFile, []byte("existing"), 0644)

	// Source file to be moved
	sourceDir := filepath.Join(tempDir, "Source")
	_ = os.MkdirAll(sourceDir, 0755)
	sourceFile := filepath.Join(sourceDir, "movie.2020.mkv")
	_ = os.WriteFile(sourceFile, []byte("source"), 0644)

	actions := []Action{
		{
			Type:       ActionMoveFile,
			SourcePath: sourceFile,
			TargetPath: existingFile,
			DisplayOld: "movie.2020.mkv",
			DisplayNew: "Movie (2020).mkv",
		},
	}

	executor := NewExecutor()
	res := executor.Execute(actions)

	if res.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", res.ErrorCount, res.Errors)
	}
	if res.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", res.SuccessCount)
	}

	expectedDisambiguated := filepath.Join(targetDir, "Movie (2020) (2).mkv")
	if _, err := os.Stat(expectedDisambiguated); os.IsNotExist(err) {
		t.Errorf("Expected disambiguated file %s to exist", expectedDisambiguated)
	}
	// And original existing file must still exist untouched
	if _, err := os.Stat(existingFile); os.IsNotExist(err) {
		t.Errorf("Original existing file %s should still exist", existingFile)
	}
}

func TestExecutorCaseOnlyRename(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smrt_exec_case_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceFile := filepath.Join(tempDir, "movie.mkv")
	_ = os.WriteFile(sourceFile, []byte("movie content"), 0644)

	targetFile := filepath.Join(tempDir, "Movie.mkv")

	actions := []Action{
		{
			Type:       ActionMoveFile,
			SourcePath: sourceFile,
			TargetPath: targetFile,
			DisplayOld: "movie.mkv",
			DisplayNew: "Movie.mkv",
		},
	}

	executor := NewExecutor()
	res := executor.Execute(actions)

	if res.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", res.ErrorCount, res.Errors)
	}
	if res.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", res.SuccessCount)
	}
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Errorf("Target file %s does not exist", targetFile)
	}
}
