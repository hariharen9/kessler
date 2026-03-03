# #!/bin/bash

# # Setup base directory
# TEST_DIR="$HOME/personal/kessler_test_env"

# echo "🧹 Clearing the launchpad..."
# rm -rf "$TEST_DIR"
# mkdir -p "$TEST_DIR"

# echo "☄️ Generating Kessler Syndrome Torture Test at $TEST_DIR..."

# # Helper to create a dummy file with size (in MB)
# make_file() {
#     local size=$(( $1 * 1048576 ))
#     head -c "$size" /dev/zero > "$2" 2>/dev/null || dd if=/dev/zero of="$2" bs=1048576 count="$1" 2>/dev/null
# }

# # --- 1. DEEP NESTED MONOREPO (Node.js) ---
# echo "   -> [Tier 1 & Danger] Creating Node Monorepo..."
# MONO_DIR="$TEST_DIR/company_monorepo"
# mkdir -p "$MONO_DIR/apps/web_frontend/src/components"
# mkdir -p "$MONO_DIR/apps/web_frontend/node_modules/react"
# mkdir -p "$MONO_DIR/apps/web_frontend/.next/server"
# mkdir -p "$MONO_DIR/services/core/api_v1/src/routes"
# mkdir -p "$MONO_DIR/services/core/api_v1/dist/routes"

# touch "$MONO_DIR/pnpm-workspace.yaml"
# touch "$MONO_DIR/pnpm-lock.yaml" # DANGER: Do Not Delete
# touch "$MONO_DIR/apps/web_frontend/package.json"
# touch "$MONO_DIR/services/core/api_v1/package.json"
# echo "SECRET=xyz" > "$MONO_DIR/services/core/api_v1/.env.production" # DANGER: Do Not Delete
# make_file 10 "$MONO_DIR/apps/web_frontend/node_modules/react/index.js" # DELETE
# make_file 5 "$MONO_DIR/apps/web_frontend/.next/server/chunk.js" # DELETE
# make_file 15 "$MONO_DIR/services/core/api_v1/dist/bundle.js" # DELETE

# # --- 2. PYTHON & ML DATASCIENCE ---
# echo "   -> [Tier 1, 2 & Maybe] Creating Python/ML Workspace..."
# PY_DIR="$TEST_DIR/ml_pipeline"
# mkdir -p "$PY_DIR/app/models"
# mkdir -p "$PY_DIR/tests"
# mkdir -p "$PY_DIR/venv/lib"
# mkdir -p "$PY_DIR/app/__pycache__"
# mkdir -p "$PY_DIR/.ipynb_checkpoints"

# touch "$PY_DIR/requirements.txt"
# touch "$PY_DIR/uv.lock" # DANGER: Do Not Delete
# echo "Fake DB data" > "$PY_DIR/app/local.sqlite3" # DANGER: Do Not Delete
# make_file 8 "$PY_DIR/venv/lib/heavy.so" # DELETE
# make_file 1 "$PY_DIR/app/__pycache__/main.cpython.pyc" # DELETE
# make_file 25 "$PY_DIR/app/models/weights.pt" # MAYBE: Prompt user
# echo "Jupyter data" > "$PY_DIR/.ipynb_checkpoints/draft.ipynb" # DELETE

# # --- 3. INFRASTRUCTURE AS CODE (Terraform) ---
# echo "   -> [Tier 1 & Danger] Creating Terraform Infrastructure..."
# IAC_DIR="$TEST_DIR/cloud_infra"
# mkdir -p "$IAC_DIR/.terraform/providers"

# touch "$IAC_DIR/main.tf"
# echo "CRITICAL STATE" > "$IAC_DIR/terraform.tfstate" # DANGER: Never delete
# echo "BACKUP STATE" > "$IAC_DIR/terraform.tfstate.backup" # DANGER: Never delete
# make_file 10 "$IAC_DIR/.terraform/providers/aws_plugin.exe" # DELETE

# # --- 4. JAVA ENTERPRISE PROJECT ---
# echo "   -> [Tier 1] Creating Java Spring Boot App..."
# JAVA_DIR="$TEST_DIR/java_backend"
# mkdir -p "$JAVA_DIR/src/main/java"
# mkdir -p "$JAVA_DIR/target/classes"

# touch "$JAVA_DIR/pom.xml"
# make_file 5 "$JAVA_DIR/target/classes/App.class" # DELETE
# make_file 20 "$JAVA_DIR/target/app-release.jar" # MAYBE: Prompt user

