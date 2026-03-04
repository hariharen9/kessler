package engine

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"strings"

	"gopkg.in/yaml.v3"
)

type Scanner struct {
	Config Config
}

func NewScanner(rulesData []byte) (*Scanner, error) {
	var config Config
	if err := yaml.Unmarshal(rulesData, &config); err != nil {
		return nil, err
	}

	return &Scanner{Config: config}, nil
}

// NewScannerMerged creates a Scanner from base rules, optionally merging user rules on top.
// If userRulesData is nil or empty, it behaves identically to NewScanner.
func NewScannerMerged(baseData []byte, userRulesData []byte) (*Scanner, error) {
	var baseConfig Config
	if err := yaml.Unmarshal(baseData, &baseConfig); err != nil {
		return nil, err
	}

	if len(userRulesData) > 0 {
		var userConfig Config
		if err := yaml.Unmarshal(userRulesData, &userConfig); err != nil {
			return nil, fmt.Errorf("user rules: %w", err)
		}
		baseConfig = MergeConfigs(baseConfig, userConfig)
	}

	return &Scanner{Config: baseConfig}, nil
}

func (s *Scanner) Scan(roots []string) ([]Project, error) {
	var projects []Project
	var lastErr error

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			for _, rule := range s.Config.Rules {
				for _, target := range rule.Targets {
					if strings.Contains(target.Path, "*") || strings.Contains(target.Path, "?") {
						if matched, _ := filepath.Match(target.Path, d.Name()); matched {
							return filepath.SkipDir
						}
					} else if d.Name() == target.Path {
						return filepath.SkipDir
					}
				}
			}

			matchedRule := s.matchRule(path)
			if matchedRule != nil {
				project := Project{
					Path: path,
					Type: matchedRule.Name,
				}

				for _, target := range matchedRule.Targets {
					// --- DANGER ZONE SAFETY NET ---
					// If the target exactly matches a known danger zone, skip it entirely
					isDangerous := false
					for _, dangerItem := range s.Config.DangerZone {
						if target.Path == dangerItem {
							isDangerous = true
							break
						}
					}
					if isDangerous {
						continue
					}

					var targetPaths []string
					if strings.Contains(target.Path, "*") || strings.Contains(target.Path, "?") {
						matches, _ := filepath.Glob(filepath.Join(path, target.Path))
						targetPaths = append(targetPaths, matches...)
					} else {
						targetPaths = append(targetPaths, filepath.Join(path, target.Path))
					}

					for _, targetPath := range targetPaths {
						if info, err := os.Stat(targetPath); err == nil {
							// --- THE GIT SAFETY NET ---
							// If the target directory contains files tracked by Git, DO NOT touch it.
							if s.isTrackedByGit(path, targetPath) {
								continue
							}

							size, modTime := s.calculateSizeAndModTime(targetPath, info)
							if size > 0 {
								project.Artifacts = append(project.Artifacts, Artifact{
									Path:    targetPath,
									Size:    size,
									Tier:    target.Tier,
									ModTime: modTime,
								})
								project.TotalSize += size

								// Update project's last modified time
								if project.LastModTime.IsZero() || modTime.After(project.LastModTime) {
									project.LastModTime = modTime
								}
							}
						}
					}
				}

				// --- GITIGNORE SCANNING ---
				// Find ignored items not already captured by rules
				ignoredArtifacts := s.scanGitIgnored(path, project.Artifacts)
				project.Artifacts = append(project.Artifacts, ignoredArtifacts...)
				for _, a := range ignoredArtifacts {
					project.TotalSize += a.Size
					if project.LastModTime.IsZero() || a.ModTime.After(project.LastModTime) {
						project.LastModTime = a.ModTime
					}
				}

				if len(project.Artifacts) > 0 {
					projects = append(projects, project)
				}
			}

			return nil
		})
		if err != nil {
			lastErr = err
		}
	}

	return projects, lastErr
}

// ScanProgress reports live scanning progress.
type ScanProgress struct {
	DirsChecked   int
	ProjectsFound int
	CurrentDir    string
	LatestProject string
	TotalSize     int64
}

