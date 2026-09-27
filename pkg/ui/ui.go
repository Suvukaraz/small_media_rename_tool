package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"smrt/pkg/renamer"
)

type screenState int

const (
	screenInput screenState = iota
	screenPreview
	screenResult
)

type focusedField int

const (
	fieldPath focusedField = iota
	fieldDropdown
	fieldButtonGo
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

	// Screen 1: Input
	inputPath    textinput.Model
	separator    renamer.SeparatorMode
	dropdownOpen bool
	dropdownIdx  int
	focus        focusedField
	inputErr     string

	// Screen 2: Preview
	scanResult   *renamer.ScanResult
	viewport     viewport.Model
	readyVP      bool
	previewErr   string

	// Screen 3: Result
	execResult   *renamer.ExecutionResult

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

// InitialModel configures and initializes the TUI model.
func InitialModel() Model {
	return InitialModelWithPath("")
}

// InitialModelWithPath configures and initializes the TUI model with an initial path.
func InitialModelWithPath(initialPath string) Model {
	ti := textinput.New()
	ti.Placeholder = "/pfad/zu/filmen_oder_serien"
	if initialPath != "" {
		ti.SetValue(initialPath)
	}
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 50

	return Model{
		state:        screenInput,
		inputPath:    ti,
		separator:    renamer.SeparatorSpace, // Default: Leerzeichen
		dropdownOpen: false,
		dropdownIdx:  0,
		focus:        fieldPath,
		width:        80,
		height:       24,
		clickAreas:   new([]clickArea),
	}
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
				// Start scan on enter
				return m.startScan()
			}

			// When text input is focused, typing (including spaces) goes strictly into textinput
			if m.focus == fieldPath && !m.dropdownOpen {
				m.inputPath, cmd = m.inputPath.Update(msg)
				return m, cmd
			}

		case screenPreview:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "esc", "b", "backspace":
				m.state = screenInput
				m.focus = fieldPath
				m.inputPath.Focus()
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
			case "ctrl+c", "q", "esc":
				return m, tea.Quit

			case "enter", "n":
				m.state = screenInput
				m.scanResult = nil
				m.execResult = nil
				m.inputErr = ""
				m.focus = fieldPath
				m.inputPath.Focus()
				return m, nil
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
			case "input":
				m.focus = fieldPath
				m.inputPath.Focus()
				m.dropdownOpen = false
				textStartX := a.minX + 4
				if x >= textStartX {
					m.inputPath.SetCursor(x - textStartX)
				}
				return *m, nil

			case "dropdown_header":
				m.focus = fieldDropdown
				m.inputPath.Blur()
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
				m.focus = fieldPath
				m.inputPath.Focus()
				return *m, nil

			case "btn_new":
				m.state = screenInput
				m.scanResult = nil
				m.execResult = nil
				m.inputErr = ""
				m.focus = fieldPath
				m.inputPath.Focus()
				return *m, nil

			case "btn_quit":
				return *m, tea.Quit
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
	m.focus = (m.focus + focusedField(delta) + 3) % 3
	if m.focus == fieldPath {
		m.inputPath.Focus()
	} else {
		m.inputPath.Blur()
	}
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
	path := strings.TrimSpace(m.inputPath.Value())
	if path == "" {
		m.inputErr = "Bitte geben Sie einen Ordnerpfad ein."
		return m, nil
	}

	scanner := renamer.NewScanner(m.separator)
	res, err := scanner.Scan(path)
	if err != nil {
		m.inputErr = err.Error()
		return m, nil
	}

	m.inputErr = ""
	m.scanResult = res
	m.state = screenPreview

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
	m.state = screenResult

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
	}

	output := b.String()

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

