#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Build Script
# Usage: bash scripts/build.sh v1.0.0

VERSION="${1:-}"
if [[ -z "$VERSION" ]] || ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Usage: $0 <version>  (e.g. v1.0.0)"
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"

echo "=== BOS3000 Build $VERSION ==="

# ---- Step 1: Update frontend version ----
SEMVER="${VERSION#v}"
echo "$SEMVER" > "$ROOT/.version"

for app in admin portal; do
  VERSION_FILE="$ROOT/$app/src/lib/version.ts"
  if [[ -f "$VERSION_FILE" ]]; then
    sed -i.bak "s/APP_VERSION = '.*'/APP_VERSION = '${SEMVER}'/" "$VERSION_FILE"
    rm -f "${VERSION_FILE}.bak"
    echo "  Updated $app version to $SEMVER"
  fi
done

# ---- Step 2: Build frontends ----
for app in admin portal; do
  echo "  Building $app frontend..."
  (cd "$ROOT/$app" && npm ci --silent && npm run build)
done

echo "  Frontend assets embedded in gateway/{admin,portal}/"

# ---- Step 3: Build Go binary via Encore ----
echo "  Building Go binary..."

# Check if encore CLI is available
if command -v encore &>/dev/null; then
  encore build docker "bos3000:${VERSION}" \
    --base ubuntu:22.04 2>/dev/null && \
    docker save "bos3000:${VERSION}" | gzip > "$DIST/bos3000-${VERSION}-docker.tar.gz" && \
    echo "  Docker image saved to dist/bos3000-${VERSION}-docker.tar.gz"
else
  echo "  WARNING: encore CLI not found, skipping docker build"
  echo "  Install Encore: curl -L https://encore.dev/install.sh | bash"
fi

# ---- Step 4: Package deployment bundle ----
mkdir -p "$DIST"
BUNDLE="$DIST/bos3000-${VERSION}.tar.gz"

tar czf "$BUNDLE" \
  --exclude='node_modules' \
  --exclude='.git' \
  --exclude='dist' \
  --exclude='*_test.go' \
  --exclude='_seed_hash.go' \
  --exclude='.ralph-tui' \
  --exclude='.beads' \
  --exclude='.playwright-mcp' \
  --exclude='admin-dashboard.png' \
  -C "$ROOT" \
  deploy/ \
  scripts/reset-admin-password.sh \
  .version

echo ""
echo "=== Build Complete ==="
echo "  Version:  $VERSION"
echo "  Bundle:   $BUNDLE"
echo "  Deploy:   Extract on server, then run: sudo bash deploy/install.sh --version $VERSION"
