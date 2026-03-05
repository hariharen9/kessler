# 🛰️ Kessler

> **Kessler Syndrome** (noun): A theoretical scenario in which the density of objects in low Earth orbit is high enough that collisions between objects could cause a cascade, generating space debris that increases the likelihood of further collisions, rendering space exploration impossible.

For developers, your hard drive is low Earth orbit. 

Over time, it gets clogged with `node_modules`, stray `build/` folders, forgotten Python virtual environments, and intermediate Rust targets. This **digital space debris** silently consumes hundreds of gigabytes of storage until your system grinds to a halt.

**Kessler** is an intelligent, blazingly fast, and incredibly safe command-line tool built in Go that clears the orbit. It finds, calculates, and safely sweeps away runtime artifacts and build caches without ever touching your source code.

---

## ✨ Features

- 🏎️ **Blazingly Fast:** Written in Go, it scans massive directory trees concurrently using a high-performance worker pool.
- 🛡️ **Active Project Protection:** Never accidentally delete artifacts while your dev server is running. Kessler detects active processes (PIDs) using a project's folder as their working directory and warns you before cleaning.
- 🚀 **Project Launchpad (New!):** Not just a cleaner, but a navigator. Kessler remembers all your projects; use Tab 4 to fuzzy-search and open them instantly in **VS Code**, **Cursor**, or your **Terminal**.
- 🧪 **Environmental Doctor:** Your system's "Spring Cleaning" assistant. Identifies unused Node.js (nvm) and Rust (rustup) toolchain versions that aren't needed by any of your current projects.
- 🔍 **Context-Aware Engine:** Doesn't just blindly delete folders. It looks for triggers (e.g., `package.json`, `Cargo.toml`) to identify project types and *only* targets known safe artifacts for that specific ecosystem.
- 🛡️ **The Git Safety Net:** Before Kessler flags *any* folder as junk, it silently queries Git (`git ls-files`). If a folder contains files actively tracked by version control, Kessler immediately aborts and ignores it.
- 🌍 **Global Cache Management:** Beyond project-level debris, Kessler identifies and safely cleans system-level caches for **Docker**, **Homebrew**, **npm**, **Cargo**, **Go modules**, and more.
- ♻️ **OS Trash Integration:** Mistakes happen. Instead of using a terrifying `rm -rf`, Kessler safely moves debris to your native OS Trash/Recycle Bin (supports macOS, Windows, and Linux), giving you an "Undo" button.
- 🎨 **Beautiful TUI & Telemetry:** Powered by Charmbracelet's Bubble Tea. Features an interactive dashboard with live "Orbital Telemetry," ecosystem icons, root drive usage, and 4 tabbed views.
- 📜 **Scan History:** Keeps track of your previous sweeps and total space freed, helping you monitor your disk's "orbital health" over time.
- 🤖 **CI / Scripting Mode:** Use `kessler scan` and `kessler clean` subcommands for non-interactive usage. Supports JSON output, dry-run, and filtering by size and age.

---

## 🚀 Installation

### Homebrew (macOS & Linux)

```bash
brew tap hariharen9/tap
brew install kessler
```

### Scoop (Windows)

```powershell
scoop bucket add hariharen9 https://github.com/hariharen9/scoop-bucket
scoop install kessler
```

### npm (All platforms)

```bash
npm install -g kessler-cli
# or use without installing
npx kessler-cli ~/Projects
```

### AUR (Arch Linux)

```bash
yay -S kessler-bin
```

### Debian / Ubuntu (.deb)