// ScanWithProgress is like Scan but sends progress updates through a channel.
func (s *Scanner) ScanWithProgress(roots []string, progress chan<- ScanProgress) ([]Project, error) {
	var projects []Project
	var dirsChecked int
	var lastErr error

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			dirsChecked++

			for _, rule := range s.Config.Rules {
				for _, target := range rule.Targets {
					if strings.Contains(target.Path, "*") || strings.Contains(target.Path, "?") {
						if matched, _ := filepath.Match(target.Path, d.Name()); matched {
							return filepath.SkipDir
						}
					} else if d.Name() == target.Path {
						return filepath.SkipDir
					}
				}
			}

			// Send progress update every 50 directories to avoid channel overhead
			if dirsChecked%50 == 0 {
				var totalSize int64
				for _, p := range projects {
					totalSize += p.TotalSize
				}
				progress <- ScanProgress{
					DirsChecked:   dirsChecked,
					ProjectsFound: len(projects),
					CurrentDir:    path,
					TotalSize:     totalSize,
				}
			}

			matchedRule := s.matchRule(path)
			if matchedRule != nil {
				project := Project{
					Path: path,
					Type: matchedRule.Name,
				}

				for _, target := range matchedRule.Targets {
					isDangerous := false
					for _, dangerItem := range s.Config.DangerZone {
						if target.Path == dangerItem {
							isDangerous = true
							break
						}
					}
					if isDangerous {
						continue
					}

					var targetPaths []string
					if strings.Contains(target.Path, "*") || strings.Contains(target.Path, "?") {
						matches, _ := filepath.Glob(filepath.Join(path, target.Path))
						targetPaths = append(targetPaths, matches...)
					} else {
						targetPaths = append(targetPaths, filepath.Join(path, target.Path))
					}

					for _, targetPath := range targetPaths {
						if info, err := os.Stat(targetPath); err == nil {
							if s.isTrackedByGit(path, targetPath) {
								continue
							}

							size, modTime := s.calculateSizeAndModTime(targetPath, info)
							if size > 0 {
								project.Artifacts = append(project.Artifacts, Artifact{
									Path:    targetPath,
									Size:    size,
									Tier:    target.Tier,
									ModTime: modTime,
								})
								project.TotalSize += size

								if project.LastModTime.IsZero() || modTime.After(project.LastModTime) {
									project.LastModTime = modTime
								}
							}
						}
					}
				}

				ignoredArtifacts := s.scanGitIgnored(path, project.Artifacts)
				project.Artifacts = append(project.Artifacts, ignoredArtifacts...)
				for _, a := range ignoredArtifacts {
					project.TotalSize += a.Size
					if project.LastModTime.IsZero() || a.ModTime.After(project.LastModTime) {
						project.LastModTime = a.ModTime
					}
				}

				if len(project.Artifacts) > 0 {
					projects = append(projects, project)

					// Send immediate update when a project is found
					var totalSize int64
					for _, p := range projects {
						totalSize += p.TotalSize
					}
					progress <- ScanProgress{
						DirsChecked:   dirsChecked,
						ProjectsFound: len(projects),
						CurrentDir:    path,
						LatestProject: filepath.Base(path),
						TotalSize:     totalSize,
					}
				}
			}

			return nil
		})
		if err != nil {
			lastErr = err
		}
	}

	return projects, lastErr
}

func (s *Scanner) matchRule(dir string) *Rule {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	entryMap := make(map[string]bool)
	for _, e := range entries {
		entryMap[e.Name()] = true
	}

	for i := range s.Config.Rules {
		rule := &s.Config.Rules[i]
		if len(rule.Triggers) == 0 {
			continue
		}
		for _, trigger := range rule.Triggers {
			if strings.Contains(trigger, "*") || strings.Contains(trigger, "?") {
				for e := range entryMap {
					if matched, _ := filepath.Match(trigger, e); matched {
						return rule
					}
				}
			} else if entryMap[trigger] {
				return rule
			}
		}
	}
	return nil
}

func (s *Scanner) calculateSizeAndModTime(path string, info fs.FileInfo) (int64, time.Time) {
	var size int64
	var latestModTime time.Time

	if !info.IsDir() {
		return info.Size(), info.ModTime()
	}

	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil {
			if !info.IsDir() {
				size += info.Size()
			}
			if modTime := info.ModTime(); modTime.After(latestModTime) {
				latestModTime = modTime
			}
		}
		return nil
	})

	// If empty directory, just return the directory's modtime
	if latestModTime.IsZero() {
		latestModTime = info.ModTime()
	}

	return size, latestModTime
}