// renderInputView renders Screen 1: Ordnerpfad, Touch-freundliches Dropdown, Let's Go Button
func (m *Model) renderInputView(b *strings.Builder) {
	isMobile := m.width < 55
	isCompact := m.height < 26
	cardWidth := max(26, min(m.width-2, 68))
	fullHitWidth := max(m.width, cardWidth)

	sepBlank := "\n\n"
	if isCompact {
		sepBlank = "\n"
	}

	// 1. Ordnerpfad Label
	var pathLabel string
	if m.focus == fieldPath {
		pathLabel = focusedLabelStyle.Render("▸ 1. Ordnerpfad (Tippen / Einfügen):")
	} else {
		pathLabel = labelStyle.Render("  1. Ordnerpfad:")
	}
	b.WriteString(pathLabel + "\n")

	// Input Box (3 lines high, easy to tap)
	var pathBox string
	if m.focus == fieldPath {
		pathBox = activeBoxStyle.Width(cardWidth).Render(m.inputPath.View())
	} else {
		pathBox = boxStyle.Width(cardWidth).Render(m.inputPath.View())
	}
	boxStartY := strings.Count(b.String(), "\n")
	boxHeight := lipgloss.Height(pathBox)
	m.addArea("input", boxStartY, boxHeight, 0, fullHitWidth)
	b.WriteString(pathBox + sepBlank)

	// 2. Trennzeichen Dropdown Label
	var dropLabel string
	if isMobile {
		if m.focus == fieldDropdown {
			dropLabel = focusedLabelStyle.Render("▸ 2. Trennzeichen (Antippen • Strg+U):")
		} else {
			dropLabel = labelStyle.Render("  2. Trennzeichen (Antippen • Strg+U):")
		}
	} else {
		if m.focus == fieldDropdown {
			dropLabel = focusedLabelStyle.Render("▸ 2. Trennzeichen (Antippen zum Aufklappen • Strg+U):")
		} else {
			dropLabel = labelStyle.Render("  2. Trennzeichen (Antippen zum Aufklappen • Strg+U):")
		}
	}
	b.WriteString(dropLabel + "\n")

	// Dropdown Rendering:
	if !m.dropdownOpen {
		// --- GESCHLOSSENES DROPDOWN (Große, leicht zu treffende Touch-Karte) ---
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
		if !isMobile {
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
		b.WriteString(renderedCard + sepBlank)
	} else {
		// --- OFFENES DROPDOWN: Fingerfreundlich und kompakt (kein Scrollen!) ---
		headerNotice := lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render("▲ Trennzeichen antippen zur Auswahl:")
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

			var cardText string
			if isCompact || isMobile {
				cardText = fmt.Sprintf("%s %-16s  %s", icon, opt.name, lipgloss.NewStyle().Foreground(subtleColor).Render(opt.example))
			} else {
				line1 := fmt.Sprintf("%s  %-18s", icon, opt.name)
				line2 := fmt.Sprintf("     Format: %s", opt.example)
				cardText = line1 + "\n" + lipgloss.NewStyle().Foreground(subtleColor).Render(line2)
			}

			var renderedOption string
			if isCompact {
				style := lipgloss.NewStyle().Padding(0, 1)
				if isSelected {
					renderedOption = style.Bold(true).Foreground(lipgloss.Color("#000000")).Background(accentColor).Width(cardWidth).Render(cardText)
				} else if isHighlighted {
					renderedOption = style.Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(primaryColor).Width(cardWidth).Render(cardText)
				} else {
					renderedOption = style.Foreground(lipgloss.Color("#EEEEEE")).Background(cardBg).Width(cardWidth).Render(cardText)
				}
			} else {
				if isSelected {
					renderedOption = selectedOptionCardStyle.Width(cardWidth).Render(cardText)
				} else if isHighlighted {
					renderedOption = activeTouchCardStyle.Width(cardWidth).Render(cardText)
				} else {
					renderedOption = touchCardStyle.Width(cardWidth).Render(cardText)
				}
			}

			optStartY := strings.Count(b.String(), "\n")
			optHeight := lipgloss.Height(renderedOption)
			m.addArea(fmt.Sprintf("opt_%d", i), optStartY, optHeight, 0, fullHitWidth)
			b.WriteString(renderedOption + "\n")
		}
		if !isCompact {
			b.WriteString("\n")
		}
	}

	// 3. LET'S GO Button (Großer, fetter Touch-Button)
	btnLabel := "▶  LET'S GO  (STARTEN)"
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

	// Error message if any
	if m.inputErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Bold(true).Render("❌ "+m.inputErr) + "\n\n")
	}

	// Responsive Help Hints
	if isMobile {
		b.WriteString(helpStyle.Render("Touch: Direkt antippen  •  Strg+U: Umschalten  •  Enter: Start"))
	} else {
		b.WriteString(helpStyle.Render("Touch / Klick: Beliebiges Feld  •  Strg+U: Trennzeichen  •  Enter: Start"))
	}
}

