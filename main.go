package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"smrt/pkg/ui"
)

func main() {
	var model ui.Model
	if len(os.Args) > 1 {
		model = ui.InitialModelWithPath(os.Args[1])
	} else {
		model = ui.InitialModel()
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fehler beim Starten der Anwendung: %v\n", err)
		os.Exit(1)
	}
}
