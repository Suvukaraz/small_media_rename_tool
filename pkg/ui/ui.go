package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"smrt/pkg/presets"
	"smrt/pkg/renamer"
)

type screenState int

const (
	screenInput screenState = iota
	screenPreview
	screenResult
	screenPresets
)

type focusedField int

const (
	fieldMoviePath focusedField = iota
	fieldShowPath
	fieldDropdown
	fieldButtonGo
	fieldButtonPresets
	fieldPath = fieldMoviePath // alias for backward compatibility
)

type presetField int

const (
	presetFieldNum presetField = iota
	presetFieldPath
	presetFieldSave
	presetFieldDelete
	presetFieldBack
)

type resultField int

const (
	resultFieldNew resultField = iota
	resultFieldQuit
)

type clickArea struct {
	id   string
	minX int
	maxX int
	minY int
	maxY int
}

// Separator options metadata with descriptive examples
var separatorOptions = []struct {
	mode    renamer.SeparatorMode
	name    string
	example string
}{
	{renamer.SeparatorSpace, "Leerzeichen ( )", "The Film Title (2009)"},
	{renamer.SeparatorDot, "Punkt (.)", "The.Film.Title.(2009)"},
	{renamer.SeparatorUnderscore, "Unterstrich (_)", "The_Film_Title_(2009)"},
}

// UI Model holds all state for the Bubble Tea application.
type Model struct {
	state screenState

	// Screen 1: Input (Two separate paths for Movies and Series)
	inputMoviePath textinput.Model
	inputShowPath  textinput.Model
	inputPath      textinput.Model // kept for backward compatibility with existing tests
	separator      renamer.SeparatorMode
	dropdownOpen   bool
	dropdownIdx    int
	focus          focusedField
	inputErr       string

	// Screen 2: Preview
	scanResult   *renamer.ScanResult
	viewport     viewport.Model
	readyVP      bool
	previewErr   string

	// Screen 3: Result
	execResult  *renamer.ExecutionResult
	resultFocus resultField

	// Presets manager
	presets *presets.Manager

	// Screen 4: Presets Submenu
	presetNumInput  textinput.Model
	presetPathInput textinput.Model
	presetFocus     presetField
	presetErr       string
	presetMsg       string

	// Terminal dimensions
	width  int
	height int

	// Touch & Mouse clickable areas
	clickAreas    *[]clickArea
	lastClickTime time.Time
	pressHandled  bool
	touchStartX   int
	touchStartY   int
	lastTouchY    int
	hasTouchDown  bool
	isDragging    bool
}

// Styling definitions with Lipgloss
var (
	subtleColor   = lipgloss.Color("#767676")
	primaryColor  = lipgloss.Color("#7D56F4")
	accentColor   = lipgloss.Color("#04B575")
	dangerColor   = lipgloss.Color("#FF5F87")
	warningColor  = lipgloss.Color("#FFD700")
	cyanColor     = lipgloss.Color("#00D7D7")
	borderColor   = lipgloss.Color("#444444")
	cardBg        = lipgloss.Color("#1E1E28")
	selectedCard  = lipgloss.Color("#2D1B69")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(primaryColor).
			Padding(0, 2)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E0E0E0"))

	focusedLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	activeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	touchCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	activeTouchCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	selectedOptionCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Background(selectedCard).
				Padding(0, 1)

	btnBigStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#2A2A35")).
			Bold(true).
			Align(lipgloss.Center)

	btnBigActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Foreground(lipgloss.Color("#000000")).
				Background(accentColor).
				Bold(true).
				Align(lipgloss.Center)

	btnSecondaryStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Foreground(lipgloss.Color("#DDDDDD")).
				Background(lipgloss.Color("#2A2A35")).
				Bold(true).
				Align(lipgloss.Center)

	btnNewActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Foreground(lipgloss.Color("#000000")).
				Background(accentColor).
				Bold(true).
				Align(lipgloss.Center)

	btnNewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#065F46")).
			Bold(true).
			Align(lipgloss.Center)

	btnQuitActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFFFFF")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#EF4444")).
				Bold(true).
				Align(lipgloss.Center)

	btnQuitStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#EF4444")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#991B1B")).
			Bold(true).
			Align(lipgloss.Center)

	btnScrollStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#555566")).
				Foreground(lipgloss.Color("#EEEEEE")).
				Background(lipgloss.Color("#20202A")).
				Bold(true).
				Align(lipgloss.Center)

	oldPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA07A"))

	newPathStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	actionTypeStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor)
)

func isDirWithEpisodes(dirPath string) bool {
	if dirPath == "" {
		return false
	}
	clean := renamer.ExpandPath(dirPath)
	entries, err := os.ReadDir(clean)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, _, _, matched := renamer.ExtractEpisodeInfoFromPath(name); matched {
			return true
		}
		if e.IsDir() {
			low := strings.ToLower(name)
			if strings.HasPrefix(low, "season") || strings.HasPrefix(low, "staffel") || strings.HasPrefix(low, "s0") || strings.HasPrefix(low, "s1") {
				return true
			}
		}
	}
	return false
}

// InitialModel configures and initializes the TUI model.
func InitialModel() Model {
	return InitialModelWithPaths("", "")
}

// InitialModelWithPath configures the TUI model with an initial path (backward compatibility).
func InitialModelWithPath(initialPath string) Model {
	lower := strings.ToLower(initialPath)
	if strings.Contains(lower, "serie") || strings.Contains(lower, "show") || strings.Contains(lower, "season") || strings.Contains(lower, "staffel") || isDirWithEpisodes(initialPath) {
		return InitialModelWithPaths("", initialPath)
	}
	return InitialModelWithPaths(initialPath, "")
}

// InitialModelWithPresets configures the TUI model with separate initial movie and show paths and a preset manager.
func InitialModelWithPresets(moviePath, showPath string, pm *presets.Manager) Model {
	if pm == nil {
		var err error
		pm, err = presets.Load()
		if err != nil || pm == nil {
			pm, _ = presets.LoadFrom(presets.DefaultConfigPath())
		}
	}

	tiMovie := textinput.New()
	tiMovie.Placeholder = "🎬 Pfad oder Nr. 1-99 (optional)"
	if moviePath != "" {
		tiMovie.SetValue(moviePath)
	}
	tiMovie.CharLimit = 1024
	tiMovie.Width = 50

	tiShow := textinput.New()
	tiShow.Placeholder = "📺 Pfad oder Nr. 1-99 (optional)"
	if showPath != "" {
		tiShow.SetValue(showPath)
	}
	tiShow.CharLimit = 1024
	tiShow.Width = 50

	tiPresetNum := textinput.New()
	tiPresetNum.Placeholder = "1-99"
	tiPresetNum.CharLimit = 2
	tiPresetNum.Width = 8

	tiPresetPath := textinput.New()
	tiPresetPath.Placeholder = "/pfad/zu/ordner"
	tiPresetPath.CharLimit = 1024
	tiPresetPath.Width = 45

	initialFocus := fieldMoviePath
	tiCompat := tiMovie
	if moviePath == "" && showPath != "" {
		tiShow.Focus()
		tiCompat = tiShow
		initialFocus = fieldShowPath
	} else {
		tiMovie.Focus()
	}

	return Model{
		state:           screenInput,
		inputMoviePath:  tiMovie,
		inputShowPath:   tiShow,
		inputPath:       tiCompat, // compatibility alias
		separator:       renamer.SeparatorSpace, // Default: Leerzeichen
		dropdownOpen:    false,
		dropdownIdx:     0,
		focus:           initialFocus,
		presets:         pm,
		presetNumInput:  tiPresetNum,
		presetPathInput: tiPresetPath,
		presetFocus:     presetFieldNum,
		width:           80,
		height:          24,
		clickAreas:      new([]clickArea),
	}
}

