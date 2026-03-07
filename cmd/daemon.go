package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/hariharen9/kessler/engine"
	"github.com/spf13/cobra"
)

var (
	daemonStart  bool
	daemonStop   bool
	daemonStatus bool
	daemonRun    bool // Hidden flag for the scheduler to call
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage Kessler's background sweep daemon",
	Long: `Kessler can run silently in the background to monitor your system.
It scans once a week and automatically sweeps away more than 1GB of 
stale debris (older than 10 days) in safe mode.`,
	RunE: runDaemon,
}

func init() {
	daemonCmd.Flags().BoolVar(&daemonStart, "start", false, "Start/Install the background daemon")
	daemonCmd.Flags().BoolVar(&daemonStop, "stop", false, "Stop/Uninstall the background daemon")
	daemonCmd.Flags().BoolVar(&daemonStatus, "status", false, "Show current schedule status")
	daemonCmd.Flags().BoolVar(&daemonRun, "run", false, "Internal use only: run the background check")
	daemonCmd.Flags().MarkHidden("run")
	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	if daemonRun {
		return runDaemonCheck()
	}

	if daemonStop {
		return uninstallDaemon()
	}

	if daemonStart {
		return installDaemon()
	}

	// Default: show status
	return showDaemonStatus()
}

func runDaemonCheck() error {
	// 1. Initialize scanner
	scanner, err := engine.NewScannerMerged(RulesData, CommunityRulesData, UserRulesData, nil)
	if err != nil {
		return err
	}

	// 2. Perform a silent scan of the user's home directory (default)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	projects, err := scanner.Scan([]string{home})
	if err != nil {
		return err
	}

	// 3. Filter for debris older than 10 days
	opts := engine.FilterOptions{
		IncludeDeep: false, // AUTO-CLEAN ONLY USES SAFE MODE
		ShowIgnored: false, // Don't auto-clean ignored files unless explicitly told
		OlderThan:   10 * 24 * time.Hour,
	}
	filtered := engine.FilterProjects(projects, opts)

	var totalDebris int64
	var artifactsToClean []engine.Artifact
	for _, p := range filtered {
		totalDebris += p.TotalSize
		artifactsToClean = append(artifactsToClean, p.Artifacts...)
	}

	// 4. Threshold: 1GB
	const threshold = 1 * 1024 * 1024 * 1024
	if totalDebris >= threshold {
		var freedSpace int64
		for _, a := range artifactsToClean {
			absPath, _ := filepath.Abs(a.Path)
			if err := engine.MoveToTrash(absPath); err == nil {
				freedSpace += a.Size
			}
		}

		if freedSpace > 0 {
			sizeStr := formatBytes(freedSpace)
			beeep.Notify(
				"🛰️ Kessler Auto-Pilot",
				fmt.Sprintf("Safely cleared %s of stale debris (untouched for 10+ days).", sizeStr),
				"",
			)
			// Log the success
			engine.SaveCleanEntry("Auto-Pilot Sweep", freedSpace)
		}
	}

	return nil
}

func installDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	absExe, err := filepath.Abs(exe)
	if err == nil {
		exe = absExe
	}

	if runtime.GOOS == "darwin" {
		return installMacLaunchAgent(exe)
	} else if runtime.GOOS == "windows" {
		return installWindowsTask(exe)
	} else {
		return installLinuxCron(exe)
	}
}

func uninstallDaemon() error {
	if runtime.GOOS == "darwin" {
		return uninstallMacLaunchAgent()
	} else if runtime.GOOS == "windows" {
		return uninstallWindowsTask()
	} else {
		return uninstallLinuxCron()
	}
}

func showDaemonStatus() error {
	fmt.Println("\n  🛰️  KESSLER DAEMON STATUS")
	fmt.Println("  " + strings.Repeat("─", 30))

	var active bool
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "kessler.debris.monitor.plist")
		_, err := os.Stat(plistPath)
		active = err == nil
	} else if runtime.GOOS == "windows" {
		out, err := exec.Command("schtasks", "/query", "/tn", "KesslerDebrisMonitor").CombinedOutput()
		active = err == nil && !strings.Contains(string(out), "ERROR")
	} else {
		out, _ := exec.Command("crontab", "-l").Output()
		active = strings.Contains(string(out), "kessler daemon --run")
	}

	if active {
		fmt.Println("  Status  : ✅ ACTIVE (Background Monitoring)")
		fmt.Println("  Schedule: Every Sunday at Midnight")
		fmt.Println("  Trigger : > 1GB of 10-day old debris (Auto-Sweep)")
		fmt.Println("  Mode    : Safe Only (no builds or binaries)")
	} else {
		fmt.Println("  Status  : 🔭 INACTIVE")
		fmt.Println("\n  Run 'kessler daemon --start' to enable background monitoring.")
	}
	fmt.Println()
	return nil
}

