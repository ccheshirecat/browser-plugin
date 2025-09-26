# Browser Automation Plugin

This repository houses the standalone Chromium runtime plugin for Volant. It packages the browser automation agent, runtime HTTP handlers, and build tooling necessary to publish a signed plugin manifest and OCI/initramfs artifacts that the Volant engine can install.

## Project Layout
```
README.md
manifest/
  browser.json
  browser.dev.json
agent/
  go.mod
  go.sum
  cmd/browser-agent/main.go
  internal/runtime/
    app/
      app.go
    browser/
      browser.go
      handlers.go
      log.go
      runtime.go
runtime/
  Dockerfile
  rootfs/
    README.md
  scripts/
    build-initramfs.sh
    build-kernel.sh
    smoke-test.sh
.github/workflows/
  build.yml
```

## Build & Test
- `make build` — compile the browser agent into `build/bin/browser-agent`
- `make test` — run unit tests for the agent runtime
- `make build-image` — build the Chromium runtime OCI image (requires Docker)
- `make build-initramfs` — bundle initramfs + kernel artifacts and checksums into `build/artifacts`
- `make smoke-test` — run integration smoke tests (requires `tests/integration` harness)