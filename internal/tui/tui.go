package tui

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hariharen9/kessler/engine"
	"github.com/shirou/gopsutil/v3/disk"
)

type state int

const (
	stateScanning state = iota
	stateResults
	stateConfirmNuke
	stateConfirmActive
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

type tabIndex int

const (
	tabProjects tabIndex = iota
	tabGlobal
	tabHistory
	tabLaunchpad
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
	scanPaths          []string
	sortMode           SortMode
	includeDeep        bool
	showIgnored        bool
	permanent          bool
	width              int
	height             int
	failedTrashCount   int
	failedTrashTargets []string

	// Preview modal
	showPreviewModal bool
	previewCursor    int

	// Post-clean feedback
	lastFreedSpace int64
	wasNukeRequest bool

	// Active project detection
	activeProjects map[string][]engine.ActiveProcess // Keyed by project path

	// Configuration
	rulesData     []byte
	userRulesData []byte

	// Progress tracking for animated cleaning
	totalToClean    int
	cleanedCount    int
	currentCleaning string
	cleaningQueue   []engine.Artifact

	// Live scanning progress
	scanDirsChecked   int
	scanProjectsFound int
	scanCurrentDir    string
	scanLatestProject string
	scanTotalSize     int64
	scanQuoteIndex    int

	// Tab navigation
	activeTab tabIndex

	// Global caches (Tab 2)
	globalCaches       []engine.GlobalCache
	globalSelected     map[int]struct{}
	globalCursor       int
	globalScanning     bool
	globalProcessed    int
	globalTotal        int
	globalCurrentCache string
	globalTotalSize    int64

	// Environmental Doctor (Tab 2)
	toolchains []engine.Toolchain
}

type cleanStepMsg struct {
	artifact   engine.Artifact
	err        error
	isFallback bool
}

type globalScanResult struct {
	caches []engine.GlobalCache
}

type globalScanProgressMsg struct {
	engine.GlobalScanProgress
	ch chan tea.Msg
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#04B575")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	deepStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5C5C")).Bold(true) // Red color for danger mode
	safeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))            // Green for safe mode
	ignoredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Bold(true) // Yellow for ignored

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

