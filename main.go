package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"smrt/pkg/presets"
	"smrt/pkg/ui"
)

const Version = "0.3.0"

func printHelp() {
	fmt.Printf(`🎬 SMRT Media Renamer v%s
Automatische Bereinigung und Umbenennung von Film- und Seriendateien.

Verwendung:
  smrt                                      Interaktive TUI starten
  smrt <pfad|nr>                            Einzelnen Ordner oder Preset-Nummer (1-99) laden
  smrt <filme|nr> <serien|nr>               Beide Ordner oder Preset-Nummern gleichzeitig laden
  smrt -m <filme|nr> -s <serien|nr>         Gezielte Pfade oder Preset-Nummern (1-99) angeben

Optionen:
  -m, --movies <pfad|nr>  Pfad oder Preset-Nummer (1-99) für Filme
  -s, --shows <pfad|nr>   Pfad oder Preset-Nummer (1-99) für Serien (Alias: --series)
  -v, --version           Version anzeigen
  -h, --help              Diese Hilfe anzeigen
`, Version)
}

func main() {
	var moviePath string
	var showPath string
	var posArgs []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-v" || arg == "--version":
			fmt.Printf("smrt version %s\n", Version)
			os.Exit(0)
		case arg == "-h" || arg == "--help":
			printHelp()
			os.Exit(0)
		case arg == "-m" || arg == "--movies":
			if i+1 < len(args) {
				i++
				moviePath = args[i]
			} else {
				fmt.Fprintln(os.Stderr, "Fehler: Option -m/--movies benötigt einen Pfad.")
				os.Exit(1)
			}
		case strings.HasPrefix(arg, "-m="):
			moviePath = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--movies="):
			moviePath = strings.TrimPrefix(arg, "--movies=")
		case arg == "-s" || arg == "--shows" || arg == "--series":
			if i+1 < len(args) {
				i++
				showPath = args[i]
			} else {
				fmt.Fprintln(os.Stderr, "Fehler: Option -s/--shows benötigt einen Pfad.")
				os.Exit(1)
			}
		case strings.HasPrefix(arg, "-s="):
			showPath = strings.TrimPrefix(arg, "-s=")
		case strings.HasPrefix(arg, "--shows="):
			showPath = strings.TrimPrefix(arg, "--shows=")
		case strings.HasPrefix(arg, "--series="):
			showPath = strings.TrimPrefix(arg, "--series=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "Unbekannte Option: %s\nVerwende -h oder --help für Hilfe.\n", arg)
			os.Exit(1)
		default:
			posArgs = append(posArgs, arg)
		}
	}

	// Resolve preset numbers (1-99) if any
	pm, _ := presets.Load()
	if pm != nil {
		if moviePath != "" {
			moviePath = pm.Resolve(moviePath)
		}
		if showPath != "" {
			showPath = pm.Resolve(showPath)
		}
		for i := range posArgs {
			posArgs[i] = pm.Resolve(posArgs[i])
		}
	}

	var model ui.Model
	if moviePath != "" || showPath != "" {
		if len(posArgs) > 0 {
			if moviePath == "" {
				moviePath = posArgs[0]
			} else if showPath == "" {
				showPath = posArgs[0]
			}
		}
		model = ui.InitialModelWithPaths(moviePath, showPath)
	} else if len(posArgs) >= 2 {
		model = ui.InitialModelWithPaths(posArgs[0], posArgs[1])
	} else if len(posArgs) == 1 {
		model = ui.InitialModelWithPath(posArgs[0])
	} else {
		model = ui.InitialModel()
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Starten der Anwendung: %v\n", err)
		os.Exit(1)
	}
}