// InitialModelWithPaths configures the TUI model with separate initial movie and show paths.
func InitialModelWithPaths(moviePath, showPath string) Model {
	return InitialModelWithPresets(moviePath, showPath, nil)
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Responsive widths for mobile & desktop
		cardWidth := max(24, min(m.width-2, 68))
		m.inputMoviePath.Width = cardWidth - 4
		m.inputShowPath.Width = cardWidth - 4
		m.inputPath.Width = cardWidth - 4

		vpWidth := max(24, m.width-4)
		if !m.readyVP {
			m.viewport = viewport.New(vpWidth, 10)
			m.readyVP = true
		} else {
			m.viewport.Width = vpWidth
		}

	case tea.MouseMsg:
		// Mouse wheel scrolling in preview (always works anywhere)
		if msg.Button == tea.MouseButtonWheelUp && m.state == screenPreview {
			m.viewport.LineUp(3)
			return m, nil
		} else if msg.Button == tea.MouseButtonWheelDown && m.state == screenPreview {
			m.viewport.LineDown(3)
			return m, nil
		}

		isPress := msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonLeft || msg.Button == tea.MouseButtonNone)
		isMotion := msg.Action == tea.MouseActionMotion
		isRelease := msg.Action == tea.MouseActionRelease && (msg.Button == tea.MouseButtonLeft || msg.Button == tea.MouseButtonNone)

		if isPress {
			area := m.findClickArea(msg.X, msg.Y)
			if isButtonOrInputArea(area) {
				// Immediate responsive trigger on press for ALL buttons and input fields across ALL screens
				m.pressHandled = true
				m.hasTouchDown = false
				m.isDragging = false
				m.lastClickTime = time.Now()
				return m.handleMouseClick(msg.X, msg.Y)
			}

			// Touch down on viewport or non-button background
			m.hasTouchDown = true
			m.isDragging = false
			m.touchStartX = msg.X
			m.touchStartY = msg.Y
			m.lastTouchY = msg.Y
			m.lastClickTime = time.Now()
			return m, nil
		}

		// Touch swipe/drag on viewport in preview (only if finger moved by >= 1 row)
		if isMotion && m.hasTouchDown && m.state == screenPreview {
			totalDragY := msg.Y - m.touchStartY
			if totalDragY < 0 {
				totalDragY = -totalDragY
			}
			if totalDragY >= 1 {
				m.isDragging = true
			}

			if m.isDragging {
				deltaY := msg.Y - m.lastTouchY
				if deltaY > 0 {
					m.viewport.LineUp(deltaY)
				} else if deltaY < 0 {
					m.viewport.LineDown(-deltaY)
				}
				m.lastTouchY = msg.Y
				return m, nil
			}
		}

		if isRelease {
			wasDragging := m.isDragging
			m.hasTouchDown = false
			m.isDragging = false

			if wasDragging {
				// User was scrolling the viewport, ignore tap!
				return m, nil
			}

			if m.pressHandled {
				// Already handled on press; ignore release
				m.pressHandled = false
				return m, nil
			}

			// In case of terminals that only send release, or viewport tap-to-scroll
			m.pressHandled = false
			return m.handleMouseClick(msg.X, msg.Y)
		}

	case tea.KeyMsg:
		m.pressHandled = false
		switch m.state {
		case screenInput:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			// Global shortcut to switch separator: Ctrl+U
			case "ctrl+u":
				m.cycleSeparator()
				return m, nil

			case "ctrl+p":
				m.state = screenPresets
				m.presetErr = ""
				m.presetMsg = ""
				m.presetNumInput.Focus()
				m.presetPathInput.Blur()
				m.presetFocus = presetFieldNum
				return m, nil

			case "esc":
				if m.dropdownOpen {
					m.dropdownOpen = false
					return m, nil
				}

			case "tab", "down":
				if m.dropdownOpen {
					m.dropdownIdx = (m.dropdownIdx + 1) % len(separatorOptions)
					m.separator = separatorOptions[m.dropdownIdx].mode
					return m, nil
				}
				m.cycleFocus(1)
				return m, nil

			case "shift+tab", "up":
				if m.dropdownOpen {
					m.dropdownIdx = (m.dropdownIdx + len(separatorOptions) - 1) % len(separatorOptions)
					m.separator = separatorOptions[m.dropdownIdx].mode
					return m, nil
				}
				m.cycleFocus(-1)
				return m, nil

			case "enter":
				if m.dropdownOpen {
					m.separator = separatorOptions[m.dropdownIdx].mode
					m.dropdownOpen = false
					return m, nil
				}
				if m.focus == fieldDropdown {
					m.dropdownOpen = true
					return m, nil
				}
				if m.focus == fieldButtonPresets {
					m.state = screenPresets
					m.presetErr = ""
					m.presetMsg = ""
					m.presetNumInput.Focus()
					m.presetPathInput.Blur()
					m.presetFocus = presetFieldNum
					return m, nil
				}
				// Start scan on enter
				return m.startScan()
			}

			// When text input is focused, typing (including spaces) goes strictly into textinput
			if !m.dropdownOpen {
				if m.focus == fieldMoviePath {
					m.inputMoviePath, cmd = m.inputMoviePath.Update(msg)
					m.inputPath = m.inputMoviePath
					return m, cmd
				} else if m.focus == fieldShowPath {
					m.inputShowPath, cmd = m.inputShowPath.Update(msg)
					return m, cmd
				}
			}

		case screenPreview:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "esc", "b", "backspace":
				m.state = screenInput
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return m, nil

			case "enter":
				if m.scanResult != nil && len(m.scanResult.Actions) > 0 {
					return m.startExecute()
				}
			}

			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd

		case screenResult:
			switch msg.String() {
			case "ctrl+c", "q", "esc", "x":
				return m, tea.Quit

			case "tab", "shift+tab", "left", "right", "up", "down":
				if m.resultFocus == resultFieldNew {
					m.resultFocus = resultFieldQuit
				} else {
					m.resultFocus = resultFieldNew
				}
				return m, nil

			case "enter":
				if m.resultFocus == resultFieldQuit {
					return m, tea.Quit
				}
				m.state = screenInput
				m.scanResult = nil
				m.execResult = nil
				m.inputErr = ""
				m.resultFocus = resultFieldNew
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return m, nil

			case "n":
				m.state = screenInput
				m.scanResult = nil
				m.execResult = nil
				m.inputErr = ""
				m.resultFocus = resultFieldNew
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return m, nil
			}

		case screenPresets:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.state = screenInput
				m.presetErr = ""
				m.presetMsg = ""
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return m, nil

			case "tab", "down":
				m.cyclePresetFocus(1)
				return m, nil

			case "shift+tab", "up":
				m.cyclePresetFocus(-1)
				return m, nil

			case "enter":
				return m.handlePresetEnter()
			}

			if m.presetFocus == presetFieldNum {
				m.presetNumInput, cmd = m.presetNumInput.Update(msg)
				return m, cmd
			} else if m.presetFocus == presetFieldPath {
				m.presetPathInput, cmd = m.presetPathInput.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m Model) findClickArea(x, y int) *clickArea {
	if m.clickAreas == nil {
		return nil
	}
	for i := range *m.clickAreas {
		a := &(*m.clickAreas)[i]
		if x >= a.minX && x <= a.maxX && y >= a.minY && y <= a.maxY {
			return a
		}
	}
	return nil
}

