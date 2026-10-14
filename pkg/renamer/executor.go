package renamer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExecutionResult holds statistics and logs of executed actions.
type ExecutionResult struct {
	SuccessCount int
	ErrorCount   int
	Errors       []string
}

// Executor performs the filesystem operations.
type Executor struct{}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute performs all actions safely in proper topological order.
func (e *Executor) Execute(actions []Action) ExecutionResult {
	res := ExecutionResult{
		Errors: []string{},
	}

	// Track renamed directories so we can adjust paths of files inside them
	renamedDirs := make(map[string]string)
	// Track directories from which files were moved, to clean them up if empty
	dirsToCleanup := make(map[string]bool)

	// Step 1: Create folders first
	for _, act := range actions {
		if act.Type == ActionCreateFolder {
			if err := os.MkdirAll(act.TargetPath, 0755); err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Ordner erstellen fehlgeschlagen %s: %v", act.DisplayNew, err))
			} else {
				res.SuccessCount++
			}
		}
	}

	// Step 2: Execute folder renames
	for _, act := range actions {
		if act.Type == ActionRenameFolder {
			if act.SourcePath == act.TargetPath {
				continue
			}

			// Ensure parent of target exists
			targetParent := filepath.Dir(act.TargetPath)
			if err := os.MkdirAll(targetParent, 0755); err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Übergeordneten Ordner erstellen fehlgeschlagen %s: %v", targetParent, err))
				continue
			}

			// Check if target already exists
			if tgtFi, err := os.Stat(act.TargetPath); err == nil {
				srcFi, errSrc := os.Stat(act.SourcePath)
				if errSrc == nil && os.SameFile(srcFi, tgtFi) {
					// Same directory on disk! Case-only rename (e.g. on Android / FAT / exFAT / sdcardfs)
					tmpDir := act.SourcePath + fmt.Sprintf(".smrt_tmp_%d", time.Now().UnixNano())
					if err := os.Rename(act.SourcePath, tmpDir); err == nil {
						if err := os.Rename(tmpDir, act.TargetPath); err == nil {
							res.SuccessCount++
							renamedDirs[act.SourcePath] = act.TargetPath
							continue
						} else {
							_ = os.Rename(tmpDir, act.SourcePath)
						}
					}
				}
				// Target directory already exists as a distinct directory on disk.
				// Files will be moved/merged into this existing target directory.
				// Crucial: DO NOT add to renamedDirs because the source directory was NOT renamed on disk!
				continue
			}

			// Rename directory
			if err := os.Rename(act.SourcePath, act.TargetPath); err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Ordner umbenennen fehlgeschlagen %s -> %s: %v", act.DisplayOld, act.DisplayNew, err))
			} else {
				res.SuccessCount++
				renamedDirs[act.SourcePath] = act.TargetPath
			}
		}
	}

	// Step 3: Execute file moves / renames
	for _, act := range actions {
		if act.Type == ActionRenameFolder || act.Type == ActionCreateFolder {
			continue // Already processed
		}

		// Adjust source path if a parent directory was renamed
		actualSource := act.SourcePath
		for oldDir, newDir := range renamedDirs {
			if strings.HasPrefix(actualSource, oldDir+string(filepath.Separator)) {
				actualSource = filepath.Join(newDir, strings.TrimPrefix(actualSource, oldDir+string(filepath.Separator)))
				break
			}
		}

		// Track parent directory of source for cleanup
		sourceDir := filepath.Dir(actualSource)
		dirsToCleanup[sourceDir] = true

		// If this is a file deletion (e.g. unwanted non-de/en subtitles or duplicates)
		if act.Type == ActionDeleteFile || act.Type == ActionDeleteDuplicate {
			if err := os.Remove(actualSource); err != nil && !os.IsNotExist(err) {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Datei löschen fehlgeschlagen %s: %v", act.DisplayOld, err))
			} else {
				res.SuccessCount++
			}
			continue
		}

		// If this is a subtitle conversion to .srt
		if act.Type == ActionConvertSubtitle {
			// Ensure target directory exists
			targetDir := filepath.Dir(act.TargetPath)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Zielordner erstellen fehlgeschlagen %s: %v", targetDir, err))
				continue
			}

			// Check if source exists
			srcFi, err := os.Stat(actualSource)
			if err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Quelldatei nicht gefunden %s: %v", act.DisplayOld, err))
				continue
			}

			targetSubPath := act.TargetPath
			if tgtFi, err := os.Stat(targetSubPath); err == nil {
				if !os.SameFile(srcFi, tgtFi) {
					ext := filepath.Ext(targetSubPath)
					base := strings.TrimSuffix(targetSubPath, ext)
					for i := 2; i <= 999; i++ {
						candidate := fmt.Sprintf("%s.%d%s", base, i, ext)
						if _, err := os.Stat(candidate); os.IsNotExist(err) {
							targetSubPath = candidate
							break
						}
					}
				}
			}

			// Convert to SRT format using astisub
			if err := ConvertSubtitleToSRT(actualSource, targetSubPath); err != nil {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Untertitel konvertieren fehlgeschlagen %s -> %s: %v", act.DisplayOld, act.DisplayNew, err))
				continue
			}

			// Delete old subtitle file after successful conversion
			if err := os.Remove(actualSource); err != nil && !os.IsNotExist(err) {
				res.Errors = append(res.Errors, fmt.Sprintf("Alte Untertiteldatei entfernen fehlgeschlagen %s: %v", act.DisplayOld, err))
			}

			// If companion .idx or .sub exists, remove it too
			srcExt := strings.ToLower(filepath.Ext(actualSource))
			if srcExt == ".sub" {
				companionIdx := strings.TrimSuffix(actualSource, srcExt) + ".idx"
				_ = os.Remove(companionIdx)
			} else if srcExt == ".idx" {
				companionSub := strings.TrimSuffix(actualSource, srcExt) + ".sub"
				_ = os.Remove(companionSub)
			}

			res.SuccessCount++
			continue
		}

		if actualSource == act.TargetPath {
			continue // Already named correctly
		}

		// Ensure target directory exists (e.g. S01, S02, or dedicated movie folder)
		targetDir := filepath.Dir(act.TargetPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			res.ErrorCount++
			res.Errors = append(res.Errors, fmt.Sprintf("Zielordner erstellen fehlgeschlagen %s: %v", targetDir, err))
			continue
		}

		// Check if source exists
		srcFi, err := os.Stat(actualSource)
		if err != nil {
			res.ErrorCount++
			res.Errors = append(res.Errors, fmt.Sprintf("Quelldatei nicht gefunden %s: %v", act.DisplayOld, err))
			continue
		}

		targetPath := act.TargetPath

		// Check if target already exists
		if tgtFi, err := os.Stat(targetPath); err == nil {
			if os.SameFile(srcFi, tgtFi) {
				// Same file on disk! Case-only rename (e.g. on Android / FAT / exFAT / sdcardfs)
				tmpFile := actualSource + fmt.Sprintf(".smrt_tmp_%d", time.Now().UnixNano())
				if err := os.Rename(actualSource, tmpFile); err != nil {
					res.ErrorCount++
					res.Errors = append(res.Errors, fmt.Sprintf("Datei umbenennen fehlgeschlagen (Tmp) %s: %v", act.DisplayOld, err))
					continue
				}
				if err := os.Rename(tmpFile, targetPath); err != nil {
					_ = os.Rename(tmpFile, actualSource)
					res.ErrorCount++
					res.Errors = append(res.Errors, fmt.Sprintf("Datei umbenennen fehlgeschlagen %s -> %s: %v", act.DisplayOld, act.DisplayNew, err))
					continue
				}
				res.SuccessCount++
				continue
			}

			// Target already exists as a DIFFERENT file!
			// Disambiguate so the file is not skipped and no data is lost!
			ext := filepath.Ext(targetPath)
			base := strings.TrimSuffix(targetPath, ext)
			disambiguated := ""
			for i := 2; i <= 999; i++ {
				candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
				if _, err := os.Stat(candidate); os.IsNotExist(err) {
					disambiguated = candidate
					break
				}
			}
			if disambiguated == "" {
				res.ErrorCount++
				res.Errors = append(res.Errors, fmt.Sprintf("Zieldatei existiert bereits, übersprungen: %s", act.DisplayNew))
				continue
			}
			targetPath = disambiguated
		}

		// Perform rename / move
		if err := os.Rename(actualSource, targetPath); err != nil {
			res.ErrorCount++
			res.Errors = append(res.Errors, fmt.Sprintf("Datei umbenennen/verschieben fehlgeschlagen %s -> %s: %v", act.DisplayOld, filepath.Base(targetPath), err))
		} else {
			res.SuccessCount++
			if targetPath != act.TargetPath {
				res.Errors = append(res.Errors, fmt.Sprintf("Zieldatei existierte bereits, umbenannt zu: %s", filepath.Base(targetPath)))
			}
		}
	}

	// Step 4: Clean up any empty directories left behind (e.g. Subs/, Subtitles/)
	for dir := range dirsToCleanup {
		cleanupIfEmpty(dir)
	}

	return res
}

// cleanupIfEmpty removes the directory only if it is completely empty.
func cleanupIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
		// Try cleaning parent if it was e.g. Subs inside a folder
		parent := filepath.Dir(dir)
		base := strings.ToLower(filepath.Base(dir))
		if base == "subs" || base == "subtitles" || base == "untertitel" {
			cleanupIfEmpty(parent)
		}
	}
}
