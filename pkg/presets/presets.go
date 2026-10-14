package presets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Preset represents a mapping between a shortcut number (1-99) and a filesystem path.
type Preset struct {
	Number int    `json:"number"`
	Path   string `json:"path"`
}

// Manager handles storing, loading, and resolving preset paths.
type Manager struct {
	mu       sync.RWMutex
	filePath string
	presets  map[int]string
}

// DefaultConfigPath returns the standard configuration file path for presets.
func DefaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "smrt", "presets.json")
	}
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".config", "smrt", "presets.json")
	}
	return ".smrt_presets.json"
}

// Load loads presets from the default configuration path.
func Load() (*Manager, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom loads presets from a specific file path.
func LoadFrom(filePath string) (*Manager, error) {
	m := &Manager{
		filePath: filePath,
		presets:  make(map[int]string),
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, fmt.Errorf("konnte Preset-Datei nicht lesen: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return m, nil
	}

	var list []Preset
	if err := json.Unmarshal(data, &list); err != nil {
		// Try fallback object map
		var objMap map[string]string
		if err2 := json.Unmarshal(data, &objMap); err2 == nil {
			for k, v := range objMap {
				if num, err3 := strconv.Atoi(k); err3 == nil && num >= 1 && num <= 99 {
					m.presets[num] = v
				}
			}
			return m, nil
		}
		return nil, fmt.Errorf("ungültiges JSON in Preset-Datei: %w", err)
	}

	for _, p := range list {
		if p.Number >= 1 && p.Number <= 99 && strings.TrimSpace(p.Path) != "" {
			m.presets[p.Number] = p.Path
		}
	}

	return m, nil
}

// Save persists the current presets to disk.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("konnte Preset-Verzeichnis nicht erstellen: %w", err)
	}

	list := make([]Preset, 0, len(m.presets))
	for num, path := range m.presets {
		list = append(list, Preset{Number: num, Path: path})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Number < list[j].Number
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("konnte Presets nicht serialisieren: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("konnte Preset-Datei nicht schreiben: %w", err)
	}

	return nil
}

// Set adds or updates a preset number (1-99).
func (m *Manager) Set(number int, path string) error {
	if number < 1 || number > 99 {
		return fmt.Errorf("ungültige Preset-Nummer %d: Erlaubt sind Zahlen von 1 bis 99", number)
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return fmt.Errorf("Pfad darf nicht leer sein")
	}

	m.mu.Lock()
	m.presets[number] = cleanPath
	m.mu.Unlock()

	return m.Save()
}

// Delete removes a preset by its number.
func (m *Manager) Delete(number int) error {
	m.mu.Lock()
	delete(m.presets, number)
	m.mu.Unlock()

	return m.Save()
}

// Get retrieves a preset path by its number.
func (m *Manager) Get(number int) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path, ok := m.presets[number]
	return path, ok
}

// List returns all presets sorted ascending by number.
func (m *Manager) List() []Preset {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]Preset, 0, len(m.presets))
	for num, path := range m.presets {
		list = append(list, Preset{Number: num, Path: path})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Number < list[j].Number
	})
	return list
}

// Count returns the number of saved presets.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.presets)
}

// ParsePresetNumber checks if the given string represents a valid preset number (1-99).
// Supports optional leading "#" or spaces (e.g. "1", "#1", " 42 ").
func ParsePresetNumber(input string) (int, bool) {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimPrefix(trimmed, "#")
	num, err := strconv.Atoi(trimmed)
	if err != nil || num < 1 || num > 99 {
		return 0, false
	}
	return num, true
}

// Resolve checks if the input is a preset number (1-99).
// If it matches a saved preset, it returns the resolved filesystem path.
// Otherwise, it returns the original input unchanged.
func (m *Manager) Resolve(input string) string {
	resolved, _ := m.ResolveWithInfo(input)
	return resolved
}

// ResolveWithInfo resolves a preset number and returns whether it was resolved.
func (m *Manager) ResolveWithInfo(input string) (string, bool) {
	num, ok := ParsePresetNumber(input)
	if !ok {
		return input, false
	}
	if p, found := m.Get(num); found {
		return p, true
	}
	return input, false
}
