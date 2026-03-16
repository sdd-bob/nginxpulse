#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs/fumadocs"
OUT_DIR="$DOCS_DIR/out"
SYNC_SCRIPT="$ROOT_DIR/scripts/sync_fumadocs_docs.sh"
DIST_DIR="$ROOT_DIR/dist"
LATEST_DIR="${LATEST_DIR:-$DIST_DIR/nginxpulse-docs}"
SKIP_SYNC="${SKIP_SYNC:-0}"
SKIP_PACKAGE="${SKIP_PACKAGE:-0}"

usage() {
  cat <<'EOF'
Usage: scripts/build_fumadocs.sh

Environment variables:
  SKIP_SYNC=1      Skip wiki -> fumadocs content sync.
  SKIP_PACKAGE=1   Skip tar.gz packaging, only build static files.

Output:
  Static site: docs/fumadocs/out
  Latest dir:  dist/nginxpulse-docs (override via LATEST_DIR)
  Package:     dist/nginxpulse-docs-<timestamp>.tar.gz
EOF
}

ensure_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd not found in PATH. Please install it and retry." >&2
    exit 1
  fi
}

ensure_node_deps() {
  local install_needed=0

  if [[ ! -d "$DOCS_DIR/node_modules" ]]; then
    install_needed=1
  elif [[ "$DOCS_DIR/package.json" -nt "$DOCS_DIR/node_modules" ]]; then
    install_needed=1
  elif [[ -f "$DOCS_DIR/pnpm-lock.yaml" && "$DOCS_DIR/pnpm-lock.yaml" -nt "$DOCS_DIR/node_modules" ]]; then
    install_needed=1
  fi

  if [[ "$install_needed" -eq 1 ]]; then
    echo "Installing Fumadocs dependencies..."
    if [[ -f "$DOCS_DIR/pnpm-lock.yaml" ]]; then
      (cd "$DOCS_DIR" && pnpm install --frozen-lockfile) || (cd "$DOCS_DIR" && pnpm install)
    else
      (cd "$DOCS_DIR" && pnpm install)
    fi
  fi
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi


if [[ ! -d "$DOCS_DIR" ]]; then
  echo "Fumadocs directory not found: $DOCS_DIR" >&2
  exit 1
fi

ensure_cmd node
ensure_cmd pnpm
ensure_cmd tar

if [[ "$SKIP_SYNC" != "1" && -f "$SYNC_SCRIPT" ]]; then
  echo "Syncing wiki docs..."
  bash "$SYNC_SCRIPT"
fi

ensure_node_deps

echo "Building static docs..."
(cd "$DOCS_DIR" && pnpm run build)

if [[ ! -d "$OUT_DIR" ]]; then
  echo "Build output not found: $OUT_DIR" >&2
  exit 1
fi

echo "Static docs generated at: $OUT_DIR"

mkdir -p "$DIST_DIR"
echo "Syncing latest static docs to: $LATEST_DIR"
rm -rf "$LATEST_DIR"
mkdir -p "$LATEST_DIR"
cp -a "$OUT_DIR"/. "$LATEST_DIR"/

if [[ "$SKIP_PACKAGE" == "1" ]]; then
  exit 0
fi

timestamp="$(date +%Y%m%d-%H%M%S)"
package_file="$DIST_DIR/nginxpulse-docs-$timestamp.tar.gz"

echo "Packing static docs..."
tar -C "$OUT_DIR" -czf "$package_file" .
echo "Package created: $package_file"
