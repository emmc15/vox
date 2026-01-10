# Diaz - Speech-to-Text Application Makefile
# Builds self-contained binaries for multiple platforms

# Application names
APP_CLI := diaz-cli
APP_MCP := diaz-mcp
VERSION := 0.1.0

# Directories
BUILD_DIR := build
CMD_CLI_DIR := cmd/cli
CMD_MCP_DIR := cmd/mcp
INTERNAL_DIR := internal
MODELS_DIR := models

# Build info
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.GitBranch=$(GIT_BRANCH)

# CGO flags for static linking
CGO_ENABLED := 1
CGO_CFLAGS := -I/usr/local/include
CGO_LDFLAGS := -L/usr/local/lib -lvosk
CGO_LDFLAGS_LINUX := -static
CGO_LDFLAGS_DARWIN :=
CGO_LDFLAGS_WINDOWS := -static

# Platform-specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Binary names
BINARY_CLI := $(BUILD_DIR)/$(APP_CLI)
BINARY_MCP := $(BUILD_DIR)/$(APP_MCP)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

.PHONY: all build build-cli build-mcp clean test fmt format vet lint deps help install run-cli run-mcp dev-cli dev-mcp

## Default target
all: clean deps build

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)Diaz - Speech-to-Text Application$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/##/$(COLOR_GREEN)/' | column -t -s ':' | sed 's/$$/$(COLOR_RESET)/'

## build: Build both CLI and MCP binaries for current platform
build: deps build-cli build-mcp

## build-cli: Build CLI binary
build-cli: deps
	@echo "$(COLOR_BLUE)Building $(APP_CLI) for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_CLI) ./$(CMD_CLI_DIR)
	@echo "$(COLOR_GREEN)Build complete: $(BINARY_CLI)$(COLOR_RESET)"

## build-mcp: Build MCP binary
build-mcp: deps
	@echo "$(COLOR_BLUE)Building $(APP_MCP) for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_MCP) ./$(CMD_MCP_DIR)
	@echo "$(COLOR_GREEN)Build complete: $(BINARY_MCP)$(COLOR_RESET)"

## install: Install binaries to system
install: build
	@echo "$(COLOR_BLUE)Installing binaries...$(COLOR_RESET)"
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 $(BINARY_CLI) $(DESTDIR)/usr/local/bin/
	install -m 755 $(BINARY_MCP) $(DESTDIR)/usr/local/bin/
	@echo "$(COLOR_GREEN)Installed to /usr/local/bin/$(COLOR_RESET)"

## uninstall: Uninstall binaries from system
uninstall:
	@echo "$(COLOR_BLUE)Uninstalling binaries...$(COLOR_RESET)"
	rm -f /usr/local/bin/$(APP_CLI)
	rm -f /usr/local/bin/$(APP_MCP)
	@echo "$(COLOR_GREEN)Uninstalled$(COLOR_RESET)"

## run-cli: Build and run the CLI application
run-cli: build-cli
	@echo "$(COLOR_BLUE)Running $(APP_CLI)...$(COLOR_RESET)"
	./$(BINARY_CLI)

## run-mcp: Build and run the MCP server
run-mcp: build-mcp
	@echo "$(COLOR_BLUE)Running $(APP_MCP)...$(COLOR_RESET)"
	./$(BINARY_MCP)

## dev-cli: Run CLI in development mode (with race detector)
dev-cli:
	@echo "$(COLOR_BLUE)Running CLI in development mode...$(COLOR_RESET)"
	go run -race ./$(CMD_CLI_DIR)

## dev-mcp: Run MCP in development mode (with race detector)
dev-mcp:
	@echo "$(COLOR_BLUE)Running MCP in development mode...$(COLOR_RESET)"
	go run -race ./$(CMD_MCP_DIR)

## test: Run all tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)Tests complete$(COLOR_RESET)"

## test-coverage: Run tests and show coverage
test-coverage: test
	@echo "$(COLOR_BLUE)Generating coverage report...$(COLOR_RESET)"
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)Coverage report: coverage.html$(COLOR_RESET)"

## bench: Run benchmarks
bench:
	@echo "$(COLOR_BLUE)Running benchmarks...$(COLOR_RESET)"
	go test -bench=. -benchmem ./...

