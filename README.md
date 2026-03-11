# 🛰️ Kessler

<p align="center">
  <img src="https://img.shields.io/github/v/release/hariharen9/kessler?style=for-the-badge&color=00ADD8&logo=go&logoColor=white" alt="Release">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge" alt="License"></a>
  <img src="https://img.shields.io/github/stars/hariharen9/kessler?style=for-the-badge&color=E3B341&logo=github&logoColor=white" alt="Stars">
  <a href="https://marketplace.visualstudio.com/items?itemName=hariharen.kessler-vscode"><img src="https://img.shields.io/visual-studio-marketplace/v/hariharen.kessler-vscode?style=for-the-badge&logo=visual-studio-code&color=007ACC&logoColor=white&label=VS%20Code%20Extension" alt="VS Code Extension"></a>
  <img src="https://img.shields.io/github/actions/workflow/status/hariharen9/kessler/release.yml?style=for-the-badge&logo=github-actions&logoColor=white" alt="Build Status">
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-333333?style=for-the-badge&logo=apple&logoColor=white" alt="Platform Support">
</p>

> **Kessler Syndrome** (noun): A theoretical scenario in which the density of objects in low Earth orbit is high enough that collisions between objects could cause a cascade, generating space debris that increases the likelihood of further collisions, rendering space exploration impossible.

**For developers, your hard drive is low Earth orbit.** 

Over time, it gets clogged with `node_modules`, `targets`, stray `build/` folders, forgotten Python virtual environments, and intermediate Rust targets. This **digital space debris** silently consumes hundreds of gigabytes until your system grinds to a halt.

**Kessler** is an intelligent, blazingly fast, and incredibly safe command-line tool built in Go that clears the orbit. It finds, calculates, and safely sweeps away runtime artifacts and build caches without ever touching your source code.

---

## ✨ Features

- 🏎️ **Blazingly Fast:** Scans massive directory trees concurrently using a high-performance Go worker pool.
- 🔍 **Context-Aware Engine:** Targets known safe artifacts based on project triggers (e.g., `package.json`, `Cargo.toml`).
- 🛡️ **Git Safety Net:** Silently queries `git ls-files` to guarantee actively tracked files are never deleted.
- 🌍 **Global Cache Management:** Safely cleans system-level caches (Docker, Homebrew, npm, Cargo, Go modules).
- ♻️ **OS Trash Integration:** Moves debris to your native OS Trash/Recycle Bin instead of a permanent `rm -rf`.
- 🛡️ **Active Project Protection:** Warns you before cleaning if a project's dev server is currently running.
- 🧪 **Environmental Doctor:** Identifies and cleans unused versions of toolchains (Node.js, Rust, Python, Ruby, Java, etc.).
- 🚀 **Project Launchpad:** A built-in navigator to fuzzy-search and instantly open projects in VS Code, Cursor, or Terminal.
- 🤖 **Background Daemon:** Runs silently to automatically sweep stale debris over 1GB weekly.
- 🌐 **Community Rules:** Dynamically updates and merges crowd-sourced cleanup rules via `kessler rules update`.
- 📜 **Scan History:** Tracks previous sweeps and total space freed to monitor your disk's health over time.
- 🤖 **CI / Scripting Mode:** Non-interactive `scan` and `clean` subcommands with JSON output, dry-run, and filtering.
- 🎨 **Beautiful TUI & Telemetry:** An interactive Charmbracelet dashboard with 4 tabbed views and live "Orbital Telemetry".

---

## ⌨️ Kessler for VS Code

