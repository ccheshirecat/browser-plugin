#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/build"
BIN_DIR="$BUILD_DIR/bin"

IMAGE_TAG=${BROWSER_PLUGIN_IMAGE:-browser-plugin-smoke:local}
PORT=${BROWSER_PLUGIN_PORT:-18080}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required" >&2
  exit 1
fi

mkdir -p "$BIN_DIR"

echo "[smoke] building agent"
GOOS=linux GOARCH=amd64 go build -o "$BIN_DIR/browser-agent" ./agent/cmd/browser-agent

pushd "$ROOT_DIR" >/dev/null

export IMAGE_TAG
export BROWSER_PLUGIN_IMAGE="$IMAGE_TAG"
export BROWSER_PLUGIN_PORT="$PORT"
export AGENT_BINARY="$BIN_DIR/browser-agent"

echo "[smoke] building image"
docker compose -f tests/integration/docker-compose.yaml build

container_name="browser-plugin-smoke"
echo "[smoke] starting container"
docker compose -f tests/integration/docker-compose.yaml up -d

cleanup() {
  docker compose -f tests/integration/docker-compose.yaml down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[smoke] waiting for health"
for _ in {1..30}; do
  if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null; then
    break
  fi
  sleep 1
done

echo "[smoke] exercising navigate"
curl -sf -X POST "http://127.0.0.1:${PORT}/v1/browser/navigate" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}' >/dev/null

echo "[smoke] taking screenshot"
curl -sf -X POST "http://127.0.0.1:${PORT}/v1/browser/screenshot" \
  -H 'Content-Type: application/json' \
  -d '{"full_page":false,"format":"png"}' >/dev/null

echo "[smoke] success"

popd >/dev/null

