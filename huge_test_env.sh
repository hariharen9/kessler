#!/bin/bash

# ==========================================
# KESSLER SYNDROME GENERATOR - LOCK & KEY FIXED
# ==========================================
DEV_DIR="/Users/hariharen/personal/Projects_Kessler_Test"

echo "🧹 Sweeping old dev environment..."
rm -rf "$DEV_DIR"
mkdir -p "$DEV_DIR"

echo "🌍 Generating massive, realistic Developer Workspace at $DEV_DIR..."

# --- HELPER FUNCTIONS ---
random_date() {
    local rand=$RANDOM
    local year
    if [ $rand -lt 3276 ]; then year="2023"
    elif [ $rand -lt 9830 ]; then year="2024"
    elif [ $rand -lt 22937 ]; then year="2025"
    else year="2026"; fi
    
    local month=$(printf "%02d" $(( (RANDOM % 12) + 1 )))
    local day=$(printf "%02d" $(( (RANDOM % 28) + 1 )))
    local hour=$(printf "%02d" $(( RANDOM % 24 )))
    local min=$(printf "%02d" $(( RANDOM % 60 )))
    
    echo "${year}${month}${day}${hour}${min}"
}

make_file() {
    local filepath="$1"
    touch "$filepath"
    touch -t $(random_date) "$filepath"
}

make_heavy_file() {
    local size=$(( $1 * 1048576 ))
    local filepath="$2"
    head -c "$size" /dev/zero > "$filepath" 2>/dev/null || dd if=/dev/zero of="$filepath" bs=1048576 count="$1" 2>/dev/null
    touch -t $(random_date) "$filepath"
}

# ==========================================
# 📂 0. THE ROOT CHAOS
# ==========================================
echo "   -> Scattering root-level junk files..."
ROOT_DIR="$DEV_DIR/00_Root_Chaos"
mkdir -p "$ROOT_DIR"

for i in {1..20}; do make_file "$ROOT_DIR/npm-debug-$i.log"; done
for i in {1..10}; do make_file "$ROOT_DIR/backup_project_v$i.zip"; done
for i in {1..5}; do make_heavy_file 10 "$ROOT_DIR/docker_export_v$i.tar"; done
make_file "$ROOT_DIR/.DS_Store"
make_file "$ROOT_DIR/.env"

# ==========================================
# 🏢 1. WORK MONOREPO (Node, React, Next.js)
# ==========================================
echo "   -> Generating Work Monorepo (Trigger: package.json)..."
MONO="$DEV_DIR/01_Work_Monorepo"
mkdir -p "$MONO/frontend/node_modules"
mkdir -p "$MONO/frontend/.next/cache/webpack"
mkdir -p "$MONO/backend/node_modules"

# TRIGGERS & DANGER
make_file "$MONO/yarn.lock"
make_file "$MONO/frontend/package.json"
make_file "$MONO/backend/package.json"
make_file "$MONO/backend/.env.production"

# ARTIFACTS
for i in {1..100}; do 
    mkdir -p "$MONO/frontend/node_modules/fake-lib-$i/dist"
    make_file "$MONO/frontend/node_modules/fake-lib-$i/dist/index.js"
done
make_heavy_file 50 "$MONO/frontend/node_modules/heavy_binary.node"
for i in {1..30}; do make_file "$MONO/frontend/.next/cache/webpack/chunk_$i.pack"; done
make_heavy_file 25 "$MONO/frontend/.next/cache/webpack/huge_cache.pack"

# ==========================================
# 🐍 2. DATA SCIENCE & MACHINE LEARNING
# ==========================================
echo "   -> Generating ML Workspace (Trigger: requirements.txt)..."
ML_DIR="$DEV_DIR/02_Data_Science_ML"
mkdir -p "$ML_DIR/.venv/lib/python3.10/site-packages"
mkdir -p "$ML_DIR/__pycache__"
mkdir -p "$ML_DIR/.ipynb_checkpoints"
mkdir -p "$ML_DIR/models"

# TRIGGERS & DANGER
make_file "$ML_DIR/requirements.txt"
make_file "$ML_DIR/poetry.lock"
make_file "$ML_DIR/local_training.sqlite3"

# ARTIFACTS
for i in {1..40}; do make_file "$ML_DIR/__pycache__/script_$i.cpython-310.pyc"; done
for i in {1..20}; do make_file "$ML_DIR/.ipynb_checkpoints/notebook_$i.ipynb"; done
for i in {1..10}; do make_heavy_file 30 "$ML_DIR/models/epoch_$i.safetensors"; done
make_heavy_file 40 "$ML_DIR/.venv/lib/python3.10/site-packages/torch_heavy.so"

