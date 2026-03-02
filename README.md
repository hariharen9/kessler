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
- 🎨 **Beautiful TUI & Telemetry:** Powered by Charmbracelet's Bubble Tea. Features an interactive dashboard with live "Orbital Telemetry," displaying your root drive usage, ecosystem breakdown, and visual space tracking.

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

Run Kessler by passing the directory you want to scan (defaults to the current directory).

```bash
# Scan your entire Projects folder
kessler ~/Projects

# Scan the current directory
kessler .
```

### The Interface
1. **Wait** a fraction of a second while Kessler analyzes the directory tree and verifies Git statuses.
2. **Review** the interactive dashboard showing all discovered projects, telemetry data, and exact byte-sizes.
3. **Filter & Sort** using `/` to search, `s` to sort by size/name, or `t` to toggle between Safe and Deep Clean modes.
4. **Select** the projects you want to clean using `Spacebar` (or `a` to select all). 
5. **Vaporize** the space junk by hitting `Enter` (Move to Trash) or `X` (Permanently Nuke).

---

## ⚙️ How the Rules Engine Works

Kessler is powered by a dynamic rules engine (`rules.yaml`). It doesn't use hardcoded `if/else` statements. 

When Kessler enters a directory, it looks for **Trigger Files**. If it finds `package.json`, it knows it's dealing with a Node.js project, and only then will it hunt for `node_modules` or `.next` folders.

Current out-of-the-box support includes:
- **Node.js:** `node_modules`, `dist`, `build`, `.next`, `.pnp.cjs`
- **Python:** `__pycache__`, `venv`, `.venv`, `env`, `.pytest_cache`
- **Rust:** `target`
- **Java:** `target`, `build`, `.gradle`
- **Go:** `vendor`
- **macOS:** `.DS_Store`

*More ecosystems are easily supported by extending the `rules.yaml` file.*

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
