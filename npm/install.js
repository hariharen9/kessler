#!/usr/bin/env node

const { execFileSync } = require("child_process");
const { createWriteStream, mkdirSync, existsSync, unlinkSync } = require("fs");
const { get } = require("https");
const { join } = require("path");

const pkg = require("./package.json");
const VERSION = pkg.version;
const REPO = "hariharen9/kessler";

const PLATFORM_MAP = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
};

const ARCH_MAP = {
    x64: "amd64",
    arm64: "arm64",
};

function getBinaryName() {
    return process.platform === "win32" ? "kessler.exe" : "kessler";
}

function getDownloadUrl() {
    const platform = PLATFORM_MAP[process.platform];
    const arch = ARCH_MAP[process.arch];

    if (!platform || !arch) {
        console.error(`Unsupported platform: ${process.platform} ${process.arch}`);
        process.exit(1);
    }

    const ext = process.platform === "win32" ? "zip" : "tar.gz";
    return `https://github.com/${REPO}/releases/download/v${VERSION}/kessler_${VERSION}_${platform}_${arch}.${ext}`;
}

function download(url, dest) {
    return new Promise((resolve, reject) => {
        get(url, (res) => {
            if (res.statusCode === 302 || res.statusCode === 301) {
                return download(res.headers.location, dest).then(resolve).catch(reject);
            }
            if (res.statusCode !== 200) {
                return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
            }
            const file = createWriteStream(dest);
            res.pipe(file);
            file.on("finish", () => {
                file.close();
                resolve();
            });
            file.on("error", reject);
        }).on("error", reject);
    });
}

async function install() {
    const binDir = join(__dirname, "bin");
    const binaryPath = join(binDir, getBinaryName());

    if (existsSync(binaryPath)) {
        return; // Already installed
    }

    mkdirSync(binDir, { recursive: true });

    const url = getDownloadUrl();
    const isWindows = process.platform === "win32";
    const archiveExt = isWindows ? "zip" : "tar.gz";
    const tmpFile = join(binDir, `kessler.${archiveExt}`);

    console.log(`Downloading kessler v${VERSION}...`);

    try {
        await download(url, tmpFile);

        // Extract using system tar (available on macOS, Linux, and modern Windows)
        if (isWindows) {
            execFileSync("tar", ["-xf", tmpFile, "-C", binDir, getBinaryName()]);
        } else {
            execFileSync("tar", ["-xzf", tmpFile, "-C", binDir, getBinaryName()]);
        }

        unlinkSync(tmpFile);

        // Make binary executable on Unix
        if (!isWindows) {
            const { chmodSync } = require("fs");
            chmodSync(binaryPath, 0o755);
        }

        console.log(`kessler v${VERSION} installed successfully!`);
    } catch (err) {
        // Clean up temp file on failure
        if (existsSync(tmpFile)) {
            try { unlinkSync(tmpFile); } catch (_) { }
        }
        console.error(`Failed to install kessler: ${err.message}`);
        console.error(`You can manually download from: https://github.com/${REPO}/releases`);
        process.exit(1);
    }
}

install();