func isButtonOrInputArea(area *clickArea) bool {
	if area == nil {
		return false
	}
	return area.id != "vp_up" && area.id != "vp_down" && area.id != "btn_scroll_up" && area.id != "btn_scroll_down"
}

func (m *Model) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	if m.clickAreas == nil {
		return *m, nil
	}

	for _, a := range *m.clickAreas {
		if x >= a.minX && x <= a.maxX && y >= a.minY && y <= a.maxY {
			switch a.id {
			case "input_movie", "input":
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				m.dropdownOpen = false
				textStartX := a.minX + 4
				if x >= textStartX {
					m.inputMoviePath.SetCursor(x - textStartX)
					m.inputPath.SetCursor(x - textStartX)
				}
				return *m, nil

			case "input_show":
				m.focus = fieldShowPath
				m.inputShowPath.Focus()
				m.inputMoviePath.Blur()
				m.dropdownOpen = false
				textStartX := a.minX + 4
				if x >= textStartX {
					m.inputShowPath.SetCursor(x - textStartX)
				}
				return *m, nil

			case "dropdown_header":
				m.focus = fieldDropdown
				m.inputMoviePath.Blur()
				m.inputShowPath.Blur()
				m.dropdownOpen = !m.dropdownOpen
				return *m, nil

			case "opt_0":
				m.separator = renamer.SeparatorSpace
				m.dropdownIdx = 0
				m.dropdownOpen = false
				return *m, nil

			case "opt_1":
				m.separator = renamer.SeparatorDot
				m.dropdownIdx = 1
				m.dropdownOpen = false
				return *m, nil

			case "opt_2":
				m.separator = renamer.SeparatorUnderscore
				m.dropdownIdx = 2
				m.dropdownOpen = false
				return *m, nil

			case "btn_go":
				m.focus = fieldButtonGo
				m.dropdownOpen = false
				return m.startScan()

			case "vp_up", "btn_scroll_up":
				m.viewport.LineUp(4)
				return *m, nil

			case "vp_down", "btn_scroll_down":
				m.viewport.LineDown(4)
				return *m, nil

			case "btn_confirm":
				return m.startExecute()

			case "btn_cancel":
				m.state = screenInput
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return *m, nil

			case "btn_new":
				m.state = screenInput
				m.scanResult = nil
				m.execResult = nil
				m.inputErr = ""
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return *m, nil

			case "btn_quit":
				return *m, tea.Quit

			case "btn_presets":
				m.state = screenPresets
				m.presetErr = ""
				m.presetMsg = ""
				m.presetNumInput.Focus()
				m.presetPathInput.Blur()
				m.presetFocus = presetFieldNum
				return *m, nil

			case "preset_input_num":
				m.presetFocus = presetFieldNum
				m.presetNumInput.Focus()
				m.presetPathInput.Blur()
				textStartX := a.minX + 2
				if x >= textStartX {
					m.presetNumInput.SetCursor(x - textStartX)
				}
				return *m, nil

			case "preset_input_path":
				m.presetFocus = presetFieldPath
				m.presetPathInput.Focus()
				m.presetNumInput.Blur()
				textStartX := a.minX + 2
				if x >= textStartX {
					m.presetPathInput.SetCursor(x - textStartX)
				}
				return *m, nil

			case "preset_btn_save":
				m.presetFocus = presetFieldSave
				return m.handlePresetEnter()

			case "preset_btn_delete":
				m.presetFocus = presetFieldDelete
				return m.handlePresetEnter()

			case "preset_btn_back":
				m.state = screenInput
				m.presetErr = ""
				m.presetMsg = ""
				m.focus = fieldMoviePath
				m.inputMoviePath.Focus()
				m.inputShowPath.Blur()
				return *m, nil

			default:
				if strings.HasPrefix(a.id, "preset_item_") {
					numStr := strings.TrimPrefix(a.id, "preset_item_")
					if num, err := strconv.Atoi(numStr); err == nil && m.presets != nil {
						if p, ok := m.presets.Get(num); ok {
							m.presetNumInput.SetValue(strconv.Itoa(num))
							m.presetPathInput.SetValue(p)
							m.presetMsg = fmt.Sprintf("Preset #%d geladen.", num)
							m.presetErr = ""
							m.presetFocus = presetFieldPath
							m.presetPathInput.Focus()
							m.presetNumInput.Blur()
							return *m, nil
						}
					}
				}
			}
		}
	}

	// Tapping outside dropdown closes it
	if m.dropdownOpen {
		m.dropdownOpen = false
	}

	return *m, nil
}

func (m *Model) cycleFocus(delta int) {
	m.dropdownOpen = false
	m.focus = (m.focus + focusedField(delta) + 5) % 5
	switch m.focus {
	case fieldMoviePath:
		m.inputMoviePath.Focus()
		m.inputShowPath.Blur()
	case fieldShowPath:
		m.inputShowPath.Focus()
		m.inputMoviePath.Blur()
	default:
		m.inputMoviePath.Blur()
		m.inputShowPath.Blur()
	}
}

func (m *Model) cyclePresetFocus(delta int) {
	m.presetFocus = (m.presetFocus + presetField(delta) + 5) % 5
	switch m.presetFocus {
	case presetFieldNum:
		m.presetNumInput.Focus()
		m.presetPathInput.Blur()
	case presetFieldPath:
		m.presetPathInput.Focus()
		m.presetNumInput.Blur()
	default:
		m.presetNumInput.Blur()
		m.presetPathInput.Blur()
	}
}

