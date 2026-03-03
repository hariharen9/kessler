package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hariharen/kessler/engine"
	"github.com/spf13/cobra"
)

var (
	scanJSON      bool
	scanSort      string
	scanMinSize   string
	scanOlderThan string
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan for project artifacts and report (no deletion)",
	Long: `Scan a directory tree for build artifacts, caches, and other debris.
Outputs a table by default, or JSON with --json for scripting.

Examples:
  kessler scan ~/Projects
  kessler scan ~/Projects --json | jq '.[] | select(.totalSize > 100000000)'
  kessler scan ~/Projects --deep --older-than 30d --min-size 100MB`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScan,
}

func init() {
	scanCmd.Flags().BoolVarP(&scanJSON, "json", "j", false, "Output as JSON")
	scanCmd.Flags().StringVarP(&scanSort, "sort", "s", "size", "Sort by: size, name")
	scanCmd.Flags().StringVarP(&scanMinSize, "min-size", "m", "", "Only show projects above this size (e.g. 100MB, 1GB)")
	scanCmd.Flags().StringVarP(&scanOlderThan, "older-than", "o", "", "Only show projects not touched in N days (e.g. 30d, 7d)")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	scanPath := "."
	if len(args) > 0 {
		scanPath = args[0]
	}

	scanner, err := engine.NewScannerMerged(RulesData, UserRulesData)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	projects, err := scanner.Scan(scanPath)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Build filter options
	opts := engine.FilterOptions{IncludeDeep: deep}

	if scanMinSize != "" {
		size, err := parseSize(scanMinSize)
		if err != nil {
			return fmt.Errorf("invalid --min-size value %q: %w", scanMinSize, err)
		}
		opts.MinSize = size
	}

	if scanOlderThan != "" {
		dur, err := parseDuration(scanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value %q: %w", scanOlderThan, err)
		}
		opts.OlderThan = dur
	}

	filtered := engine.FilterProjects(projects, opts)

	// Sort
	switch strings.ToLower(scanSort) {
	case "name":
		sort.Slice(filtered, func(i, j int) bool {
			return filepath.Base(filtered[i].Path) < filepath.Base(filtered[j].Path)
		})
	default: // size
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TotalSize > filtered[j].TotalSize
		})
	}

	if scanJSON {
		return outputJSON(filtered)
	}
	outputTable(filtered)
	return nil
}

// --- Output Formatters ---

type jsonArtifact struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Tier string `json:"tier"`
}

type jsonProject struct {
	Path           string         `json:"path"`
	Type           string         `json:"type"`
	TotalSize      int64          `json:"totalSize"`
	TotalSizeHuman string         `json:"totalSizeHuman"`
	LastModified   string         `json:"lastModified"`
	Artifacts      []jsonArtifact `json:"artifacts"`
}

func outputJSON(projects []engine.Project) error {
	var out []jsonProject
	for _, p := range projects {
		jp := jsonProject{
			Path:           p.Path,
			Type:           p.Type,
			TotalSize:      p.TotalSize,
			TotalSizeHuman: formatBytes(p.TotalSize),
			LastModified:   p.LastModTime.Format(time.RFC3339),
		}
		for _, a := range p.Artifacts {
			jp.Artifacts = append(jp.Artifacts, jsonArtifact{
				Path: filepath.Base(a.Path),
				Size: a.Size,
				Tier: string(a.Tier),
			})
		}
		out = append(out, jp)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputTable(projects []engine.Project) {
	if len(projects) == 0 {
		fmt.Println("No project artifacts found. Orbit is clear! 🛰️")
		return
	}

	// Header
	fmt.Printf("\n  %-30s %-25s %10s   %s\n", "PROJECT", "TYPE", "SIZE", "LAST TOUCHED")
	fmt.Println("  " + strings.Repeat("─", 85))

	var totalSize int64
	for _, p := range projects {
		name := filepath.Base(p.Path)
		if len(name) > 28 {
			name = name[:25] + "..."
		}

		age := "unknown"
		if !p.LastModTime.IsZero() {
			days := int(time.Since(p.LastModTime).Hours() / 24)
			if days == 0 {
				age = "today"
			} else {
				age = fmt.Sprintf("%dd ago", days)
			}
		}

		fmt.Printf("  %-30s %-25s %10s   %s\n", name, p.Type, formatBytes(p.TotalSize), age)
		totalSize += p.TotalSize
	}

	fmt.Println("  " + strings.Repeat("─", 85))
	fmt.Printf("  Total: %d projects, %s of debris\n\n", len(projects), formatBytes(totalSize))
}

// --- Helpers ---

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

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			var val float64
			if _, err := fmt.Sscanf(numStr, "%f", &val); err != nil {
				return 0, fmt.Errorf("cannot parse number: %s", numStr)
			}
			return int64(val * float64(mult)), nil
		}
	}
	// Try as plain number (bytes)
	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, fmt.Errorf("cannot parse size: %s", s)
	}
	return val, nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, fmt.Errorf("cannot parse days: %s", numStr)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	// Fallback to Go's time.ParseDuration for hours, minutes, etc.
	return time.ParseDuration(s)
}
