# 🛰️ Kessler CLI

> Intelligent CLI tool to find and safely clean build artifacts & caches

Kessler finds and sweeps away `node_modules`, `__pycache__`, `target/`, `dist/`, `.venv`, and dozens more — without ever touching your source code.

## Install

```bash
npm install -g kessler-cli
```

## Usage

```bash
# Launch interactive TUI
kessler ~/Projects

# Scan and report (no deletion)
kessler scan ~/Projects --json

# Clean with confirmation
kessler clean ~/Projects

# Deep clean + force (CI-friendly)
kessler clean ~/Projects --deep --force --older-than 30d
```

## Features

- 🏎️ **Blazingly fast** — Written in Go, scans massive directory trees in milliseconds
- 🧠 **Context-aware** — Identifies project types via trigger files, only targets known safe artifacts
- 🔍 **`.gitignore` intelligence** — Discovers ignored directories unique to your project
- 🛡️ **Git safety net** — Never touches files tracked by version control
- ♻️ **OS trash integration** — Moves to Trash instead of `rm -rf`
- 🎨 **Beautiful TUI** — Interactive dashboard with live telemetry

## Other Install Methods

- **Homebrew:** `brew tap hariharen9/tap && brew install kessler`
- **Scoop (Windows):** `scoop install kessler`
- **Go:** `go install github.com/hariharen9/kessler@latest`
- **Binary:** [GitHub Releases](https://github.com/hariharen9/kessler/releases)

## Links

- [GitHub](https://github.com/hariharen9/kessler)
- [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=hariharen.kessler-vscode)
- [Full Documentation](https://github.com/hariharen9/kessler#readme)
