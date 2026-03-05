# 🛰️ Kessler

<p align="center">
  <img src="https://img.shields.io/github/v/release/hariharen9/kessler?style=flat-square" alt="Release">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/hariharen9/kessler?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/github/stars/hariharen9/kessler?style=flat-square" alt="Stars">
  <a href="https://goreportcard.com/report/github.com/hariharen9/kessler"><img src="https://goreportcard.com/badge/github.com/hariharen9/kessler?style=flat-square" alt="Go Report Card"></a>
  <img src="https://img.shields.io/github/actions/workflow/status/hariharen9/kessler/release.yml?style=flat-square" alt="Build Status">
  <img src="https://img.shields.io/github/go-mod/go-version/hariharen9/kessler?style=flat-square" alt="Go Version">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey?style=flat-square" alt="Platform Support">
</p>

> **Kessler Syndrome** (noun): A theoretical scenario in which the density of objects in low Earth orbit is high enough that collisions between objects could cause a cascade, generating space debris that increases the likelihood of further collisions, rendering space exploration impossible.

**For developers, your hard drive is low Earth orbit.** 

Over time, it gets clogged with `node_modules`, `targets`, stray `build/` folders, forgotten Python virtual environments, and intermediate Rust targets. This **digital space debris** silently consumes hundreds of gigabytes until your system grinds to a halt.

**Kessler** is an intelligent, blazingly fast, and incredibly safe command-line tool built in Go that clears the orbit. It finds, calculates, and safely sweeps away runtime artifacts and build caches without ever touching your source code.

---

## ✨ Features

- 🏎️ **Blazingly Fast:** Scans massive directory trees concurrently using a high-performance Go worker pool.
- 🛡️ **Active Project Protection:** Warns you before cleaning if a project's dev server is currently running.
- 🚀 **Project Launchpad:** A built-in navigator to fuzzy-search and instantly open projects in VS Code, Cursor, or Terminal.
- 🧪 **Environmental Doctor:** Identifies and cleans unused versions of toolchains (Node.js, Rust, Python, Ruby, Java, etc.).
- 🔍 **Context-Aware Engine:** Targets known safe artifacts based on project triggers (e.g., `package.json`, `Cargo.toml`).
- 🛡️ **Git Safety Net:** Silently queries `git ls-files` to guarantee actively tracked files are never deleted.
- 🌍 **Global Cache Management:** Safely cleans system-level caches (Docker, Homebrew, npm, Cargo, Go modules).
- ♻️ **OS Trash Integration:** Moves debris to your native OS Trash/Recycle Bin instead of a permanent `rm -rf`.
- 🎨 **Beautiful TUI & Telemetry:** An interactive Charmbracelet dashboard with 4 tabbed views and live "Orbital Telemetry".
- 📜 **Scan History:** Tracks previous sweeps and total space freed to monitor your disk's health over time.
- 🤖 **CI / Scripting Mode:** Non-interactive `scan` and `clean` subcommands with JSON output, dry-run, and filtering.

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
```

**TUI Controls:**
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Space` | Toggle selection |
| `a` | Select / deselect all |
| `e` | **Select Ecosystem** (Bulk select same project types) |
| `S` | **Select Stale** (Select projects untouched for >30 days) |
| `t` | **Toggle Tier** (Safe ↔ Deep mode in Tab 1) |
| `o` | **Open in Editor** (VS Code / Cursor in Tab 4) |
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
kessler clean ~/Projects --deep                  # Deep clean
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

Kessler is powered by a dynamic rules engine (`rules.yaml`). When Kessler enters a directory, it looks for **Trigger Files** (like `package.json` or `Cargo.toml`). By understanding context, it only targets known safe artifacts for that specific ecosystem.

Out of the box, Kessler provides **support for over 30 ecosystems and global package managers**—zero configuration required. Really! 😉

### Supported Ecosystems

