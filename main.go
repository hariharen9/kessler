package main

import (
	_ "embed"

	"github.com/hariharen/kessler/cmd"
)

//go:embed rules.yaml
var defaultRules []byte

func main() {
	cmd.RulesData = defaultRules
	cmd.Execute()
}
