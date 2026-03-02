package engine

import "time"

type Tier string

const (
	TierSafe   Tier = "safe"
	TierDeep   Tier = "deep"
	TierDanger Tier = "danger"
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
