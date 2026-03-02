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
	"github.com/laurent22/go-trash"
	"github.com/shirou/gopsutil/v3/disk"
)

type state int

const (
	stateScanning state = iota
	stateResults
	stateCleaning
	stateConfirmFallback
	stateDone
	stateQuitting
)

type SortMode int

const (
	SortSize SortMode = iota
	SortName
)

type UIModel struct {
	scanner            *engine.Scanner
	projects           []engine.Project
	spinner            spinner.Model
	textInput          textinput.Model
	state              state
	cursor             int
	selected           map[string]struct{} // Keyed by project path to persist across filters
	freedSpace         int64
	scanPath           string
	sortMode           SortMode
	includeDeep        bool
	permanent          bool
	width              int
	height             int
	failedTrashCount   int
	failedTrashTargets []string
	
	// Progress tracking for animated cleaning
	totalToClean    int
	cleanedCount    int
	currentCleaning string
	cleaningQueue   []engine.Artifact
}

type cleanStepMsg struct {
	artifact   engine.Artifact
	err        error
	isFallback bool
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#04B575")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	deepStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C5C")).Bold(true) // Red color for danger mode
	safeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))            // Green for safe mode

	paneStyle     = lipgloss.NewStyle().Padding(1, 2)
	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			MarginLeft(2)
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

func (m *UIModel) startCleaning() tea.Cmd {
	m.cleaningQueue = []engine.Artifact{}
	for _, proj := range m.projects {
		if _, ok := m.selected[proj.Path]; ok {
			for _, artifact := range proj.Artifacts {
				if !m.includeDeep && artifact.Tier == engine.TierDeep {
					continue
				}
				m.cleaningQueue = append(m.cleaningQueue, artifact)
			}
		}
	}
	m.totalToClean = len(m.cleaningQueue)
	m.cleanedCount = 0

	if m.totalToClean == 0 {
		m.state = stateDone
		return tea.Quit
	}

	return m.cleanNext()
}

func (m *UIModel) cleanNext() tea.Cmd {
	if len(m.cleaningQueue) == 0 {
		return func() tea.Msg {
			return cleanResult{
				freedSpace:         m.freedSpace,
				failedTrashTargets: m.failedTrashTargets,
			}
		}
	}

	artifact := m.cleaningQueue[0]
	m.cleaningQueue = m.cleaningQueue[1:]
	m.currentCleaning = artifact.Path

	return func() tea.Msg {
		var err error
		if m.permanent {
			err = os.RemoveAll(artifact.Path)
		} else {
			absPath, pathErr := filepath.Abs(artifact.Path)
			if pathErr == nil {
				_, err = trash.MoveToTrash(absPath)
			} else {
				err = pathErr
			}
		}

		// Small delay to make animation visible for small files
		time.Sleep(50 * time.Millisecond)

		return cleanStepMsg{artifact: artifact, err: err}
	}
}

func (m *UIModel) startFallbackCleaning() tea.Cmd {
	m.cleaningQueue = []engine.Artifact{}
	for _, path := range m.failedTrashTargets {
		// Find artifact details
		for _, p := range m.projects {
			for _, a := range p.Artifacts {
				if a.Path == path {
					m.cleaningQueue = append(m.cleaningQueue, a)
					break
				}
			}
		}
	}
	m.failedTrashTargets = []string{} // Reset
	m.totalToClean = len(m.cleaningQueue)
	m.cleanedCount = 0
	m.permanent = true

	return m.cleanNextFallback()
}

func (m *UIModel) cleanNextFallback() tea.Cmd {
	if len(m.cleaningQueue) == 0 {
		return func() tea.Msg {
			return m.freedSpace
		}
	}

	artifact := m.cleaningQueue[0]
	m.cleaningQueue = m.cleaningQueue[1:]
	m.currentCleaning = artifact.Path

	return func() tea.Msg {
		err := os.RemoveAll(artifact.Path)
		time.Sleep(50 * time.Millisecond)
		return cleanStepMsg{artifact: artifact, err: err, isFallback: true}
	}
}

