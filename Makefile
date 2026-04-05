# Makefile — devforge-mcp
# Requires: CGO_ENABLED=1 (libsql / go-sqlite3)
# Usage: make help

# ── Variables ──────────────────────────────────────────────────────────────────
BINARY_MCP   := devforge-mcp
BINARY_TUI   := devforge
DIST_DIR     := dist
BIN_DIR      := bin
VERSION      := $(shell cat VERSION 2>/dev/null | tr -d '[:space:]' || echo "0.0.0")

# DB path used by the app at runtime (relative to project root).
# The canonical distribution DB lives inside dist/ so the binary and DB ship together.
DB_PATH      := $(DIST_DIR)/devforge.db

# Seeds directory
SEEDS_DIR    := db/seeds

# Ollama settings — override on the command line:
#   make db-embeddings OLLAMA_URL=http://myhost:11434 OLLAMA_MODEL=mxbai-embed-large
OLLAMA_URL   := http://localhost:11434
OLLAMA_MODEL := nomic-embed-text

# Install destination
INSTALL_DIR  := $(HOME)/.local/bin

# Go build flags
GO_BUILD     := CGO_ENABLED=1 go build
GO_RUN       := CGO_ENABLED=1 go run
GO_TEST      := CGO_ENABLED=1 go test

# Helper scripts (Go programs, run directly via go run — no pre-compilation needed)
INIT_RUNNER  := ./scripts/init_db_runner
SEED_RUNNER  := ./scripts/seed_runner

# Legacy seed script (kept for standalone / post-install use)
SEED_SCRIPT  := scripts/seed.sh

.DEFAULT_GOAL := help

# ── Phony targets ──────────────────────────────────────────────────────────────
.PHONY: build build-mcp build-tui install uninstall dist \
        db-init db-seed db-bootstrap db-embeddings seed \
        clean test run tui release-bundle render-homebrew-formula \
        install-dpf build-rust build-rust-static help

# ── Seed file list (sorted) ─────────────────────────────────────────────────
SEED_FILES := $(sort $(wildcard $(SEEDS_DIR)/*.sql))
RELEASE_OUT_DIR := $(DIST_DIR)/release
FORMULA_TEMPLATE := packaging/homebrew/Formula/devforge.rb
FORMULA_OUTPUT ?= $(RELEASE_OUT_DIR)/devforge.rb
CHECKSUMS_FILE ?= $(RELEASE_OUT_DIR)/checksums.txt

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

## dist: Build binaries + initialize and seed the distribution DB (no embeddings)
dist: build db-bootstrap
	@if [ -f $(BIN_DIR)/dpf ]; then \
		chmod +x $(BIN_DIR)/dpf; \
		cp $(BIN_DIR)/dpf $(DIST_DIR)/dpf; \
	fi
	@echo "Distribution package ready in $(DIST_DIR)/"
	@echo "  binary : $(DIST_DIR)/$(BINARY_MCP)"
	@echo "  database: $(DB_PATH)"

# ── Database ───────────────────────────────────────────────────────────────────

## db-init: Create/migrate the libSQL DB using the project's own db.Open() / RunMigrations()
##          Writes to DB_PATH (default: dist/devforge.db). Idempotent.
db-init:
	@echo "Initializing database at $(DB_PATH)..."
	@mkdir -p $(DIST_DIR)
	$(GO_RUN) $(INIT_RUNNER) -db "$(DB_PATH)"

## db-seed: Apply all db/seeds/*.sql files to DB_PATH in numeric order (idempotent)
db-seed:
	@if [ ! -f "$(DB_PATH)" ]; then \
		echo "Database not found at $(DB_PATH) — run 'make db-init' first"; \
		exit 1; \
	fi
	@echo "Seeding database at $(DB_PATH)..."
	$(GO_RUN) $(SEED_RUNNER) -db "$(DB_PATH)" $(foreach f,$(SEED_FILES),-sql "$(f)")
	@echo "Seed complete."

## db-bootstrap: Create/migrate and seed the DB without embeddings
db-bootstrap: db-init db-seed

## db-embeddings: Generate Ollama embeddings for rows with embedding IS NULL
db-embeddings:
	@bash $(SEED_SCRIPT) --embeddings-only \
		--db "$(DB_PATH)" \
		--ollama-url "$(OLLAMA_URL)" \
		--ollama-model "$(OLLAMA_MODEL)"

## seed: Full seed pipeline — db-init + db-seed + db-embeddings (idempotent)
seed: db-init db-seed db-embeddings

## release-bundle: Build the canonical Linux amd64 release bundle in dist/release/
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

## test: Run all tests with CGO_ENABLED=1 (required for libsql / FTS5)
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
	@echo "devforge-mcp — available make targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'
	@echo ""
	@echo "Variables (override on command line):"
	@echo "  OLLAMA_URL=$(OLLAMA_URL)"
	@echo "  OLLAMA_MODEL=$(OLLAMA_MODEL)"
	@echo "  DB_PATH=$(DB_PATH)"
	@echo "  INSTALL_DIR=$(INSTALL_DIR)"
