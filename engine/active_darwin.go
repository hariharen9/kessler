//go:build darwin

package engine

import (
	"bufio"
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
)

// ActiveProcess represents a running process that is active in a project directory.
type ActiveProcess struct {
	PID  int32
	Name string
}

// GetActiveProcessesInPath returns a list of processes that have their CWD inside the given path.
// This is used to warn users before cleaning "active" projects.
// On Darwin, we use 'lsof' to avoid CGO warnings from gopsutil's CPU dependencies.
func GetActiveProcessesInPath(targetPath string) []ActiveProcess {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		absTarget = targetPath
	}

	// +D finds all processes with open files/CWD in the directory
	// -F pnc returns PID (p) and Name (n)
	cmd := exec.Command("lsof", "-a", "+D", absTarget, "-d", "cwd", "-F", "pn")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()

	var active []ActiveProcess
	scanner := bufio.NewScanner(&out)
	var currentProc ActiveProcess

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		prefix := line[0]
		value := line[1:]

		switch prefix {
		case 'p':
			pid, _ := strconv.ParseInt(value, 10, 32)
			currentProc.PID = int32(pid)
		case 'n':
			currentProc.Name = value
			if currentProc.PID != 0 {
				active = append(active, currentProc)
				currentProc = ActiveProcess{}
			}
		}
	}
	return active
}
