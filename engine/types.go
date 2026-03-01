package engine

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
	Rules []Rule `yaml:"rules"`
}

type Artifact struct {
	Path string
	Size int64
	Tier Tier
}

type Project struct {
	Path      string
	Type      string // e.g., "Node.js", "Python"
	Artifacts []Artifact
	TotalSize int64
}
