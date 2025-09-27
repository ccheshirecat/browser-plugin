GO ?= go
ROOT_DIR := $(shell pwd)
BUILD_DIR ?= $(ROOT_DIR)/build
BIN_DIR ?= $(BUILD_DIR)/bin
ARTIFACTS_DIR ?= $(BUILD_DIR)/artifacts
IMAGE_TAG ?= ghcr.io/volant-plugins/browser:dev
INITRAMFS_NAME ?= browser-initramfs.cpio.gz

.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-agent ## Build the browser agent binary

.PHONY: build-agent
build-agent: ## Compile the browser agent (linux/amd64)
	mkdir -p $(BIN_DIR)
	cd agent && GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/browser-agent ./cmd/browser-agent

.PHONY: test
test: ## Run unit tests
	cd agent && $(GO) test ./...

.PHONY: fmt
fmt: ## Format Go sources
	cd agent && $(GO) fmt ./...

.PHONY: lint
lint: ## Run go vet
	cd agent && $(GO) vet ./...

.PHONY: build-image
build-image: build-agent ## Build OCI image for the browser runtime
	docker build --build-arg AGENT_BINARY=build/bin/browser-agent -t $(IMAGE_TAG) runtime

.PHONY: build-initramfs
build-initramfs: build-image ## Produce initramfs archive and kernel snapshot
	mkdir -p $(ARTIFACTS_DIR)
	OUTPUT_DIR=$(ARTIFACTS_DIR) AGENT_BIN=$(BIN_DIR)/browser-agent IMAGE_TAG=$(IMAGE_TAG) INITRAMFS_NAME=$(INITRAMFS_NAME) \
		runtime/scripts/build-initramfs.sh

.PHONY: build-kernel
build-kernel: ## Fetch pinned kernel if needed
	runtime/scripts/build-kernel.sh $(ARTIFACTS_DIR)

.PHONY: smoke-test
smoke-test: build-agent ## Run smoke tests against local runtime
	IMAGE_TAG=$(IMAGE_TAG) AGENT_BINARY=build/bin/browser-agent tests/integration/smoke-test.sh

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
