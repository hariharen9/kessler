package engine

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GlobalCache represents a system-level package manager cache.
type GlobalCache struct {
	Name         string // e.g. "npm"
	Description  string // e.g. "Node.js package cache"
	Path         string // Absolute path to cache directory
	Size         int64
	Exists       bool
	CleanCommand string // Equivalent CLI command for transparency
	Icon         string
}

// GlobalScanProgress reports live progress for global cache scanning.
type GlobalScanProgress struct {
	CachesProcessed int
	TotalCaches     int
	CurrentCache    string
	TotalSize       int64
}

// ScanGlobalCaches detects known system-level caches and returns their sizes.
func ScanGlobalCaches() []GlobalCache {
	caches := getGlobalCacheList()
	return performGlobalScan(caches, nil)
}

// ScanGlobalCachesWithProgress is like ScanGlobalCaches but sends progress updates.
func ScanGlobalCachesWithProgress(progress chan<- GlobalScanProgress) []GlobalCache {
	caches := getGlobalCacheList()
	return performGlobalScan(caches, progress)
}

func getGlobalCacheList() []GlobalCache {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	caches := []GlobalCache{
		{
			Name:         "npm",
			Description:  "Node.js package download cache",
			Path:         npmCachePath(home),
			CleanCommand: "npm cache clean --force",
			Icon:         "📦",
		},
		{
			Name:         "Yarn",
			Description:  "Yarn package cache",
			Path:         yarnCachePath(home),
			CleanCommand: "yarn cache clean",
			Icon:         "🧶",
		},
		{
			Name:         "pnpm",
			Description:  "pnpm content-addressable store",
			Path:         pnpmCachePath(home),
			CleanCommand: "pnpm store prune",
			Icon:         "📦",
		},
		{
			Name:         "pip",
			Description:  "Python package download cache",
			Path:         pipCachePath(home),
			CleanCommand: "pip cache purge",
			Icon:         "🐍",
		},
		{
			Name:         "Go Modules",
			Description:  "Go module download cache",
			Path:         goCachePath(home),
			CleanCommand: "go clean -modcache",
			Icon:         "🐹",
		},
		{
			Name:         "Gradle",
			Description:  "Gradle dependency cache",
			Path:         filepath.Join(home, ".gradle", "caches"),
			CleanCommand: "rm -rf ~/.gradle/caches",
			Icon:         "🐘",
		},
		{
			Name:         "Cargo",
			Description:  "Rust crate registry cache",
			Path:         filepath.Join(home, ".cargo", "registry", "cache"),
			CleanCommand: "cargo cache --autoclean",
			Icon:         "🦀",
		},
		{
			Name:         "Bun",
			Description:  "Bun global package cache",
			Path:         filepath.Join(home, ".bun", "install", "cache"),
			CleanCommand: "bun pm cache rm",
			Icon:         "🥟",
		},
		{
			Name:         "Nix",
			Description:  "Nix package store (requires sudo for GC)",
			Path:         "/nix/store",
			CleanCommand: "nix-collect-garbage -d",
			Icon:         "❄️",
		},
		{
			Name:         "Vagrant",
			Description:  "Vagrant box cache",
			Path:         filepath.Join(home, ".vagrant.d", "boxes"),
			CleanCommand: "vagrant box prune",
			Icon:         "📦",
		},
		{
			Name:         "Composer",
			Description:  "PHP Composer cache",
			Path:         composerCachePath(home),
			CleanCommand: "composer clear-cache",
			Icon:         "🐘",
		},
		{
			Name:         "Maven",
			Description:  "Maven local repository",
			Path:         filepath.Join(home, ".m2", "repository"),
			CleanCommand: "rm -rf ~/.m2/repository",
			Icon:         "🪶",
		},
	}

	// Platform-specific caches
	if runtime.GOOS == "darwin" {
		caches = append(caches,
			GlobalCache{
				Name:         "CocoaPods",
				Description:  "iOS/macOS pod cache",
				Path:         filepath.Join(home, "Library", "Caches", "CocoaPods"),
				CleanCommand: "pod cache clean --all",
				Icon:         "🍫",
			},
			GlobalCache{
				Name:         "Homebrew",
				Description:  "Downloaded bottles and formulae",
				Path:         filepath.Join(home, "Library", "Caches", "Homebrew"),
				CleanCommand: "brew cleanup -s",
				Icon:         "🍺",
			},
			GlobalCache{
				Name:         "Xcode",
				Description:  "Xcode DerivedData (rebuild on compile)",
				Path:         filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData"),
				CleanCommand: "rm -rf ~/Library/Developer/Xcode/DerivedData",
				Icon:         "🔨",
			},
		)
	}

	// JetBrains IDE caches (cross-platform)
	jetbrainsPath := filepath.Join(home, ".cache", "JetBrains")
	if runtime.GOOS == "darwin" {
		jetbrainsPath = filepath.Join(home, "Library", "Caches", "JetBrains")
	}
	caches = append(caches, GlobalCache{
		Name:         "JetBrains",
		Description:  "IntelliJ/WebStorm/PyCharm IDE caches",
		Path:         jetbrainsPath,
		CleanCommand: "rm -rf " + jetbrainsPath,
		Icon:         "🧠",
	})

	// Docker (only if docker is installed)
	if cmdExists("docker") {
		caches = append(caches, GlobalCache{
			Name:         "Docker",
			Description:  "Dangling images, stopped containers, build cache",
			Path:         "(managed by Docker daemon)",
			CleanCommand: "docker system prune -f",
			Icon:         "🐳",
		})
	}

	return caches
}

