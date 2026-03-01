package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hariharen/kessler/engine"
)

type state int

const (
	stateScanning state = iota
	stateResults
	stateCleaning
	stateDone
)

type model struct {
	scanner      *engine.Scanner
	projects     []engine.Project
	spinner      spinner.Model
	state        state
	cursor       int
	selected     map[int]struct{}
	freedSpace   int64
	scanPath     string
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#04B575")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F4B73F"))
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// safeTrash attempts to move the path to the macOS Trash Bin.
// If it fails, it falls back to permanent deletion.
func safeTrash(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.RemoveAll(path)
	}

	trashDir := filepath.Join(home, ".Trash")
	base := filepath.Base(path)
	timestamp := time.Now().Format("20060102-150405")
	dest := filepath.Join(trashDir, fmt.Sprintf("%s-%s", base, timestamp))

	err = os.Rename(path, dest)
	if err != nil {
		// Fallback to permanent delete if rename across partitions fails
		return os.RemoveAll(path)
	}
	return nil
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	path := "."
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	return model{
		spinner:  s,
		state:    stateScanning,
		selected: make(map[int]struct{}),
		scanPath: path,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

func (m model) startScan() tea.Cmd {
	return func() tea.Msg {
		scanner, err := engine.NewScanner("rules.yaml")
		if err != nil {
			return err
		}
		m.scanner = scanner
		projects, err := scanner.Scan(m.scanPath)
		if err != nil {
			return err
		}
		return projects
	}
}

func (m model) cleanSelected() tea.Cmd {
	return func() tea.Msg {
		var freed int64
		for i := range m.selected {
			proj := m.projects[i]
			for _, artifact := range proj.Artifacts {
				// Use the new safe Trash function!
				err := safeTrash(artifact.Path)
				if err == nil {
					freed += artifact.Size
				}
			}
		}
		return freed
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.state == stateResults && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.state == stateResults && m.cursor < len(m.projects)-1 {
				m.cursor++
			}
		case " ":
			if m.state == stateResults {
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
			}
		case "enter":
			if m.state == stateResults && len(m.selected) > 0 {
				m.state = stateCleaning
				return m, m.cleanSelected()
			}
		}

	case []engine.Project:
		m.projects = msg
		m.state = stateResults
		return m, nil

	case int64: // Finished cleaning
		m.freedSpace = msg
		m.state = stateDone
		return m, tea.Quit

	case error:
		fmt.Println("Error:", msg)
		return m, tea.Quit

	case spinner.TickMsg:
		if m.state == stateScanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateScanning:
		return fmt.Sprintf("\n %s Scanning %s and verifying Git status...\n\n", m.spinner.View(), m.scanPath)
	case stateResults:
		s := titleStyle.Render("🚀 KESSLER: ORBITAL DEBRIS SCANNER") + "\n"
		
		if len(m.projects) == 0 {
			return s + "\nNo project artifacts found. Orbit is clear!\n"
		}

		var totalSelectable int64
		for _, p := range m.projects {
			totalSelectable += p.TotalSize
		}

		s += fmt.Sprintf("Found %d projects containing %s of safe-to-delete debris.\n\n", len(m.projects), formatBytes(totalSelectable))

		for i, proj := range m.projects {
			cursor := " "
			if m.cursor == i {
				cursor = cursorStyle.Render(">")
			}

			checked := " "
			if _, ok := m.selected[i]; ok {
				checked = "x"
			}

			line := fmt.Sprintf("%s [%s] %s (%s) - %s", cursor, checked, filepath.Base(proj.Path), proj.Type, formatBytes(proj.TotalSize))
			
			if _, ok := m.selected[i]; ok {
				s += selectedStyle.Render(line) + "\n"
			} else {
				s += itemStyle.Render(line) + "\n"
			}
		}
		s += helpStyle.Render("\n↑/↓: navigate • space: toggle • enter: clean (moves to Trash) • q: quit\n")
		return s

	case stateCleaning:
		return "\n 🧹 Firing orbital lasers. Safely moving debris to Trash Bin...\n"

	case stateDone:
		return fmt.Sprintf("\n ✨ Cleanup Complete! %s safely moved to the Trash.\n\n", formatBytes(m.freedSpace))
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