Bring the full power of Kessler directly into your editor. [**Kessler for VS Code**](https://marketplace.visualstudio.com/items?itemName=hariharen.kessler-vscode) is a blazingly fast, lightweight extension that lives in your status bar, giving you real-time telemetry and one-click cleanup without ever leaving your workspace.

- **📡 Real-time Telemetry:** Live debris weight tracking in your status bar.
- **🔄 Auto-Pilot:** Automatically cleans build caches on Git branch switches.
- **⚡ Zero-Config:** Intelligent, ecosystem-aware scanning out of the box.

[**Install from Marketplace →**](https://marketplace.visualstudio.com/items?itemName=hariharen.kessler-vscode)

---

## 🚀 Installation
<table width="100%">
  <tr>
    <td width="25%">🍺 <b>Homebrew</b><br><sup>macOS & Linux</sup></td>
    <td>

```bash
brew tap hariharen9/tap
brew install kessler
```
</td>
  </tr>
  <tr>
    <td>🪟 <b>Scoop</b><br><sup>Windows</sup></td>
    <td>

```powershell
scoop bucket add hariharen9 https://github.com/hariharen9/scoop-bucket
scoop install kessler
```
</td>
  </tr>
  <tr>
    <td>📦 <b>npm</b><br><sup>All Platforms</sup></td>
    <td>

```bash
npm install -g kessler-cli
# Or run directly without installing:
npx kessler-cli ~/Projects
```
</td>
  </tr>
  <tr>
    <td>🐹 <b>Go</b><br><sup>From Source</sup></td>
    <td>

```bash
go install github.com/hariharen9/kessler@latest
```
</td>
  </tr>
  <tr>
    <td>📦 <b>Debian / Ubuntu</b><br><sup>.deb Package</sup></td>
    <td>Download the <code>.deb</code> from the <a href="https://github.com/hariharen9/kessler/releases/latest">latest release</a>, then:<br>

```bash
sudo dpkg -i kessler_*.deb
```
</td>
  </tr>
</table>
---

## 🎮 Usage

<details open>
<summary><b>🕹️ Interactive TUI (Default)</b></summary>

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

</details>

<details>
<summary><b>🤖 CLI Subcommands</b></summary>

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

#### `kessler daemon`

Kessler can run silently in the background to monitor your system. It scans once a week and automatically sweeps away more than 1GB of stale debris (older than 10 days) in safe mode.

```bash
kessler daemon --start                           # Install and start the background daemon
kessler daemon --status                          # Show current schedule status
kessler daemon --stop                            # Uninstall the daemon
```

#### `kessler rules update`

Fetch the latest community-provided project cleanup rules from GitHub and merge them locally.

```bash
kessler rules update
```

</details>

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
Your custom rules will automatically merge with the default rules engine and the community rules.

### Community Rules

Kessler maintains a set of community-driven rules that are updated independently from the binary. You can fetch the latest community rules at any time:

```bash
kessler rules update
```
These rules are stored in `community-rules.yaml` in your config directory and are automatically applied during scans.

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

## 🛰️ Kessler vs. The Alternatives

While tools like **Kondo**, **npkill**, or ecosystem-specific commands like `cargo clean` are great for basic cleaning, Kessler is engineered as a **Safety-First, Polyglot Disk Management Dashboard**.

| Feature | cargo clean | npkill | Kondo | **Kessler** |
|:---|:---:|:---:|:---:|:---:|
| **Ecosystem Support** | Single (Rust) | Single (Node) | Polyglot | ✅ **Polyglot (30+)** |
| **The Git Safety Net** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (`git ls-files`) |
| **Active Project Protection**| ❌ No | ❌ No | ❌ No | ✅ **Yes** (Warns on running PIDs) |
| **Environmental Doctor** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (Detects unused toolchains) |
| **Global Cache Cleaning** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (npm, docker, homebrew) |
| **Tiered Cleaning** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (Safe vs. Deep mode) |
| **`.gitignore` Scanning** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (Finds hidden ignored dirs) |
| **High-Fidelity TUI** | ❌ No | ✅ Yes | ⚠️ Basic | ✅ **Yes** (Modern Bubble Tea) |
| **OS Trash Integration** | ❌ No | ❌ No | ✅ Yes | ✅ **Yes** (+ Windows support) |
| **Project Launchpad** | ❌ No | ❌ No | ❌ No | ✅ **Yes** (Fuzzy-search & Open) |

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
