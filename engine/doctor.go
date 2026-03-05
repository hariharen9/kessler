package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Toolchain struct {
	Name        string
	Installed   []string
	InUse       []string
	Unused      []string
	Path        string
	Description string
}

// GetUnusedToolchains scans the system for toolchains and identifies versions not used by scanned projects.
func GetUnusedToolchains(projects []Project) []Toolchain {
	var toolchains []Toolchain

	// 1. Node.js (via nvm)
	if node := getUnusedNode(projects); node != nil {
		toolchains = append(toolchains, *node)
	}

	// 2. Rust (via rustup)
	if rust := getUnusedRust(projects); rust != nil {
		toolchains = append(toolchains, *rust)
	}

	// 3. Python (via pyenv)
	if python := getUnusedPython(projects); python != nil {
		toolchains = append(toolchains, *python)
	}

	// 4. Ruby (via rbenv)
	if ruby := getUnusedRuby(projects); ruby != nil {
		toolchains = append(toolchains, *ruby)
	}

	// 5. Java (via SDKMAN)
	if java := getUnusedJava(projects); java != nil {
		toolchains = append(toolchains, *java)
	}

	// 6. Universal (via asdf / mise)
	if multi := getUnusedMulti(projects); multi != nil {
		toolchains = append(toolchains, *multi)
	}

	return toolchains
}

func getUnusedJava(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	sdkPath := filepath.Join(home, ".sdkman", "candidates", "java")
	
	entries, err := os.ReadDir(sdkPath)
	if err != nil {
		return nil
	}

	var installed []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "current" {
			installed = append(installed, e.Name())
		}
	}

	if len(installed) == 0 {
		return nil
	}

	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, ".sdkmanrc")); err == nil {
			content := string(data)
			if strings.Contains(content, "java=") {
				ver := strings.Split(strings.Split(content, "java=")[1], "\n")[0]
				inUseMap[strings.TrimSpace(ver)] = true
			}
		}
	}

	var unused []string
	for _, inst := range installed {
		if !inUseMap[inst] {
			// Skip active
			if out, err := exec.Command("java", "-version").CombinedOutput(); err == nil {
				if strings.Contains(string(out), inst) {
					continue
				}
			}
			unused = append(unused, inst)
		}
	}

	return &Toolchain{
		Name:        "Java",
		Description: "Versions managed by SDKMAN",
		Installed:   installed,
		Unused:      unused,
		Path:        sdkPath,
	}
}

func getUnusedMulti(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	
	// Try asdf first, then mise
	multiPath := filepath.Join(home, ".asdf", "installs")
	name := "asdf"
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "mise", "installs")); err == nil {
		multiPath = filepath.Join(home, ".local", "share", "mise", "installs")
		name = "mise"
	}

	plugins, err := os.ReadDir(multiPath)
	if err != nil {
		return nil
	}

	// For asdf/mise, we collect ALL unused versions across ALL plugins
	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, ".tool-versions")); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					// parts[0] is plugin, parts[1] is version
					inUseMap[parts[0]+":"+parts[1]] = true
				}
			}
		}
	}

	var unused []string
	for _, plugin := range plugins {
		if !plugin.IsDir() {
			continue
		}
		versions, _ := os.ReadDir(filepath.Join(multiPath, plugin.Name()))
		for _, v := range versions {
			if v.IsDir() {
				if !inUseMap[plugin.Name()+":"+v.Name()] {
					unused = append(unused, plugin.Name()+" "+v.Name())
				}
			}
		}
	}

	if len(unused) == 0 {
		return nil
	}

	return &Toolchain{
		Name:        name,
		Description: "Universal version manager",
		Unused:      unused,
		Path:        multiPath,
	}
}