// isTrackedByGit checks if a specific artifact path contains files tracked by Git.
func (s *Scanner) isTrackedByGit(projectRoot, artifactPath string) bool {
	// git ls-files <path> returns a list of tracked files inside that path.
	cmd := exec.Command("git", "ls-files", artifactPath)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		// If git errors out (e.g. not a git repo), we assume it's untracked.
		return false
	}
	// If the output contains text, it means Git is tracking something inside this folder!
	return len(bytes.TrimSpace(out)) > 0
}

// scanGitIgnored finds gitignored directories not already covered by rule-based artifacts.
// SAFETY: Only considers ignored DIRECTORIES, not individual files.
// Individual ignored files (like .env, lockfiles) are never surfaced.
func (s *Scanner) scanGitIgnored(projectRoot string, existingArtifacts []Artifact) []Artifact {
	// Get list of ignored directories
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "--directory")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil // Not a git repo or git error
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil
	}

	// Build set of existing artifact base names for fast lookup
	existingSet := make(map[string]bool)
	for _, a := range existingArtifacts {
		existingSet[filepath.Base(a.Path)] = true
	}

	// Build set of all known rule target paths to filter out
	ruleTargetSet := make(map[string]bool)
	for _, rule := range s.Config.Rules {
		for _, target := range rule.Targets {
			ruleTargetSet[target.Path] = true
		}
		// Also exclude trigger files (package.json, Cargo.toml, etc.)
		for _, trigger := range rule.Triggers {
			ruleTargetSet[trigger] = true
		}
	}

	// Also add danger zone items
	for _, d := range s.Config.DangerZone {
		ruleTargetSet[d] = true
	}

	var artifacts []Artifact
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// SAFETY: Only consider directory entries (they have a trailing slash).
		// Individual files like .env, lockfiles, configs must NEVER be surfaced.
		if !strings.HasSuffix(line, "/") {
			continue
		}

		// Remove trailing slash from directory entries
		cleanName := strings.TrimSuffix(line, "/")

		// Skip if already covered by rules or existing artifacts
		if existingSet[cleanName] || ruleTargetSet[cleanName] {
			continue
		}

		// Also check against globs in ruleTargetSet
		isMatchedGlob := false
		for k := range ruleTargetSet {
			if strings.Contains(k, "*") || strings.Contains(k, "?") {
				if matched, _ := filepath.Match(k, cleanName); matched {
					isMatchedGlob = true
					break
				}
			}
		}
		if isMatchedGlob {
			continue
		}

		// Skip danger zone items
		isDangerous := false
		for _, d := range s.Config.DangerZone {
			if cleanName == d {
				isDangerous = true
				break
			}
		}
		if isDangerous {
			continue
		}

		targetPath := filepath.Join(projectRoot, cleanName)
		info, err := os.Stat(targetPath)
		if err != nil {
			continue
		}

		size, modTime := s.calculateSizeAndModTime(targetPath, info)
		if size > 0 {
			artifacts = append(artifacts, Artifact{
				Path:    targetPath,
				Size:    size,
				Tier:    TierIgnored,
				ModTime: modTime,
			})
		}
	}

	return artifacts
}

// FilterOptions controls post-scan filtering for non-interactive mode.
type FilterOptions struct {
	IncludeDeep bool
	ShowIgnored bool
	MinSize     int64         // bytes, 0 = no filter
	OlderThan   time.Duration // 0 = no filter
}

// FilterProjects applies tier, min-size, and older-than filters on scanned projects.
// It recalculates TotalSize per project based on the active tier.
func FilterProjects(projects []Project, opts FilterOptions) []Project {
	var filtered []Project

	for _, p := range projects {
		var activeSize int64
		var activeArtifacts []Artifact

		for _, a := range p.Artifacts {
			if a.Tier == TierIgnored && !opts.ShowIgnored {
				continue
			}
			if a.Tier == TierDeep && !opts.IncludeDeep {
				continue
			}
			activeSize += a.Size
			activeArtifacts = append(activeArtifacts, a)
		}

		// Skip projects with no matching artifacts
		if activeSize == 0 {
			continue
		}

		// Apply min-size filter
		if opts.MinSize > 0 && activeSize < opts.MinSize {
			continue
		}

		// Apply older-than filter (skip projects that are too recent)
		if opts.OlderThan > 0 && !p.LastModTime.IsZero() {
			if time.Since(p.LastModTime) < opts.OlderThan {
				continue
			}
		}

		p.TotalSize = activeSize
		p.Artifacts = activeArtifacts
		filtered = append(filtered, p)
	}

	return filtered
}
