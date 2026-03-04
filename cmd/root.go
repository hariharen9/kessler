package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hariharen9/kessler/internal/tui"
	"github.com/spf13/cobra"
)

// RulesData is set by main.go from the embedded rules.yaml.
var RulesData []byte

// UserRulesData is loaded from ~/.config/kessler/rules.yaml if it exists.
var UserRulesData []byte

var deep bool

var rootCmd = &cobra.Command{
	Use:   "kessler [paths...]",
	Short: "🛰️ Kessler — Clear the orbital debris from your filesystem",
	Long: `Kessler is an intelligent, blazingly fast CLI tool that finds and safely
sweeps away runtime artifacts and build caches (node_modules, __pycache__,
target/, etc.) without ever touching your source code.

Run without a subcommand to launch the interactive TUI dashboard.
Custom rules can be added at ~/.config/kessler/rules.yaml`,
	Args: cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		loadUserRules()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		scanPaths := []string{"."}
		if len(args) > 0 {
			scanPaths = args
		}

		// For the TUI, we pass both base and user rules.
		// The TUI's scanner will need to use NewScannerMerged.
		// For now, we pre-merge and pass the base rules (TUI uses NewScanner internally).
		// We update the TUI to accept user rules separately.
		p := tea.NewProgram(tui.InitialModel(scanPaths, deep, RulesData, UserRulesData), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&deep, "deep", "d", false, "Include deep-tier artifacts (builds, binaries)")
}

func loadUserRules() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	configPath := filepath.Join(home, ".config", "kessler", "rules.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return // File doesn't exist or unreadable — that's fine
	}

	UserRulesData = data
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