func getUnusedPython(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	pyenvPath := filepath.Join(home, ".pyenv", "versions")
	
	entries, err := os.ReadDir(pyenvPath)
	if err != nil {
		return nil
	}

	var installed []string
	for _, e := range entries {
		if e.IsDir() {
			installed = append(installed, e.Name())
		}
	}

	if len(installed) == 0 {
		return nil
	}

	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, ".python-version")); err == nil {
			inUseMap[strings.TrimSpace(string(data))] = true
		}
	}

	var unused []string
	for _, inst := range installed {
		if !inUseMap[inst] {
			// Skip active
			if out, err := exec.Command("python", "--version").Output(); err == nil {
				if strings.Contains(string(out), inst) {
					continue
				}
			}
			unused = append(unused, inst)
		}
	}

	return &Toolchain{
		Name:        "Python",
		Description: "Versions managed by pyenv",
		Installed:   installed,
		Unused:      unused,
		Path:        pyenvPath,
	}
}

func getUnusedRuby(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	rbenvPath := filepath.Join(home, ".rbenv", "versions")
	
	entries, err := os.ReadDir(rbenvPath)
	if err != nil {
		return nil
	}

	var installed []string
	for _, e := range entries {
		if e.IsDir() {
			installed = append(installed, e.Name())
		}
	}

	if len(installed) == 0 {
		return nil
	}

	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, ".ruby-version")); err == nil {
			inUseMap[strings.TrimSpace(string(data))] = true
		}
	}

	var unused []string
	for _, inst := range installed {
		if !inUseMap[inst] {
			// Skip active
			if out, err := exec.Command("ruby", "-v").Output(); err == nil {
				if strings.Contains(string(out), inst) {
					continue
				}
			}
			unused = append(unused, inst)
		}
	}

	return &Toolchain{
		Name:        "Ruby",
		Description: "Versions managed by rbenv",
		Installed:   installed,
		Unused:      unused,
		Path:        rbenvPath,
	}
}

func getUnusedNode(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	nvmPath := filepath.Join(home, ".nvm", "versions", "node")
	if runtime.GOOS == "windows" {
		nvmPath = filepath.Join(os.Getenv("APPDATA"), "nvm")
	}

	entries, err := os.ReadDir(nvmPath)
	if err != nil {
		return nil
	}

	var installed []string
	for _, e := range entries {
		if e.IsDir() {
			installed = append(installed, e.Name())
		}
	}

	if len(installed) == 0 {
		return nil
	}

	// Find versions in use
	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, ".nvmrc")); err == nil {
			ver := strings.TrimSpace(string(data))
			inUseMap[ver] = true
			if !strings.HasPrefix(ver, "v") {
				inUseMap["v"+ver] = true
			} else {
				inUseMap[strings.TrimPrefix(ver, "v")] = true
			}
		}
	}

	var unused []string
	for _, inst := range installed {
		if !inUseMap[inst] {
			if out, err := exec.Command("node", "-v").Output(); err == nil {
				if strings.TrimSpace(string(out)) == inst {
					continue
				}
			}
			unused = append(unused, inst)
		}
	}

	return &Toolchain{
		Name:        "Node.js",
		Description: "Versions managed by nvm",
		Installed:   installed,
		Unused:      unused,
		Path:        nvmPath,
	}
}

func getUnusedRust(projects []Project) *Toolchain {
	home, _ := os.UserHomeDir()
	rustupPath := filepath.Join(home, ".rustup", "toolchains")
	
	entries, err := os.ReadDir(rustupPath)
	if err != nil {
		return nil
	}

	var installed []string
	for _, e := range entries {
		if e.IsDir() {
			installed = append(installed, e.Name())
		}
	}

	if len(installed) == 0 {
		return nil
	}

	inUseMap := make(map[string]bool)
	for _, p := range projects {
		if data, err := os.ReadFile(filepath.Join(p.Path, "rust-toolchain")); err == nil {
			inUseMap[strings.TrimSpace(string(data))] = true
		}
		if data, err := os.ReadFile(filepath.Join(p.Path, "rust-toolchain.toml")); err == nil {
			inUseMap[strings.TrimSpace(string(data))] = true
		}
	}

	var unused []string
	for _, inst := range installed {
		if !inUseMap[inst] {
			if out, err := exec.Command("rustc", "--version").Output(); err == nil {
				if strings.Contains(string(out), inst) {
					continue
				}
			}
			unused = append(unused, inst)
		}
	}

	return &Toolchain{
		Name:        "Rust",
		Description: "Toolchains managed by rustup",
		Installed:   installed,
		Unused:      unused,
		Path:        rustupPath,
	}
}
