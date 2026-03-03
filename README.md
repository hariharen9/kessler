# 🛰️ Kessler

> **Kessler Syndrome** (noun): A theoretical scenario in which the density of objects in low Earth orbit is high enough that collisions between objects could cause a cascade, generating space debris that increases the likelihood of further collisions, rendering space exploration impossible.

For developers, your hard drive is low Earth orbit. 

Over time, it gets clogged with `node_modules`, stray `build/` folders, forgotten Python virtual environments, and intermediate Rust targets. This **digital space debris** silently consumes hundreds of gigabytes of storage until your system grinds to a halt.

**Kessler** is an intelligent, blazingly fast, and incredibly safe command-line tool built in Go that clears the orbit. It finds, calculates, and safely sweeps away runtime artifacts and build caches without ever touching your source code.

---

## ✨ Features

- 🏎️ **Blazingly Fast:** Written in Go, it scans massive directory trees concurrently in milliseconds.
- 🧠 **Context-Aware Engine:** Doesn't just blindly delete folders. It looks for triggers (e.g., `package.json`, `Cargo.toml`) to identify project types and *only* targets known safe artifacts for that specific ecosystem.
- 🛡️ **The Git Safety Net:** Before Kessler flags *any* folder as junk, it silently queries Git (`git ls-files`). If a folder contains files actively tracked by version control, Kessler immediately aborts and ignores it.
- ♻️ **OS Trash Integration:** Mistakes happen. Instead of using a terrifying `rm -rf`, Kessler safely moves debris to your native OS Trash/Recycle Bin (supports macOS, Windows, and Linux), giving you an "Undo" button.
- 🎨 **Beautiful TUI & Telemetry:** Powered by Charmbracelet's Bubble Tea. Features an interactive dashboard with live "Orbital Telemetry," ecosystem icons, root drive usage, and visual space tracking.
- 🤖 **CI / Scripting Mode:** Use `kessler scan` and `kessler clean` subcommands for non-interactive usage in cron jobs, CI pipelines, and shell scripts. Supports JSON output, dry-run, and filtering by size and age.
- 🔧 **Custom User Rules:** Extend the built-in rules engine with your own `~/.config/kessler/rules.yaml` — add new ecosystems or extra targets without forking.

---

## 🚀 Installation

You can install Kessler directly using Go:

```bash
go install github.com/hariharen/kessler@latest
```

Or clone and build it manually:

```bash
git clone https://github.com/hariharen/kessler.git
cd kessler
go build -o kessler
sudo mv kessler /usr/local/bin/
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
| `t` | Toggle Safe ↔ Deep mode |
| `s` | Sort by Size ↔ Name |
| `/` | Search projects |
| `Enter` | Move selected to Trash |
| `X` | Permanently delete |
| `q` | Quit |

### Non-Interactive / CI Mode

#### `kessler scan`

Scan and report — no deletion.

```bash
kessler scan ~/Projects                          # Table output
kessler scan ~/Projects --json                   # JSON output (pipe to jq)
kessler scan ~/Projects --deep --older-than 30d  # Stale projects with builds
kessler scan ~/Projects --min-size 100MB         # Only large projects
kessler scan ~/Projects --sort name              # Sort alphabetically
```

#### `kessler clean`

Scan and clean — shows a preview and asks for confirmation.

```bash
kessler clean ~/Projects                         # Preview + confirm
kessler clean ~/Projects --deep                  # Deep clean (extra warning)
kessler clean ~/Projects --force                 # Skip confirmation
kessler clean ~/Projects --dry-run               # Preview only, no deletion
kessler clean ~/Projects --permanent             # rm -rf instead of trash
kessler clean ~/Projects --older-than 30d --force  # Cron job friendly
```

---

## ⚙️ How the Rules Engine Works

Kessler is powered by a dynamic rules engine (`rules.yaml`). It doesn't use hardcoded `if/else` statements. 

When Kessler enters a directory, it looks for **Trigger Files**. If it finds `package.json`, it knows it's dealing with a Node.js project, and only then will it hunt for `node_modules` or `.next` folders.

Current out-of-the-box support includes:
- **Node.js:** `node_modules`, `dist`, `build`, `.next`, `.nuxt`, `.svelte-kit`, `coverage`
- **Python:** `__pycache__`, `venv`, `.venv`, `.pytest_cache`, `.mypy_cache`, `wandb`
- **Rust:** `target`
- **Go:** `vendor`
- **Java / JVM:** `target`, `build`, `.gradle`
- **PHP:** `vendor`
- **Ruby:** `vendor/bundle`, `.bundle`
- **.NET / C#:** `bin`, `obj`, `packages`
- **Elixir:** `deps`, `_build`
- **Terraform / IaC:** `.terraform`, `cdk.out`, `.serverless`
- **OS & Editor:** `.DS_Store`, `Thumbs.db`, `.idea`, `.vscode`

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
      - path: "DerivedData"
        tier: "deep"

  # Add extra targets to an existing ecosystem (merged by name)
  - name: "Node.js / JS Ecosystem"
    targets:
      - path: ".cache"
        tier: "safe"

# Add extra items to the danger zone (never deletable)
danger_zone:
  - "secrets.json"
```

User rules are **merged** with the defaults — matching rule names get their targets appended, new rules are added, and danger zone entries are unioned.

---

## ⚠️ The Safety Philosophy (Why Kessler is Different)

There are other tools that delete `node_modules`. Kessler is built with **developer trust** as its core tenet:

1. **It respects Git:** A folder named `vendor/` might be a junk cache in one project, but actively committed source code in another. If Git tracks it, Kessler won't touch it.
2. **It respects State:** It never targets files required to reproduce a build (like `package-lock.json` or `Cargo.lock`) or environment secrets (like `.env`).
3. **It respects the OS:** By moving files to the Trash Bin instead of permanent deletion, a mistaken sweep is an easy fix, not a catastrophic data loss event. It will safely prompt you if cross-drive trashing fails.

---

## 🤝 Contributing

Contributions are welcome! If you want to add new ecosystem rules (e.g., Elixir, C#, Swift) or improve the TUI, feel free to open a Pull Request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.