func performGlobalScan(caches []GlobalCache, progress chan<- GlobalScanProgress) []GlobalCache {
	var result []GlobalCache
	var totalSize int64

	for i, cache := range caches {
		if progress != nil {
			progress <- GlobalScanProgress{
				CachesProcessed: i,
				TotalCaches:     len(caches),
				CurrentCache:    cache.Name,
				TotalSize:       totalSize,
			}
		}

		if cache.Name == "Docker" {
			cache.Size = getDockerReclaimableSize()
			cache.Exists = cache.Size > 0
		} else {
			info, err := os.Stat(cache.Path)
			if err == nil && info.IsDir() {
				cache.Exists = true
				cache.Size = dirSize(cache.Path)
			}
		}

		if cache.Exists && cache.Size > 0 {
			result = append(result, cache)
			totalSize += cache.Size
		}
	}

	if progress != nil {
		progress <- GlobalScanProgress{
			CachesProcessed: len(caches),
			TotalCaches:     len(caches),
			CurrentCache:    "Complete",
			TotalSize:       totalSize,
		}
	}

	return result
}


// CleanGlobalCache cleans a global cache, preferring native commands.
func CleanGlobalCache(cache GlobalCache) error {
	// Try native clean command first
	switch cache.Name {
	case "npm":
		if cmdExists("npm") {
			return exec.Command("npm", "cache", "clean", "--force").Run()
		}
	case "Yarn":
		if cmdExists("yarn") {
			return exec.Command("yarn", "cache", "clean").Run()
		}
	case "pnpm":
		if cmdExists("pnpm") {
			return exec.Command("pnpm", "store", "prune").Run()
		}
	case "pip":
		if cmdExists("pip3") {
			return exec.Command("pip3", "cache", "purge").Run()
		}
		if cmdExists("pip") {
			return exec.Command("pip", "cache", "purge").Run()
		}
	case "Go Modules":
		if cmdExists("go") {
			return exec.Command("go", "clean", "-modcache").Run()
		}
	case "CocoaPods":
		if cmdExists("pod") {
			return exec.Command("pod", "cache", "clean", "--all").Run()
		}
	case "Homebrew":
		if cmdExists("brew") {
			return exec.Command("brew", "cleanup", "-s").Run()
		}
	case "Docker":
		if cmdExists("docker") {
			return exec.Command("docker", "system", "prune", "-f").Run()
		}
		return nil // No fallback for Docker
	case "Bun":
		if cmdExists("bun") {
			return exec.Command("bun", "pm", "cache", "rm").Run()
		}
	case "Nix":
		if cmdExists("nix-collect-garbage") {
			return exec.Command("nix-collect-garbage", "-d").Run()
		}
	case "Vagrant":
		if cmdExists("vagrant") {
			return exec.Command("vagrant", "global-status", "--prune").Run()
		}
	case "Cargo":
		if cmdExists("cargo") {
			// Cargo cache clean requires an external plugin (cargo-cache), fallback to removing dir if plugin doesn't exist
			if err := exec.Command("cargo", "cache", "--autoclean").Run(); err == nil {
				return nil
			}
		}
	case "Composer":
		if cmdExists("composer") {
			return exec.Command("composer", "clear-cache").Run()
		}
	}

	// Fallback: remove directory contents
	return os.RemoveAll(cache.Path)
}

func yarnCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "Yarn")
	} else if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Yarn", "Cache")
	}
	return filepath.Join(home, ".cache", "yarn")
}

func npmCachePath(home string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "npm-cache")
	}
	return filepath.Join(home, ".npm", "_cacache")
}

func pnpmCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "pnpm", "store")
	} else if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "pnpm", "store")
	}
	return filepath.Join(home, ".local", "share", "pnpm", "store")
}

func goCachePath(home string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "go", "pkg", "mod", "cache") // Windows usually uses GOPATH\pkg\mod or LOCALAPPDATA\go-build
	}
	return filepath.Join(home, "go", "pkg", "mod", "cache")
}

func composerCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "composer")
	} else if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Composer")
	}
	return filepath.Join(home, ".cache", "composer")
}

func pipCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "pip")
	} else if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "pip", "Cache")
	}
	return filepath.Join(home, ".cache", "pip")
}

func dirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}

func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// getDockerReclaimableSize estimates reclaimable Docker space via `docker system df`.
func getDockerReclaimableSize() int64 {
	out, err := exec.Command("docker", "system", "df", "--format", "{{.Reclaimable}}").Output()
	if err != nil {
		return 0
	}

	var total int64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// Lines look like "1.2GB (30%)" or "500MB" or "0B"
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip parenthetical percentage
		if idx := strings.Index(line, "("); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}

		total += parseDockerSize(line)
	}

	return total
}

// parseDockerSize converts Docker's size strings (e.g., "1.2GB", "500MB", "10kB") to bytes.
func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(s)
	multiplier := int64(1)

	if strings.HasSuffix(s, "GB") {
		multiplier = 1000 * 1000 * 1000
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1000 * 1000
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "kB") {
		multiplier = 1000
		s = strings.TrimSuffix(s, "kB")
	} else if strings.HasSuffix(s, "B") {
		s = strings.TrimSuffix(s, "B")
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multiplier))
}

// SaveCleanEntry saves a history entry for a global clean operation.
func SaveCleanEntry(cacheName string, freedSpace int64) {
	SaveEntry(ScanHistoryEntry{
		Timestamp:  time.Now(),
		ScanPath:   fmt.Sprintf("[global] %s", cacheName),
		FreedSpace: freedSpace,
	})
}