func InitialModel(scanPaths []string, includeDeep bool, rulesData []byte, userRulesData []byte) UIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Search projects... (press '/' to focus)"
	ti.CharLimit = 156
	ti.Width = 40

	return UIModel{
		spinner:        s,
		textInput:      ti,
		state:          stateScanning,
		selected:       make(map[string]struct{}),
		globalSelected: make(map[int]struct{}),
		activeProjects: make(map[string][]engine.ActiveProcess),
		scanPaths:      scanPaths,
		sortMode:       SortSize,
		includeDeep:    includeDeep,
		rulesData:      rulesData,
		userRulesData:  userRulesData,
		scanQuoteIndex: rand.Intn(len(spaceQuotes)),
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

type scanComplete struct {
	projects []engine.Project
	err      error
}

// scanProgressMsg wraps progress + the channel for the next read
type scanProgressMsg struct {
	engine.ScanProgress
	ch chan tea.Msg
}

var spaceQuotes = []string{
	"Space debris travels at 17,500 mph. Your node_modules aren't far behind.",
	"Houston, we have a storage problem.",
	"One small sweep for dev, one giant save for disk space.",
	"In space, no one can hear your SSD scream.",
	"The universe is expanding. So is your build folder.",
	"Ground control to Major Dev... your disk is 87% full.",
	"Ad astra per aspera — through junk to the stars.",
	"That's one small delete for man, one giant cleanup for mankind.",
}

func (m UIModel) startScan() tea.Cmd {
	ch := make(chan tea.Msg, 50)

	go func() {
		scanner, err := engine.NewScannerMerged(m.rulesData, m.userRulesData)
		if err != nil {
			ch <- scanComplete{err: err}
			return
		}

		progress := make(chan engine.ScanProgress, 10)
		var projects []engine.Project
		var scanErr error

		go func() {
			projects, scanErr = scanner.ScanWithProgress(m.scanPaths, progress)
			close(progress)
		}()

		for p := range progress {
			ch <- scanProgressMsg{ScanProgress: p, ch: ch}
		}

		ch <- scanComplete{projects: projects, err: scanErr}
	}()

	return waitForProgress(ch)
}

func (m UIModel) scanGlobalCachesCmd() tea.Cmd {
	ch := make(chan tea.Msg, 50)

	go func() {
		progress := make(chan engine.GlobalScanProgress, 10)
		var caches []engine.GlobalCache

		go func() {
			caches = engine.ScanGlobalCachesWithProgress(progress)
			close(progress)
		}()

		for p := range progress {
			ch <- globalScanProgressMsg{GlobalScanProgress: p, ch: ch}
		}

		ch <- globalScanResult{caches: caches}
	}()

	return waitForProgress(ch)
}

// waitForProgress reads one message from the channel and returns it
func waitForProgress(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
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
				if !m.showIgnored && artifact.Tier == engine.TierIgnored {
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
				err = engine.MoveToTrash(absPath)
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
			if a.Tier == engine.TierIgnored && !m.showIgnored {
				continue
			}
			// Only sum the sizes of artifacts that match the current Tier mode
			if m.includeDeep || a.Tier == engine.TierSafe || (a.Tier == engine.TierIgnored && m.showIgnored) {
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

func (m *UIModel) checkActiveProjects() bool {
	m.activeProjects = make(map[string][]engine.ActiveProcess)
	found := false
	for path := range m.selected {
		procs := engine.GetActiveProcessesInPath(path)
		if len(procs) > 0 {
			m.activeProjects[path] = procs
			found = true
		}
	}
	return found
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

		if m.state == stateConfirmActive {
			switch strings.ToLower(msg.String()) {
			case "y":
				if m.wasNukeRequest {
					m.state = stateConfirmNuke
				} else {
					m.state = stateCleaning
					m.permanent = false
					return m, m.startCleaning()
				}
				return m, nil
			case "n", "esc", "q":
				m.state = stateResults
				return m, nil
			}
			return m, nil
		}

		if m.state == stateConfirmNuke {
			switch strings.ToLower(msg.String()) {
			case "y":
				if m.activeTab == tabGlobal {
					// Execute global cache cleaning
					var freedSpace int64
					for idx := range m.globalSelected {
						if idx < len(m.globalCaches) {
							cache := m.globalCaches[idx]
							freedSpace += cache.Size
							engine.CleanGlobalCache(cache)
							engine.SaveCleanEntry(cache.Name, cache.Size)
						}
					}
					m.lastFreedSpace = freedSpace
					m.globalSelected = make(map[int]struct{})
					m.globalCursor = 0
					m.globalScanning = true
					m.state = stateResults
					return m, m.scanGlobalCachesCmd()
				}
				m.state = stateCleaning
				m.permanent = true
				return m, m.startCleaning()
			case "n", "esc", "q":
				m.state = stateResults
				return m, nil
			}
			return m, nil
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
			if m.state == stateResults && m.showPreviewModal {
				if m.previewCursor > 0 {
					m.previewCursor--
				}
				return m, nil
			}
			if m.state == stateResults {
				if m.activeTab == tabProjects && m.cursor > 0 {
					m.cursor--
				} else if m.activeTab == tabGlobal && m.globalCursor > 0 {
					m.globalCursor--
				} else if m.activeTab == tabLaunchpad && m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.state == stateResults && m.showPreviewModal {
				filtered := m.getFilteredProjects()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					activeProj := filtered[m.cursor]
					var visibleArtifacts []engine.Artifact
					for _, artifact := range activeProj.Artifacts {
						if !m.includeDeep && artifact.Tier == engine.TierDeep {
							continue
						}
						if !m.showIgnored && artifact.Tier == engine.TierIgnored {
							continue
						}
						visibleArtifacts = append(visibleArtifacts, artifact)
					}
					if m.previewCursor < len(visibleArtifacts)-1 {
						m.previewCursor++
					}
				}
				return m, nil
			}
			if m.state == stateResults {
				if m.activeTab == tabProjects || m.activeTab == tabLaunchpad {
					filtered := m.getFilteredProjects()
					if m.cursor < len(filtered)-1 {
						m.cursor++
					}
				} else if m.activeTab == tabGlobal {
					if m.globalCursor < len(m.globalCaches)-1 {
						m.globalCursor++
					}
				}
			}
		case "/":
			if m.state == stateResults {
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "o": // Open in VS Code
			if m.state == stateResults && m.activeTab == tabLaunchpad {
				filtered := m.getFilteredProjects()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					path := filtered[m.cursor].Path
					if runtime.GOOS == "windows" {
						exec.Command("cmd", "/c", "code", path).Run()
						exec.Command("cmd", "/c", "cursor", path).Run()
					} else {
						exec.Command("code", path).Run()
						exec.Command("cursor", path).Run()
					}
				}
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
		case "t": // Open in Terminal (Launchpad) OR Toggle Tiers (Projects)
			if m.state == stateResults {
				if m.activeTab == tabLaunchpad {
					filtered := m.getFilteredProjects()
					if len(filtered) > 0 && m.cursor < len(filtered) {
						path := filtered[m.cursor].Path
						if runtime.GOOS == "darwin" {
							exec.Command("open", "-a", "Terminal", path).Run()
							exec.Command("open", "-a", "iTerm", path).Run()
						} else if runtime.GOOS == "windows" {
							exec.Command("cmd", "/c", "start", "cmd", "/K", "cd /d", path).Run()
						} else {
							// Linux: try xdg-open (file manager) or common terminals
							exec.Command("xdg-open", path).Run()
						}
					}
				} else {
					m.includeDeep = !m.includeDeep
					m.cursor = 0 // Reset cursor as list changes
				}
			}
		case "i": // Toggle Ignored files
			if m.state == stateResults {
				m.showIgnored = !m.showIgnored
				m.cursor = 0
			}
		case "p": // Toggle preview modal
			if m.state == stateResults {
				m.showPreviewModal = !m.showPreviewModal
				m.previewCursor = 0
			}
		case " ": // Toggle selection
			if m.state == stateResults && m.activeTab == tabProjects {
				filtered := m.getFilteredProjects()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					path := filtered[m.cursor].Path
					if _, ok := m.selected[path]; ok {
						delete(m.selected, path)
					} else {
						m.selected[path] = struct{}{}
					}
				}
			} else if m.state == stateResults && m.activeTab == tabGlobal {
				if m.globalCursor < len(m.globalCaches) {
					if _, ok := m.globalSelected[m.globalCursor]; ok {
						delete(m.globalSelected, m.globalCursor)
					} else {
						m.globalSelected[m.globalCursor] = struct{}{}
					}
				}
			}
		case "a": // Select all visible
			if m.state == stateResults && m.activeTab == tabProjects {
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
		case "e": // Select all of same type
			if m.state == stateResults && m.activeTab == tabProjects {
				filtered := m.getFilteredProjects()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					targetType := filtered[m.cursor].Type
					for _, p := range filtered {
						if p.Type == targetType {
							m.selected[p.Path] = struct{}{}
						}
					}
				}
			}
		case "S": // Select all stale (>30 days)
			if m.state == stateResults && m.activeTab == tabProjects {
				filtered := m.getFilteredProjects()
				staleThreshold := 30 * 24 * time.Hour
				for _, p := range filtered {
					if !p.LastModTime.IsZero() && time.Since(p.LastModTime) > staleThreshold {
						m.selected[p.Path] = struct{}{}
					}
				}
			}
		case "enter":
			if m.state == stateResults && m.activeTab == tabProjects && len(m.selected) > 0 {
				m.wasNukeRequest = false
				if m.checkActiveProjects() {
					m.state = stateConfirmActive
					return m, nil
				}
				m.state = stateCleaning
				m.permanent = false
				return m, m.startCleaning()
			} else if m.state == stateResults && m.activeTab == tabGlobal && len(m.globalSelected) > 0 {
				// Show confirmation for global cleaning
				m.state = stateConfirmNuke // Reuse nuke confirmation with global context
				m.permanent = false
				return m, nil
			}
		case "X": // Capital X for Nuke — show confirmation first
			if m.state == stateResults && m.activeTab == tabProjects && len(m.selected) > 0 {
				m.wasNukeRequest = true
				if m.checkActiveProjects() {
					m.state = stateConfirmActive
					return m, nil
				}
				m.state = stateConfirmNuke
				return m, nil
			}
		case "1":
			if m.state == stateResults {
				m.activeTab = tabProjects
			}
		case "2":
			if m.state == stateResults {
				m.activeTab = tabGlobal
				if m.globalCaches == nil && !m.globalScanning {
					m.globalScanning = true
					return m, m.scanGlobalCachesCmd()
				}
			}
		case "3":
			if m.state == stateResults {
				m.activeTab = tabHistory
			}
		case "4":
			if m.state == stateResults {
				m.activeTab = tabLaunchpad
			}
		case "tab":
			if m.state == stateResults {
				m.activeTab = (m.activeTab + 1) % 4
				if m.activeTab == tabGlobal && m.globalCaches == nil && !m.globalScanning {
					m.globalScanning = true
					return m, m.scanGlobalCachesCmd()
				}
			}
		}

	case globalScanResult:
		m.globalCaches = msg.caches
		m.globalScanning = false
		return m, nil

	case globalScanProgressMsg:
		m.globalProcessed = msg.CachesProcessed
		m.globalTotal = msg.TotalCaches
		m.globalCurrentCache = msg.CurrentCache
		m.globalTotalSize = msg.TotalSize
		return m, waitForProgress(msg.ch)

	case []engine.Project:
		m.projects = msg
		m.state = stateResults
		return m, nil

	case scanProgressMsg:
		m.scanDirsChecked = msg.DirsChecked
		m.scanProjectsFound = msg.ProjectsFound
		m.scanCurrentDir = msg.CurrentDir
		m.scanTotalSize = msg.TotalSize
		if msg.LatestProject != "" {
			m.scanLatestProject = msg.LatestProject
		}
		return m, waitForProgress(msg.ch)

	case scanComplete:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.projects = msg.projects
		m.state = stateResults
		m.toolchains = engine.GetUnusedToolchains(msg.projects)

		// Save scan to history
		var totalSize int64
		for _, p := range msg.projects {
			totalSize += p.TotalSize
		}
		engine.SaveEntry(engine.ScanHistoryEntry{
			Timestamp:    time.Now(),
			ScanPath:     strings.Join(m.scanPaths, ", "),
			ProjectCount: len(msg.projects),
			TotalSize:    totalSize,
		})
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
		m.lastFreedSpace = m.freedSpace
		m.freedSpace = 0
		m.selected = make(map[string]struct{})
		m.cursor = 0
		m.cleanedCount = 0
		m.totalToClean = 0
		m.state = stateScanning
		return m, m.startScan()

	case int64: // Finished fallback cleaning
		m.lastFreedSpace = m.freedSpace + msg
		m.freedSpace = 0
		m.selected = make(map[string]struct{})
		m.cursor = 0
		m.cleanedCount = 0
		m.totalToClean = 0
		m.state = stateScanning
		return m, m.startScan()

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
		// Truncate current dir for display
		shortDir := m.scanCurrentDir
		if len(shortDir) > 45 {
			shortDir = "..." + shortDir[len(shortDir)-42:]
		}

		quote := spaceQuotes[m.scanQuoteIndex]
		quoteStyled := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#626262")).Render("\"" + quote + "\"")

		latestProj := m.scanLatestProject
		if latestProj == "" {
			latestProj = "—"
		}

		// Scan history line
		var historyLine string
		history := engine.LoadHistory()
		if last := history.LastEntry(); last != nil {
			ago := time.Since(last.Timestamp)
			var agoStr string
			if ago.Hours() < 1 {
				agoStr = fmt.Sprintf("%dm ago", int(ago.Minutes()))
			} else if ago.Hours() < 24 {
				agoStr = fmt.Sprintf("%dh ago", int(ago.Hours()))
			} else {
				agoStr = fmt.Sprintf("%dd ago", int(ago.Hours()/24))
			}
			historyLine = fmt.Sprintf("\n  📜 Last scan  : %s — %d projects, %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(agoStr),
				last.ProjectCount,
				formatBytes(last.TotalSize))
		}

		scanBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 3).
			Width(60).
			Render(fmt.Sprintf(
				"%s  Scanning orbital debris...\n"+
					"\n"+
					"  📡 Path       : %s\n"+
					"  📂 Dirs checked: %s\n"+
					"  🔭 Projects    : %s\n"+
					"  💾 Debris found: %s\n"+
					"  🚀 Latest      : %s%s\n"+
					"\n"+
					"  %s",
				m.spinner.View(),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render(strings.Join(m.scanPaths, ", ")),
				lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", m.scanDirsChecked)),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5C07B")).Render(fmt.Sprintf("%d", m.scanProjectsFound)),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5C5C")).Render(formatBytes(m.scanTotalSize)),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(latestProj),
				historyLine,
				quoteStyled,
			))

		banner := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render(
			"  🛰️  K E S S L E R")

		return fmt.Sprintf("\n%s\n\n%s\n", banner, scanBox)

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
		tierNote := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#626262")).Render(" Safe mode targets caches/artifacts/temp junk. Deep mode adds builds.")
		if m.includeDeep {
			tierText = deepStyle.Render("DEEP CLEAN MODE (Includes Binaries & Builds)")
		}
		if m.showIgnored {
			tierText += " " + ignoredStyle.Render("[+Ignored]")
		}

		// Build Left Pane
		var leftContent strings.Builder
		leftContent.WriteString(titleStyle.Render("🚀 KESSLER: Clear the Debris") + "\n\n")

		// Show freed space banner from last clean
		if m.lastFreedSpace > 0 {
			freedBanner := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Render(
				fmt.Sprintf(" ✨ Freed %s in the last sweep!", formatBytes(m.lastFreedSpace)))
			leftContent.WriteString(freedBanner + "\n\n")
		}
		leftContent.WriteString(fmt.Sprintf(" %s\n\n", m.textInput.View()))
		leftContent.WriteString(fmt.Sprintf(" Found %d projects | Total Debris: %s | Sort: %s\n Mode: %s\n %s\n\n", len(filtered), formatBytes(totalSelectable), sortText, tierText, tierNote))

		if len(filtered) == 0 {
			leftContent.WriteString(" No project artifacts match the current filters. Orbit is clear!\n")
		} else {
			for i, proj := range filtered {
				// Dynamic pagination based on terminal height
				visibleRows := m.height - 12 // subtract header, footer, search, summary lines
				if visibleRows < 5 {
					visibleRows = 5
				}
				halfVisible := visibleRows / 2
				if i < m.cursor-halfVisible || i > m.cursor+halfVisible {
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

				// Sparkline: show relative size bar
				var maxSize int64
				for _, fp := range filtered {
					if fp.TotalSize > maxSize {
						maxSize = fp.TotalSize
					}
				}
				sparkWidth := 8
				var sparkFilled int
				if maxSize > 0 {
					sparkFilled = int(float64(proj.TotalSize) / float64(maxSize) * float64(sparkWidth))
				}
				if sparkFilled < 1 && proj.TotalSize > 0 {
					sparkFilled = 1
				}
				sparkBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(strings.Repeat("█", sparkFilled)) + strings.Repeat("░", sparkWidth-sparkFilled)

				// We want a clean columnar look. Format it with padding.
				baseNamePadded := fmt.Sprintf("%-26s", baseName)
				timeStrPadded := fmt.Sprintf("[%5s]", timeStr)
				sizeStr := formatBytes(proj.TotalSize)

				line := fmt.Sprintf("%s [%s] %s %s %s %s %s - %s", cursor, checked, getIcon(proj.Type), baseNamePadded, sparkBar, timeStrPadded, proj.Type, sizeStr)

				if _, ok := m.selected[proj.Path]; ok {
					leftContent.WriteString(selectedStyle.Render(line) + "\n")
				} else {
					leftContent.WriteString(itemStyle.Render(line) + "\n")
				}
			}
		}
		// Full project path tooltip
		if len(filtered) > 0 && m.cursor < len(filtered) {
			fullPath := filtered[m.cursor].Path
			leftContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).Render(fmt.Sprintf("\n 📍 %s", fullPath)) + "\n")
		}
		leftContent.WriteString(helpStyle.Render(
			"\n ↑/↓: nav        •  space: select          •  a: select all          •  e: select ecosystem  •  S: select stale" +
				"\n t: toggle mode  •  i: user-ignored        •  s: sort                •  /: search            •  p: preview" +
				"\n q: quit         •  enter: trash them      •  X: nuke them\n"))

		// Build Right Pane (Stats)
		var rightContent strings.Builder
		rightContent.WriteString(lipgloss.NewStyle().Bold(true).Render("📊 ORBITAL TELEMETRY") + "\n")
		rightContent.WriteString(lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#626262")).Render(" Projects identified by ecosystem 'triggers'.") + "\n\n")
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
					if !m.showIgnored && artifact.Tier == engine.TierIgnored {
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

			// Collect visible artifacts
			var visibleArtifacts []engine.Artifact
			for _, artifact := range activeProj.Artifacts {
				if !m.includeDeep && artifact.Tier == engine.TierDeep {
					continue
				}
				if !m.showIgnored && artifact.Tier == engine.TierIgnored {
					continue
				}
				visibleArtifacts = append(visibleArtifacts, artifact)
			}

			// Limit preview height based on terminal (leave room for header/footer/stats)
			maxPreviewItems := m.height - 30
			if maxPreviewItems < 3 {
				maxPreviewItems = 3
			}

			for i, artifact := range visibleArtifacts {
				if i >= maxPreviewItems {
					remaining := len(visibleArtifacts) - maxPreviewItems
					rightContent.WriteString(fmt.Sprintf(" └─ %s\n",
						lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).Render(
							fmt.Sprintf("+%d more items (press p to expand)", remaining))))
					break
				}

				tierColor := lipgloss.Color("#04B575") // Safe Green
				tierLabel := ""
				if artifact.Tier == engine.TierDeep {
					tierColor = lipgloss.Color("#E5C07B") // Deep Yellow
				} else if artifact.Tier == engine.TierDanger {
					tierColor = lipgloss.Color("#FF5C5C") // Danger Red
				} else if artifact.Tier == engine.TierIgnored {
					tierColor = lipgloss.Color("#E5C07B") // Ignored Yellow
					tierLabel = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Render("[user ignored]")
				}

				coloredPath := lipgloss.NewStyle().Foreground(tierColor).Render(filepath.Base(artifact.Path))
				rightContent.WriteString(fmt.Sprintf(" ├─ %-15s : %s%s\n", coloredPath, formatBytes(artifact.Size), tierLabel))
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

		mainView := paneStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftPaneContent, statsBox))

		// Preview Modal Overlay
		if m.showPreviewModal && len(filtered) > 0 && m.cursor < len(filtered) {
			activeProj := filtered[m.cursor]

			// Collect visible artifacts for the active project
			var visibleArtifacts []engine.Artifact
			for _, artifact := range activeProj.Artifacts {
				if !m.includeDeep && artifact.Tier == engine.TierDeep {
					continue
				}
				if !m.showIgnored && artifact.Tier == engine.TierIgnored {
					continue
				}
				visibleArtifacts = append(visibleArtifacts, artifact)
			}

			// Left Pane: Artifact List
			var listContent strings.Builder
			listContent.WriteString(lipgloss.NewStyle().Bold(true).Render("🎯 ARTIFACTS — "+filepath.Base(activeProj.Path)) + "\n\n")

			for i, artifact := range visibleArtifacts {
				cursor := "  "
				if m.previewCursor == i {
					cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("> ")
				}

				tierColor := lipgloss.Color("#04B575")
				tierLabel := ""
				if artifact.Tier == engine.TierDeep {
					tierColor = lipgloss.Color("#E5C07B")
					tierLabel = " [deep]"
				} else if artifact.Tier == engine.TierDanger {
					tierColor = lipgloss.Color("#FF5C5C")
					tierLabel = " [danger]"
				} else if artifact.Tier == engine.TierIgnored {
					tierColor = lipgloss.Color("#E5C07B")
					tierLabel = " [ignored]"
				}

				name := filepath.Base(artifact.Path)
				if len(name) > 30 {
					name = name[:27] + "..."
				}

				coloredName := lipgloss.NewStyle().Foreground(tierColor).Render(name)
				line := fmt.Sprintf("%s%-32s %8s%s", cursor, coloredName, formatBytes(artifact.Size), tierLabel)

				if m.previewCursor == i {
					listContent.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#2D2D2D")).Render(line) + "\n")
				} else {
					listContent.WriteString(line + "\n")
				}
			}

			// Better modal sizing
			modalWidth := int(float64(m.width) * 0.9)
			if modalWidth > 110 {
				modalWidth = 110
			}
			if modalWidth < 80 {
				modalWidth = 80
			}
			modalHeight := int(float64(m.height) * 0.8)
			if modalHeight > 30 {
				modalHeight = 30
			}
			if modalHeight < 15 {
				modalHeight = 15
			}

			// Right Pane: File Tree
			var treeContent strings.Builder
			if m.previewCursor < len(visibleArtifacts) {
				selectedArtifact := visibleArtifacts[m.previewCursor]
				treeContent.WriteString(lipgloss.NewStyle().Bold(true).Render("🌳 QUICK-LOOK: "+filepath.Base(selectedArtifact.Path)) + "\n\n")
				
				// Calculate max lines for tree based on modal height
				maxTreeLines := modalHeight - 10
				if maxTreeLines < 5 {
					maxTreeLines = 5
				}
				treeContent.WriteString(renderFileTree(selectedArtifact.Path, maxTreeLines))
			}

			leftPane := lipgloss.NewStyle().
				Width(modalWidth/2 - 3).
				Padding(0, 1).
				Render(listContent.String())

			rightPane := lipgloss.NewStyle().
				Width(modalWidth/2 - 3).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("#444444")).
				Render(treeContent.String())

			modalContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
			
			modalBox := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1, 2).
				Width(modalWidth).
				Height(modalHeight).
				Render(modalContent + "\n\n" +
					lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).Render("  ↑/↓: navigate   •   p: close preview"))

			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
		}

		// ===== TAB BAR =====
		tabNames := []string{"1: Projects", "2: Global", "3: History", "4: Launchpad"}
		var tabParts []string
		for i, name := range tabNames {
			if tabIndex(i) == m.activeTab {
				tabParts = append(tabParts, lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#7D56F4")).
					Background(lipgloss.Color("#2D2D2D")).
					Padding(0, 2).
					Render("[ "+name+" ]"))
			} else {
				tabParts = append(tabParts, lipgloss.NewStyle().
					Foreground(lipgloss.Color("#626262")).
					Padding(0, 2).
					Render("  "+name+"  "))
			}
		}
		tabBar := strings.Join(tabParts, "") + "\n\n"

		switch m.activeTab {
		case tabProjects:
			return tabBar + mainView

		case tabLaunchpad:
			filtered := m.getFilteredProjects()
			var lpView strings.Builder
			lpView.WriteString(lipgloss.NewStyle().Bold(true).Render("🚀 PROJECT LAUNCHPAD") + "\n")
			lpView.WriteString(lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#626262")).Render(" Since Kessler already scanned your orbit, use this to jump into your projects.") + "\n\n")
			lpView.WriteString(fmt.Sprintf(" %s\n\n", m.textInput.View()))

			if len(filtered) == 0 {
				lpView.WriteString("  No projects found matching your search. Try a different query!\n")
			} else {
				for i, proj := range filtered {
					visibleRows := m.height - 10
					halfVisible := visibleRows / 2
					if i < m.cursor-halfVisible || i > m.cursor+halfVisible {
						continue
					}

					cursor := "  "
					style := itemStyle
					if m.cursor == i {
						cursor = cursorStyle.Render("> ")
						style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
					}

					name := filepath.Base(proj.Path)
					if len(name) > 40 {
						name = name[:37] + "..."
					}
					
					lpView.WriteString(fmt.Sprintf("%s%s %-40s %s\n", cursor, getIcon(proj.Type), style.Render(name), lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(proj.Path)))
				}
			}

			lpView.WriteString(helpStyle.Render("\n ↑/↓: nav   •   o: open in VS Code   •   t: open terminal   •   1/2/3/4: switch tab   •   q: quit\n"))
			lpContent := lipgloss.NewStyle().Padding(1, 2).Render(lpView.String())
			return tabBar + lpContent

		case tabGlobal:
			var globalView strings.Builder
			globalView.WriteString(lipgloss.NewStyle().Bold(true).Render("🌍 GLOBAL CACHES") + "\n\n")

			// Caution banner
			cautionStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF5C5C")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF5C5C")).
				Padding(0, 2).
				Width(70)
			globalView.WriteString(cautionStyle.Render(
				"⚠️  CAUTION: Global caches are shared system resources. Deleting them affects ALL projects.") + "\n\n")

			if m.globalScanning {
				// Show scanning progress
				progress := 0.0
				if m.globalTotal > 0 {
					progress = float64(m.globalProcessed) / float64(m.globalTotal)
				}
				width := 40
				filled := int(progress * float64(width))
				if filled > width {
					filled = width
				}
				bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
				coloredBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(bar)

				globalView.WriteString(fmt.Sprintf(
					"  %s  Scanning global caches...\n\n"+
						"  %s [%d/%d]\n\n"+
						"  Current: %s\n"+
						"  Total Found: %s\n",
					m.spinner.View(),
					coloredBar,
					m.globalProcessed,
					m.globalTotal,
					lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(m.globalCurrentCache),
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5C5C")).Render(formatBytes(m.globalTotalSize)),
				))
			} else if len(m.globalCaches) == 0 {
				globalView.WriteString("  No global caches detected. Your system is clean! 🎉\n")
			} else {
				// Find max size for sparklines
				var maxGlobalSize int64
				var totalGlobalSize int64
				for _, c := range m.globalCaches {
					if c.Size > maxGlobalSize {
						maxGlobalSize = c.Size
					}
					totalGlobalSize += c.Size
				}

				globalView.WriteString(fmt.Sprintf("  Found %d caches | Total: %s\n\n",
					len(m.globalCaches),
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5C5C")).Render(formatBytes(totalGlobalSize))))

				for i, cache := range m.globalCaches {
					cursor := "  "
					if m.globalCursor == i {
						cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("> ")
					}

					checked := " "
					if _, ok := m.globalSelected[i]; ok {
						checked = "x"
					}

					// Sparkline
					sparkWidth := 8
					var sparkFilled int
					if maxGlobalSize > 0 {
						sparkFilled = int(float64(cache.Size) / float64(maxGlobalSize) * float64(sparkWidth))
					}
					if sparkFilled < 1 && cache.Size > 0 {
						sparkFilled = 1
					}
					sparkBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(strings.Repeat("█", sparkFilled)) + strings.Repeat("░", sparkWidth-sparkFilled)

					name := fmt.Sprintf("%-14s", cache.Name)
					sizeStr := fmt.Sprintf("%8s", formatBytes(cache.Size))

					line := fmt.Sprintf("%s[%s] %s %s %s  %s", cursor, checked, cache.Icon, name, sparkBar, sizeStr)

					if _, ok := m.globalSelected[i]; ok {
						globalView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render(line) + "\n")
					} else if m.globalCursor == i {
						globalView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Render(line) + "\n")
					} else {
						globalView.WriteString(line + "\n")
					}

					// Show description and CLI equivalent for selected item
					if m.globalCursor == i {
						globalView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).Render(
							fmt.Sprintf("     %s\n     Equivalent: %s", cache.Description, cache.CleanCommand)) + "\n")
					}
				}
			}

			// --- Environmental Doctor Section ---
			if len(m.toolchains) > 0 {
				globalView.WriteString("\n" + lipgloss.NewStyle().Bold(true).Render("🧪 ENVIRONMENTAL DOCTOR") + "\n")
				globalView.WriteString(lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#626262")).Render(" Kessler checks if any of your scanned projects actually use these versions.") + "\n\n")
				for _, t := range m.toolchains {
					if len(t.Unused) > 0 {
						globalView.WriteString(fmt.Sprintf("  %-10s: %d versions appear UNUSED\n", t.Name, len(t.Unused)))
						globalView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(
							fmt.Sprintf("    Unused: %s", strings.Join(t.Unused, ", "))) + "\n")
						
						// Cleanup advice
						advice := ""
						if t.Name == "Node.js" {
							advice = fmt.Sprintf("nvm uninstall %s", t.Unused[0])
						} else if t.Name == "Rust" {
							advice = fmt.Sprintf("rustup toolchain uninstall %s", t.Unused[0])
						} else if t.Name == "Python" {
							advice = fmt.Sprintf("pyenv uninstall %s", t.Unused[0])
						} else if t.Name == "Ruby" {
							advice = fmt.Sprintf("rbenv uninstall %s", t.Unused[0])
						} else if t.Name == "Java" {
							advice = fmt.Sprintf("sdk uninstall java %s", t.Unused[0])
						} else if t.Name == "asdf" {
							parts := strings.Fields(t.Unused[0])
							advice = fmt.Sprintf("asdf uninstall %s %s", parts[0], parts[1])
						} else if t.Name == "mise" {
							parts := strings.Fields(t.Unused[0])
							advice = fmt.Sprintf("mise uninstall %s@%s", parts[0], parts[1])
						}
						globalView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Italic(true).Render(
							fmt.Sprintf("    Tip: Run '%s' to save space", advice)) + "\n\n")
					}
				}
			}

			globalView.WriteString(helpStyle.Render("\n ↑/↓: nav   •   space: select   •   enter: clean selected   •   1/2/3/4: switch tab   •   q: quit\n"))

			globalContent := lipgloss.NewStyle().Padding(1, 2).Render(globalView.String())
			return tabBar + globalContent

		case tabHistory:
			var histView strings.Builder
			histView.WriteString(lipgloss.NewStyle().Bold(true).Render("📜 SCAN HISTORY") + "\n\n")

			history := engine.LoadHistory()
			if len(history.Entries) == 0 {
				histView.WriteString("  No scan history yet. Run your first sweep!\n")
			} else {
				// Header
				headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
				histView.WriteString(headerStyle.Render(fmt.Sprintf("  %-20s  %-30s  %8s  %10s  %10s", "Date", "Path", "Projects", "Debris", "Freed")) + "\n")
				histView.WriteString("  " + strings.Repeat("─", 85) + "\n")

				// Show entries in reverse order (newest first)
				for i := len(history.Entries) - 1; i >= 0; i-- {
					entry := history.Entries[i]
					dateStr := entry.Timestamp.Format("2006-01-02 15:04")
					pathStr := entry.ScanPath
					if len(pathStr) > 28 {
						pathStr = "..." + pathStr[len(pathStr)-25:]
					}

					freedStr := ""
					if entry.FreedSpace > 0 {
						freedStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render(formatBytes(entry.FreedSpace))
					} else {
						freedStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("—")
					}

					debrisStr := ""
					if entry.TotalSize > 0 {
						debrisStr = formatBytes(entry.TotalSize)
					} else {
						debrisStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("—")
					}

					histView.WriteString(fmt.Sprintf("  %-20s  %-30s  %8d  %10s  %10s\n",
						dateStr, pathStr, entry.ProjectCount, debrisStr, freedStr))
				}
			}

			histView.WriteString(helpStyle.Render("\n 1/2/3: switch tab   •   q: quit\n"))

			histContent := lipgloss.NewStyle().Padding(1, 2).Render(histView.String())
			return tabBar + histContent
		}

		return tabBar + mainView

	case stateConfirmNuke:
		if m.activeTab == tabGlobal {
			// Global cache clean confirmation
			var cacheNames []string
			var totalCleanSize int64
			for idx := range m.globalSelected {
				if idx < len(m.globalCaches) {
					cacheNames = append(cacheNames, m.globalCaches[idx].Name)
					totalCleanSize += m.globalCaches[idx].Size
				}
			}

			var cmdList string
			for idx := range m.globalSelected {
				if idx < len(m.globalCaches) {
					cmdList += fmt.Sprintf("  • %s (%s)\n    → %s\n", m.globalCaches[idx].Name, formatBytes(m.globalCaches[idx].Size), m.globalCaches[idx].CleanCommand)
				}
			}

			confirmBox := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#E5C07B")).
				Padding(1, 3).
				Width(60).
				Render(fmt.Sprintf(
					"%s\n\n"+
						"  You are about to clean %d global cache(s):\n\n%s\n"+
						"  Total: %s\n\n"+
						"  ⚠️  This affects ALL projects using these caches.\n"+
						"  Caches will regenerate on next install/build.\n\n"+
						"  %s",
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5C07B")).Render("⚠️  CONFIRM GLOBAL CLEAN"),
					len(m.globalSelected),
					cmdList,
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5C07B")).Render(formatBytes(totalCleanSize)),
					lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press y to confirm, n to cancel"),
				))

			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, confirmBox)
		}

		// Project nuke confirmation (existing)
		var nukeSize int64
		var nukeCount int
		for _, proj := range m.projects {
			if _, ok := m.selected[proj.Path]; ok {
				for _, a := range proj.Artifacts {
					if !m.includeDeep && a.Tier == engine.TierDeep {
						continue
					}
					if !m.showIgnored && a.Tier == engine.TierIgnored {
						continue
					}
					nukeSize += a.Size
					nukeCount++
				}
			}
		}

		nukeBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF5C5C")).
			Padding(1, 3).
			Width(55).
			Render(fmt.Sprintf(
				"%s\n\n"+
					"  You are about to PERMANENTLY DELETE:\n\n"+
					"  • %d artifacts from %d projects\n"+
					"  • Total size: %s\n\n"+
					"  ⚠️  This CANNOT be undone. Files will NOT go to Trash.\n\n"+
					"  %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5C5C")).Render("☢️  CONFIRM PERMANENT NUKE"),
				nukeCount,
				len(m.selected),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5C5C")).Render(formatBytes(nukeSize)),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press y to confirm, n to cancel"),
			))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, nukeBox)

	case stateConfirmActive:
		var activeList strings.Builder
		for path, procs := range m.activeProjects {
			activeList.WriteString(fmt.Sprintf("  📁 %s\n", filepath.Base(path)))
			for _, p := range procs {
				activeList.WriteString(fmt.Sprintf("     └─ %s (PID: %d)\n", p.Name, p.PID))
			}
		}

		activeBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#E5C07B")).
			Padding(1, 3).
			Width(60).
			Render(fmt.Sprintf(
				"%s\n\n"+
					"  The following projects have ACTIVE processes running:\n\n%s\n"+
					"  ⚠️  Cleaning active projects may cause build errors or\n"+
					"  crashes in your dev servers.\n\n"+
					"  Do you want to proceed anyway?\n\n"+
					"  %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5C07B")).Render("⚠️  ACTIVE PROJECTS DETECTED"),
				activeList.String(),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press y to proceed, n to cancel"),
			))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, activeBox)

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

