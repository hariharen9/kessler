#!/usr/bin/env node

const { execFileSync } = require("child_process");
const { createWriteStream, mkdirSync, chmodSync, existsSync, unlinkSync } = require("fs");
const { get } = require("https");
const { join } = require("path");
const { createGunzip } = require("zlib");
const { Extract } = require("tar") || {};

const pkg = require("./package.json");
const VERSION = pkg.version;
const REPO = "hariharen/kessler";

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

function download(url) {
    return new Promise((resolve, reject) => {
        get(url, (res) => {
            if (res.statusCode === 302 || res.statusCode === 301) {
                return download(res.headers.location).then(resolve).catch(reject);
            }
            if (res.statusCode !== 200) {
                return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
            }
            resolve(res);
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
    console.log(`Downloading kessler v${VERSION}...`);

    try {
        const res = await download(url);

        if (process.platform === "win32") {
            // For Windows zip, download to temp and extract
            const tmpPath = join(binDir, "kessler.zip");
            const file = createWriteStream(tmpPath);
            await new Promise((resolve, reject) => {
                res.pipe(file);
                file.on("finish", resolve);
                file.on("error", reject);
            });
            // Use tar to extract zip (available on modern Windows)
            execFileSync("tar", ["-xf", tmpPath, "-C", binDir, getBinaryName()]);
            unlinkSync(tmpPath);
        } else {
            // For Unix tar.gz, pipe through gunzip and tar
            await new Promise((resolve, reject) => {
                const gunzip = createGunzip();
                const extractor = require("tar").extract({ cwd: binDir, strip: 0 });

                // Only extract the binary
                extractor.on("entry", (entry) => {
                    if (entry.path !== "kessler") {
                        entry.resume();
                    }
                });

                res.pipe(gunzip).pipe(extractor);
                extractor.on("finish", resolve);
                extractor.on("error", reject);
                gunzip.on("error", reject);
            });
        }

        chmodSync(binaryPath, 0o755);
        console.log(`kessler v${VERSION} installed successfully!`);
    } catch (err) {
        console.error(`Failed to install kessler: ${err.message}`);
        console.error(`You can manually download from: https://github.com/${REPO}/releases`);
        process.exit(1);
    }
}

install();