// --- macOS Implementation (LaunchAgents) ---

func installMacLaunchAgent(exePath string) error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "kessler.debris.monitor.plist")

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>kessler.debris.monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>--run</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>0</integer>
        <key>Minute</key>
        <integer>0</integer>
        <key>Weekday</key>
        <integer>7</integer> <!-- Sunday -->
    </dict>
    <key>StandardErrorPath</key>
    <string>%s/.kessler/daemon.err</string>
    <key>StandardOutPath</key>
    <string>%s/.kessler/daemon.out</string>
</dict>
</plist>`, exePath, home, home)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return err
	}

	// Load the agent
	exec.Command("launchctl", "unload", plistPath).Run() // unload first just in case
	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %s", string(out))
	}

	fmt.Println("\n  ✨ Kessler Daemon installed as a macOS LaunchAgent. See you on Sunday! 🚀")
	return nil
}

func uninstallMacLaunchAgent() error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "kessler.debris.monitor.plist")

	exec.Command("launchctl", "unload", plistPath).Run()
	os.Remove(plistPath)
	fmt.Println("\n  🧹 Kessler Daemon uninstalled.")
	return nil
}

// --- Windows Implementation (SchTasks) ---

func installWindowsTask(exePath string) error {
	args := []string{
		"/create", "/sc", "weekly", "/d", "SUN", "/st", "00:00",
		"/tn", "KesslerDebrisMonitor",
		"/tr", fmt.Sprintf("\"%s\" daemon --run", exePath),
		"/f",
	}
	out, err := exec.Command("schtasks", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to schedule task: %s", string(out))
	}
	fmt.Println("\n  ✨ Kessler Daemon scheduled as a Windows Task. See you on Sunday! 🚀")
	return nil
}

func uninstallWindowsTask() error {
	out, err := exec.Command("schtasks", "/delete", "/tn", "KesslerDebrisMonitor", "/f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove task: %s", string(out))
	}
	fmt.Println("\n  🧹 Kessler Daemon uninstalled.")
	return nil
}

// --- Linux Implementation (Crontab) ---

func installLinuxCron(exePath string) error {
	cronLine := fmt.Sprintf("0 0 * * 0 %s daemon --run >> ~/.kessler/daemon.log 2>&1", exePath)
	
	current, _ := exec.Command("crontab", "-l").Output()
	currentStr := string(current)
	
	if strings.Contains(currentStr, "kessler daemon --run") {
		// Replace existing
		lines := strings.Split(currentStr, "\n")
		var newLines []string
		for _, line := range lines {
			if line != "" && !strings.Contains(line, "kessler daemon --run") {
				newLines = append(newLines, line)
			}
		}
		newLines = append(newLines, cronLine)
		currentStr = strings.Join(newLines, "\n") + "\n"
	} else {
		currentStr += cronLine + "\n"
	}

	tmpFile := filepath.Join(os.TempDir(), "kessler-daemon-cron")
	if err := os.WriteFile(tmpFile, []byte(currentStr), 0644); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	if out, err := exec.Command("crontab", tmpFile).CombinedOutput(); err != nil {
		return fmt.Errorf("crontab update failed: %s", string(out))
	}

	fmt.Println("\n  ✨ Kessler Daemon added to crontab. See you on Sunday! 🚀")
	return nil
}

func uninstallLinuxCron() error {
	current, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(current), "\n")
	var newLines []string
	found := false
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "kessler daemon --run") {
			newLines = append(newLines, line)
		} else if strings.Contains(line, "kessler daemon --run") {
			found = true
		}
	}

	if !found {
		fmt.Println("\n  🔭 No Kessler daemon schedule found.")
		return nil
	}

	if len(newLines) == 0 {
		exec.Command("crontab", "-r").Run()
	} else {
		tmpFile := filepath.Join(os.TempDir(), "kessler-daemon-cron")
		os.WriteFile(tmpFile, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
		defer os.Remove(tmpFile)
		exec.Command("crontab", tmpFile).Run()
	}

	fmt.Println("\n  🧹 Kessler Daemon uninstalled.")
	return nil
}
