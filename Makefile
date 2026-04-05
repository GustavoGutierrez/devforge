# Makefile — devforge
# Usage: make help

# ── Variables ──────────────────────────────────────────────────────────────────
BINARY_MCP   := devforge-mcp
BINARY_TUI   := devforge
DIST_DIR     := dist
BIN_DIR      := bin
VERSION      := $(shell cat VERSION 2>/dev/null | tr -d '[:space:]' || echo "0.0.0")

# Install destination
INSTALL_DIR  := $(HOME)/.local/bin

# Go build flags
GO_BUILD     := go build
GO_TEST      := go test

RELEASE_OUT_DIR := $(DIST_DIR)/release
FORMULA_TEMPLATE := packaging/homebrew/Formula/devforge.rb
FORMULA_OUTPUT ?= $(RELEASE_OUT_DIR)/devforge.rb
CHECKSUMS_FILE ?= $(RELEASE_OUT_DIR)/checksums.txt

.DEFAULT_GOAL := help

# ── Phony targets ──────────────────────────────────────────────────────────────
.PHONY: build build-mcp build-tui install uninstall dist \
        clean test run tui release-bundle render-homebrew-formula \
        install-dpf build-rust build-rust-static help

# ── Build ──────────────────────────────────────────────────────────────────────

## build: Compile both binaries into ./dist/
build: build-mcp build-tui

## build-mcp: Compile the MCP server binary to ./dist/devforge-mcp
build-mcp:
	@mkdir -p $(DIST_DIR)
	$(GO_BUILD) -o $(DIST_DIR)/$(BINARY_MCP) ./cmd/devforge-mcp/
	@echo "Built $(DIST_DIR)/$(BINARY_MCP)"

## build-tui: Compile the CLI/TUI binary to ./dist/devforge
build-tui:
	@mkdir -p $(DIST_DIR)
	$(GO_BUILD) -o $(DIST_DIR)/$(BINARY_TUI) ./cmd/devforge/
	@echo "Built $(DIST_DIR)/$(BINARY_TUI)"

# ── Install ────────────────────────────────────────────────────────────────────

## install: Install to ~/.local/share/devforge/versions/$(VERSION)/ with symlinks in ~/.local/bin/
install:
	@bash scripts/install.sh

## uninstall: Remove all devforge binaries, data, and symlinks
uninstall:
	@bash scripts/uninstall.sh

# ── Distribution package ───────────────────────────────────────────────────────

## dist: Build binaries and copy dpf into ./dist/
dist: build
	@if [ -f $(BIN_DIR)/dpf ]; then \
		chmod +x $(BIN_DIR)/dpf; \
		cp $(BIN_DIR)/dpf $(DIST_DIR)/dpf; \
	fi
	@echo "Distribution package ready in $(DIST_DIR)/"
	@echo "  binary : $(DIST_DIR)/$(BINARY_MCP)"
	@echo "  binary : $(DIST_DIR)/$(BINARY_TUI)"

## release-bundle: Build the canonical release bundle for the current platform
release-bundle:
	@bash scripts/package_release_bundle.sh --version "$(VERSION)" --output-dir "$(RELEASE_OUT_DIR)"

## render-homebrew-formula: Render the release formula from packaging/homebrew/Formula/devforge.rb
render-homebrew-formula:
	@python3 scripts/render_homebrew_formula.py \
		--template "$(FORMULA_TEMPLATE)" \
		--version "$(VERSION)" \
		--checksums-file "$(CHECKSUMS_FILE)" \
		--output "$(FORMULA_OUTPUT)"

# ── Run ───────────────────────────────────────────────────────────────────────

## run: Build and run the MCP server (stdio transport)
run: build-mcp
	@echo "Starting MCP server (stdio transport)..."
	./$(DIST_DIR)/$(BINARY_MCP)

## tui: Build and run the CLI/TUI
tui: build-tui
	./$(DIST_DIR)/$(BINARY_TUI)

# ── Test ───────────────────────────────────────────────────────────────────────

## test: Run all tests
test:
	$(GO_TEST) ./...

# ── DevPixelForge binary (dpf) ─────────────────────────────────────────────────
# The Rust source lives in https://github.com/GustavoGutierrez/devpixelforge
# Use the provided script to download a pre-built release:

## install-dpf: Download the latest DevPixelForge release to bin/dpf
install-dpf:
	@bash scripts/install-dpf.sh

## install-dpf VERSION: Download a specific DevPixelForge release to bin/dpf
install-dpf-%:
	@bash scripts/install-dpf.sh $*

## build-rust: (Deprecated — clone https://github.com/GustavoGutierrez/devpixelforge and use its Makefile)
build-rust:
	@echo "The Rust source is no longer in this repository."
	@echo "Clone devpixelforge: git clone https://github.com/GustavoGutierrez/devpixelforge.git"
	@echo "Then run 'make build-rust' inside that directory."

## build-rust-static: (Deprecated — clone https://github.com/GustavoGutierrez/devpixelforge and use its Makefile)
build-rust-static:
	@echo "The Rust source is no longer in this repository."
	@echo "Clone devpixelforge: git clone https://github.com/GustavoGutierrez/devpixelforge.git"
	@echo "Then run 'make build-rust-static' inside that directory."

# ── Clean ──────────────────────────────────────────────────────────────────────

## clean: Remove dist/ and compiled binaries
clean:
	rm -rf $(DIST_DIR)
	@echo "Cleaned $(DIST_DIR)/"

# ── Help ───────────────────────────────────────────────────────────────────────

## help: Show this help message
help:
	@echo "devforge — available make targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'
	@echo ""
	@echo "Variables (override on command line):"
	@echo "  INSTALL_DIR=$(INSTALL_DIR)"