func (m *Model) handlePresetEnter() (tea.Model, tea.Cmd) {
	if m.presetFocus == presetFieldBack {
		m.state = screenInput
		m.presetErr = ""
		m.presetMsg = ""
		m.focus = fieldMoviePath
		m.inputMoviePath.Focus()
		m.inputShowPath.Blur()
		return *m, nil
	}

	numStr := strings.TrimSpace(m.presetNumInput.Value())
	num, ok := presets.ParsePresetNumber(numStr)
	if !ok {
		m.presetErr = "Bitte eine gültige Nummer von 1 bis 99 eingeben."
		m.presetMsg = ""
		return *m, nil
	}

	if m.presetFocus == presetFieldDelete {
		if m.presets != nil {
			if err := m.presets.Delete(num); err != nil {
				m.presetErr = err.Error()
				m.presetMsg = ""
			} else {
				m.presetMsg = fmt.Sprintf("Preset #%d gelöscht.", num)
				m.presetErr = ""
				m.presetNumInput.SetValue("")
				m.presetPathInput.SetValue("")
			}
		}
		return *m, nil
	}

	// Save preset
	path := strings.TrimSpace(m.presetPathInput.Value())
	if path == "" {
		m.presetErr = "Bitte einen Pfad für das Preset angeben."
		m.presetMsg = ""
		return *m, nil
	}

	if m.presets != nil {
		if err := m.presets.Set(num, path); err != nil {
			m.presetErr = err.Error()
			m.presetMsg = ""
			return *m, nil
		}
	}

	m.presetMsg = fmt.Sprintf("Preset #%d erfolgreich gespeichert!", num)
	m.presetErr = ""
	m.presetNumInput.SetValue("")
	m.presetPathInput.SetValue("")
	m.presetFocus = presetFieldNum
	m.presetNumInput.Focus()
	m.presetPathInput.Blur()
	return *m, nil
}

func (m *Model) cycleSeparator() {
	switch m.separator {
	case renamer.SeparatorSpace:
		m.separator = renamer.SeparatorDot
		m.dropdownIdx = 1
	case renamer.SeparatorDot:
		m.separator = renamer.SeparatorUnderscore
		m.dropdownIdx = 2
	default:
		m.separator = renamer.SeparatorSpace
		m.dropdownIdx = 0
	}
}

func (m Model) startScan() (tea.Model, tea.Cmd) {
	movieP := strings.TrimSpace(m.inputMoviePath.Value())
	showP := strings.TrimSpace(m.inputShowPath.Value())

	if m.presets != nil {
		movieP = m.presets.Resolve(movieP)
		showP = m.presets.Resolve(showP)
	}

	// Backward compatibility and single path smart-routing
	if movieP == "" && showP == "" && m.inputPath.Value() != "" {
		legacyVal := strings.TrimSpace(m.inputPath.Value())
		if m.presets != nil {
			legacyVal = m.presets.Resolve(legacyVal)
		}
		lower := strings.ToLower(legacyVal)
		if strings.Contains(lower, "serie") || strings.Contains(lower, "show") || strings.Contains(lower, "season") || strings.Contains(lower, "staffel") || isDirWithEpisodes(legacyVal) {
			showP = legacyVal
		} else {
			movieP = legacyVal
		}
	} else if movieP != "" && showP == "" {
		// If user entered only movie path but it's a TV series folder (e.g. pasted into first field)
		lower := strings.ToLower(movieP)
		if strings.Contains(lower, "serie") || strings.Contains(lower, "show") || strings.Contains(lower, "season") || strings.Contains(lower, "staffel") || isDirWithEpisodes(movieP) {
			showP = movieP
			movieP = ""
		}
	} else if showP != "" && movieP == "" {
		// If user entered only show path but it's a movie folder
		lower := strings.ToLower(showP)
		if (strings.Contains(lower, "film") || strings.Contains(lower, "movie")) && !strings.Contains(lower, "serie") && !strings.Contains(lower, "show") && !isDirWithEpisodes(showP) {
			movieP = showP
			showP = ""
		}
	}

	if movieP == "" && showP == "" {
		m.inputErr = "Bitte mindestens einen Ordnerpfad (Filme oder Serien) angeben."
		return m, nil
	}

	scanner := renamer.NewScanner(m.separator)
	res, err := scanner.ScanPaths(movieP, showP)
	if err != nil {
		m.inputErr = err.Error()
		return m, nil
	}

	m.inputErr = ""
	m.scanResult = res
	m.state = screenPreview
	m.inputMoviePath.Blur()
	m.inputShowPath.Blur()
	m.inputPath.Blur()

	content := m.renderPreviewContent()
	m.viewport.SetContent(content)
	m.viewport.GotoTop()

	return m, nil
}

func (m Model) startExecute() (tea.Model, tea.Cmd) {
	if m.scanResult == nil || len(m.scanResult.Actions) == 0 {
		return m, nil
	}

	executor := renamer.NewExecutor()
	execRes := executor.Execute(m.scanResult.Actions)
	m.execResult = &execRes
	m.resultFocus = resultFieldNew
	m.state = screenResult
	m.inputMoviePath.Blur()
	m.inputShowPath.Blur()
	m.inputPath.Blur()
	m.presetNumInput.Blur()
	m.presetPathInput.Blur()

	return m, nil
}

// addArea registers a clickable / touchable bounding box.
// Setting maxX to a large value or screen width ensures easy thumb-reachability on mobile!
func (m *Model) addArea(id string, startY, height, startX, width int) {
	if m.clickAreas == nil {
		m.clickAreas = new([]clickArea)
	}
	*m.clickAreas = append(*m.clickAreas, clickArea{
		id:   id,
		minY: startY,
		maxY: startY + height - 1,
		minX: startX,
		maxX: startX + width - 1,
	})
}

func (m Model) View() string {
	// Reset click areas for current frame
	if m.clickAreas != nil {
		*m.clickAreas = (*m.clickAreas)[:0]
	} else {
		m.clickAreas = new([]clickArea)
	}

	var b strings.Builder

	// 1. Header Banner
	bannerText := "🎬 SMRT Media Renamer"
	if m.width < 45 {
		bannerText = "🎬 SMRT Renamer"
	}
	renderedTitle := titleStyle.Render(bannerText)
	if m.height < 26 {
		b.WriteString(renderedTitle + "\n")
	} else {
		b.WriteString(renderedTitle + "\n\n")
	}

	switch m.state {
	case screenInput:
		m.renderInputView(&b)
	case screenPreview:
		m.renderPreviewView(&b)
	case screenResult:
		m.renderResultView(&b)
	case screenPresets:
		m.renderPresetsView(&b)
	}

	output := b.String()

	// Hide hardware cursor on Preview and Result screens to prevent blinking cursor artifacts
	// and close mobile soft keyboards; restore cursor on Input and Presets screens.
	if m.state == screenPreview || m.state == screenResult {
		output = "\x1b[?25l" + output
	} else if m.state == screenInput || m.state == screenPresets {
		output = "\x1b[?25h" + output
	}

	// Terminal Scroll Offset Correction:
	// A terminal of height H displays H rows.
	// If output has more than H rows, the terminal emulator scrolls up by: len(lines) - m.height.
	// Adjusting all hitboxes guarantees 100% alignment with the physical screen rows!
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if m.height > 0 && len(lines) > m.height {
		scrollOffset := len(lines) - m.height
		if m.clickAreas != nil {
			for i := range *m.clickAreas {
				(*m.clickAreas)[i].minY -= scrollOffset
				(*m.clickAreas)[i].maxY -= scrollOffset
			}
		}
	}

	return output
}

