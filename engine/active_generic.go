//go:build !darwin

package engine

import (
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// ActiveProcess represents a running process that is active in a project directory.
type ActiveProcess struct {
	PID  int32
	Name string
}

// GetActiveProcessesInPath returns a list of processes that have their CWD inside the given path.
// This is used to warn users before cleaning "active" projects.
func GetActiveProcessesInPath(targetPath string) []ActiveProcess {
	var active []ActiveProcess
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		absTarget = targetPath
	}

	for _, p := range procs {
		cwd, err := p.Cwd()
		if err != nil {
			continue
		}

		absCwd, err := filepath.Abs(cwd)
		if err != nil {
			absCwd = cwd
		}

		if strings.HasPrefix(absCwd, absTarget) {
			name, _ := p.Name()
			active = append(active, ActiveProcess{
				PID:  p.Pid,
				Name: name,
			})
		}
	}

	return active
}
