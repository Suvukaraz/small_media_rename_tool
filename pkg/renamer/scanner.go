package renamer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scanner handles analyzing directories and planning renaming actions.
type Scanner struct {
	Separator SeparatorMode
}

// NewScanner creates a new Scanner instance with the given separator mode.
func NewScanner(sep SeparatorMode) *Scanner {
	if sep == "" {
		sep = SeparatorSpace
	}
	return &Scanner{
		Separator: sep,
	}
}

// NewScannerWithSpaces creates a Scanner instance from a boolean for backward compatibility.
func NewScannerWithSpaces(useSpaces bool) *Scanner {
	if useSpaces {
		return NewScanner(SeparatorSpace)
	}
	return NewScanner(SeparatorDot)
}

// ExpandPath expands leading ~ to user's home directory.
func ExpandPath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return filepath.Clean(path)
}

// Scan examines the given directory and generates a planned list of actions.
func (s *Scanner) Scan(inputPath string) (*ScanResult, error) {
	cleanPath := ExpandPath(inputPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("ungültiger Pfad: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("Pfad nicht gefunden: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("angegebener Pfad ist kein Ordner")
	}

	res := &ScanResult{
		RootPath:  absPath,
		Separator: s.Separator,
		UseSpaces: s.Separator == SeparatorSpace,
		Actions:   []Action{},
		Warnings:  []string{},
	}

	// Read entries of the root directory
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("konnte Ordner nicht lesen: %w", err)
	}

	// Categorize direct entries
	var subDirs []string
	var videoFiles []string
	var subtitleFiles []string

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden files
		}
		full := filepath.Join(absPath, name)
		if entry.IsDir() {
			subDirs = append(subDirs, full)
		} else {
			if IsMediaFile(name) {
				videoFiles = append(videoFiles, full)
			} else if IsSubtitleFile(name) {
				subtitleFiles = append(subtitleFiles, full)
			}
		}
	}

	// Check if the root directory itself is a single Show or Movie folder
	if len(videoFiles) > 0 {
		isSingleItemRoot := s.checkIfFolderIsSingleItem(absPath, videoFiles)
		if isSingleItemRoot {
			err := s.planActionsForFolder(res, absPath, true)
			if err != nil {
				return nil, err
			}
			return res, nil
		}
	}

	// Process subdirectories
	for _, subDir := range subDirs {
		err := s.planActionsForFolder(res, subDir, false)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", filepath.Base(subDir), err))
		}
	}

	// Process any loose video and subtitle files in root directory
	if len(videoFiles) > 0 || len(subtitleFiles) > 0 {
		s.planActionsForLooseFiles(res, absPath, videoFiles, subtitleFiles)
	}

	return res, nil
}

// checkIfFolderIsSingleItem checks if folder itself represents a single movie or TV show release.
func (s *Scanner) checkIfFolderIsSingleItem(folderPath string, videoFiles []string) bool {
	// If all video files are TV episodes belonging to one show
	var seasons []int
	epCount := 0
	for _, vf := range videoFiles {
		sNum, _, _, matched := ExtractEpisodeInfo(filepath.Base(vf))
		if matched {
			epCount++
			seasons = append(seasons, sNum)
		}
	}

	if epCount > 0 && epCount == len(videoFiles) && !isGenericFolder(filepath.Base(folderPath)) {
		return true // It's a TV show folder
	}

	// If there's 1 or 2 video files (e.g. CD1/CD2 or main movie) and matches movie year pattern
	if len(videoFiles) <= 2 {
		folderName := filepath.Base(folderPath)
		if !isGenericFolder(folderName) {
			_, year := ExtractMovieYearAndTitle(folderName)
			if year > 0 {
				return true
			}
			_, fYear := ExtractMovieYearAndTitle(strings.TrimSuffix(filepath.Base(videoFiles[0]), filepath.Ext(videoFiles[0])))
			if fYear > 0 {
				return true
			}
		}
	}

	return false
}