// renderInputView renders Screen 1: Ordnerpfade (Filme & Serien), Touch-freundliches Dropdown, Let's Go Button
func (m *Model) renderInputView(b *strings.Builder) {
	// Sync legacy inputPath if set externally
	if m.inputPath.Value() != "" && m.inputMoviePath.Value() == "" && m.inputShowPath.Value() == "" {
		m.inputMoviePath.SetValue(m.inputPath.Value())
		m.inputMoviePath.SetCursor(m.inputPath.Position())
	}

	isMobile := m.width < 55
	isCompact := m.height < 26
	isVeryCompact := m.height <= 22
	cardWidth := max(24, min(m.width-2, 68))
	fullHitWidth := max(m.width, cardWidth)

	sepBlank := "\n"
	if !isCompact {
		sepBlank = "\n\n"
	}

	// 1. Filme-Ordnerpfad Label
	var movieLabel string
	if m.focus == fieldMoviePath {
		movieLabel = focusedLabelStyle.Render("▸ 1. Filme-Ordnerpfad (Tippen / Einfügen):")
	} else {
		movieLabel = labelStyle.Render("  1. Filme-Ordnerpfad:")
	}
	b.WriteString(movieLabel + "\n")

	// Filme Input Box
	var movieBox string
	if m.focus == fieldMoviePath {
		movieBox = activeBoxStyle.Width(cardWidth).Render(m.inputMoviePath.View())
	} else {
		movieBox = boxStyle.Width(cardWidth).Render(m.inputMoviePath.View())
	}
	mStartY := strings.Count(b.String(), "\n")
	mHeight := lipgloss.Height(movieBox)
	m.addArea("input_movie", mStartY, mHeight, 0, fullHitWidth)
	m.addArea("input", mStartY, mHeight, 0, fullHitWidth) // legacy fallback hitbox
	b.WriteString(movieBox + "\n")
	if m.presets != nil && strings.TrimSpace(m.inputMoviePath.Value()) != "" {
		if resM, isP := m.presets.ResolveWithInfo(m.inputMoviePath.Value()); isP {
			b.WriteString(lipgloss.NewStyle().Foreground(accentColor).Render("  ↳ Preset: "+resM) + "\n")
		}
	}

	// 2. Serien-Ordnerpfad Label
	var showLabel string
	if m.focus == fieldShowPath {
		showLabel = focusedLabelStyle.Render("▸ 2. Serien-Ordnerpfad (Tippen / Einfügen):")
	} else {
		showLabel = labelStyle.Render("  2. Serien-Ordnerpfad:")
	}
	b.WriteString(showLabel + "\n")

	// Serien Input Box
	var showBox string
	if m.focus == fieldShowPath {
		showBox = activeBoxStyle.Width(cardWidth).Render(m.inputShowPath.View())
	} else {
		showBox = boxStyle.Width(cardWidth).Render(m.inputShowPath.View())
	}
	sStartY := strings.Count(b.String(), "\n")
	sHeight := lipgloss.Height(showBox)
	m.addArea("input_show", sStartY, sHeight, 0, fullHitWidth)
	b.WriteString(showBox + "\n")
	if m.presets != nil && strings.TrimSpace(m.inputShowPath.Value()) != "" {
		if resS, isP := m.presets.ResolveWithInfo(m.inputShowPath.Value()); isP {
			b.WriteString(lipgloss.NewStyle().Foreground(accentColor).Render("  ↳ Preset: "+resS) + "\n")
		}
	}

	// 3. Trennzeichen Dropdown Label (omitted if isVeryCompact to guarantee Screen 1 <= 20 lines)
	if !isVeryCompact {
		var dropLabel string
		if isMobile {
			if m.focus == fieldDropdown {
				dropLabel = focusedLabelStyle.Render("▸ 3. Trennzeichen (Antippen • Strg+U):")
			} else {
				dropLabel = labelStyle.Render("  3. Trennzeichen (Antippen • Strg+U):")
			}
		} else {
			if m.focus == fieldDropdown {
				dropLabel = focusedLabelStyle.Render("▸ 3. Trennzeichen (Antippen zum Aufklappen • Strg+U):")
			} else {
				dropLabel = labelStyle.Render("  3. Trennzeichen (Antippen zum Aufklappen • Strg+U):")
			}
		}
		b.WriteString(dropLabel + "\n")
	}

	// Dropdown Rendering:
	if !m.dropdownOpen {
		// --- GESCHLOSSENES DROPDOWN ---
		activeName := "Leerzeichen ( )"
		activeExample := "The Film Title (2009)"
		for _, opt := range separatorOptions {
			if opt.mode == m.separator {
				activeName = opt.name
				activeExample = opt.example
				break
			}
		}

		headerContent := fmt.Sprintf("▼ %-18s  [ %s ]", activeName, "Tippen zum Ändern ▾")
		if !isMobile && !isCompact {
			headerContent = fmt.Sprintf("▼ %-18s  Format: %-22s  [ Ändern ▾ ]", activeName, activeExample)
		}

		var renderedCard string
		if m.focus == fieldDropdown {
			renderedCard = activeTouchCardStyle.Width(cardWidth).Render(headerContent)
		} else {
			renderedCard = touchCardStyle.Width(cardWidth).Render(headerContent)
		}

		cardStartY := strings.Count(b.String(), "\n")
		cardHeight := lipgloss.Height(renderedCard)
		m.addArea("dropdown_header", cardStartY, cardHeight, 0, fullHitWidth)
		b.WriteString(renderedCard + "\n")
	} else {
		// --- OFFENES DROPDOWN: Kompakt und fingerfreundlich ---
		headerNotice := lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render("▲ Trennzeichen antippen:")
		hdrStartY := strings.Count(b.String(), "\n")
		m.addArea("dropdown_header", hdrStartY, 1, 0, fullHitWidth)
		b.WriteString(headerNotice + "\n")

		for i, opt := range separatorOptions {
			isSelected := opt.mode == m.separator
			isHighlighted := m.focus == fieldDropdown && m.dropdownIdx == i

			icon := "[○]"
			if isSelected {
				icon = "[●]"
			}

			cardText := fmt.Sprintf("%s %-14s %s", icon, opt.name, lipgloss.NewStyle().Foreground(subtleColor).Render(opt.example))
			style := lipgloss.NewStyle().Padding(0, 1)
			var renderedOption string
			if isSelected {
				renderedOption = style.Bold(true).Foreground(lipgloss.Color("#000000")).Background(accentColor).Width(cardWidth).Render(cardText)
			} else if isHighlighted {
				renderedOption = style.Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(primaryColor).Width(cardWidth).Render(cardText)
			} else {
				renderedOption = style.Foreground(lipgloss.Color("#EEEEEE")).Background(cardBg).Width(cardWidth).Render(cardText)
			}

			optStartY := strings.Count(b.String(), "\n")
			optHeight := lipgloss.Height(renderedOption)
			m.addArea(fmt.Sprintf("opt_%d", i), optStartY, optHeight, 0, fullHitWidth)
			b.WriteString(renderedOption + "\n")
		}
	}

	if !isCompact && !m.dropdownOpen {
		b.WriteString("\n")
	}

	// 4. LET'S GO Button
	hasMovies := strings.TrimSpace(m.inputMoviePath.Value()) != ""
	hasShows := strings.TrimSpace(m.inputShowPath.Value()) != ""
	btnLabel := "▶  LET'S GO  (STARTEN)"
	if hasMovies && hasShows {
		btnLabel = "▶  BEIDE ORDNER SCANNEN (LET'S GO)"
	} else if hasShows {
		btnLabel = "▶  SERIEN SCANNEN (LET'S GO)"
	} else if hasMovies {
		btnLabel = "▶  FILME SCANNEN (LET'S GO)"
	}

	var renderedBtn string
	if m.focus == fieldButtonGo {
		renderedBtn = btnBigActiveStyle.Width(cardWidth).Render(btnLabel)
	} else {
		renderedBtn = btnBigStyle.Width(cardWidth).Render(btnLabel)
	}
	goStartY := strings.Count(b.String(), "\n")
	goHeight := lipgloss.Height(renderedBtn)
	m.addArea("btn_go", goStartY, goHeight, 0, fullHitWidth)
	b.WriteString(renderedBtn + sepBlank)

	// 5. Presets Submenu Button (hidden while dropdown is open to maximize compact spacing)
	if !m.dropdownOpen {
		presetCount := 0
		if m.presets != nil {
			presetCount = m.presets.Count()
		}
		presetLabel := fmt.Sprintf("⚙  PRESETS (1-99) VERWALTEN  [%d GESPEICHERT]", presetCount)
		if isMobile {
			presetLabel = fmt.Sprintf("⚙ Presets (1-99) [%d]", presetCount)
		}

		var renderedPresetsBtn string
		if m.focus == fieldButtonPresets {
			renderedPresetsBtn = btnSecondaryStyle.BorderForeground(cyanColor).Foreground(lipgloss.Color("#FFFFFF")).Width(cardWidth).Render(presetLabel)
		} else {
			renderedPresetsBtn = btnSecondaryStyle.Width(cardWidth).Render(presetLabel)
		}
		pStartY := strings.Count(b.String(), "\n")
		pHeight := lipgloss.Height(renderedPresetsBtn)
		m.addArea("btn_presets", pStartY, pHeight, 0, fullHitWidth)
		b.WriteString(renderedPresetsBtn + sepBlank)
	}

	// Error message if any
	if m.inputErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Bold(true).Render("❌ "+m.inputErr) + "\n")
	}

	// Responsive Help Hints
	if isMobile {
		b.WriteString(helpStyle.Render("Touch: Direkt antippen  •  Strg+P: Presets  •  Enter: Start"))
	} else {
		b.WriteString(helpStyle.Render("Touch / Klick: Beliebiges Feld  •  Strg+P: Presets  •  Enter: Start"))
	}
}

