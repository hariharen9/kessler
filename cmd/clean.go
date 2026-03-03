package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hariharen/kessler/engine"
	"github.com/laurent22/go-trash"
	"github.com/spf13/cobra"
)

var (
	cleanForce          bool
	cleanPermanent      bool
	cleanDryRun         bool
	cleanMinSize        string
	cleanOlderThan      string
	cleanIncludeIgnored bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [path]",
	Short: "Non-interactively clean project artifacts",
	Long: `Scan and clean build artifacts without the TUI.

In safe mode (default), cleaning proceeds without asking.
In deep mode (--deep), a Y/n confirmation is shown unless --force is used.

Examples:
  kessler clean ~/Projects
  kessler clean ~/Projects --deep --force
  kessler clean ~/Projects --older-than 30d --dry-run
  kessler clean ~/Projects --permanent`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompts")
	cleanCmd.Flags().BoolVarP(&cleanPermanent, "permanent", "p", false, "Permanently delete instead of moving to trash")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without deleting")
	cleanCmd.Flags().StringVarP(&cleanMinSize, "min-size", "m", "", "Only clean projects above this size (e.g. 100MB, 1GB)")
	cleanCmd.Flags().StringVarP(&cleanOlderThan, "older-than", "o", "", "Only clean projects not touched in N days (e.g. 30d)")
	cleanCmd.Flags().BoolVar(&cleanIncludeIgnored, "include-ignored", false, "Include gitignored artifacts in cleaning")

	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	scanPath := "."
	if len(args) > 0 {
		scanPath = args[0]
	}

	scanner, err := engine.NewScannerMerged(RulesData, UserRulesData)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	fmt.Println("\n  🛰️  Scanning for debris...")
	projects, err := scanner.Scan(scanPath)
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
		fmt.Println("  No project artifacts match the current filters. Orbit is clear! 🛰️\n")
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

	// Dry run: stop here
	if cleanDryRun {
		fmt.Println("\n  ── DRY RUN (no files were deleted) ──\n")
		return nil
	}

	// Ask for confirmation unless --force
	if !cleanForce {
		label := "Proceed?"
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
			fmt.Println("  Aborted.")
			return nil
		}
	}

	// Clean
	method := "moving to Trash"
	if cleanPermanent {
		method = "permanently deleting"
	}
	fmt.Printf("\n  🧹 Cleaning %d artifacts (%s)...\n\n", len(artifacts), method)

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
				_, delErr = trash.MoveToTrash(absPath)
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
	fmt.Println("\n")

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

	return nil
}
