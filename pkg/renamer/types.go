package renamer

// ActionType denotes what kind of filesystem operation will be performed.
type ActionType string

const (
	ActionRenameFolder    ActionType = "RENAME_FOLDER"
	ActionCreateFolder    ActionType = "CREATE_FOLDER"
	ActionMoveFile        ActionType = "MOVE_FILE"
	ActionRenameFile      ActionType = "RENAME_FILE"
	ActionDeleteFile      ActionType = "DELETE_FILE"
	ActionConvertSubtitle ActionType = "CONVERT_SUBTITLE"
	ActionDeleteDuplicate ActionType = "DELETE_DUPLICATE"
)

// Action represents a single planned file or directory rename/move.
type Action struct {
	Type        ActionType
	SourcePath  string // Absolute path of source
	TargetPath  string // Absolute path of destination
	DisplayOld  string // Formatted for preview UI
	DisplayNew  string // Formatted for preview UI
	Description string // Optional explanation (e.g. "Staffel 2 Ordner erstellen")
}

// ItemType indicates whether an item is a Movie or a TV Show.
type ItemType string

const (
	ItemTypeMovie   ItemType = "Film"
	ItemTypeTVShow  ItemType = "Serie"
	ItemTypeUnknown ItemType = "Unbekannt"
)

// EpisodeFile holds metadata for an individual TV show episode file.
type EpisodeFile struct {
	OriginalPath string
	Filename     string
	Extension    string
	Season       int
	Episode      int
	SecondEp     int
	Subtitles    []string // Accompanying subtitle files
}

// TVShowGroup represents a detected TV show collection in a directory.
type TVShowGroup struct {
	Title        string
	FolderPath   string // Directory containing the episodes
	IsRootFolder bool   // Whether FolderPath is the root scan directory
	Episodes     []*EpisodeFile
	Seasons      map[int]bool
}

// MovieItem represents a detected movie file or folder.
type MovieItem struct {
	Title        string
	Year         int
	FolderPath   string
	IsFolder     bool // true if whole folder is dedicated to this movie
	FilePath     string
	Extension    string
	Subtitles    []string
	IsRootFolder bool
}

// ScanResult holds all detected items and planned actions.
type ScanResult struct {
	RootPath     string
	Separator    SeparatorMode
	UseSpaces    bool
	TotalMovies  int
	TotalShows   int
	TotalFiles   int
	Actions      []Action
	Warnings     []string
}
