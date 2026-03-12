#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Build Script — produces a single deployment tarball
# Usage: bash scripts/build.sh v1.0.0

VERSION="${1:-}"
if [[ -z "$VERSION" ]] || ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Usage: $0 <version>  (e.g. v1.0.0)"
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
STAGE="$DIST/stage"
SEMVER="${VERSION#v}"

echo "=== BOS3000 Build $VERSION ==="

# ---- Step 1: Update frontend version ----
echo "$SEMVER" > "$ROOT/.version"
for app in admin portal; do
  VERSION_FILE="$ROOT/$app/src/lib/version.ts"
  if [[ -f "$VERSION_FILE" ]]; then
    sed -i.bak "s/APP_VERSION = '.*'/APP_VERSION = '${SEMVER}'/" "$VERSION_FILE"
    rm -f "${VERSION_FILE}.bak"
    echo "  Updated $app version to $SEMVER"
  fi
done

# ---- Step 2: Build frontends → gateway/{admin,portal}/ ----
for app in admin portal; do
  echo "  Building $app frontend..."
  (cd "$ROOT/$app" && npm ci --silent && npm run build)
done
echo "  Frontend assets embedded in gateway/{admin,portal}/"

# ---- Step 3: Build binary via Encore docker (then extract) ----
echo "  Compiling linux/amd64 binary..."
if ! command -v encore &>/dev/null; then
  echo "  ERROR: encore CLI not found"
  echo "  Install: curl -L https://encore.dev/install.sh | bash"
  exit 1
fi

IMAGE_TAG="bos3000-build:${VERSION}"
encore build docker "$IMAGE_TAG" --arch amd64 --base ubuntu:22.04 --skip-config 2>/dev/null

# Extract binary from docker image
CONTAINER_ID=$(docker create --platform linux/amd64 "$IMAGE_TAG")
mkdir -p "$DIST"
docker cp "$CONTAINER_ID":/artifacts/0/build/encore_app_out "$DIST/bos3000"
docker cp "$CONTAINER_ID":/encore/meta "$DIST/encore-meta"
docker rm "$CONTAINER_ID" >/dev/null
docker rmi "$IMAGE_TAG" >/dev/null 2>&1 || true

chmod +x "$DIST/bos3000"
echo "  Binary: dist/bos3000 ($(du -h "$DIST/bos3000" | cut -f1))"

# ---- Step 4: Package deployment tarball ----
echo "  Packaging deployment bundle..."
rm -rf "$STAGE"
mkdir -p "$STAGE/bos3000"

# Binary
cp "$DIST/bos3000" "$STAGE/bos3000/bos3000"
cp "$DIST/encore-meta" "$STAGE/bos3000/encore-meta"

# Deploy scripts and config
cp -r "$ROOT/deploy/"* "$STAGE/bos3000/"
cp "$ROOT/scripts/reset-admin-password.sh" "$STAGE/bos3000/"
echo "$SEMVER" > "$STAGE/bos3000/.version"

BUNDLE="$DIST/bos3000-${VERSION}-linux-amd64.tar.gz"
tar czf "$BUNDLE" -C "$STAGE" bos3000
rm -rf "$STAGE"

echo ""
echo "=== Build Complete ==="
echo "  Version:  $VERSION"
echo "  Binary:   dist/bos3000 (linux/amd64)"
echo "  Bundle:   $BUNDLE ($(du -h "$BUNDLE" | cut -f1))"
echo ""
echo "  Deploy to server:"
echo "    scp $BUNDLE root@<server>:/tmp/"
echo "    ssh root@<server>"
echo "    tar xzf /tmp/bos3000-${VERSION}-linux-amd64.tar.gz -C /tmp"
echo "    sudo bash /tmp/bos3000/install.sh --eip <public-ip>"
