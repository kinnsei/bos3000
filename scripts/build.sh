#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Build Script
# Produces both a binary tarball and a Docker image.
#
# Usage:
#   bash scripts/build.sh v1.0.0            # both binary + docker
#   bash scripts/build.sh v1.0.0 --binary   # binary only
#   bash scripts/build.sh v1.0.0 --docker   # docker only

VERSION="${1:-}"
if [[ -z "$VERSION" ]] || ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Usage: $0 <version> [--binary|--docker]"
  echo "  e.g. $0 v1.0.0"
  exit 1
fi

MODE="${2:-both}"  # both, --binary, --docker
BUILD_BINARY=true
BUILD_DOCKER=true
case "$MODE" in
  --binary) BUILD_DOCKER=false ;;
  --docker) BUILD_BINARY=false ;;
  both|"")  ;;
  *) echo "Unknown flag: $MODE"; exit 1 ;;
esac

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
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

# ---- Step 3: Compile via Encore ----
echo "  Compiling linux/amd64..."
if ! command -v encore &>/dev/null; then
  echo "  ERROR: encore CLI not found"
  echo "  Install: curl -L https://encore.dev/install.sh | bash"
  exit 1
fi

mkdir -p "$DIST"
IMAGE_TAG="bos3000:${VERSION}"

# Build docker image (needed for both modes — binary is extracted from it)
encore build docker "$IMAGE_TAG" --arch amd64 --base ubuntu:22.04 --skip-config

# ---- Step 4: Extract binary ----
if $BUILD_BINARY; then
  echo "  Extracting binary..."
  CONTAINER_ID=$(docker create --platform linux/amd64 "$IMAGE_TAG")
  docker cp "$CONTAINER_ID":/artifacts/0/build/encore_app_out "$DIST/bos3000"
  docker cp "$CONTAINER_ID":/encore/meta "$DIST/encore-meta"
  docker rm "$CONTAINER_ID" >/dev/null
  chmod +x "$DIST/bos3000"

  # Package deployment tarball
  STAGE="$DIST/stage"
  rm -rf "$STAGE"
  mkdir -p "$STAGE/bos3000"
  cp "$DIST/bos3000" "$STAGE/bos3000/bos3000"
  cp "$DIST/encore-meta" "$STAGE/bos3000/encore-meta"
  cp -r "$ROOT/deploy/"* "$STAGE/bos3000/"
  cp "$ROOT/scripts/reset-admin-password.sh" "$STAGE/bos3000/"
  echo "$SEMVER" > "$STAGE/bos3000/.version"

  BINARY_BUNDLE="$DIST/bos3000-${VERSION}-linux-amd64.tar.gz"
  tar czf "$BINARY_BUNDLE" -C "$STAGE" bos3000
  rm -rf "$STAGE"
  echo "  Binary bundle: $BINARY_BUNDLE ($(du -h "$BINARY_BUNDLE" | cut -f1))"
fi

# ---- Step 5: Save Docker image ----
if $BUILD_DOCKER; then
  DOCKER_BUNDLE="$DIST/bos3000-${VERSION}-docker-amd64.tar.gz"
  echo "  Saving Docker image..."
  docker save "$IMAGE_TAG" | gzip > "$DOCKER_BUNDLE"
  echo "  Docker bundle: $DOCKER_BUNDLE ($(du -h "$DOCKER_BUNDLE" | cut -f1))"
fi

# Clean up build image if only binary was needed
if ! $BUILD_DOCKER; then
  docker rmi "$IMAGE_TAG" >/dev/null 2>&1 || true
fi

# ---- Summary ----
echo ""
echo "=== Build Complete ==="
echo "  Version: $VERSION"
echo ""
if $BUILD_BINARY; then
  echo "  Binary tarball:  $BINARY_BUNDLE"
  echo "    Deploy: scp to server → tar xzf → sudo bash install.sh --eip <ip>"
fi
if $BUILD_DOCKER; then
  echo "  Docker image:    $IMAGE_TAG"
  echo "    Local:  docker run -p 4000:4000 $IMAGE_TAG"
  echo "    Export: $DOCKER_BUNDLE"
  echo "    Load:   docker load < $DOCKER_BUNDLE"
fi
echo ""
echo "  Release:"
echo "    gh release create $VERSION dist/bos3000-${VERSION}-*.tar.gz"
