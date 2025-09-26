#!/usr/bin/env bash
set -euo pipefail

ARTIFACTS_DIR=${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../build/artifacts" && pwd)}
KERNEL_URL=${KERNEL_URL:-https://github.com/cloud-hypervisor/linux/releases/download/ch-release-v6.12.8-20250613/vmlinux-x86_64}

mkdir -p "$ARTIFACTS_DIR"

dest="$ARTIFACTS_DIR/vmlinux-x86_64"
if [ -f "$dest" ]; then
  echo "Kernel already present at $dest"
  exit 0
fi

echo "Downloading kernel to $dest"
curl -L "$KERNEL_URL" -o "$dest"
sha256sum "$dest" > "$dest.sha256"