// renderPreviewView renders Screen 2: Vorschau der Änderungen
func (m *Model) renderPreviewView(b *strings.Builder) {
	isMobile := m.width < 55
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

	// 1. Summary Header (bounded to never wrap)
	var summary string
	if isMobile {
		summary = fmt.Sprintf("Erkannt: %d Filme, %d Serien (%d Aktionen)",
			m.scanResult.TotalMovies, m.scanResult.TotalShows, totalActions)
	} else {
		rootDisplay := filepath.Base(m.scanResult.RootPath)
		summary = fmt.Sprintf("Ordner: %s  |  %d Filme, %d Serien  |  %d Aktionen geplant",
			rootDisplay, m.scanResult.TotalMovies, m.scanResult.TotalShows, totalActions)
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render(summary) + "\n\n")

	// 2. Show warnings if any
	if len(m.scanResult.Warnings) > 0 {
		for _, w := range m.scanResult.Warnings {
			b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Render("⚠ "+w) + "\n")
		}
		b.WriteString("\n")
	}

	// 3. PROMINENT ACTION BUTTONS: Huge, bold touch cards for 100% reliable touch access!
	btnWidth := cardWidth

	var confirmBtn string
	var cancelBtn string

	if m.height >= 26 {
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

		// Secondary Cancel Touch Card (3 lines tall with coral border)
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
		// Compact yet big touch buttons for smaller screens (e.g. 20-24 rows)
		confirmContent := fmt.Sprintf("✔  JETZT UMBENENNEN (STARTEN)\n(%d Aktionen sofort ausführen)", totalActions)
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
	b.WriteString(cancelBtn + "\n\n")

	// Continuous hitboxes without dead rows between buttons:
	m.addArea("btn_confirm", confStartY, cancStartY-confStartY, 0, fullHitWidth)
	m.addArea("btn_cancel", cancStartY, cancHeight+1, 0, fullHitWidth)

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
	if vpHeight < 3 {
		vpHeight = 3
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

// renderResultView renders Screen 3: Erfolgsmeldung & Details
func (m *Model) renderResultView(b *strings.Builder) {
	isMobile := m.width < 55
	cardWidth := max(26, min(m.width-2, 68))
	fullHitWidth := max(m.width, cardWidth)

	if m.execResult == nil {
		return
	}

	if m.execResult.ErrorCount == 0 {
		successMsg := fmt.Sprintf("🎉 Alles erledigt!\n%d Aktionen wurden erfolgreich ausgeführt.", m.execResult.SuccessCount)
		renderedSuccess := lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			Width(cardWidth).
			Render(successMsg)
		b.WriteString(renderedSuccess + "\n\n")
	} else {
		partialMsg := fmt.Sprintf("⚠️ %d erfolgreich, %d Fehler aufgetreten:", m.execResult.SuccessCount, m.execResult.ErrorCount)
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(warningColor).
			Render(partialMsg) + "\n")

		for _, errStr := range m.execResult.Errors {
			b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Render("• "+errStr) + "\n")
		}
		b.WriteString("\n")
	}

	if isMobile {
		// On mobile: Stack large touch cards across full width
		newBtn := btnBigActiveStyle.Width(cardWidth).Render("✔  [ Neuer Ordner ]")
		newStartY := strings.Count(b.String(), "\n")
		newHeight := lipgloss.Height(newBtn)
		m.addArea("btn_new", newStartY, newHeight, 0, fullHitWidth)
		b.WriteString(newBtn + "\n\n")

		quitBtn := btnSecondaryStyle.Width(cardWidth).Render("✖  [ Beenden ]")
		quitStartY := strings.Count(b.String(), "\n")
		quitHeight := lipgloss.Height(quitBtn)
		m.addArea("btn_quit", quitStartY, quitHeight, 0, fullHitWidth)
		b.WriteString(quitBtn + "\n\n")
	} else {
		// On desktop: Side-by-side large touch buttons with exact hitboxes
		halfWidth := max(16, (cardWidth-2)/2)
		newBtn := btnBigActiveStyle.Width(halfWidth).Render("✔  [ Neuer Ordner ]")
		quitBtn := btnSecondaryStyle.Width(halfWidth).Render("✖  [ Beenden ]")

		btnRowStartY := strings.Count(b.String(), "\n")
		btnRowHeight := max(lipgloss.Height(newBtn), lipgloss.Height(quitBtn))

		m.addArea("btn_new", btnRowStartY, btnRowHeight, 0, halfWidth+2)
		m.addArea("btn_quit", btnRowStartY, btnRowHeight, halfWidth+3, cardWidth+4)
		b.WriteString(newBtn + "  " + quitBtn + "\n\n")
	}

	b.WriteString(helpStyle.Render("Touch: Button antippen  •  Weiter: Enter  •  Beenden: q / Esc"))
}
