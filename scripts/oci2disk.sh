#!/usr/bin/env bash
set -euo pipefail

# Usage: ./oci2disk.sh <image_ref> <output.img> [size] [fstype]
# Example:
#   ./oci2disk.sh docker.io/library/nginx:latest nginx.img 2G ext4
#   ./oci2disk.sh docker.io/library/nginx:latest nginx.img   # auto sparse

IMAGE_REF="${1:-}"
OUT_IMG="${2:-rootfs.img}"
SIZE="${3:-}"          # optional
FSTYPE="${4:-ext4}"

WORKDIR=$(mktemp -d)
LAYOUT="$WORKDIR/layout"
ROOTFS="$WORKDIR/rootfs"

cleanup() {
    umount "$WORKDIR/mnt" 2>/dev/null || true
    losetup -d "$LOOP" 2>/dev/null || true
    rm -rf "$WORKDIR"
}
trap cleanup EXIT

if [ -z "$IMAGE_REF" ]; then
    echo "Usage: $0 <image_ref> <output.img> [size] [fstype]"
    exit 1
fi

echo "[*] Pulling image $IMAGE_REF to OCI layout..."
skopeo copy "docker://$IMAGE_REF" "oci:$LAYOUT:latest"

echo "[*] Unpacking to rootfs..."
umoci unpack --image "$LAYOUT:latest" "$ROOTFS"

mkdir -p "$WORKDIR/mnt"

if [ -n "$SIZE" ]; then
    echo "[*] Creating fixed-size image ($SIZE) using fallocate..."
    fallocate -l "$SIZE" "$OUT_IMG"
else
    ROOTFS_BYTES=$(du -sb "$ROOTFS/rootfs" | cut -f1)
    SPARSE_SIZE=$(( ROOTFS_BYTES + 500*1024*1024 ))   # +500MB headroom
    echo "[*] Creating sparse image (~$((SPARSE_SIZE/1024/1024)) MB) using truncate..."
    truncate -s $SPARSE_SIZE "$OUT_IMG"
fi

echo "[*] Formatting $FSTYPE filesystem..."
case "$FSTYPE" in
    ext4) mkfs.ext4 -F "$OUT_IMG" >/dev/null ;;
    xfs)  mkfs.xfs -f "$OUT_IMG" >/dev/null ;;
    btrfs) mkfs.btrfs -f "$OUT_IMG" >/dev/null ;;
    *) echo "Unsupported fstype: $FSTYPE" && exit 1 ;;
esac

LOOP=$(losetup -f --show "$OUT_IMG")
mount "$LOOP" "$WORKDIR/mnt"

echo "[*] Copying rootfs contents..."
rsync -aHAX "$ROOTFS/rootfs/" "$WORKDIR/mnt/"

echo "[*] Finalizing image..."
umount "$WORKDIR/mnt"
losetup -d "$LOOP"

echo "[+] Done. Image ready at $OUT_IMG"