// planActionsForFolder inspects a folder (which could be the root or a subfolder).
func (s *Scanner) planActionsForFolder(res *ScanResult, folderPath string, isRoot bool) error {
	folderName := filepath.Base(folderPath)

	// Collect all video and subtitle files inside folderPath (including season subdirs and Subs/ folders)
	var videoFiles []string
	var subtitleFiles []string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path != folderPath {
				base := strings.ToLower(info.Name())
				isSeasonDir := strings.HasPrefix(base, "season") || strings.HasPrefix(base, "staffel") || (len(base) >= 2 && base[0] == 's' && isDigit(base[1]))
				isSubsDir := base == "subs" || base == "subtitles" || base == "untertitel"
				if !isSeasonDir && !isSubsDir {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		if IsMediaFile(info.Name()) {
			videoFiles = append(videoFiles, path)
		} else if IsSubtitleFile(info.Name()) {
			subtitleFiles = append(subtitleFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(videoFiles) == 0 {
		return nil // No videos in folder
	}

	// Determine if this folder contains TV show episodes
	var episodes []*EpisodeFile
	seasonsSet := make(map[int]bool)

	for _, vf := range videoFiles {
		filename := filepath.Base(vf)
		season, ep, ep2, matched := ExtractEpisodeInfo(filename)
		if matched {
			seasonsSet[season] = true
			episodes = append(episodes, &EpisodeFile{
				OriginalPath: vf,
				Filename:     filename,
				Extension:    filepath.Ext(vf),
				Season:       season,
				Episode:      ep,
				SecondEp:     ep2,
			})
		}
	}

	// If majority or all are TV episodes -> treat as TV Show
	if len(episodes) > 0 && len(episodes) >= len(videoFiles)/2 {
		return s.planTVShow(res, folderPath, folderName, isRoot, episodes, subtitleFiles, seasonsSet)
	}

	// Otherwise treat as Movies
	return s.planMovie(res, folderPath, folderName, isRoot, videoFiles, subtitleFiles)
}

func (s *Scanner) planTVShow(res *ScanResult, folderPath, folderName string, isRoot bool, episodes []*EpisodeFile, subtitles []string, seasonsSet map[int]bool) error {
	res.TotalShows++
	res.TotalFiles += len(episodes)

	// Determine Show Title
	cleanFolderName := CleanTitle(StripSceneTags(folderName))

	var candidateFileTitle string
	if len(episodes) > 0 {
		epName := episodes[0].Filename
		loc := tvPatternSxxExx.FindStringIndex(epName)
		if loc == nil {
			loc = tvPatternNxNN.FindStringIndex(epName)
		}
		if loc != nil && loc[0] > 0 {
			prefix := epName[:loc[0]]
			if hyphenIdx := strings.Index(prefix, "-"); hyphenIdx > 0 {
				prefix = prefix[:hyphenIdx]
			}
			candidateFileTitle = CleanTitle(prefix)
		}
	}

	showTitle := cleanFolderName
	if showTitle == "" || isGenericFolder(showTitle) {
		if candidateFileTitle != "" {
			showTitle = candidateFileTitle
		}
	}
	if showTitle == "" {
		showTitle = "Unknown Show"
	}

	formattedShowTitle := FormatSeparator(showTitle, s.Separator)
	parentDir := filepath.Dir(folderPath)
	targetShowFolderPath := filepath.Join(parentDir, formattedShowTitle)

	if folderName != formattedShowTitle {
		res.Actions = append(res.Actions, Action{
			Type:        ActionRenameFolder,
			SourcePath:  folderPath,
			TargetPath:  targetShowFolderPath,
			DisplayOld:  folderName,
			DisplayNew:  formattedShowTitle,
			Description: fmt.Sprintf("Serien-Ordner: %s", formattedShowTitle),
		})
	}

	hasMultipleSeasons := len(seasonsSet) > 1

	sort.Slice(episodes, func(i, j int) bool {
		if episodes[i].Season != episodes[j].Season {
			return episodes[i].Season < episodes[j].Season
		}
		return episodes[i].Episode < episodes[j].Episode
	})

	// Map each episode to its planned target directory and base name
	type episodeTargetInfo struct {
		targetDirPath string
		baseName      string
		seasonStr     string
		epStr         string
	}
	epTargets := make(map[*EpisodeFile]episodeTargetInfo)

	for _, ep := range episodes {
		seasonStr := fmt.Sprintf("S%02d", ep.Season)
		var epStr string
		if ep.SecondEp > 0 {
			epStr = fmt.Sprintf("S%02dE%02d-E%02d", ep.Season, ep.Episode, ep.SecondEp)
		} else {
			epStr = fmt.Sprintf("S%02dE%02d", ep.Season, ep.Episode)
		}

		rawBaseName := fmt.Sprintf("%s - %s", showTitle, epStr)
		newBaseName := FormatSeparator(rawBaseName, s.Separator)

		var targetDirPath string
		var relTarget string
		if hasMultipleSeasons {
			targetDirPath = filepath.Join(targetShowFolderPath, seasonStr)
			relTarget = filepath.Join(formattedShowTitle, seasonStr, newBaseName+ep.Extension)
		} else {
			targetDirPath = targetShowFolderPath
			relTarget = filepath.Join(formattedShowTitle, newBaseName+ep.Extension)
		}

		finalFilePath := filepath.Join(targetDirPath, newBaseName+ep.Extension)
		relSource, _ := filepath.Rel(parentDir, ep.OriginalPath)

		actionType := ActionRenameFile
		if hasMultipleSeasons || filepath.Dir(ep.OriginalPath) != targetDirPath {
			actionType = ActionMoveFile
		}

		res.Actions = append(res.Actions, Action{
			Type:        actionType,
			SourcePath:  ep.OriginalPath,
			TargetPath:  finalFilePath,
			DisplayOld:  relSource,
			DisplayNew:  relTarget,
			Description: fmt.Sprintf("Episode %s", epStr),
		})

		epTargets[ep] = episodeTargetInfo{
			targetDirPath: targetDirPath,
			baseName:      newBaseName,
			seasonStr:     seasonStr,
			epStr:         epStr,
		}
	}

	// Match and plan subtitles for TV episodes
	// Filter subtitles: ONLY keep German and English, delete all others, deduplicate
	validSubs := filterAndDedupSubtitles(res, subtitles, parentDir)

	// Match and plan remaining DE/EN subtitles for TV episodes
	usedSubPaths := make(map[string]int)

	for _, subInfo := range validSubs {
		var matchedEp *EpisodeFile

		// 1. Try matching by season and episode number
		if subInfo.HasTVEpisode {
			for _, ep := range episodes {
				if ep.Season == subInfo.Season && ep.Episode == subInfo.Episode {
					matchedEp = ep
					break
				}
			}
		}

		// 2. Try matching by filename prefix
		if matchedEp == nil {
			subBase := strings.TrimSuffix(subInfo.Filename, subInfo.Extension)
			for _, ep := range episodes {
				epBase := strings.TrimSuffix(ep.Filename, ep.Extension)
				if strings.HasPrefix(strings.ToLower(subBase), strings.ToLower(epBase)) {
					matchedEp = ep
					break
				}
			}
		}

		// 3. Fallback: If only 1 episode exists
		if matchedEp == nil && len(episodes) == 1 {
			matchedEp = episodes[0]
		}

		if matchedEp != nil {
			info := epTargets[matchedEp]
			subSuffix := SubtitleSuffix(subInfo)
			newSubFilename := info.baseName + subSuffix + ".srt"

			// Handle collision if multiple subtitles have same lang/forced status
			targetSubPath := filepath.Join(info.targetDirPath, newSubFilename)
			if count, exists := usedSubPaths[targetSubPath]; exists {
				usedSubPaths[targetSubPath] = count + 1
				indexedSuffix := fmt.Sprintf("%s.%d", subSuffix, count+1)
				newSubFilename = info.baseName + indexedSuffix + ".srt"
				targetSubPath = filepath.Join(info.targetDirPath, newSubFilename)
			} else {
				usedSubPaths[targetSubPath] = 1
			}

			relSubSource, _ := filepath.Rel(parentDir, subInfo.OriginalPath)
			var relSubTarget string
			if hasMultipleSeasons {
				relSubTarget = filepath.Join(formattedShowTitle, info.seasonStr, newSubFilename)
			} else {
				relSubTarget = filepath.Join(formattedShowTitle, newSubFilename)
			}

			actType, desc := subtitleActionTypeAndDesc(subInfo)

			res.Actions = append(res.Actions, Action{
				Type:        actType,
				SourcePath:  subInfo.OriginalPath,
				TargetPath:  targetSubPath,
				DisplayOld:  relSubSource,
				DisplayNew:  relSubTarget,
				Description: desc,
			})
		}
	}

	return nil
}

func (s *Scanner) planMovie(res *ScanResult, folderPath, folderName string, isRoot bool, videoFiles, subtitles []string) error {
	parentDir := filepath.Dir(folderPath)

	// If there is exactly 1 movie and folderPath is a dedicated movie folder (not generic)
	isDedicatedFolder := len(videoFiles) == 1 && !isGenericFolder(folderName)

	if isDedicatedFolder {
		// Single movie in its own folder: rename folder and file
		vf := videoFiles[0]
		fileName := filepath.Base(vf)
		ext := filepath.Ext(vf)
		baseName := strings.TrimSuffix(fileName, ext)

		title, year := ExtractMovieYearAndTitle(baseName)
		if year == 0 {
			fTitle, fYear := ExtractMovieYearAndTitle(folderName)
			if fYear > 0 {
				year = fYear
				if title == "" {
					title = fTitle
				}
			}
		}
		if title == "" {
			title = CleanTitle(StripSceneTags(baseName))
		}
		if title == "" {
			title = "Movie"
		}

		var cleanMovieName string
		if year > 0 {
			cleanMovieName = fmt.Sprintf("%s (%d)", title, year)
		} else {
			cleanMovieName = title
		}

		formattedMovieName := FormatSeparator(cleanMovieName, s.Separator)
		targetMovieFolderPath := filepath.Join(parentDir, formattedMovieName)

		res.TotalMovies++
		res.TotalFiles++

		if folderName != formattedMovieName {
			res.Actions = append(res.Actions, Action{
				Type:        ActionRenameFolder,
				SourcePath:  folderPath,
				TargetPath:  targetMovieFolderPath,
				DisplayOld:  folderName,
				DisplayNew:  formattedMovieName,
				Description: fmt.Sprintf("Film-Ordner: %s", formattedMovieName),
			})
		}

		newFileName := formattedMovieName + ext
		finalFilePath := filepath.Join(targetMovieFolderPath, newFileName)
		relSource, _ := filepath.Rel(parentDir, vf)
		relTarget := filepath.Join(formattedMovieName, newFileName)

		res.Actions = append(res.Actions, Action{
			Type:        ActionRenameFile,
			SourcePath:  vf,
			TargetPath:  finalFilePath,
			DisplayOld:  relSource,
			DisplayNew:  relTarget,
			Description: fmt.Sprintf("Film: %s", cleanMovieName),
		})

		// Filter subtitles: ONLY keep German and English, delete all others, deduplicate
		validSubs := filterAndDedupSubtitles(res, subtitles, parentDir)

		// Subtitles for this single movie
		usedSubPaths := make(map[string]int)
		for _, subInfo := range validSubs {
			subSuffix := SubtitleSuffix(subInfo)
			newSubName := formattedMovieName + subSuffix + ".srt"

			targetSubPath := filepath.Join(targetMovieFolderPath, newSubName)
			if count, exists := usedSubPaths[targetSubPath]; exists {
				usedSubPaths[targetSubPath] = count + 1
				newSubName = fmt.Sprintf("%s%s.%d.srt", formattedMovieName, subSuffix, count+1)
				targetSubPath = filepath.Join(targetMovieFolderPath, newSubName)
			} else {
				usedSubPaths[targetSubPath] = 1
			}

			relSubSource, _ := filepath.Rel(parentDir, subInfo.OriginalPath)
			relSubTarget := filepath.Join(formattedMovieName, newSubName)

			actType, desc := subtitleActionTypeAndDesc(subInfo)

			res.Actions = append(res.Actions, Action{
				Type:        actType,
				SourcePath:  subInfo.OriginalPath,
				TargetPath:  targetSubPath,
				DisplayOld:  relSubSource,
				DisplayNew:  relSubTarget,
				Description: desc,
			})
		}

		return nil
	}

	// Filter subtitles: ONLY keep German and English, delete all others, deduplicate
	validSubs := filterAndDedupSubtitles(res, subtitles, folderPath)

	// If there are multiple movies (or loose movies in a container folder):
	// "wird für jeden film ein ordner erstellt gleich benannt wie der film. in diesen ordner liegt dann der film."
	for _, vf := range videoFiles {
		fileName := filepath.Base(vf)
		ext := filepath.Ext(vf)
		baseName := strings.TrimSuffix(fileName, ext)

		title, year := ExtractMovieYearAndTitle(baseName)
		if title == "" {
			title = CleanTitle(StripSceneTags(baseName))
		}
		if title == "" {
			title = "Movie"
		}

		var cleanMovieName string
		if year > 0 {
			cleanMovieName = fmt.Sprintf("%s (%d)", title, year)
		} else {
			cleanMovieName = title
		}

		formattedMovieName := FormatSeparator(cleanMovieName, s.Separator)
		targetMovieFolderPath := filepath.Join(folderPath, formattedMovieName)

		res.TotalMovies++
		res.TotalFiles++

		// Create dedicated folder for this movie
		res.Actions = append(res.Actions, Action{
			Type:        ActionCreateFolder,
			SourcePath:  "",
			TargetPath:  targetMovieFolderPath,
			DisplayOld:  "(Neu)",
			DisplayNew:  formattedMovieName,
			Description: fmt.Sprintf("Ordner für Film: %s", formattedMovieName),
		})

		newFileName := formattedMovieName + ext
		finalFilePath := filepath.Join(targetMovieFolderPath, newFileName)
		relSource, _ := filepath.Rel(folderPath, vf)
		relTarget := filepath.Join(formattedMovieName, newFileName)

		res.Actions = append(res.Actions, Action{
			Type:        ActionMoveFile,
			SourcePath:  vf,
			TargetPath:  finalFilePath,
			DisplayOld:  relSource,
			DisplayNew:  relTarget,
			Description: fmt.Sprintf("Film: %s", cleanMovieName),
		})

		// Find matching subtitles for this movie
		usedSubPaths := make(map[string]int)
		for _, subInfo := range validSubs {
			if matchSubtitleToMovie(subInfo.OriginalPath, baseName, len(videoFiles) == 1) {
				subSuffix := SubtitleSuffix(subInfo)
				newSubName := formattedMovieName + subSuffix + ".srt"

				targetSubPath := filepath.Join(targetMovieFolderPath, newSubName)
				if count, exists := usedSubPaths[targetSubPath]; exists {
					usedSubPaths[targetSubPath] = count + 1
					newSubName = fmt.Sprintf("%s%s.%d.srt", formattedMovieName, subSuffix, count+1)
					targetSubPath = filepath.Join(targetMovieFolderPath, newSubName)
				} else {
					usedSubPaths[targetSubPath] = 1
				}

				relSubSource, _ := filepath.Rel(folderPath, subInfo.OriginalPath)
				relSubTarget := filepath.Join(formattedMovieName, newSubName)

				actType, desc := subtitleActionTypeAndDesc(subInfo)

				res.Actions = append(res.Actions, Action{
					Type:        actType,
					SourcePath:  subInfo.OriginalPath,
					TargetPath:  targetSubPath,
					DisplayOld:  relSubSource,
					DisplayNew:  relSubTarget,
					Description: desc,
				})
			}
		}
	}

	return nil
}

func (s *Scanner) planActionsForLooseFiles(res *ScanResult, rootDir string, videoFiles, subtitleFiles []string) {
	// Filter subtitles: ONLY keep German and English, delete all others, deduplicate
	validSubs := filterAndDedupSubtitles(res, subtitleFiles, rootDir)

	for _, vf := range videoFiles {
		filename := filepath.Base(vf)
		ext := filepath.Ext(vf)
		baseName := strings.TrimSuffix(filename, ext)

		// Check if TV episode
		season, ep, ep2, isTV := ExtractEpisodeInfo(filename)
		if isTV {
			res.TotalShows++
			res.TotalFiles++

			loc := tvPatternSxxExx.FindStringIndex(filename)
			if loc == nil {
				loc = tvPatternNxNN.FindStringIndex(filename)
			}
			var rawShowTitle string
			if loc != nil && loc[0] > 0 {
				prefix := filename[:loc[0]]
				if hyphenIdx := strings.Index(prefix, "-"); hyphenIdx > 0 {
					prefix = prefix[:hyphenIdx]
				}
				rawShowTitle = prefix
			}
			showTitle := CleanTitle(rawShowTitle)
			if showTitle == "" {
				showTitle = "TV Show"
			}

			var epStr string
			if ep2 > 0 {
				epStr = fmt.Sprintf("S%02dE%02d-E%02d", season, ep, ep2)
			} else {
				epStr = fmt.Sprintf("S%02dE%02d", season, ep)
			}

			rawBaseName := fmt.Sprintf("%s - %s", showTitle, epStr)
			formattedBaseName := FormatSeparator(rawBaseName, s.Separator)
			newFilename := formattedBaseName + ext

			finalFilePath := filepath.Join(rootDir, newFilename)
			res.Actions = append(res.Actions, Action{
				Type:        ActionRenameFile,
				SourcePath:  vf,
				TargetPath:  finalFilePath,
				DisplayOld:  filename,
				DisplayNew:  newFilename,
				Description: fmt.Sprintf("Serie: %s %s", showTitle, epStr),
			})

			// Match subtitles for this loose TV episode
			usedSubPaths := make(map[string]int)
			for _, subInfo := range validSubs {
				matched := false
				if subInfo.HasTVEpisode && subInfo.Season == season && subInfo.Episode == ep {
					matched = true
				} else {
					subBase := strings.TrimSuffix(subInfo.Filename, subInfo.Extension)
					epBase := strings.TrimSuffix(filename, ext)
					if strings.HasPrefix(strings.ToLower(subBase), strings.ToLower(epBase)) {
						matched = true
					}
				}

				if matched {
					subSuffix := SubtitleSuffix(subInfo)
					newSubName := formattedBaseName + subSuffix + ".srt"
					targetSubPath := filepath.Join(rootDir, newSubName)
					if count, exists := usedSubPaths[targetSubPath]; exists {
						usedSubPaths[targetSubPath] = count + 1
						newSubName = fmt.Sprintf("%s%s.%d.srt", formattedBaseName, subSuffix, count+1)
						targetSubPath = filepath.Join(rootDir, newSubName)
					} else {
						usedSubPaths[targetSubPath] = 1
					}

					actType, desc := subtitleActionTypeAndDesc(subInfo)
					if actType == ActionMoveFile {
						actType = ActionRenameFile
					}

					res.Actions = append(res.Actions, Action{
						Type:        actType,
						SourcePath:  subInfo.OriginalPath,
						TargetPath:  targetSubPath,
						DisplayOld:  filepath.Base(subInfo.OriginalPath),
						DisplayNew:  newSubName,
						Description: desc,
					})
				}
			}
			continue
		}

		// Otherwise Movie:
		// "wird für jeden film ein ordner erstellt gleich benannt wie der film. in diesen ordner liegt dann der film."
		res.TotalMovies++
		res.TotalFiles++

		title, year := ExtractMovieYearAndTitle(baseName)
		if title == "" {
			title = CleanTitle(StripSceneTags(baseName))
		}
		if title == "" {
			title = "Movie"
		}

		var cleanName string
		if year > 0 {
			cleanName = fmt.Sprintf("%s (%d)", title, year)
		} else {
			cleanName = title
		}

		formattedName := FormatSeparator(cleanName, s.Separator)
		targetMovieFolderPath := filepath.Join(rootDir, formattedName)

		// Create dedicated folder for this movie
		res.Actions = append(res.Actions, Action{
			Type:        ActionCreateFolder,
			SourcePath:  "",
			TargetPath:  targetMovieFolderPath,
			DisplayOld:  "(Neu)",
			DisplayNew:  formattedName,
			Description: fmt.Sprintf("Ordner für Film: %s", formattedName),
		})

		newFileName := formattedName + ext
		finalFilePath := filepath.Join(targetMovieFolderPath, newFileName)

		res.Actions = append(res.Actions, Action{
			Type:        ActionMoveFile,
			SourcePath:  vf,
			TargetPath:  finalFilePath,
			DisplayOld:  filename,
			DisplayNew:  filepath.Join(formattedName, newFileName),
			Description: fmt.Sprintf("Film: %s", cleanName),
		})

		// Find matching loose subtitles
		usedSubPaths := make(map[string]int)
		for _, subInfo := range validSubs {
			if matchSubtitleToMovie(subInfo.OriginalPath, baseName, len(videoFiles) == 1) {
				subSuffix := SubtitleSuffix(subInfo)
				newSubName := formattedName + subSuffix + ".srt"

				targetSubPath := filepath.Join(targetMovieFolderPath, newSubName)
				if count, exists := usedSubPaths[targetSubPath]; exists {
					usedSubPaths[targetSubPath] = count + 1
					newSubName = fmt.Sprintf("%s%s.%d.srt", formattedName, subSuffix, count+1)
					targetSubPath = filepath.Join(targetMovieFolderPath, newSubName)
				} else {
					usedSubPaths[targetSubPath] = 1
				}

				actType, desc := subtitleActionTypeAndDesc(subInfo)

				res.Actions = append(res.Actions, Action{
					Type:        actType,
					SourcePath:  subInfo.OriginalPath,
					TargetPath:  targetSubPath,
					DisplayOld:  filepath.Base(subInfo.OriginalPath),
					DisplayNew:  filepath.Join(formattedName, newSubName),
					Description: desc,
				})
			}
		}
	}
}

// matchSubtitleToMovie checks whether a subtitle belongs to a specific movie base name.
func matchSubtitleToMovie(subPath, movieBaseName string, isSoleMovie bool) bool {
	if isSoleMovie {
		return true // If only 1 movie exists, all non-TV subtitles belong to it
	}

	subFile := filepath.Base(subPath)
	subBase := strings.TrimSuffix(subFile, filepath.Ext(subFile))

	// Direct prefix match: sub starts with movie base name
	if strings.HasPrefix(strings.ToLower(subBase), strings.ToLower(movieBaseName)) {
		return true
	}

	// Check clean title match
	cleanMovie, _ := ExtractMovieYearAndTitle(movieBaseName)
	cleanMovie = strings.ToLower(strings.ReplaceAll(cleanMovie, " ", ""))
	cleanSub := strings.ToLower(strings.ReplaceAll(subBase, " ", ""))
	cleanSub = strings.ReplaceAll(cleanSub, "_", "")
	cleanSub = strings.ReplaceAll(cleanSub, ".", "")

	if cleanMovie != "" && strings.Contains(cleanSub, cleanMovie) {
		idx := strings.Index(cleanSub, cleanMovie)
		afterIdx := idx + len(cleanMovie)
		if afterIdx >= len(cleanSub) {
			return true // Exact match or end of string
		}
		nextChar := cleanSub[afterIdx]
		// Allow non-alphanumeric after match (language codes, tags, etc.)
		if !((nextChar >= 'a' && nextChar <= 'z') || (nextChar >= '0' && nextChar <= '9')) {
			return true
		}
	}

	return false
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isGenericFolder(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "downloads", "movies", "filme", "series", "serien", "tv", "tv shows", "media", "torrents", "complete", "incoming", "videos":
		return true
	default:
		return false
	}
}

func subtitleActionTypeAndDesc(subInfo SubtitleInfo) (ActionType, string) {
	if strings.ToLower(subInfo.Extension) != ".srt" {
		origExt := strings.ToUpper(strings.TrimPrefix(subInfo.Extension, "."))
		return ActionConvertSubtitle, fmt.Sprintf("Untertitel (%s, %s ➔ SRT konvertieren)", subInfo.LangCode, origExt)
	}
	return ActionMoveFile, fmt.Sprintf("Untertitel (%s)", subInfo.LangCode)
}

// filterAndDedupSubtitles filters subtitles to DE/EN, creates delete actions for
// foreign languages, deduplicates identical content, and creates delete actions
// for duplicates. Returns the surviving subtitles.
func filterAndDedupSubtitles(res *ScanResult, subtitles []string, relBaseDir string) []SubtitleInfo {
	// Step 1: Language filtering
	var validSubs []SubtitleInfo
	for _, sf := range subtitles {
		subInfo := ParseSubtitle(sf)
		langRes := DetectSubtitleLanguage(sf, subInfo)
		if langRes.Language != "de" && langRes.Language != "en" {
			relSubSource, _ := filepath.Rel(relBaseDir, sf)
			res.Actions = append(res.Actions, Action{
				Type:        ActionDeleteFile,
				SourcePath:  sf,
				TargetPath:  "",
				DisplayOld:  relSubSource,
				DisplayNew:  "(Wird gelöscht)",
				Description: fmt.Sprintf("Untertitel entfernen (%s, nicht DE/EN)", langRes.Details),
			})
			continue
		}
		subInfo.LangCode = langRes.Language
		validSubs = append(validSubs, subInfo)
	}

	// Step 2: Deduplication
	kept, duplicates := DeduplicateSubtitles(validSubs)
	for _, dup := range duplicates {
		relDupSource, _ := filepath.Rel(relBaseDir, dup.Duplicate.OriginalPath)
		keptName := filepath.Base(dup.KeptOriginal.OriginalPath)
		res.Actions = append(res.Actions, Action{
			Type:        ActionDeleteDuplicate,
			SourcePath:  dup.Duplicate.OriginalPath,
			TargetPath:  "",
			DisplayOld:  relDupSource,
			DisplayNew:  "(Wird gelöscht)",
			Description: fmt.Sprintf("Doppelter Untertitel (%s, identisch mit %s)", dup.Duplicate.LangCode, keptName),
		})
	}

	return kept
}