# # --- 5. THE "FAKE OUT" DIRECTORY ---
# echo "   -> [Guardrail] Creating Fake Out Directory (Should be ignored)..."
# FAKE_DIR="$TEST_DIR/not_a_project_just_files"
# mkdir -p "$FAKE_DIR/dist"
# mkdir -p "$FAKE_DIR/build"
# mkdir -p "$FAKE_DIR/node_modules"

# echo "No trigger files here" > "$FAKE_DIR/dist/readme.txt"
# echo "Should not be deleted" > "$FAKE_DIR/node_modules/notes.txt"

# # --- 6. RUST PROJECT WITH MIXED TRACKING ---
# echo "   -> [Git Guardrail] Creating Rust Project with mixed Git tracking..."
# RUST_DIR="$TEST_DIR/rust_service"
# mkdir -p "$RUST_DIR/src"
# mkdir -p "$RUST_DIR/target/debug"
# mkdir -p "$RUST_DIR/target/release"

# touch "$RUST_DIR/Cargo.toml"
# touch "$RUST_DIR/Cargo.lock" # DANGER: Do Not Delete
# make_file 12 "$RUST_DIR/target/debug/app.exe" # DELETE

# echo "Intentionally committed" > "$RUST_DIR/target/release/custom_binary.txt"

# cd "$RUST_DIR" || exit
# git init -q
# git add Cargo.toml Cargo.lock src/ target/release/custom_binary.txt
# git commit -q -m "Initial commit"
# cd - > /dev/null || exit

# # --- 7. SCATTERED OS JUNK & LOGS ---
# echo "   -> [Tier 1] Scattering OS Junk and Crash Logs..."
# touch "$TEST_DIR/.DS_Store"
# touch "$MONO_DIR/.DS_Store"
# touch "$JAVA_DIR/Thumbs.db"
# touch "$PY_DIR/npm-debug.log"
# mkdir -p "$PY_DIR/.idea"
# echo "IDE settings" > "$PY_DIR/.idea/workspace.xml" # DELETE

# echo "✅ Orbit is heavily polluted. Kessler is ready to fire."

#!/bin/bash

# Setup base directory strictly to the requested path
DEV_DIR="/Users/hariharen/personal/Projects_Kessler_Test"

echo "🧹 Sweeping old dev environment..."
rm -rf "$DEV_DIR"
mkdir -p "$DEV_DIR"

echo "🌍 Generating a highly realistic, chaotic Developer Workspace at $DEV_DIR..."

# Helper to create a dummy file with size (in MB)
make_file() {
    local size=$(( $1 * 1048576 ))
    head -c "$size" /dev/zero > "$2" 2>/dev/null || dd if=/dev/zero of="$2" bs=1048576 count="$1" 2>/dev/null
}

# ==========================================
# 📂 THE ROOT FOLDER CHAOS
# ==========================================
echo "   -> Scattering root-level junk (logs, zips, temp files)..."
touch "$DEV_DIR/.DS_Store"
touch "$DEV_DIR/Thumbs.db"
touch "$DEV_DIR/test.js" # Random valid script
make_file 50 "$DEV_DIR/backup-2023_FINAL.zip" # MAYBE DELETE
make_file 2 "$DEV_DIR/npm-debug.log" # DELETE

# ==========================================
# 🏢 1. WORK PROJECTS (Heavy, complex, IaC, Monorepos)
# ==========================================
WORK_DIR="$DEV_DIR/work"
mkdir -p "$WORK_DIR"
echo "   -> [Work] Generating Corporate Monorepo..."

MONO="$WORK_DIR/acme-corp-monorepo"
mkdir -p "$MONO/frontend/node_modules/react"
mkdir -p "$MONO/frontend/.next/cache"
mkdir -p "$MONO/backend/venv/lib/python3.9/site-packages"
mkdir -p "$MONO/backend/__pycache__"

touch "$MONO/yarn.lock" # DANGER
touch "$MONO/backend/poetry.lock" # DANGER
echo "SECRET=prod_key_123" > "$MONO/backend/.env" # DANGER
make_file 15 "$MONO/frontend/node_modules/react/index.js" # DELETE
make_file 30 "$MONO/frontend/.next/cache/webpack.pack" # DELETE
make_file 20 "$MONO/backend/venv/lib/python3.9/site-packages/heavy.so" # DELETE