// renderPresetsView renders Screen 4: Preset-Verwaltung (Nummern 1-99 zuweisen)
func (m *Model) renderPresetsView(b *strings.Builder) {
	isMobile := m.width < 55
	isCompact := m.height < 26
	cardWidth := max(24, min(m.width-2, 68))
	fullHitWidth := max(m.width, cardWidth)

	// Submenu Title
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render("🔖 Pfad-Presets (Nummern 1 - 99)") + "\n")
	if !isCompact {
		b.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render("Vergib Nummern für Pfade, um sie im Hauptmenü schnell abzurufen.") + "\n\n")
	} else {
		b.WriteString("\n")
	}

	// Input fields for Preset: Number and Path
	numLabel := "Nummer (1-99):"
	if m.presetFocus == presetFieldNum {
		numLabel = focusedLabelStyle.Render("▸ Nr. (1-99):")
	} else {
		numLabel = labelStyle.Render("  Nr. (1-99):")
	}

	var numBox string
	if m.presetFocus == presetFieldNum {
		numBox = activeBoxStyle.Width(10).Render(m.presetNumInput.View())
	} else {
		numBox = boxStyle.Width(10).Render(m.presetNumInput.View())
	}

	pathLabel := "Speicherpfad:"
	if m.presetFocus == presetFieldPath {
		pathLabel = focusedLabelStyle.Render("▸ Speicherpfad:")
	} else {
		pathLabel = labelStyle.Render("  Speicherpfad:")
	}

	pathBoxWidth := max(16, cardWidth-14)
	var pathBox string
	if m.presetFocus == presetFieldPath {
		pathBox = activeBoxStyle.Width(pathBoxWidth).Render(m.presetPathInput.View())
	} else {
		pathBox = boxStyle.Width(pathBoxWidth).Render(m.presetPathInput.View())
	}

	if isMobile {
		b.WriteString(numLabel + "\n")
		nStartY := strings.Count(b.String(), "\n")
		nHeight := lipgloss.Height(numBox)
		m.addArea("preset_input_num", nStartY, nHeight, 0, fullHitWidth)
		b.WriteString(numBox + "\n")

		b.WriteString(pathLabel + "\n")
		pStartY := strings.Count(b.String(), "\n")
		pHeight := lipgloss.Height(pathBox)
		m.addArea("preset_input_path", pStartY, pHeight, 0, fullHitWidth)
		b.WriteString(pathBox + "\n")
	} else {
		inputsRow := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left, numLabel, numBox),
			"  ",
			lipgloss.JoinVertical(lipgloss.Left, pathLabel, pathBox),
		)
		inStartY := strings.Count(b.String(), "\n") + 1
		m.addArea("preset_input_num", inStartY, 3, 0, 14)
		m.addArea("preset_input_path", inStartY, 3, 14, cardWidth)
		b.WriteString(inputsRow + "\n")
	}

	// Action Buttons: Speichern, Löschen, Zurück
	btnSaveStyle := btnBigStyle
	if m.presetFocus == presetFieldSave {
		btnSaveStyle = btnBigActiveStyle
	}
	saveBtn := btnSaveStyle.Render(" ✔ Speichern ")

	btnDelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Foreground(dangerColor).
		Background(lipgloss.Color("#2A2A35")).
		Bold(true)
	if m.presetFocus == presetFieldDelete {
		btnDelStyle = btnDelStyle.BorderForeground(dangerColor).Foreground(lipgloss.Color("#FFFFFF")).Background(dangerColor)
	}
	delBtn := btnDelStyle.Render(" 🗑 Löschen ")

	btnBackStyle := btnSecondaryStyle
	if m.presetFocus == presetFieldBack {
		btnBackStyle = btnBigActiveStyle
	}
	backBtn := btnBackStyle.Render(" ✖ Zurück (Esc) ")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Top, saveBtn, " ", delBtn, " ", backBtn)
	bStartY := strings.Count(b.String(), "\n")
	bHeight := lipgloss.Height(btnRow)
	m.addArea("preset_btn_save", bStartY, bHeight, 0, cardWidth/3)
	m.addArea("preset_btn_delete", bStartY, bHeight, cardWidth/3, cardWidth/3)
	m.addArea("preset_btn_back", bStartY, bHeight, (cardWidth*2)/3, cardWidth/3+6)
	b.WriteString(btnRow + "\n")

	// Status messages
	if m.presetErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Bold(true).Render("❌ "+m.presetErr) + "\n")
	} else if m.presetMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("✔ "+m.presetMsg) + "\n")
	} else {
		b.WriteString("\n")
	}

	// Saved presets list
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#DDDDDD")).Render("Gespeicherte Presets (Antippen zum Übernehmen):") + "\n")

	presetsList := []presets.Preset{}
	if m.presets != nil {
		presetsList = m.presets.List()
	}

	if len(presetsList) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render("  (Noch keine Presets gespeichert. Oben Nummer 1-99 und Pfad eingeben)") + "\n")
	} else {
		maxItems := 8
		if isCompact {
			maxItems = 3
		}
		for i, p := range presetsList {
			if i >= maxItems {
				b.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(fmt.Sprintf("  ... und %d weitere Presets", len(presetsList)-maxItems)) + "\n")
				break
			}
			tag := fmt.Sprintf("[#%02d]", p.Number)
			tagRendered := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(tag)
			pathDisplay := p.Path
			maxLen := cardWidth - 10
			if len(pathDisplay) > maxLen && maxLen > 10 {
				pathDisplay = "..." + pathDisplay[len(pathDisplay)-(maxLen-3):]
			}
			line := fmt.Sprintf(" %s %s", tagRendered, pathDisplay)
			lStartY := strings.Count(b.String(), "\n")
			m.addArea(fmt.Sprintf("preset_item_%d", p.Number), lStartY, 1, 0, fullHitWidth)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("Tab: Wechseln  •  Enter: Ausführen  •  Preset antippen: Bearbeiten  •  Esc: Zurück"))
}

