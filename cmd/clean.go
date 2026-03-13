package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hariharen9/kessler/engine"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	cleanForce          bool
	cleanConfirm        bool
	cleanPermanent      bool
	cleanDryRun         bool
	cleanMinSize        string
	cleanOlderThan      string
	cleanIncludeIgnored bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [paths...]",
	Short: "Non-interactively clean project artifacts",
	Long: `Scan and clean build artifacts without the TUI.

Kessler defaults to a dry-run preview. To execute the cleaning:
- In an interactive terminal, it will ask for a [y/N] confirmation.
- In scripts (non-interactive), you must use the --force or --confirm flag.

Examples:
  kessler clean ~/Projects                         # Shows preview + asks y/n
  kessler clean ~/Projects --confirm                # Clean without asking (good for scripts)
  kessler clean ~/Projects --deep --force           # Deep clean (builds/binaries) without asking
  kessler clean ~/Projects --older-than 30d         # Shows preview only if not a terminal`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompts (force execute)")
	cleanCmd.Flags().BoolVarP(&cleanConfirm, "confirm", "c", false, "Confirm execution (bypass dry-run)")
	cleanCmd.Flags().BoolVarP(&cleanPermanent, "permanent", "p", false, "Permanently delete instead of moving to trash")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without deleting")
	cleanCmd.Flags().StringVarP(&cleanMinSize, "min-size", "m", "", "Only clean projects above this size (e.g. 100MB, 1GB)")
	cleanCmd.Flags().StringVarP(&cleanOlderThan, "older-than", "o", "", "Only clean projects not touched in N days (e.g. 30d)")
	cleanCmd.Flags().BoolVar(&cleanIncludeIgnored, "include-ignored", false, "Include gitignored artifacts in cleaning")

	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	scanPaths := []string{"."}
	if len(args) > 0 {
		scanPaths = args
	}

	scanner, err := engine.NewScannerMerged(RulesData, CommunityRulesData, UserRulesData, excludes)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	fmt.Println("\n  🛰️  Scanning for debris...")
	projects, err := scanner.Scan(scanPaths)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Build filter options
	opts := engine.FilterOptions{IncludeDeep: deep, ShowIgnored: cleanIncludeIgnored}

	if cleanMinSize != "" {
		size, err := parseSize(cleanMinSize)
		if err != nil {
			return fmt.Errorf("invalid --min-size value %q: %w", cleanMinSize, err)
		}
		opts.MinSize = size
	}

	if cleanOlderThan != "" {
		dur, err := parseDuration(cleanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value %q: %w", cleanOlderThan, err)
		}
		opts.OlderThan = dur
	}

	filtered := engine.FilterProjects(projects, opts)

	// Sort by size (largest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TotalSize > filtered[j].TotalSize
	})

	if len(filtered) == 0 {
		fmt.Println("  No project artifacts match the current filters. Orbit is clear! 🛰️")
		fmt.Println()
		return nil
	}

	// Collect all artifacts to clean
	var artifacts []engine.Artifact
	var totalSize int64
	for _, p := range filtered {
		for _, a := range p.Artifacts {
			artifacts = append(artifacts, a)
			totalSize += a.Size
		}
	}

	// Show summary and preview
	modeStr := "Safe"
	if deep {
		modeStr = "Deep"
	}
	fmt.Printf("  Found %d projects with %s of debris (%s mode)\n\n", len(filtered), formatBytes(totalSize), modeStr)

	// Always show what will be cleaned
	for _, p := range filtered {
		fmt.Printf("  📁 %s (%s) — %s\n", filepath.Base(p.Path), p.Type, formatBytes(p.TotalSize))
		for _, a := range p.Artifacts {
			fmt.Printf("     └─ %s [%s] %s\n", filepath.Base(a.Path), a.Tier, formatBytes(a.Size))
		}
	}
	fmt.Printf("\n  Total: %d artifacts, %s\n", len(artifacts), formatBytes(totalSize))

	// If explicit dry-run, stop here
	if cleanDryRun {
		fmt.Println("\n  ── DRY RUN (no files were deleted) ──")
		fmt.Println()
		return nil
	}

	// Logic for default dry-run vs execution
	shouldExecute := cleanForce || cleanConfirm

	// If not forced/confirmed, ask if interactive, or dry-run if scripted
	if !shouldExecute {
		// Check if we are in an interactive terminal
		isTerminal := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())

		if !isTerminal {
			fmt.Println("\n  ── DRY RUN (No --force or --confirm flag provided in non-interactive mode) ──")
			fmt.Println("  Use --confirm to execute this cleaning in scripts.")
			fmt.Println()
			return nil
		}

		// Ask for confirmation in interactive mode
		label := "Proceed with cleaning?"
		if deep {
			label = "⚠️  Deep clean includes builds & binaries. Proceed?"
		}
		if cleanIncludeIgnored {
			label = "⚠️  Includes gitignored files. Proceed?"
			if deep {
				label = "⚠️  Deep clean + gitignored files. Proceed?"
			}
		}
		fmt.Printf("\n  %s [y/N] ", label)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("  Aborted. No files were deleted.")
			fmt.Println()
			return nil
		}
		shouldExecute = true
	}

	// Clean
	method := "moving to Trash"
	if cleanPermanent {
		method = "permanently deleting"
	}
	fmt.Printf("\n  🧹 Cleaning %d artifacts (%s)...\n\n", len(artifacts), method)

	// Check for active processes across all projects being cleaned
	var activeProcsFound bool
	for _, p := range filtered {
		procs := engine.GetActiveProcessesInPath(p.Path)
		if len(procs) > 0 {
			activeProcsFound = true
			fmt.Printf("  ⚠️  ACTIVE PROCESSES in %s:\n", filepath.Base(p.Path))
			for _, proc := range procs {
				fmt.Printf("     └─ %s (PID: %d)\n", proc.Name, proc.PID)
			}
		}
	}

	if activeProcsFound && !cleanForce {
		isTerminal := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
		if isTerminal {
			fmt.Printf("\n  ⚠️  Proceed with cleaning active projects? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				fmt.Println("  Aborted. No files were deleted.")
				fmt.Println()
				return nil
			}
		} else {
			fmt.Println("\n  ❌ Error: Active processes detected. Use --force to clean anyway in non-interactive mode.")
			fmt.Println()
			return nil
		}
	}

	var freedSpace int64
	var failedCount int
	var failedPaths []string

	for _, a := range artifacts {
		var delErr error
		if cleanPermanent {
			delErr = os.RemoveAll(a.Path)
		} else {
			absPath, pathErr := filepath.Abs(a.Path)
			if pathErr == nil {
				delErr = engine.MoveToTrash(absPath)
			} else {
				delErr = pathErr
			}
		}

		if delErr != nil {
			failedCount++
			failedPaths = append(failedPaths, a.Path)
			fmt.Printf("  ✗ %s — %v\n", filepath.Base(a.Path), delErr)
		} else {
			freedSpace += a.Size
			fmt.Printf("  ✓ %s (%s)\n", filepath.Base(a.Path), formatBytes(a.Size))
		}
	}

	fmt.Printf("\n  ✨ Done! Freed %s", formatBytes(freedSpace))
	if failedCount > 0 {
		fmt.Printf(" (%d failed)", failedCount)
	}
	fmt.Println()
	fmt.Println()

	// Offer fallback for failed trash items
	if failedCount > 0 && !cleanPermanent {
		if cleanForce {
			// With --force, auto-fallback to permanent deletion
			fmt.Println("  Retrying failed items with permanent deletion...")
			for _, path := range failedPaths {
				if err := os.RemoveAll(path); err != nil {
					fmt.Printf("  ✗ %s — %v\n", filepath.Base(path), err)
				} else {
					fmt.Printf("  ✓ %s (permanently deleted)\n", filepath.Base(path))
				}
			}
			fmt.Println()
		} else {
			fmt.Printf("  ⚠️  %d items failed to trash. Use --permanent to force delete.\n\n", failedCount)
		}
	}

	// Save to history
	var totalBefore int64
	for _, p := range filtered {
		totalBefore += p.TotalSize
	}
	engine.SaveEntry(engine.ScanHistoryEntry{
		Timestamp:    time.Now(),
		ScanPath:     strings.Join(scanPaths, ", "),
		ProjectCount: len(filtered),
		TotalSize:    totalBefore,
		FreedSpace:   freedSpace,
	})

	return nil
}
