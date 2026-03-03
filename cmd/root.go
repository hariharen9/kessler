package cmd

import (
	_ "embed"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hariharen/kessler/internal/tui"
	"github.com/spf13/cobra"
)

// RulesData is set by main.go from the embedded rules.yaml.
var RulesData []byte

var deep bool

var rootCmd = &cobra.Command{
	Use:   "kessler [path]",
	Short: "🛰️ Kessler — Clear the orbital debris from your filesystem",
	Long: `Kessler is an intelligent, blazingly fast CLI tool that finds and safely
sweeps away runtime artifacts and build caches (node_modules, __pycache__,
target/, etc.) without ever touching your source code.

Run without a subcommand to launch the interactive TUI dashboard.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scanPath := "."
		if len(args) > 0 {
			scanPath = args[0]
		}

		p := tea.NewProgram(tui.InitialModel(scanPath, deep, RulesData), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&deep, "deep", "d", false, "Include deep-tier artifacts (builds, binaries)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