// renderPreviewView renders Screen 2: Vorschau der Änderungen
func (m *Model) renderPreviewView(b *strings.Builder) {
	isMobile := m.width < 55
	isCompact := m.height < 28
	cardWidth := max(24, min(m.width-2, 68))
	fullHitWidth := cardWidth + 4
	if isMobile {
		fullHitWidth = max(m.width, cardWidth)
	}

	if m.scanResult == nil {
		b.WriteString("Keine Vorschau verfügbar.\n")
		return
	}

	totalActions := len(m.scanResult.Actions)
	if totalActions == 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render("ℹ Keine umzubenennenden Dateien oder Ordner gefunden.\n\n"))
		b.WriteString("Alle Dateien entsprechen bereits dem Schema oder es wurden keine Mediendateien erkannt.\n\n")

		backBtn := btnBigActiveStyle.Width(cardWidth).Render("✖  [ Zurück zum Anpassen ]")
		btnStartY := strings.Count(b.String(), "\n")
		btnHeight := lipgloss.Height(backBtn)
		m.addArea("btn_cancel", btnStartY, btnHeight+1, 0, fullHitWidth)
		b.WriteString(backBtn + "\n\n")

		b.WriteString(helpStyle.Render("Touch: Button antippen  •  Zurück: Esc / Enter"))
		return
	}

	// 1. Summary Header
	var summary string
	if m.scanResult.MoviePath != "" && m.scanResult.ShowPath != "" {
		mBase := filepath.Base(m.scanResult.MoviePath)
		sBase := filepath.Base(m.scanResult.ShowPath)
		if isMobile {
			summary = fmt.Sprintf("🎬 %s • 📺 %s (%d Aktionen)", mBase, sBase, totalActions)
		} else {
			summary = fmt.Sprintf("Filme: %s | Serien: %s | %d Filme, %d Serien | %d Aktionen",
				mBase, sBase, m.scanResult.TotalMovies, m.scanResult.TotalShows, totalActions)
		}
	} else {
		singleRoot := m.scanResult.RootPath
		if singleRoot == "" {
			if m.scanResult.MoviePath != "" {
				singleRoot = m.scanResult.MoviePath
			} else {
				singleRoot = m.scanResult.ShowPath
			}
		}
		rootDisplay := filepath.Base(singleRoot)
		if isMobile {
			summary = fmt.Sprintf("%s: %d Filme, %d Serien (%d Aktionen)",
				rootDisplay, m.scanResult.TotalMovies, m.scanResult.TotalShows, totalActions)
		} else {
			summary = fmt.Sprintf("Ordner: %s | %d Filme, %d Serien | %d Aktionen geplant",
				rootDisplay, m.scanResult.TotalMovies, m.scanResult.TotalShows, totalActions)
		}
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render(summary) + "\n")

	// 2. Show warnings if any
	if len(m.scanResult.Warnings) > 0 {
		maxWarn := 2
		if len(m.scanResult.Warnings) < maxWarn {
			maxWarn = len(m.scanResult.Warnings)
		}
		for i := 0; i < maxWarn; i++ {
			b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Render("⚠ "+m.scanResult.Warnings[i]) + "\n")
		}
	}

	// 3. PROMINENT ACTION BUTTONS
	btnWidth := cardWidth

	var confirmBtn string
	var cancelBtn string

	if !isMobile && !isCompact {
		// Huge, Vibrant Emerald Green Confirm Touch Card (6 lines tall)
		confirmContent := fmt.Sprintf("\n✔  JETZT UMBENENNEN (STARTEN)\n   (%d Aktionen sofort ausführen)\n", totalActions)
		confirmBtn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Foreground(lipgloss.Color("#000000")).
			Background(accentColor).
			Bold(true).
			Align(lipgloss.Center).
			Width(btnWidth).
			Render(confirmContent)

		// Secondary Cancel Touch Card
		cancelContent := "✖  Abbrechen / Zurück (Esc)"
		cancelBtn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5F87")).
			Foreground(lipgloss.Color("#EEEEEE")).
			Background(lipgloss.Color("#2A2A35")).
			Bold(true).
			Align(lipgloss.Center).
			Width(btnWidth).
			Render(cancelContent)
	} else {
		// Compact yet big touch buttons for mobile / compact screens (3 lines tall each)
		confirmContent := fmt.Sprintf("✔  JETZT UMBENENNEN (%d Aktionen)", totalActions)
		confirmBtn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Foreground(lipgloss.Color("#000000")).
			Background(accentColor).
			Bold(true).
			Align(lipgloss.Center).
			Width(btnWidth).
			Render(confirmContent)

		cancelContent := "✖  Abbrechen / Zurück (Esc)"
		cancelBtn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5F87")).
			Foreground(lipgloss.Color("#EEEEEE")).
			Background(lipgloss.Color("#2A2A35")).
			Bold(true).
			Align(lipgloss.Center).
			Width(btnWidth).
			Render(cancelContent)
	}

	confStartY := strings.Count(b.String(), "\n")
	b.WriteString(confirmBtn + "\n")

	cancStartY := strings.Count(b.String(), "\n")
	cancHeight := lipgloss.Height(cancelBtn)
	b.WriteString(cancelBtn + "\n")

	// Continuous hitboxes without dead rows between buttons:
	m.addArea("btn_confirm", confStartY, cancStartY-confStartY, 0, fullHitWidth)
	m.addArea("btn_cancel", cancStartY, cancHeight, 0, fullHitWidth)

	// 4. Viewport Header / Scroll hints
	vpHdrText := "▼ Details der Änderungen (Wischen/Tasten zum Scrollen):"
	if isMobile {
		vpHdrText = "▼ Details (Wischen zum Scrollen):"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(vpHdrText) + "\n")

	// 5. Dynamic Viewport Height: Computes remaining lines precisely so view NEVER overflows m.height!
	usedLines := strings.Count(b.String(), "\n")
	// Viewport takes: 2 border lines + help line (1) = 3 lines
	vpHeight := m.height - usedLines - 3
	if vpHeight < 2 {
		vpHeight = 2
	}
	m.viewport.Height = vpHeight
	m.viewport.Width = max(20, m.width-6)

	// Render Viewport
	vpView := boxStyle.Width(m.viewport.Width).Render(m.viewport.View())
	vpStartY := strings.Count(b.String(), "\n")
	vpHeightRendered := lipgloss.Height(vpView)

	// Touch scroll hitboxes on viewport itself:
	halfVp := vpHeightRendered / 2
	if halfVp > 0 {
		m.addArea("vp_up", vpStartY, halfVp, 0, fullHitWidth)
		m.addArea("vp_down", vpStartY+halfVp, vpHeightRendered-halfVp, 0, fullHitWidth)
	}
	b.WriteString(vpView + "\n")

	// 6. Help hints at bottom
	if isMobile {
		b.WriteString(helpStyle.Render("Touch: Buttons oben antippen  •  Start: Enter  •  Zurück: Esc"))
	} else {
		b.WriteString(helpStyle.Render("Touch: Buttons antippen  •  Scrollen: Mausrad / Tasten  •  Start: Enter"))
	}
}

