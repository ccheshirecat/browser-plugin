GO ?= go
ROOT_DIR := $(shell pwd)
BUILD_DIR ?= $(ROOT_DIR)/build
ARTIFACTS_DIR ?= $(BUILD_DIR)/build
SCRIPTS_DIR ?= $(ROOT_DIR)/scripts
IMAGE_TAG ?= docker.io/chromedp/headless-shell:latest
IMAGE_DIGEST ?= sha256:8a59f11326194bd44e7ae86041e33aa22603291c329b02e0c8031c2d68574cc0


.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests
	cd agent && $(GO) test ./...

.PHONY: fmt
fmt: ## Format Go sources
	cd agent && $(GO) fmt ./...

.PHONY: lint
lint: ## Run go vet
	cd agent && $(GO) vet ./...

.PHONY: build
build-image:  ## Build OCI image for the browser runtime
	$(SCRIPTS_DIR)/oci2disk.sh $(IMAGE_TAG) $(ARTIFACTS_DIR)/rootfs.img
	sed -i 's|"image": "",|"image": "$(IMAGE_TAG)",|' manifest/browser.json
	sed -i 's|"image_digest": "",|"image_digest": "$(IMAGE_DIGEST)",|' manifest/browser.json
	sed -i '/"rootfs":/,/"}/{ s|"url": "",|"url": "$(ARTIFACTS_DIR)/rootfs.img",|; }' manifest/browser.json
	sed -i '/"rootfs":/,/"}/{ s|"checksum": "",|"checksum": "$(IMAGE_DIGEST)",|; }' manifest/browser.json
	echo "Build complete, run install-plugin to install the plugin with the built manifest"


.PHONY: install-plugin
install-plugin: ## Fetch pinned kernel if needed
	volar plugins install --manifest ./manifest/browser.json
	echo "Plugin installed, run test-plugin to test the plugin"

.PHONY: test-plugin	
test-plugin:  ## Should already have CLI binary installed
	volar plugins install --manifest ./manifest/browser.json
	volar plugins enable browser
	volar plugins test browser
	echo "Plugin tested"


.PHONY: uninstall-plugin
uninstall-plugin: ## Should already have CLI binary installed
	volar plugins remove browser
	echo "Plugin removed"

.PHONY: disable-plugin
disable-plugin: ## Should already have CLI binary installed
	volar plugins disable browser
	echo "Plugin disabled"


.PHONY: enable-plugin
enable-plugin: ## Should already have CLI binary installed
	volar plugins enable browser	
	echo "Plugin enabled"

.PHONY: show-plugin
show-plugin: ## Should already have CLI binary installed
	volar plugins show browser
	echo "Plugin shown"


.PHONY: list-plugins
list-plugins: ## Should already have CLI binary installed
	volar plugins list
	echo "Plugins listed"


.PHONY: status-plugin
status-plugin: ## Should already have CLI binary installed
	volar plugins status browser
	echo "Plugin status shown"

.PHONY: test-e2e-with-vm #Includes cleanup
spawn-vm: ## Should already have CLI binary installed
	volar vms create test-vm --plugin browser --cpu 2 --memory 2048
	volar vms navigate test-vm browser --url https://example.com
	volar vms screenshot test-vm --output screenshot.png
	volar vms delete test-vm
	volar vms list
	echo "VM spawned"	


.PHONY: delete-vm
delete-vm: ## Should already have CLI binary installed
	volar vms delete test-vm
	echo "VM deleted"

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