Download the `.deb` from the [latest release](https://github.com/hariharen9/kessler/releases/latest), then:

```bash
sudo dpkg -i kessler_*.deb
```

### Go Install

```bash
go install github.com/hariharen9/kessler@latest
```

---

## 🎮 Usage

### Interactive TUI (Default)

Run Kessler without a subcommand to launch the interactive dashboard.

```bash
kessler ~/Projects      # Scan your Projects folder
kessler .               # Scan the current directory
kessler . --deep        # Include build outputs (dist, build, bin)
```

**TUI Controls:**
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Space` | Toggle selection |
| `a` | Select / deselect all |
| `e` | **Select Ecosystem** (Bulk select same project types) |
| `S` | **Select Stale** (Select projects untouched for >30 days) |
| `t` | Toggle Safe ↔ Deep mode (Tab 1) |
| `o` | **Open in VS Code / Cursor** (Tab 4) |
| `t` | **Open in Terminal** (Tab 4) |
| `i` | Toggle gitignored artifacts |
| `s` | Sort by Size ↔ Name |
| `/` | Search projects |
| `p` | Toggle full Project Preview |
| `1` - `4` | Switch tabs (Projects / Global / History / Launchpad) |
| `Tab` | Cycle through tabs |
| `Enter` | Move selected to Trash |
| `X` | Permanently delete |
| `q` | Quit |

### Non-Interactive / CI Mode

#### `kessler scan`

Scan and report — no deletion.

```bash
kessler scan ~/Projects                          # Table output
kessler scan ~/Projects --json                   # JSON output (pipe to jq)
```

#### `kessler clean`

Scan and clean — shows a preview and asks for confirmation. Includes **Active Project Protection**.

```bash
kessler clean ~/Projects                         # Preview + confirm
kessler clean ~/Projects --deep                  # Deep clean (extra warning)
kessler clean ~/Projects --force                 # Skip confirmation + safety checks
```

---

## 🛰️ Kessler vs. Kondo

While tools like **Kondo** or **npkill** are great for basic cleaning, Kessler is engineered as a **Safety-First Disk Management Dashboard** for polyglot developers.

| Feature | Kondo | **Kessler** |
|:---|:---:|:---:|
| **The Git Safety Net** | ❌ No | ✅ **Yes** (`git ls-files` check) |
| **Active Project Protection**| ❌ No | ✅ **Yes** (Warns if dev server is running) |
| **Project Launchpad** | ❌ No | ✅ **Yes** (Tab 4 Project Navigator) |
| **Environmental Doctor** | ❌ No | ✅ **Yes** (Detects unused toolchains) |
| **Global Cache Cleaning** | ❌ No | ✅ **Yes** (npm, pip, go, docker, etc.) |
| **Tiered Cleaning** | ❌ No | ✅ **Yes** (Safe vs. Deep mode) |
| **`.gitignore` Scanning** | ❌ No | ✅ **Yes** (Finds hidden ignored dirs) |
| **High-Fidelity TUI** | ⚠️ Basic | ✅ **Yes** (Modern Charm/Bubble Tea) |
| **OS Trash Integration** | ✅ Yes | ✅ **Yes** (+ Windows PowerShell support) |

---

## ⚙️ How the Rules Engine Works

Kessler is powered by a dynamic rules engine (`rules.yaml`). When Kessler enters a directory, it looks for **Trigger Files**. If it finds `package.json`, it identifies it as a Node.js project and targets known safe artifacts.

Current support includes:
- **Node.js:** `node_modules`, `dist`, `build`, `.next`, `.nuxt`, `.svelte-kit`, `coverage`
- **Python:** `__pycache__`, `venv`, `.venv`, `.pytest_cache`, `.mypy_cache`, `wandb`
- **Rust:** `target`
- **Go:** `vendor`, `bin`, `.gocache`
- **Java / JVM:** `target`, `build`, `.gradle`
- **Infrastructure & Cloud:** `.terraform`, `cdk.out`, `.serverless`, `.aws-sam`, `.wrangler`, `.amplify`, `supabase/.temp`
- **PHP:** `vendor`
- **Ruby:** `vendor/bundle`, `.bundle`
- **.NET / C#:** `bin`, `obj`, `packages`
- **Mobile:** Swift (`DerivedData`, `Pods`), Flutter (`build`, `.dart_tool`), Android (`.gradle`, `build`)
- **Game Engines:** Unreal Engine (`Binaries`, `Intermediate`, `Saved`), Godot (`.godot`)

### `.gitignore` Intelligence

For every detected project, it runs `git ls-files --ignored --directory` to discover **directories your `.gitignore` is hiding** that aren't already covered by Kessler's rules. These appear as `[user ignored]` artifacts.

**Safety guarantees:**
- ✅ Only ignored **directories** are surfaced — individual files (`.env`, lockfiles, configs) are never shown
- ✅ Trigger files (`package.json`, `Cargo.toml`, etc.) are always excluded
- ✅ Danger zone items can never appear, even if gitignored

### Custom User Rules

Extend or override the built-in rules by creating `~/.config/kessler/rules.yaml`:

```yaml
rules:
  # Add a brand new ecosystem
  - name: "Swift"
    triggers: ["Package.swift"]
    targets:
      - path: ".build"
        tier: "safe"

  # Add extra targets to an existing ecosystem (merged by name)
  - name: "Node.js / JS Ecosystem"
    targets:
      - path: ".cache"
        tier: "safe"

danger_zone:
  - "secrets.json"
```

---

## ⚠️ The Safety Philosophy

Kessler is built with **developer trust** as its core tenet:

1. **Active Process Check:** It won't let you delete a project's `node_modules` while its dev server is still running.
2. **It respects Git:** If Git tracks a folder (even if named `bin`), Kessler won't touch it.
3. **It respects State:** It never targets files required to reproduce a build (like lockfiles) or environment secrets (like `.env`).
4. **It respects the OS:** Moving files to the Trash Bin instead of permanent deletion gives you an "Undo" button.

---

## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.
