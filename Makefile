GO ?= go
ROOT_DIR := $(shell pwd)
BUILD_DIR ?= $(ROOT_DIR)/build
BIN_DIR ?= $(BUILD_DIR)/bin
ARTIFACTS_DIR ?= $(BUILD_DIR)/artifacts
IMAGE_TAG ?= ghcr.io/volant-plugins/browser:dev
INITRAMFS_NAME ?= browser-initramfs.cpio.gz
MANIFEST_DIR ?= $(ROOT_DIR)/manifest

.PHONY: build
build: build-agent ## Build the browser agent binary

.PHONY: build-agent
build-agent: ## Compile the browser agent
	mkdir -p $(BIN_DIR)
	cd agent && $(GO) build -o $(BIN_DIR)/browser-agent ./cmd/browser-agent

.PHONY: test
test: ## Run unit tests
	cd agent && $(GO) test ./...

.PHONY: fmt
fmt: ## Format Go sources
	cd agent && $(GO) fmt ./...

.PHONY: lint
lint: ## Run go vet on the agent module
	cd agent && $(GO) vet ./...

.PHONY: build-image
build-image: build-agent ## Build OCI image for the browser runtime
	cd runtime && IMAGE_TAG=$(IMAGE_TAG) bash -c 'set -euo pipefail; cp ../build/bin/browser-agent browser-agent.bin; trap "rm -f browser-agent.bin" EXIT; docker build --build-arg AGENT_BINARY=browser-agent.bin -t "$$IMAGE_TAG" .'

.PHONY: build-initramfs
build-initramfs: build-image ## Produce initramfs archive and kernel
	mkdir -p $(ARTIFACTS_DIR)
	OUTPUT_DIR=$(ARTIFACTS_DIR) AGENT_BIN=$(BIN_DIR)/browser-agent IMAGE_TAG=$(IMAGE_TAG) INITRAMFS_NAME=$(INITRAMFS_NAME) \
		bash runtime/scripts/build-initramfs.sh

.PHONY: build-kernel
build-kernel: ## Fetch pinned kernel if needed
	bash runtime/scripts/build-kernel.sh $(ARTIFACTS_DIR)

.PHONY: smoke-test
smoke-test: ## Run smoke tests against local runtime
	bash runtime/scripts/smoke-test.sh

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
