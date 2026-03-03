#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/verify_docker_latest.sh -v <version> [-r <repo>] [-l <latest_tag>]

Options:
  -v, --version     Version tag to compare with latest (e.g. v1.6.13)
  -r, --repo        Docker Hub repo (default: magiccoders/nginxpulse)
  -l, --latest-tag  Latest tag name to compare (default: latest)
  -h, --help        Show help

Exit code:
  0  latest digest equals version digest
  2  latest digest does not equal version digest
  1  other errors
EOF
}

REPO="magiccoders/nginxpulse"
VERSION_TAG=""
LATEST_TAG="latest"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -v|--version)
      VERSION_TAG="${2:-}"
      shift 2
      ;;
    -r|--repo)
      REPO="${2:-}"
      shift 2
      ;;
    -l|--latest-tag)
      LATEST_TAG="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$VERSION_TAG" ]]; then
  echo "Missing version tag. Use -v <version>." >&2
  usage >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker CLI not found." >&2
  exit 1
fi

get_digest() {
  local image_ref="$1"
  local output
  if ! output="$(docker buildx imagetools inspect "$image_ref" 2>&1)"; then
    echo "Failed to inspect $image_ref" >&2
    echo "$output" >&2
    exit 1
  fi

  local digest
  digest="$(printf '%s\n' "$output" | awk '/^Digest: / {print $2; exit}')"
  if [[ -z "$digest" ]]; then
    echo "Failed to parse digest for $image_ref" >&2
    echo "$output" >&2
    exit 1
  fi
  printf '%s' "$digest"
}

LATEST_REF="${REPO}:${LATEST_TAG}"
VERSION_REF="${REPO}:${VERSION_TAG}"

echo "Inspecting $LATEST_REF ..."
latest_digest="$(get_digest "$LATEST_REF")"
echo "Inspecting $VERSION_REF ..."
version_digest="$(get_digest "$VERSION_REF")"

echo "latest  digest: $latest_digest"
echo "version digest: $version_digest"

if [[ "$latest_digest" == "$version_digest" ]]; then
  echo "OK: ${LATEST_REF} points to ${VERSION_REF}"
  exit 0
fi

echo "NOT OK: ${LATEST_REF} does not point to ${VERSION_REF}" >&2
exit 2