## fmt: Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	go fmt ./...
	@echo "$(COLOR_GREEN)Code formatted$(COLOR_RESET)"

## format: Alias for fmt
format: fmt

## vet: Run go vet
vet:
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	go vet ./...
	@echo "$(COLOR_GREEN)Vet complete$(COLOR_RESET)"

## lint: Run linters (requires golangci-lint)
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(COLOR_GREEN)Lint complete$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint not installed. Skipping...$(COLOR_RESET)"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## deps: Download and tidy dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	go mod download
	go mod tidy
	@echo "$(COLOR_GREEN)Dependencies ready$(COLOR_RESET)"

## deps-update: Update all dependencies
deps-update:
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	go get -u ./...
	go mod tidy
	@echo "$(COLOR_GREEN)Dependencies updated$(COLOR_RESET)"

## clean: Remove build artifacts
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "$(COLOR_GREEN)Clean complete$(COLOR_RESET)"

## clean-all: Remove build artifacts and downloaded models
clean-all: clean
	@echo "$(COLOR_BLUE)Cleaning models...$(COLOR_RESET)"
	rm -rf $(MODELS_DIR)/*.zip
	@echo "$(COLOR_GREEN)Full clean complete$(COLOR_RESET)"

## models-download: Download speech recognition models
models-download:
	@echo "$(COLOR_BLUE)Downloading models...$(COLOR_RESET)"
	@mkdir -p $(MODELS_DIR)
	@echo "$(COLOR_YELLOW)Model download to be implemented$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Manual download: https://alphacephei.com/vosk/models$(COLOR_RESET)"

## size: Show binary sizes
size:
	@echo "$(COLOR_BOLD)Binary sizes:$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/ 2>/dev/null || echo "No binaries found. Run 'make build' first."

## info: Show build information
info:
	@echo "$(COLOR_BOLD)Build Information:$(COLOR_RESET)"
	@echo "  CLI Binary:  $(APP_CLI)"
	@echo "  MCP Binary:  $(APP_MCP)"
	@echo "  Version:     $(VERSION)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Git Branch:  $(GIT_BRANCH)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  GOOS:        $(GOOS)"
	@echo "  GOARCH:      $(GOARCH)"
	@echo "  CGO Enabled: $(CGO_ENABLED)"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(COLOR_GREEN)All checks passed!$(COLOR_RESET)"

## release: Build release binaries
release: clean check build
	@echo "$(COLOR_GREEN)Release build complete!$(COLOR_RESET)"
	@$(MAKE) size

## quick: Quick build without dependencies check
quick:
	@echo "$(COLOR_BLUE)Quick build...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_CLI) ./$(CMD_CLI_DIR)
	CGO_ENABLED=$(CGO_ENABLED) \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_MCP) ./$(CMD_MCP_DIR)
	@echo "$(COLOR_GREEN)Quick build complete$(COLOR_RESET)"

## install-vosk: Install Vosk library (Linux x86_64 only)
install-vosk:
	@echo "$(COLOR_BLUE)Installing Vosk library...$(COLOR_RESET)"
	@if [ ! -f scripts/install-vosk-lib.sh ]; then \
		echo "$(COLOR_YELLOW)Error: scripts/install-vosk-lib.sh not found$(COLOR_RESET)"; \
		exit 1; \
	fi
	@bash scripts/install-vosk-lib.sh
	@echo "$(COLOR_GREEN)Vosk installation complete$(COLOR_RESET)"

## check-vosk: Check if Vosk library is installed
check-vosk:
	@echo "$(COLOR_BLUE)Checking for Vosk library...$(COLOR_RESET)"
	@if [ -f /usr/local/lib/libvosk.so ] && [ -f /usr/local/include/vosk_api.h ]; then \
		echo "$(COLOR_GREEN)✓ Vosk library found$(COLOR_RESET)"; \
		echo "  Library: /usr/local/lib/libvosk.so"; \
		echo "  Header: /usr/local/include/vosk_api.h"; \
	else \
		echo "$(COLOR_YELLOW)✗ Vosk library not found$(COLOR_RESET)"; \
		echo "  Run 'make install-vosk' to install it"; \
		exit 1; \
	fi

.DEFAULT_GOAL := help
