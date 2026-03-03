package main

import (
	_ "embed"

	"github.com/hariharen9/kessler/cmd"
)

//go:embed rules.yaml
var defaultRules []byte

func main() {
	cmd.RulesData = defaultRules
	cmd.Execute()
}
