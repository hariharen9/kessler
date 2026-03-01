package engine

import (
	"bytes"
	_ "embed"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed default_rules.yaml
var defaultRules []byte

type Scanner struct {
	Config Config
}

func NewScanner() (*Scanner, error) {
	var config Config
	if err := yaml.Unmarshal(defaultRules, &config); err != nil {
		return nil, err
	}

	return &Scanner{Config: config}, nil
}

func (s *Scanner) Scan(root string) ([]Project, error) {
	var projects []Project

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil 
		}

		if !d.IsDir() {
			return nil
		}

		for _, rule := range s.Config.Rules {
			for _, target := range rule.Targets {
				if d.Name() == target.Path {
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

				targetPath := filepath.Join(path, target.Path)
				if info, err := os.Stat(targetPath); err == nil {
					// --- THE GIT SAFETY NET ---
					// If the target directory contains files tracked by Git, DO NOT touch it.
					if s.isTrackedByGit(path, targetPath) {
						continue
					}

					size := s.calculateSize(targetPath, info)
					if size > 0 {
						project.Artifacts = append(project.Artifacts, Artifact{
							Path: targetPath,
							Size: size,
							Tier: target.Tier,
						})
						project.TotalSize += size
					}
				}
			}

			if len(project.Artifacts) > 0 {
				projects = append(projects, project)
			}
		}

		return nil
	})

	return projects, err
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
			if entryMap[trigger] {
				return rule
			}
		}
	}
	return nil
}

func (s *Scanner) calculateSize(path string, info fs.FileInfo) int64 {
	if !info.IsDir() {
		return info.Size()
	}

	var size int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
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
