package engine

import "time"

type Tier string

const (
	TierSafe    Tier = "safe"
	TierDeep    Tier = "deep"
	TierDanger  Tier = "danger"
	TierIgnored Tier = "ignored"
)

type RuleTarget struct {
	Path string `yaml:"path"`
	Tier Tier   `yaml:"tier"`
}

type Rule struct {
	Name     string       `yaml:"name"`
	Triggers []string     `yaml:"triggers"`
	Targets  []RuleTarget `yaml:"targets"`
}

type Config struct {
	Rules      []Rule   `yaml:"rules"`
	DangerZone []string `yaml:"danger_zone"`
}

// MergeConfigs merges a user config into a base config.
// - Rules with the same Name are merged (user targets are appended).
// - Rules with new Names are appended.
// - DangerZone entries are unioned.
func MergeConfigs(base, user Config) Config {
	merged := base

	for _, userRule := range user.Rules {
		found := false
		for i, baseRule := range merged.Rules {
			if baseRule.Name == userRule.Name {
				// Merge triggers (union)
				triggerSet := make(map[string]bool)
				for _, t := range baseRule.Triggers {
					triggerSet[t] = true
				}
				for _, t := range userRule.Triggers {
					if !triggerSet[t] {
						merged.Rules[i].Triggers = append(merged.Rules[i].Triggers, t)
					}
				}
				// Merge targets (override tier or append new)
				for _, ut := range userRule.Targets {
					foundTarget := false
					for j, mt := range merged.Rules[i].Targets {
						if mt.Path == ut.Path {
							merged.Rules[i].Targets[j].Tier = ut.Tier
							foundTarget = true
							break
						}
					}
					if !foundTarget {
						merged.Rules[i].Targets = append(merged.Rules[i].Targets, ut)
					}
				}
				found = true
				break
			}
		}
		if !found {
			merged.Rules = append(merged.Rules, userRule)
		}
	}

	// Union danger zone
	dangerSet := make(map[string]bool)
	for _, d := range merged.DangerZone {
		dangerSet[d] = true
	}
	for _, d := range user.DangerZone {
		if !dangerSet[d] {
			merged.DangerZone = append(merged.DangerZone, d)
		}
	}

	return merged
}

type Artifact struct {
	Path    string
	Size    int64
	Tier    Tier
	ModTime time.Time
}

type Project struct {
	Path        string
	Type        string // e.g., "Node.js", "Python"
	Artifacts   []Artifact
	TotalSize   int64
	LastModTime time.Time
}
