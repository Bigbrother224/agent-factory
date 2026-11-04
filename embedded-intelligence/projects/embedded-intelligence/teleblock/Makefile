# teleblock — build targets, including the Raspberry Pi image pipeline.
#
#   make build        # host binary in dist/
#   make build-pi     # static linux/arm64 binary for the Pi
#   make pi-bundle    # boot overlay for Raspberry Pi Imager
#   make verify       # the project arbiter

SHELL         := /bin/bash
.DEFAULT_GOAL := build

BINARY        ?= teleblock
GO            ?= go
DIST          ?= dist

# First main package of the module, so the Makefile does not hard-code the
# directory layout. `go list` prints an empty line for every non-main package,
# so those are dropped before taking the first match. Override it when needed:
#   make build-pi MAIN=./cmd/teleblock
MAIN          ?= $(shell $(GO) list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... 2>/dev/null | grep -v '^$$' | head -n 1)

# -s -w drops the symbol table; nothing debugs a binary on a Pi.
LDFLAGS       ?= -s -w
BUILD_FLAGS   ?= -trimpath -ldflags '$(LDFLAGS)'

# This Makefile's own directory, absolute, wherever make was invoked from.
PROJECT_DIR   := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# The arbiter sits at the repository root, above this project directory — and
# this directory ships a verify.sh shim, so starting the search at $(CURDIR)
# would resolve to that shim and exec it forever. Start above the project.
# Override when the arbiter lives elsewhere:
#   make verify VERIFY=/path/to/verify.sh
VERIFY        ?= $(shell p="$(PROJECT_DIR)"; while [ "$$p" != "/" ] && [ "$$p" != "." ]; do p=$$(dirname "$$p"); if [ -f "$$p/verify.sh" ]; then echo "$$p/verify.sh"; break; fi; done)

.PHONY: build build-pi cross-check pi-bundle test verify clean help

help: ## List the available targets
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build the host binary into dist/
	@mkdir -p $(DIST)
	@test -n "$(MAIN)" || { echo "build: no main package found, set MAIN=./path/to/cmd" >&2; exit 1; }
	$(GO) build $(BUILD_FLAGS) -o $(DIST)/$(BINARY) $(MAIN)

build-pi: ## Cross-compile a static linux/arm64 binary for the Pi
	@mkdir -p $(DIST)
	@test -n "$(MAIN)" || { echo "build-pi: no main package found, set MAIN=./path/to/cmd" >&2; exit 1; }
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-linux-arm64 $(MAIN)

cross-check: ## Compile every package for linux/arm64, without an artefact
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build ./...

pi-bundle: build-pi ## Assemble dist/pi-bootfs/ for Raspberry Pi Imager
	$(PROJECT_DIR)/deploy/pi/build-image.sh --out $(DIST)/pi-bootfs --binary $(DIST)/$(BINARY)-linux-arm64

test: ## Run the Go test suite
	$(GO) test ./...

verify: ## Run the project arbiter
	@test -n "$(VERIFY)" || { echo "verify: no verify.sh above $(PROJECT_DIR); set VERIFY=/path/to/verify.sh" >&2; exit 1; }
	"$(VERIFY)" teleblock

clean: ## Remove build artefacts
	rm -rf $(DIST)