<p>
  <img src="https://img.shields.io/badge/Node.js-43853D?style=for-the-badge&logo=node.js&logoColor=white" alt="Node.js" />
  <img src="https://img.shields.io/badge/Python-3776AB?style=for-the-badge&logo=python&logoColor=white" alt="Python" />
  <img src="https://img.shields.io/badge/Rust-000000?style=for-the-badge&logo=rust&logoColor=white" alt="Rust" />
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Java-ED8B00?style=for-the-badge&logo=openjdk&logoColor=white" alt="Java" />
  <img src="https://img.shields.io/badge/PHP-777BB4?style=for-the-badge&logo=php&logoColor=white" alt="PHP" />
  <img src="https://img.shields.io/badge/Ruby-CC342D?style=for-the-badge&logo=ruby&logoColor=white" alt="Ruby" />
  <img src="https://img.shields.io/badge/C%23-239120?style=for-the-badge&logo=c-sharp&logoColor=white" alt="C#" />
  <img src="https://img.shields.io/badge/Elixir-4B275F?style=for-the-badge&logo=elixir&logoColor=white" alt="Elixir" />
  <img src="https://img.shields.io/badge/C%2FC%2B%2B-00599C?style=for-the-badge&logo=c%2B%2B&logoColor=white" alt="C/C++" />
  <img src="https://img.shields.io/badge/Swift-FA7343?style=for-the-badge&logo=swift&logoColor=white" alt="Swift" />
  <img src="https://img.shields.io/badge/Flutter-02569B?style=for-the-badge&logo=flutter&logoColor=white" alt="Flutter" />
  <img src="https://img.shields.io/badge/Android-3DDC84?style=for-the-badge&logo=android&logoColor=white" alt="Android" />
  <img src="https://img.shields.io/badge/Scala-DC322F?style=for-the-badge&logo=scala&logoColor=white" alt="Scala" />
  <img src="https://img.shields.io/badge/Haskell-5E5086?style=for-the-badge&logo=haskell&logoColor=white" alt="Haskell" />
  <img src="https://img.shields.io/badge/Zig-F7A41D?style=for-the-badge&logo=zig&logoColor=white" alt="Zig" />
  <img src="https://img.shields.io/badge/R-276DC3?style=for-the-badge&logo=r&logoColor=white" alt="R" />
  <img src="https://img.shields.io/badge/LaTeX-008080?style=for-the-badge&logo=latex&logoColor=white" alt="LaTeX" />
  <img src="https://img.shields.io/badge/Unreal_Engine-0E1128?style=for-the-badge&logo=unrealengine&logoColor=white" alt="Unreal Engine" />
  <img src="https://img.shields.io/badge/Godot-478CBF?style=for-the-badge&logo=godotengine&logoColor=white" alt="Godot" />
  <img src="https://img.shields.io/badge/Terraform-7B42BC?style=for-the-badge&logo=terraform&logoColor=white" alt="Terraform" />
  <img src="https://img.shields.io/badge/Astro-BC52EE?style=for-the-badge&logo=astro&logoColor=white" alt="Astro" />
  <img src="https://img.shields.io/badge/Nx-143055?style=for-the-badge&logo=nx&logoColor=white" alt="Nx" />
</p>

### Global Caches & Package Managers

Beyond project directories, Kessler identifies and safely cleans massive system-level caches that silently eat your hard drive.

<p>
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/Homebrew-F2A900?style=for-the-badge&logo=homebrew&logoColor=black" alt="Homebrew" />
  <img src="https://img.shields.io/badge/npm-CB3837?style=for-the-badge&logo=npm&logoColor=white" alt="npm" />
  <img src="https://img.shields.io/badge/Yarn-2C8EBB?style=for-the-badge&logo=yarn&logoColor=white" alt="Yarn" />
  <img src="https://img.shields.io/badge/pnpm-F69220?style=for-the-badge&logo=pnpm&logoColor=white" alt="pnpm" />
  <img src="https://img.shields.io/badge/Bun-000000?style=for-the-badge&logo=bun&logoColor=white" alt="Bun" />
  <img src="https://img.shields.io/badge/Deno-000000?style=for-the-badge&logo=deno&logoColor=white" alt="Deno" />
  <img src="https://img.shields.io/badge/Composer-885630?style=for-the-badge&logo=composer&logoColor=white" alt="Composer" />
</p>

### Custom User Rules

Need to clean up proprietary build artifacts or custom frameworks? You can extend Kessler's intelligence without modifying the source code.

Simply create a `rules.yaml` file in your config directory (e.g., `~/.config/kessler/rules.yaml` on macOS/Linux or `%APPDATA%\kessler\rules.yaml` on Windows).

```yaml
rules:
  - name: "My Custom Framework"
    triggers: ["my-framework.config"]
    targets:
      - path: ".custom-cache"
        tier: "safe"
      - path: "out_binaries"
        tier: "deep"
```
Your custom rules will automatically merge with the default rules engine.

### `.gitignore` Intelligence

For every detected project, it runs `git ls-files --ignored --directory` to discover **directories your `.gitignore` is hiding** that aren't already covered by Kessler's rules. These appear as `[user ignored]` artifacts.

**Safety guarantees:**
- ✅ Only ignored **directories** are surfaced — individual files (`.env`, lockfiles, configs) are never shown
- ✅ Trigger files (`package.json`, `Cargo.toml`, etc.) are always excluded
- ✅ Danger zone items can never appear, even if gitignored

---

## ⚠️ The Safety Philosophy

Kessler is built with **developer trust** as its core tenet:

1. **Active Process Check:** It won't let you delete a project's `node_modules` while its dev server is still running.
2. **It respects Git:** If Git tracks a folder (even if named `bin`), Kessler won't touch it.
3. **It respects State:** It never targets files required to reproduce a build (like lockfiles) or environment secrets (like `.env`).
4. **It respects the OS:** Moving files to the Trash Bin instead of permanent deletion gives you an "Undo" button.


---

## 📈 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=hariharen9/kessler&type=Date)](https://star-history.com/#hariharen9/kessler&Date)

---

## 📝 License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.

---

<p align="center">
  Built with ❤️ by <b><a href="https://hariharen.site">Hariharen</a></b>
  <br><br>
  <a href="https://www.buymeacoffee.com/hariharen">
    <img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" width="160">
  </a>
</p>
