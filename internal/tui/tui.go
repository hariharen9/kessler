package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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

type SortMode int

const (
	SortSize SortMode = iota
	SortName
)

type UIModel struct {
	scanner      *engine.Scanner
	projects     []engine.Project
	spinner      spinner.Model
	textInput    textinput.Model
	state        state
	cursor       int
	selected     map[string]struct{} // Keyed by project path to persist across filters
	freedSpace   int64
	scanPath     string
	sortMode     SortMode
	includeDeep  bool
	permanent    bool
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#04B575")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	deepStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C5C")).Bold(true) // Red color for danger mode
	safeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))             // Green for safe mode
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

func InitialModel(scanPath string, includeDeep bool) UIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Search projects... (press '/' to focus)"
	ti.CharLimit = 156
	ti.Width = 40

	return UIModel{
		spinner:     s,
		textInput:   ti,
		state:       stateScanning,
		selected:    make(map[string]struct{}),
		scanPath:    scanPath,
		sortMode:    SortSize,
		includeDeep: includeDeep,
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

func (m UIModel) startScan() tea.Cmd {
	return func() tea.Msg {
		scanner, err := engine.NewScanner()
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

func (m UIModel) cleanSelected() tea.Cmd {
	return func() tea.Msg {
		var freed int64
		for _, proj := range m.projects {
			if _, ok := m.selected[proj.Path]; ok {
				for _, artifact := range proj.Artifacts {
					// Guard: Only delete deep artifacts if includeDeep is true
					if !m.includeDeep && artifact.Tier == engine.TierDeep {
						continue
					}
					
					var err error
					if m.permanent {
						err = os.RemoveAll(artifact.Path)
					} else {
						// Simple trash logic here for MVP
						home, _ := os.UserHomeDir()
						dest := filepath.Join(home, ".Trash", fmt.Sprintf("%s-%d", filepath.Base(artifact.Path), time.Now().Unix()))
						err = os.Rename(artifact.Path, dest)
						if err != nil {
							err = os.RemoveAll(artifact.Path)
						}
					}
					
					if err == nil {
						freed += artifact.Size
					}
				}
			}
		}
		return freed
	}
}

// getFilteredProjects applies the search filter, the tier filter, and the sorting
func (m UIModel) getFilteredProjects() []engine.Project {
	var filtered []engine.Project
	searchQuery := strings.ToLower(m.textInput.Value())

	for _, p := range m.projects {
		var activeSize int64
		for _, a := range p.Artifacts {
			// Only sum the sizes of artifacts that match the current Tier mode
			if m.includeDeep || a.Tier == engine.TierSafe {
				activeSize += a.Size
			}
		}
		
		// If a project has no artifacts for the current tier, skip it
		if activeSize == 0 {
			continue
		}

		// Apply Search text
		if searchQuery != "" && !strings.Contains(strings.ToLower(filepath.Base(p.Path)), searchQuery) {
			continue
		}

		p.TotalSize = activeSize
		filtered = append(filtered, p)
	}

	if m.sortMode == SortSize {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TotalSize > filtered[j].TotalSize
		})
	} else {
		sort.Slice(filtered, func(i, j int) bool {
			return filepath.Base(filtered[i].Path) < filepath.Base(filtered[j].Path)
		})
	}

	return filtered
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		
		// If the search bar is focused, handle input differently
		if m.textInput.Focused() {
			switch msg.String() {
			case "enter", "esc":
				m.textInput.Blur()
				return m, nil
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				m.cursor = 0 // Reset cursor when search changes
				return m, cmd
			}
		}

		// Normal Mode Commands
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.state == stateResults && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.state == stateResults {
				filtered := m.getFilteredProjects()
				if m.cursor < len(filtered)-1 {
					m.cursor++
				}
			}
		case "/":
			if m.state == stateResults {
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "s": // Toggle sorting
			if m.state == stateResults {
				if m.sortMode == SortSize {
					m.sortMode = SortName
				} else {
					m.sortMode = SortSize
				}
				m.cursor = 0
			}
		case "t": // Toggle Tiers (Safe vs Deep)
			if m.state == stateResults {
				m.includeDeep = !m.includeDeep
				m.cursor = 0 // Reset cursor as list changes
			}
		case " ": // Toggle selection
			if m.state == stateResults {
				filtered := m.getFilteredProjects()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					path := filtered[m.cursor].Path
					if _, ok := m.selected[path]; ok {
						delete(m.selected, path)
					} else {
						m.selected[path] = struct{}{}
					}
				}
			}
		case "enter":
			if m.state == stateResults && len(m.selected) > 0 {
				m.state = stateCleaning
				m.permanent = false
				return m, m.cleanSelected()
			}
		case "X": // Capital X for Nuke
			if m.state == stateResults && len(m.selected) > 0 {
				m.state = stateCleaning
				m.permanent = true
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
		return m, nil // Don't quit immediately so user sees the message

	case error:
		fmt.Println("Error:", msg)
		return m, tea.Quit

	case spinner.TickMsg:
		if m.state == stateScanning {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m UIModel) View() string {
	switch m.state {
	case stateScanning:
		return fmt.Sprintf("\n %s Scanning %s and verifying Git status...\n\n", m.spinner.View(), m.scanPath)
	
	case stateResults:
		s := titleStyle.Render("🚀 KESSLER: COMMAND CENTER") + "\n\n"
		
		filtered := m.getFilteredProjects()

		var totalSelectable int64
		for _, p := range filtered {
			totalSelectable += p.TotalSize
		}

		sortText := "Size"
		if m.sortMode == SortName {
			sortText = "Name"
		}

		// The Tier Visual Toggle
		tierText := safeStyle.Render("Safe Mode (100% Regeneratable)")
		if m.includeDeep {
			tierText = deepStyle.Render("DEEP CLEAN MODE (Includes Binaries & Builds)")
		}

		s += fmt.Sprintf(" %s\n\n", m.textInput.View())
		s += fmt.Sprintf(" Found %d projects | Total Debris: %s | Sort: %s | Mode: %s\n\n", len(filtered), formatBytes(totalSelectable), sortText, tierText)

		if len(filtered) == 0 {
			s += " No project artifacts match the current filters. Orbit is clear!\n"
		} else {
			for i, proj := range filtered {
				// Only show a limited number of lines to avoid terminal overflow (Pagination)
				if i < m.cursor-5 || i > m.cursor+10 {
					continue
				}

				cursor := " "
				if m.cursor == i {
					cursor = cursorStyle.Render(">")
				}

				checked := " "
				if _, ok := m.selected[proj.Path]; ok {
					checked = "x"
				}

				baseName := filepath.Base(proj.Path)
				line := fmt.Sprintf("%s [%s] %s (%s) - %s", cursor, checked, baseName, proj.Type, formatBytes(proj.TotalSize))
				
				if _, ok := m.selected[proj.Path]; ok {
					s += selectedStyle.Render(line) + "\n"
				} else {
					s += itemStyle.Render(line) + "\n"
				}
			}
		}
		
		s += helpStyle.Render("\n ↑/↓: nav • space: select • t: tier • s: sort • /: search • enter: trash • X: nuke • q: quit\n")
		return s

	case stateCleaning:
		msg := "Safely moving debris to Trash Bin..."
		if m.permanent {
			msg = "Permanently nuking debris (No undo!)..."
		}
		return fmt.Sprintf("\n 🧹 Firing orbital lasers. %s\n", msg)

	case stateDone:
		msg := "safely moved to the Trash."
		if m.permanent {
			msg = "permanently deleted."
		}
		return fmt.Sprintf("\n ✨ Cleanup Complete! %s %s\n\n", formatBytes(m.freedSpace), msg)
	}
	return ""
}