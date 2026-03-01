package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hariharen/kessler/internal/tui"
)

func main() {
	// Simple argument parsing for MVP, later switch to Cobra entirely
	scanPath := "."
	if len(os.Args) > 1 {
		scanPath = os.Args[1]
	}

	p := tea.NewProgram(tui.InitialModel(scanPath, false))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
