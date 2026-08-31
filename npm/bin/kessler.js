#!/usr/bin/env node

/**
 * Cross-platform shim for kessler.
 * The postinstall script downloads the native binary into this same `bin/`
 * directory as `kessler` (Unix) or `kessler.exe` (Windows).
 * npm registers THIS file as the `kessler` CLI entry point, which works on
 * every platform without needing a `.exe` extension in package.json "bin".
 */

const { spawnSync } = require("child_process");
const { join } = require("path");

const binaryName = process.platform === "win32" ? "kessler.exe" : "kessler";
const binaryPath = join(__dirname, binaryName);

const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: "inherit",
    windowsHide: false,
});

if (result.error) {
    if (result.error.code === "ENOENT") {
        console.error(
            `[kessler-cli] Binary not found at: ${binaryPath}\n` +
            `Try reinstalling: npm install -g kessler-cli`
        );
    } else {
        console.error(`[kessler-cli] Failed to run kessler: ${result.error.message}`);
    }
    process.exit(1);
}

process.exit(result.status ?? 0);
