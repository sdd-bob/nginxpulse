#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs/fumadocs"
DOCS_PORT="${DOCS_PORT:-4173}"
DOCS_HOST="${DOCS_HOST:-0.0.0.0}"
SYNC_SCRIPT="$ROOT_DIR/scripts/sync_fumadocs_docs.sh"

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

if [[ ! -d "$DOCS_DIR" ]]; then
  echo "Fumadocs directory not found: $DOCS_DIR" >&2
  exit 1
fi

ensure_cmd node
ensure_cmd pnpm
if [[ -f "$SYNC_SCRIPT" ]]; then
  bash "$SYNC_SCRIPT"
fi
ensure_node_deps

echo "Starting Fumadocs on http://localhost:${DOCS_PORT}"
echo "LAN access: http://<your-ip>:${DOCS_PORT}"
cd "$DOCS_DIR"
exec pnpm run dev -- --hostname "$DOCS_HOST" --port "$DOCS_PORT"