func (m Model) renderPreviewContent() string {
	if m.scanResult == nil {
		return ""
	}

	var sb strings.Builder
	for i, act := range m.scanResult.Actions {
		tag := "[DATEI]"
		renderedTag := actionTypeStyle.Render(tag)
		if act.Type == renamer.ActionRenameFolder {
			tag = "[ORDNER]"
			renderedTag = actionTypeStyle.Render(tag)
		} else if act.Type == renamer.ActionCreateFolder {
			tag = "[NEUER ORDNER]"
			renderedTag = actionTypeStyle.Render(tag)
		} else if act.Type == renamer.ActionMoveFile {
			tag = "[VERSCHIEBEN]"
			renderedTag = actionTypeStyle.Render(tag)
		} else if act.Type == renamer.ActionDeleteFile {
			tag = "[LÖSCHEN]"
			renderedTag = lipgloss.NewStyle().Bold(true).Foreground(dangerColor).Render(tag)
		} else if act.Type == renamer.ActionConvertSubtitle {
			tag = "[KONVERTIEREN]"
			renderedTag = lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(tag)
		} else if act.Type == renamer.ActionDeleteDuplicate {
			tag = "[DUPLIKAT]"
			renderedTag = lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render(tag)
		}

		sb.WriteString(fmt.Sprintf("%d. %s  %s\n", i+1, renderedTag, act.Description))
		if act.DisplayOld != "" && act.DisplayOld != "(Neu)" {
			sb.WriteString(fmt.Sprintf("   Alt:  %s\n", oldPathStyle.Render(act.DisplayOld)))
		}
		if act.Type == renamer.ActionDeleteFile || act.Type == renamer.ActionDeleteDuplicate {
			sb.WriteString(fmt.Sprintf("   Neu:  %s\n", lipgloss.NewStyle().Bold(true).Foreground(dangerColor).Render(act.DisplayNew)))
		} else {
			sb.WriteString(fmt.Sprintf("   Neu:  %s\n", newPathStyle.Render(act.DisplayNew)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderResultView renders Screen 3: Erfolgsmeldung & Details (Brand-new Mobile-First Touch Design)
func (m *Model) renderResultView(b *strings.Builder) {
	isMobile := m.width < 55
	isVeryCompact := m.height < 22
	cardWidth := max(24, min(m.width-4, 60))
	fullHitWidth := cardWidth + 4
	if m.width <= 75 {
		fullHitWidth = max(m.width, cardWidth)
	}

	if m.execResult == nil {
		return
	}

	// 1. Result Status Hero Card
	if m.execResult.ErrorCount == 0 {
		var titleLine string
		var subLine string
		if isMobile {
			titleLine = "✨ Alles erledigt!"
			subLine = fmt.Sprintf("✓ %d Aktionen erfolgreich ausgeführt.", m.execResult.SuccessCount)
		} else {
			titleLine = "✨ Alles erledigt! Erfolgreich umbenannt"
			subLine = fmt.Sprintf("✓ %d Datei- & Ordneraktionen erfolgreich ausgeführt.", m.execResult.SuccessCount)
		}

		heroText := fmt.Sprintf("%s\n%s", titleLine, subLine)
		heroCard := lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Foreground(accentColor).
			Background(lipgloss.Color("#162620")).
			Padding(0, 2).
			Width(cardWidth).
			Align(lipgloss.Center).
			Render(heroText)
		b.WriteString(heroCard + "\n\n")
	} else {
		summaryLine := fmt.Sprintf("⚠️ %d erfolgreich  •  %d Hinweise", m.execResult.SuccessCount, m.execResult.ErrorCount)
		var heroLines []string
		heroLines = append(heroLines, summaryLine)

		maxErrs := 2
		if !isVeryCompact && !isMobile {
			maxErrs = 4
		}
		for i, errStr := range m.execResult.Errors {
			if i >= maxErrs {
				heroLines = append(heroLines, fmt.Sprintf("• ... und %d weitere Meldungen", len(m.execResult.Errors)-maxErrs))
				break
			}
			heroLines = append(heroLines, "• "+errStr)
		}

		heroCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Foreground(warningColor).
			Background(lipgloss.Color("#2A2218")).
			Padding(0, 1).
			Width(cardWidth).
			Render(strings.Join(heroLines, "\n"))
		b.WriteString(heroCard + "\n\n")
	}

	// 2. High-Contrast Full-Width Touch Buttons (Consistent vertical stack for Mobile & Desktop)
	var renderedNewBtn string
	var renderedQuitBtn string

	btnNewText := "↻  [ NEUER ORDNER ] "
	btnQuitText := "✖  [ BEENDEN ] "

	if m.resultFocus == resultFieldNew {
		renderedNewBtn = btnNewActiveStyle.Width(cardWidth).Render(btnNewText)
		renderedQuitBtn = btnQuitStyle.Width(cardWidth).Render(btnQuitText)
	} else {
		renderedNewBtn = btnNewStyle.Width(cardWidth).Render(btnNewText)
		renderedQuitBtn = btnQuitActiveStyle.Width(cardWidth).Render(btnQuitText)
	}

	newStartY := strings.Count(b.String(), "\n")
	b.WriteString(renderedNewBtn + "\n\n")

	quitStartY := strings.Count(b.String(), "\n")
	quitHeight := lipgloss.Height(renderedQuitBtn)
	b.WriteString(renderedQuitBtn + "\n\n")

	// Seamless Touch Hitboxes across the entire width:
	// Upper area covers btn_new, lower area covers btn_quit down to the bottom
	m.addArea("btn_new", newStartY, quitStartY-newStartY, 0, fullHitWidth)
	bottomCoverage := max(quitHeight+2, m.height-quitStartY)
	m.addArea("btn_quit", quitStartY, bottomCoverage, 0, fullHitWidth)

	// 3. Helpful Responsive Footer (Always fits on mobile screens without cutting off)
	if m.width < 75 {
		b.WriteString(helpStyle.Render("Touch: Button antippen  •  Wechseln: Tab") + "\n")
		b.WriteString(helpStyle.Render("Ausführen: Enter  •  Beenden: q / Esc"))
	} else {
		b.WriteString(helpStyle.Render("Touch: Button antippen  •  Wechseln: Tab  •  Ausführen: Enter  •  Beenden: q / Esc"))
	}
}
