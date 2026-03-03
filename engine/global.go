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
			Path:         filepath.Join(home, ".npm", "_cacache"),
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
			Path:         filepath.Join(home, ".local", "share", "pnpm", "store"),
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
			Path:         filepath.Join(home, "go", "pkg", "mod", "cache"),
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
	}

	// Fallback: remove directory contents
	return os.RemoveAll(cache.Path)
}

func yarnCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "Yarn")
	}
	return filepath.Join(home, ".cache", "yarn")
}

func pipCachePath(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "pip")
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