type cleanResult struct {
	freedSpace         int64
	failedTrashTargets []string
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:

		// If the search bar is focused, handle input differently
		if m.state == stateResults && m.textInput.Focused() {
			switch msg.String() {
			case "enter", "esc":
				m.textInput.Blur()
				return m, nil
			case "ctrl+c":
				m.state = stateQuitting
				return m, tea.Quit
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				m.cursor = 0 // Reset cursor when search changes
				return m, cmd
			}
		}

		if m.state == stateConfirmFallback {
			switch strings.ToLower(msg.String()) {
			case "y":
				m.state = stateCleaning
				return m, m.startFallbackCleaning()
			case "n", "esc", "q", "ctrl+c":
				m.state = stateDone
				return m, tea.Quit
			}
			return m, nil
		}

		// Normal Mode Commands
		switch msg.String() {
		case "ctrl+c", "q":
			m.state = stateQuitting
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
		case "a": // Select all visible
			if m.state == stateResults {
				filtered := m.getFilteredProjects()
				allSelected := true

				// First pass to see if everything is already selected
				for _, p := range filtered {
					if _, ok := m.selected[p.Path]; !ok {
						allSelected = false
						break
					}
				}

				if allSelected {
					// If all selected, deselect all visible
					for _, p := range filtered {
						delete(m.selected, p.Path)
					}
				} else {
					// If not all selected, select all visible
					for _, p := range filtered {
						m.selected[p.Path] = struct{}{}
					}
				}
			}
		case "enter":
			if m.state == stateResults && len(m.selected) > 0 {
				m.state = stateCleaning
				m.permanent = false
				return m, m.startCleaning()
			}
		case "X": // Capital X for Nuke
			if m.state == stateResults && len(m.selected) > 0 {
				m.state = stateCleaning
				m.permanent = true
				return m, m.startCleaning()
			}
		}

	case []engine.Project:
		m.projects = msg
		m.state = stateResults
		return m, nil

	case cleanStepMsg:
		if msg.err == nil {
			m.freedSpace += msg.artifact.Size
		} else if !msg.isFallback {
			// Save failed trash targets if not already in fallback
			m.failedTrashTargets = append(m.failedTrashTargets, msg.artifact.Path)
		}
		m.cleanedCount++
		
		if msg.isFallback {
			return m, m.cleanNextFallback()
		}
		return m, m.cleanNext()

	case cleanResult:
		m.freedSpace = msg.freedSpace
		if len(msg.failedTrashTargets) > 0 {
			m.failedTrashTargets = msg.failedTrashTargets
			m.state = stateConfirmFallback
			return m, nil
		}
		m.state = stateDone
		return m, tea.Quit

	case int64: // Finished fallback cleaning
		m.freedSpace += msg
		m.state = stateDone
		return m, tea.Quit

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

func getIcon(projectType string) string {
	switch projectType {
	case "Node.js / JS Ecosystem":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#83CD29")).Render("") // Node green
	case "Python":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#3776AB")).Render("") // Python blue
	case "Rust":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#DEA584")).Render("") // Rust orange
	case "Go":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Render("") // Go cyan
	case "Java / JVM":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ED8B00")).Render("") // Java orange
	case "PHP":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#777BB4")).Render("") // PHP purple
	case "Ruby":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CC342D")).Render("") // Ruby red
	case ".NET / C#":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#512BD4")).Render("") // .NET purple
	case "Elixir":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#4E2A8E")).Render("") // Elixir deep purple
	case "Terraform / IaC":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7B42BC")).Render("󱁢") // Terraform purple
	case "OS & Editor Caches":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("⚙️")
	default:
		return "📁"
	}
}

func (m UIModel) View() string {
	switch m.state {
	case stateScanning:
		return fmt.Sprintf("\n %s Scanning %s and verifying Git status...\n\n", m.spinner.View(), m.scanPath)

	case stateResults:
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

		// Build Left Pane
		var leftContent strings.Builder
		leftContent.WriteString(titleStyle.Render("🚀 KESSLER: COMMAND CENTER") + "\n\n")
		leftContent.WriteString(fmt.Sprintf(" %s\n\n", m.textInput.View()))
		leftContent.WriteString(fmt.Sprintf(" Found %d projects | Total Debris: %s | Sort: %s | Mode: %s\n\n", len(filtered), formatBytes(totalSelectable), sortText, tierText))

		if len(filtered) == 0 {
			leftContent.WriteString(" No project artifacts match the current filters. Orbit is clear!\n")
		} else {
			for i, proj := range filtered {
				// Pagination
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
				// Truncate baseName if it's too long
				if len(baseName) > 25 {
					baseName = baseName[:22] + "..."
				}

				// Calculate time since last touch
				var timeStr string
				if !proj.LastModTime.IsZero() {
					duration := time.Since(proj.LastModTime)
					days := int(duration.Hours() / 24)
					if days > 365 {
						timeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C5C")).Render(fmt.Sprintf("%dy", days/365)) // Red if > 1 year
					} else if days > 30 {
						timeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Render(fmt.Sprintf("%dd", days)) // Yellow if > 1 month
					} else if days == 0 {
						timeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("today")
					} else {
						timeStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(fmt.Sprintf("%dd", days)) // Grey if recent
					}
				}

				// We want a clean columnar look. Format it with padding.
				baseNamePadded := fmt.Sprintf("%-26s", baseName)
				timeStrPadded := fmt.Sprintf("[%5s]", timeStr)
				sizeStr := formatBytes(proj.TotalSize)

				line := fmt.Sprintf("%s [%s] %s %s %s %s - %s", cursor, checked, getIcon(proj.Type), baseNamePadded, timeStrPadded, proj.Type, sizeStr)

				if _, ok := m.selected[proj.Path]; ok {
					leftContent.WriteString(selectedStyle.Render(line) + "\n")
				} else {
					leftContent.WriteString(itemStyle.Render(line) + "\n")
				}
			}
		}
		leftContent.WriteString(helpStyle.Render("\n ↑/↓: nav   •   space: select   •   a: select all   •   t: toggle tier   •   s: sort   •   /: search   •   q: quit\n\n enter: trash them   •   X: nuke them\n"))

		// Build Right Pane (Stats)
		var rightContent strings.Builder
		rightContent.WriteString(lipgloss.NewStyle().Bold(true).Render("📊 ORBITAL TELEMETRY") + "\n\n")
		// Disk Usage
		usage, err := disk.Usage("/")
		if err == nil {
			usedPercent := usage.UsedPercent
			rightContent.WriteString(fmt.Sprintf("Drive Total : %s\n", formatBytes(int64(usage.Total))))
			rightContent.WriteString(fmt.Sprintf("Drive Used  : %s (%.1f%%)\n", formatBytes(int64(usage.Used)), usedPercent))
			rightContent.WriteString(fmt.Sprintf("Drive Free  : %s\n\n", formatBytes(int64(usage.Free))))

			// Simple progress bar
			barWidth := 25
			usedBlocks := int((usedPercent / 100.0) * float64(barWidth))
			if usedBlocks > barWidth {
				usedBlocks = barWidth
			}
			if usedBlocks < 0 {
				usedBlocks = 0
			}
			bar := strings.Repeat("█", usedBlocks) + strings.Repeat("░", barWidth-usedBlocks)

			barColor := lipgloss.Color("#04B575") // Green
			if usedPercent > 90 {
				barColor = lipgloss.Color("#FF5C5C") // Red
			} else if usedPercent > 75 {
				barColor = lipgloss.Color("#E5C07B") // Yellow
			}
			coloredBar := lipgloss.NewStyle().Foreground(barColor).Render(bar)
			rightContent.WriteString(fmt.Sprintf("[%s]\n\n", coloredBar))
		} else {
			rightContent.WriteString("Drive Stats : Unavailable\n\n")
		}

		// Selected Debris Size
		var selectedSize int64
		for _, proj := range m.projects {
			if _, ok := m.selected[proj.Path]; ok {
				for _, artifact := range proj.Artifacts {
					if !m.includeDeep && artifact.Tier == engine.TierDeep {
						continue
					}
					selectedSize += artifact.Size
				}
			}
		}

		rightContent.WriteString(fmt.Sprintf("Total Debris: %s\n", formatBytes(totalSelectable)))
		rightContent.WriteString(fmt.Sprintf("Selected    : %s\n", selectedStyle.Render(formatBytes(selectedSize))))
		rightContent.WriteString(fmt.Sprintf("Projects    : %d\n\n", len(filtered)))

		// Type Breakdown
		breakdown := make(map[string]int64)
		for _, p := range filtered {
			breakdown[p.Type] += p.TotalSize
		}

		if len(breakdown) > 0 {
			rightContent.WriteString("Breakdown by Type:\n")
			type kv struct {
				Key   string
				Value int64
			}
			var ss []kv
			for k, v := range breakdown {
				ss = append(ss, kv{k, v})
			}
			sort.Slice(ss, func(i, j int) bool {
				return ss[i].Value > ss[j].Value
			})
			for i, kv := range ss {
				if i > 5 { // Show top 6 types to avoid making the box too tall
					rightContent.WriteString(" • ...\n")
					break
				}
				rightContent.WriteString(fmt.Sprintf(" • %-15s: %s\n", kv.Key, formatBytes(kv.Value)))
			}
		}

		// Preview Pane (Deep Context)
		if len(filtered) > 0 && m.cursor < len(filtered) {
			activeProj := filtered[m.cursor]

			rightContent.WriteString("\n")
			rightContent.WriteString(lipgloss.NewStyle().Bold(true).Render("🎯 TARGET PREVIEW") + "\n\n")
			rightContent.WriteString(fmt.Sprintf("%s %s\n", getIcon(activeProj.Type), filepath.Base(activeProj.Path)))

			for _, artifact := range activeProj.Artifacts {
				if !m.includeDeep && artifact.Tier == engine.TierDeep {
					continue
				}

				tierColor := lipgloss.Color("#04B575") // Safe Green
				if artifact.Tier == engine.TierDeep {
					tierColor = lipgloss.Color("#E5C07B") // Deep Yellow
				} else if artifact.Tier == engine.TierDanger {
					tierColor = lipgloss.Color("#FF5C5C") // Danger Red
				}

				coloredPath := lipgloss.NewStyle().Foreground(tierColor).Render(filepath.Base(artifact.Path))
				rightContent.WriteString(fmt.Sprintf(" ├─ %-15s : %s\n", coloredPath, formatBytes(artifact.Size)))
			}
		}

		statsBox := statsBoxStyle.Render(rightContent.String())

		// We need to calculate widths so the layout works nicely.
		// If width is 0 (WindowSizeMsg hasn't arrived yet), we'll guess a width.
		windowWidth := m.width
		if windowWidth == 0 {
			windowWidth = 100
		}

		statsBoxWidth := lipgloss.Width(statsBox)
		leftPaneWidth := windowWidth - statsBoxWidth - 4
		if leftPaneWidth < 50 {
			leftPaneWidth = 50 // fallback width to ensure content doesn't crash formatting
		}

		leftPaneContent := lipgloss.NewStyle().Width(leftPaneWidth).Render(leftContent.String())

		return paneStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftPaneContent, statsBox))

	case stateCleaning:
		msg := "Safely moving debris to Trash Bin..."
		if m.permanent {
			msg = "Permanently nuking debris (No undo!)..."
		}
		
		// Progress bar
		width := 40
		if m.width > 0 && m.width < 50 {
			width = m.width - 10
		}
		
		progress := float64(m.cleanedCount) / float64(m.totalToClean)
		filled := int(progress * float64(width))
		if filled > width {
			filled = width
		}
		
		bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		coloredBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(bar)
		
		// Truncate current cleaning path
		current := m.currentCleaning
		if len(current) > 50 {
			current = "..." + current[len(current)-47:]
		}
		
		return fmt.Sprintf(
			"\n 🧹 Firing orbital lasers...\n\n %s\n\n %s [%d/%d]\n\n Vaporizing: %s\n",
			msg,
			coloredBar,
			m.cleanedCount,
			m.totalToClean,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C5C")).Render(current),
		)

	case stateDone:
		msg := "safely moved to the Trash."
		if m.permanent {
			msg = "permanently deleted."
		}

		var failMsg string
		if len(m.failedTrashTargets) > 0 && !m.permanent {
			failMsg = fmt.Sprintf("\n ⚠️  Note: %d items could not be deleted.\n", len(m.failedTrashTargets))
		}

		return fmt.Sprintf("\n ✨ Cleanup Complete! %s %s%s\n\n", formatBytes(m.freedSpace), msg, failMsg)

	case stateConfirmFallback:
		s := deepStyle.Render("\n ⚠️  OS TRASH FAILED") + "\n\n"
		s += fmt.Sprintf(" Kessler tried to safely trash %d item(s), but your OS rejected the operation.\n", len(m.failedTrashTargets))
		s += " This usually happens when clearing items across different drive partitions.\n\n"
		s += deepStyle.Render(" Do you want to PERMANENTLY NUKE these remaining items? (y/n)") + "\n\n"
		return s

	case stateQuitting:
		return "\n 👋 Orbit clear. Catch you on the next sweep!\n\n"
	}
	return ""
}
