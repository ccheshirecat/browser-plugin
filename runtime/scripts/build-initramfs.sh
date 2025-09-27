#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOTFS_DIR="$REPO_ROOT/runtime/rootfs"

OUTPUT_DIR=${OUTPUT_DIR:-$REPO_ROOT/build/artifacts}
AGENT_BIN=${AGENT_BIN:-$REPO_ROOT/build/bin/browser-agent}
IMAGE_TAG=${IMAGE_TAG:-volant-browser-runtime:dev}
INITRAMFS_NAME=${INITRAMFS_NAME:-browser-initramfs.cpio.gz}
KERNEL_URL=${KERNEL_URL:-https://github.com/cloud-hypervisor/linux/releases/download/ch-release-v6.12.8-20250613/vmlinux-x86_64}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

if [ ! -f "$AGENT_BIN" ]; then
  echo "Agent binary not found at $AGENT_BIN" >&2
  exit 1
fi

STAGED_AGENT="$REPO_ROOT/runtime/browser-agent.bin"
TMPDIR=""
CID=""

cleanup_all() {
  rm -f "$STAGED_AGENT" 2>/dev/null || true
  if [ -n "$CID" ]; then
    docker rm -f "$CID" >/dev/null 2>&1 || true
  fi
  if [ -n "$TMPDIR" ] && [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup_all EXIT

cp "$AGENT_BIN" "$STAGED_AGENT"

printf 'Building image... ' >&2
if ! (cd "$REPO_ROOT/runtime" && docker build --build-arg AGENT_BINARY="$(basename "$STAGED_AGENT")" -t "$IMAGE_TAG" . >/dev/null); then
  echo 'failed' >&2
  exit 1
fi
rm -f "$STAGED_AGENT"
echo 'done' >&2

TMPDIR=$(mktemp -d)

CID=$(docker create "$IMAGE_TAG")

docker export "$CID" | tar -C "$TMPDIR" -xf -

pushd "$TMPDIR" >/dev/null
find . | cpio -o -H newc | gzip -9 > "$OUTPUT_DIR/$INITRAMFS_NAME"
popd >/dev/null

docker rm -f "$CID" >/dev/null
CID=""

KERNEL_DEST="$OUTPUT_DIR/vmlinux-x86_64"
if [ ! -f "$KERNEL_DEST" ]; then
  echo "Downloading kernel to $KERNEL_DEST" >&2
  curl -L "$KERNEL_URL" -o "$KERNEL_DEST"
fi

pushd "$OUTPUT_DIR" >/dev/null
sha256sum "$INITRAMFS_NAME" vmlinux-x86_64 > checksums.txt
popd >/dev/null

echo "Artifacts written to $OUTPUT_DIR"