# ==========================================
# ☁️ 3. INFRASTRUCTURE & BACKEND (Terraform, Go, Rust)
# ==========================================
echo "   -> Generating Cloud & Systems (Triggers: main.tf, Cargo.toml, go.mod)..."
SYS_DIR="$DEV_DIR/03_Systems_And_Cloud"

# Terraform
mkdir -p "$SYS_DIR/aws-infra/.terraform/providers/registry.terraform.io/hashicorp/aws/5.0.0/darwin_arm64"
make_file "$SYS_DIR/aws-infra/main.tf"
make_file "$SYS_DIR/aws-infra/terraform.tfstate"
make_file "$SYS_DIR/aws-infra/terraform.tfstate.backup"
for i in {1..10}; do make_heavy_file 5 "$SYS_DIR/aws-infra/.terraform/providers/aws_plugin_$i.exe"; done

# Rust
mkdir -p "$SYS_DIR/rust_microservice/target/debug/deps"
make_file "$SYS_DIR/rust_microservice/Cargo.toml"
make_file "$SYS_DIR/rust_microservice/Cargo.lock"
for i in {1..50}; do make_file "$SYS_DIR/rust_microservice/target/debug/deps/lib_chunk_$i.rlib"; done
make_heavy_file 80 "$SYS_DIR/rust_microservice/target/debug/rusty_app"

# Go
mkdir -p "$SYS_DIR/go_api/vendor/github.com/gin-gonic"
make_file "$SYS_DIR/go_api/go.mod"
make_file "$SYS_DIR/go_api/go.sum"
make_heavy_file 25 "$SYS_DIR/go_api/compiled_api_mac.bin"

# ==========================================
# ☕ 4. ENTERPRISE JAVA & C# (.NET)
# ==========================================
echo "   -> Generating Enterprise Graveyard (Triggers: pom.xml, *.csproj)..."
ENT_DIR="$DEV_DIR/04_Enterprise_Graveyard"

# Java
mkdir -p "$ENT_DIR/LegacyBankingApp/target/classes/com/bank/core"
mkdir -p "$ENT_DIR/LegacyBankingApp/.gradle/caches"
make_file "$ENT_DIR/LegacyBankingApp/pom.xml"
for i in {1..30}; do make_file "$ENT_DIR/LegacyBankingApp/target/classes/com/bank/core/Service$i.class"; done
make_heavy_file 60 "$ENT_DIR/LegacyBankingApp/target/BankingApp-1.0-SNAPSHOT.jar"

# C# / .NET
mkdir -p "$ENT_DIR/WindowsWPFApp/bin/Debug/net8.0"
mkdir -p "$ENT_DIR/WindowsWPFApp/obj/Debug/net8.0"
make_file "$ENT_DIR/WindowsWPFApp/WindowsWPFApp.csproj"
make_file "$ENT_DIR/WindowsWPFApp/appsettings.json"
for i in {1..20}; do make_file "$ENT_DIR/WindowsWPFApp/obj/Debug/net8.0/module_$i.g.cs"; done
make_heavy_file 45 "$ENT_DIR/WindowsWPFApp/bin/Debug/net8.0/WindowsWPFApp.dll"

# ==========================================
# 📱 5. MOBILE APP CEMETERY (iOS, Android, Flutter)
# ==========================================
echo "   -> Generating Mobile Cemetery (Triggers: pubspec.yaml, Podfile)..."
MOB_DIR="$DEV_DIR/05_Mobile_Cemetery"

# Flutter/Android
mkdir -p "$MOB_DIR/flutter_social/.dart_tool/flutter_build"
mkdir -p "$MOB_DIR/flutter_social/build/app/outputs/flutter-apk"
make_file "$MOB_DIR/flutter_social/pubspec.yaml"
make_file "$MOB_DIR/flutter_social/pubspec.lock"
for i in {1..15}; do make_file "$MOB_DIR/flutter_social/.dart_tool/flutter_build/kernel_snapshot_$i.bin"; done
make_heavy_file 30 "$MOB_DIR/flutter_social/build/app/outputs/flutter-apk/app-release.apk"

# iOS / Xcode
mkdir -p "$MOB_DIR/ios_swift_app/DerivedData/Build/Products/Debug-iphonesimulator"
make_file "$MOB_DIR/ios_swift_app/Podfile"
make_file "$MOB_DIR/ios_swift_app/Podfile.lock"
make_heavy_file 100 "$MOB_DIR/ios_swift_app/DerivedData/Build/Products/Debug-iphonesimulator/SwiftApp.app"

# ==========================================
# 🏁 FINALIZE
# ==========================================
echo ""
echo "✅ CORRECTED KESSLER TORTURE TEST DEPLOYED AT: $DEV_DIR"
echo "📊 Total files generated: $(find "$DEV_DIR" -type f | wc -l)"
echo "💾 Total bloat weight: $(du -sh "$DEV_DIR" | awk '{print $1}')"