echo "   -> [Work] Generating Cloud Infrastructure..."
INFRA="$WORK_DIR/aws-infrastructure"
mkdir -p "$INFRA/.terraform/providers"
touch "$INFRA/main.tf"
echo "CRITICAL" > "$INFRA/terraform.tfstate" # DANGER
make_file 15 "$INFRA/.terraform/providers/aws_plugin" # DELETE

# ==========================================
# 🚀 2. SIDE PROJECTS (Mobile, Rust, Go)
# ==========================================
SIDE_DIR="$DEV_DIR/side_projects"
mkdir -p "$SIDE_DIR"

echo "   -> [Side] Generating Flutter/Mobile App..."
MOBILE="$SIDE_DIR/startup_idea_app"
mkdir -p "$MOBILE/.dart_tool/flutter_build"
mkdir -p "$MOBILE/build/ios/Debug-iphonesimulator"
mkdir -p "$MOBILE/ios/Pods"

touch "$MOBILE/pubspec.lock" # DANGER
make_file 40 "$MOBILE/build/ios/Debug-iphonesimulator/App.framework" # DELETE
make_file 10 "$MOBILE/.dart_tool/flutter_build/kernel.bin" # DELETE
make_file 25 "$MOBILE/app-release.apk" # MAYBE DELETE

echo "   -> [Side] Generating Rust Game Engine..."
RUST="$SIDE_DIR/rusty_engine"
mkdir -p "$RUST/target/release/deps"
touch "$RUST/Cargo.lock" # DANGER
make_file 80 "$RUST/target/release/deps/lib_heavy_render.rlib" # DELETE

echo "   -> [Side] Generating Go Microservice..."
GO_SVC="$SIDE_DIR/fast_api_go"
mkdir -p "$GO_SVC/vendor/github.com/gorilla/mux"
mkdir -p "$GO_SVC/bin"
touch "$GO_SVC/go.sum" # DANGER
make_file 12 "$GO_SVC/bin/fast_api_go_linux_amd64" # DELETE

# ==========================================
# 🪦 3. THE GRAVEYARD (Abandoned tutorials & tests)
# ==========================================
GRAVE="$DEV_DIR/graveyard"
mkdir -p "$GRAVE"

echo "   -> [Graveyard] Generating half-finished ML tutorial..."
ML="$GRAVE/pytorch_cats_vs_dogs"
mkdir -p "$ML/.ipynb_checkpoints"
mkdir -p "$ML/wandb/run-2023"
touch "$ML/requirements.txt"
make_file 1 "$ML/.ipynb_checkpoints/Untitled.ipynb" # DELETE
make_file 150 "$ML/model_epoch_50.safetensors" # MAYBE DELETE

echo "   -> [Graveyard] Generating forgotten React tutorial..."
VITE="$GRAVE/todo-app-vite-test"
mkdir -p "$VITE/node_modules"
mkdir -p "$VITE/dist"
touch "$VITE/package-lock.json" # DANGER
make_file 25 "$VITE/node_modules/bloat.js" # DELETE
make_file 5 "$VITE/dist/index.html" # DELETE

echo "   -> [Graveyard] Generating C++ OpenGL test..."
CPP="$GRAVE/cpp_triangle"
mkdir -p "$CPP/build/CMakeFiles"
touch "$CPP/CMakeLists.txt"
make_file 8 "$CPP/build/main.o" # DELETE
make_file 1 "$CPP/build/CMakeCache.txt" # DELETE

# ==========================================
# 📦 4. OPEN SOURCE CLONES (Heavy history, Java/Android)
# ==========================================
CLONES="$DEV_DIR/open_source"
mkdir -p "$CLONES"

echo "   -> [Clones] Generating massive cloned Java/Android project..."
JAVA="$CLONES/elasticsearch-fork"
mkdir -p "$JAVA/.gradle/caches"
mkdir -p "$JAVA/build/classes/java/main"
mkdir -p "$JAVA/target/surefire-reports"
mkdir -p "$JAVA/.idea"

touch "$JAVA/build.gradle"
make_file 60 "$JAVA/.gradle/caches/dependencies.jar" # DELETE
make_file 20 "$JAVA/build/classes/java/main/Core.class" # DELETE
make_file 2 "$JAVA/target/surefire-reports/TEST-Core.xml" # DELETE
echo "user settings" > "$JAVA/.idea/workspace.xml" # DELETE

# ==========================================
# 🏁 FINALIZE
# ==========================================
echo ""
echo "✅ Developer Workspace Created Successfully at: $DEV_DIR"
echo "📊 Run 'du -sh $DEV_DIR' to see the bloat."
echo "🚀 Kessler target locked and ready."