func renderFileTree(root string, maxTotalLines int) string {
	var sb strings.Builder
	depthLimit := 3
	maxItemsPerDir := 15
	lineCount := 0

	var walk func(path string, depth int, prefix string)
	walk = func(path string, depth int, prefix string) {
		if depth > depthLimit || lineCount >= maxTotalLines-1 {
			return
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		// Sort entries: directories first, then files
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() && !entries[j].IsDir() {
				return true
			}
			if !entries[i].IsDir() && entries[j].IsDir() {
				return false
			}
			return entries[i].Name() < entries[j].Name()
		})

		count := 0
		for i, entry := range entries {
			if lineCount >= maxTotalLines-1 {
				if count < len(entries) {
					sb.WriteString(prefix + "└── ... (more items)\n")
					lineCount++
				}
				return
			}

			if count >= maxItemsPerDir {
				sb.WriteString(prefix + "└── ... (" + fmt.Sprintf("%d", len(entries)-maxItemsPerDir) + " more items)\n")
				lineCount++
				break
			}

			isLast := i == len(entries)-1 || count == maxItemsPerDir-1
			connector := "├── "
			newPrefix := prefix + "│   "
			if isLast {
				connector = "└── "
				newPrefix = prefix + "    "
			}

			name := entry.Name()
			if entry.IsDir() {
				name = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render(name + "/")
			}

			sb.WriteString(prefix + connector + name + "\n")
			lineCount++

			if entry.IsDir() && depth < depthLimit {
				walk(filepath.Join(path, entry.Name()), depth+1, newPrefix)
			}
			count++
		}
	}

	walk(root, 1, "")
	return sb.String()
}
